package controller

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"

	fabricv1alpha1 "github.com/charchess/vixens/apps/60-services/txo-fabric/operator/api/v1alpha1"
	"github.com/charchess/vixens/apps/60-services/txo-fabric/operator/internal/authentik"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

// trustedHumanOIDCProof MUST be constructed only after a real server-side
// signed Authentik ID-token verification and a single-use OAuth nonce/state
// check. It is deliberately private to Fabric, never a CRD, browser JSON or
// an untrusted HTTP header. Future BFF→Fabric transport MUST be authenticated
// and authoritatively bind its tenant; that transport is NOT implemented here.
type trustedHumanOIDCProof struct {
	globalIssuer  string
	globalSubject string
	tenantIssuer  string
	tenantSubject string
	authentikUUID string
}

type authoritativeHumanAccountReader interface {
	SnapshotUserAccount(context.Context, string) (authentik.UserAccountSnapshot, error)
}

type authoritativeVerifiedHumanEnrollmentSource interface {
	authoritativeHumanAccountReader
	authoritativeIAMGroupReader
}

func allocateOpaqueFabricUserID() (string, error) {
	var random [16]byte
	if _, err := rand.Read(random[:]); err != nil {
		return "", errors.New("cannot allocate Fabric user identity")
	}
	return "usr" + hex.EncodeToString(random[:]), nil
}

func decodeRetainedIdentityBindings(raw string) ([]authentik.IdentityBinding, error) {
	if len(raw) == 0 || len(raw) > maxIdentityBindingJSONBytes {
		return nil, errors.New("missing or oversized retained Fabric identity ledger")
	}
	var bindings []authentik.IdentityBinding
	dec := json.NewDecoder(strings.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&bindings); err != nil || bindings == nil {
		return nil, errors.New("malformed retained Fabric identity ledger")
	}
	var extra any
	if dec.Decode(&extra) != io.EOF {
		return nil, errors.New("trailing data in Fabric identity ledger")
	}
	return bindings, nil
}

// verifiedEnrollmentGroupSnapshots accepts a verified UUID only when it is a
// current member of at least one exact tenant-declared IAM group. Group names
// and immutable UUID pins come solely from the tenant-owned, UID-bound ledger;
// no browser value, static human identity, or CRD allowlist selects a group.
func (r *TenantBundleReconciler) verifiedEnrollmentGroupSnapshots(
	ctx context.Context, bundle *fabricv1alpha1.TenantBundle, uuid string, source authoritativeVerifiedHumanEnrollmentSource,
) (map[string]authentik.GroupMembership, error) {
	if source == nil || uuid == "" {
		return nil, errors.New("verified Fabric enrollment group source unavailable")
	}
	pins, err := r.readOpenFGAIAMGroupPins(ctx, bundle)
	if err != nil {
		return nil, err
	}
	names, err := checkedIAMGroupNames(bundle)
	if err != nil || len(pins) != len(names) {
		return nil, errors.New("verified Fabric enrollment group pins unavailable")
	}
	snapshots := make(map[string]authentik.GroupMembership, len(names))
	member := false
	for _, name := range names {
		snapshot, err := source.SnapshotGroupMembership(ctx, name)
		if err != nil || snapshot.Name != name || snapshot.ActiveAuthentikUserUUIDs == nil ||
			pins[name] == "" || pins[name] != snapshot.AuthentikGroupUUID {
			return nil, errors.New("verified Fabric enrollment group changed or unavailable")
		}
		for _, candidate := range snapshot.ActiveAuthentikUserUUIDs {
			if candidate == uuid {
				member = true
			}
		}
		snapshots[name] = snapshot
	}
	if !member {
		return nil, errors.New("verified active Authentik account is outside tenant IAM groups")
	}
	return snapshots, nil
}

func enrollmentGroupSnapshotsUnchanged(
	ctx context.Context, snapshots map[string]authentik.GroupMembership, source authoritativeVerifiedHumanEnrollmentSource,
) error {
	for name, previous := range snapshots {
		current, err := source.SnapshotGroupMembership(ctx, name)
		if err != nil || !reflect.DeepEqual(current, previous) {
			return errors.New("verified Fabric enrollment group changed during enrollment")
		}
	}
	return nil
}

// enrollHumanFromVerifiedIdentity is a STAGED privileged write primitive;
// it is not wired into an HTTP endpoint or the TenantBundle reconcile loop.
//
// The signed Authentik claim provides (issuer, opaque sub, immutable UUID).
// Fabric proves the UUID is still an ACTIVE Authentik user before accepting
// this association, and rechecks after writing. It does not infer membership,
// issue a chat grant, create an Authentik account, or change the OIDC sub_mode.
//
// The retained store+model+tenant-UID association is verified against OpenFGA
// before every write, and the ConfigMap Update is resourceVersion/CAS guarded.
// Concurrent updates return Conflict, never blind overwrite: the trusted
// enrollment caller must retry from a fresh read and revalidate its proof.
// User IDs are generated from cryptographic randomness, not from login, sub,
// UUID, workspace paths or browser values. An existing UUID/subject cannot be
// silently rebound (account replacement requires a reviewed migration).
func (r *TenantBundleReconciler) enrollHumanFromVerifiedIdentity(
	ctx context.Context, bundle *fabricv1alpha1.TenantBundle,
	proof trustedHumanOIDCProof, source authoritativeVerifiedHumanEnrollmentSource,
) (string, error) {
	if source == nil || bundle == nil || !bundle.DeletionTimestamp.IsZero() ||
		bundle.Spec.HumanAccess == nil || bundle.Spec.HumanAccess.Web == nil {
		return "", errors.New("trusted Fabric human enrollment dependencies unavailable")
	}
	if err := validateTenantIAM(bundle); err != nil {
		return "", err
	}
	tenantIssuer := bundle.Spec.HumanAccess.Web.OIDC.Issuer
	if proof.globalIssuer == "" || proof.globalSubject == "" || proof.authentikUUID == "" ||
		proof.tenantIssuer != tenantIssuer || proof.tenantSubject == "" {
		return "", errors.New("verified global and tenant OIDC proofs are incomplete or tenant-mismatched")
	}
	// The global proof selects a reviewed intent only. The tenant tuple comes
	// exclusively from the separately verified tenant issuer proof.
	claim := authentik.IdentityBinding{
		TenantID: bundle.Spec.TenantID, Issuer: proof.tenantIssuer,
		OIDCSubject: proof.tenantSubject, AuthentikUserUUID: proof.authentikUUID,
		FabricUserID: "usr-validation-only",
	}
	if _, err := authentik.NewIdentityRegistry(bundle.Name, bundle.Spec.TenantID, tenantIssuer,
		bundle.Spec.HumanAccess.Web.IAMGroups, []authentik.IdentityBinding{claim}); err != nil {
		return "", errors.New("invalid reviewed tenant identity mapping")
	}

	account, err := source.SnapshotUserAccount(ctx, proof.authentikUUID)
	if err != nil {
		return "", fmt.Errorf("trusted Authentik account unavailable: %w", err)
	}
	if account.UUID != proof.authentikUUID || !account.Active {
		return "", errors.New("AuthentiK account inactive or inconsistent")
	}
	groupSnapshots, err := r.verifiedEnrollmentGroupSnapshots(ctx, bundle, proof.authentikUUID, source)
	if err != nil {
		return "", err
	}

	// Validate the retained tenantUID/store/model against its *live* model
	// before accessing an identity ledger; source-only ledger presence is
	// never evidence that a human is authorized to chat.
	if _, err := r.readOpenFGAHumanIdentityRegistry(ctx, bundle); err != nil {
		return "", fmt.Errorf("Fabric human enrollment ledger untrusted: %w", err)
	}
	approved, err := r.readOpenFGAAuthorizationBinding(ctx, bundle)
	if err != nil {
		return "", err
	}
	name, err := openFGABindingName(bundle.Spec.TenantID)
	if err != nil {
		return "", err
	}
	var cm corev1.ConfigMap
	if err := r.Get(ctx, types.NamespacedName{Namespace: openFGAPlatformNamespace, Name: name}, &cm); err != nil {
		return "", err
	}
	if cm.Data["tenantID"] != bundle.Spec.TenantID ||
		cm.Data["tenantName"] != bundle.Name ||
		cm.Data["tenantUID"] != string(bundle.UID) ||
		cm.Data["storeID"] != approved.Store.ID ||
		cm.Data["modelID"] != approved.ModelID ||
		cm.Data["modelFingerprint"] != approved.ModelFingerprint ||
		cm.Data[identityIssuerKey] != tenantIssuer ||
		cm.Data[identitySchemaKey] != "v1" {
		return "", errors.New("retained Fabric identity ledger changed ownership before enrollment")
	}
	existing, err := decodeRetainedIdentityBindings(cm.Data[identityBindingsKey])
	if err != nil {
		return "", err
	}
	if _, err := authentik.NewIdentityRegistry(bundle.Name, bundle.Spec.TenantID, tenantIssuer,
		bundle.Spec.HumanAccess.Web.IAMGroups, existing); err != nil {
		return "", errors.New("retained Fabric identity mappings inconsistent")
	}
	for _, previous := range existing {
		if previous.OIDCSubject == claim.OIDCSubject || previous.AuthentikUserUUID == proof.authentikUUID {
			if previous.OIDCSubject != claim.OIDCSubject || previous.AuthentikUserUUID != proof.authentikUUID {
				return "", errors.New("identity replacement requires explicit Fabric migration")
			}
			// Idempotent: a repeated trusted login does not modify the ledger
			// or rotate a stable Fabric user ID, but must still observe a fresh
			// active account and stable declared-group snapshot.
			after, err := source.SnapshotUserAccount(ctx, proof.authentikUUID)
			if err != nil || after.UUID != proof.authentikUUID || !after.Active ||
				enrollmentGroupSnapshotsUnchanged(ctx, groupSnapshots, source) != nil {
				return "", errors.New("Authentik account or group changed during enrollment")
			}
			return previous.FabricUserID, nil
		}
	}
	if len(existing) >= 10000 {
		return "", errors.New("Fabric human identity ledger is full")
	}
	seenIDs := map[string]bool{}
	for _, old := range existing {
		seenIDs[old.FabricUserID] = true
	}
	var allocated string
	for attempt := 0; attempt < 8; attempt++ {
		allocated, err = allocateOpaqueFabricUserID()
		if err != nil {
			return "", err
		}
		if !seenIDs[allocated] {
			break
		}
		allocated = ""
	}
	if allocated == "" {
		return "", errors.New("Fabric user ID allocation collision")
	}
	claim.FabricUserID = allocated
	next := append(existing, claim)
	if _, err := authentik.NewIdentityRegistry(bundle.Name, bundle.Spec.TenantID, tenantIssuer,
		bundle.Spec.HumanAccess.Web.IAMGroups, next); err != nil {
		return "", errors.New("new Fabric identity binding invalid")
	}
	raw, err := json.Marshal(next)
	if err != nil || len(raw) > maxIdentityBindingJSONBytes {
		return "", errors.New("Fabric human identity ledger capacity exceeded")
	}
	// Kubernetes performs optimistic concurrency control via resourceVersion.
	cm.Data[identityBindingsKey] = string(raw)
	if err := r.Update(ctx, &cm); err != nil {
		return "", fmt.Errorf("Fabric human identity enrollment CAS write failed: %w", err)
	}
	fresh, err := r.readOpenFGAHumanIdentityRegistry(ctx, bundle)
	if err != nil {
		return "", fmt.Errorf("Fabric human identity enrollment unconfirmed: %w", err)
	}
	confirmed, err := fresh.ResolveOIDCSubject(tenantIssuer, claim.OIDCSubject)
	if err != nil || confirmed != allocated {
		return "", errors.New("Fabric human enrollment binding changed after write")
	}
	after, err := source.SnapshotUserAccount(ctx, proof.authentikUUID)
	if err != nil || after.UUID != proof.authentikUUID || !after.Active ||
		enrollmentGroupSnapshotsUnchanged(ctx, groupSnapshots, source) != nil {
		// A persisted association alone is never a grant. The future BFF
		// must always recheck live active status + fresh FGA authorization.
		return "", errors.New("Authentik user disabled or group changed during enrollment")
	}
	return allocated, nil
}
