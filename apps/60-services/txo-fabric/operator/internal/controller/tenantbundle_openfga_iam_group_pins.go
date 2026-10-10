package controller

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"

	fabricv1alpha1 "github.com/charchess/vixens/apps/60-services/txo-fabric/operator/api/v1alpha1"
	"github.com/charchess/vixens/apps/60-services/txo-fabric/operator/internal/authentik"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

// Group UUIDs are platform-owned immutable identities. A name is not enough
// to adopt a deleted/recreated Authentik group, even if Fabric generated it.
const (
	iamGroupPinsSchemaKey = "iamGroupPinsSchema"
	iamGroupPinsOwnerUIDKey = "iamGroupPinsOwnerUID"
	iamGroupPinsDataKey   = "iamGroupPins"
	iamGroupPinsSchemaV1  = "v1"
	maxIAMGroupPinsBytes  = 65536
	authentikTokenSecret  = "txo-authentik-runtime"
)

var immutableAuthentikUUID = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[1-8][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

type iamGroupPin struct {
	Name              string `json:"name"`
	AuthentikGroupUUID string `json:"authentikGroupUUID"`
}

func (r *TenantBundleReconciler) authentikMembershipSource(ctx context.Context) (authoritativeIAMGroupReader, error) {
	if r.AuthentikMembershipClient != nil {
		return r.AuthentikMembershipClient, nil
	}
	// The credential MUST originate from a separately reviewed OpenBao
	// ExternalSecret in the platform namespace; never from TenantBundle or
	// a browser. It is not provisioned by this feature slice.
	var secret corev1.Secret
	if err := r.Get(ctx, types.NamespacedName{
		Namespace: openFGAPlatformNamespace, Name: authentikTokenSecret,
	}, &secret); err != nil {
		return nil, errors.New("Fabric Authentik membership credential unavailable")
	}
	source, err := authentik.NewClient(string(secret.Data["token"]), nil)
	if err != nil {
		return nil, errors.New("Fabric Authentik membership credential invalid")
	}
	return source, nil
}

func checkedIAMGroupNames(bundle *fabricv1alpha1.TenantBundle) ([]string, error) {
	if bundle == nil || !bundle.DeletionTimestamp.IsZero() ||
		bundle.Spec.HumanAccess == nil || bundle.Spec.HumanAccess.Web == nil {
		return nil, errors.New("no active tenant-owned human IAM group scope")
	}
	if err := validateTenantIAM(bundle); err != nil { return nil, err }
	names := make([]string, 0, len(bundle.Spec.HumanAccess.Web.IAMGroups))
	for _, key := range bundle.Spec.HumanAccess.Web.IAMGroups {
		names = append(names, authentikGroupName(bundle.Name, key))
	}
	sort.Strings(names)
	return names, nil
}

func decodeIAMGroupPins(raw string) (map[string]string, error) {
	if len(raw) == 0 || len(raw) > maxIAMGroupPinsBytes {
		return nil, errors.New("missing or oversized retained Fabric IAM group pins")
	}
	var pins []iamGroupPin
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&pins); err != nil || pins == nil {
		return nil, errors.New("invalid Fabric IAM group pin ledger")
	}
	var extra any
	if decoder.Decode(&extra) != io.EOF {
		return nil, errors.New("trailing data in Fabric IAM group pins")
	}
	result := make(map[string]string, len(pins))
	seenUUID := map[string]bool{}
	for _, pin := range pins {
		if pin.Name == "" || !immutableAuthentikUUID.MatchString(pin.AuthentikGroupUUID) ||
			seenUUID[pin.AuthentikGroupUUID] {
			return nil, errors.New("invalid or duplicated Fabric Authentik group pin")
		}
		if _, duplicated := result[pin.Name]; duplicated {
			return nil, errors.New("duplicated Fabric IAM group name pin")
		}
		seenUUID[pin.AuthentikGroupUUID] = true
		result[pin.Name] = pin.AuthentikGroupUUID
	}
	return result, nil
}

func encodeIAMGroupPins(pins map[string]string) (string, error) {
	keys := make([]string, 0, len(pins))
	for key := range pins { keys = append(keys, key) }
	sort.Strings(keys)
	sorted := make([]iamGroupPin, 0, len(keys))
	for _, key := range keys {
		sorted = append(sorted, iamGroupPin{Name: key, AuthentikGroupUUID: pins[key]})
	}
	raw, err := json.Marshal(sorted)
	if err != nil { return "", errors.New("cannot encode Fabric IAM group pins") }
	if _, err := decodeIAMGroupPins(string(raw)); err != nil { return "", err }
	return string(raw), nil
}

// readOpenFGAIAMGroupPins returns an EXACT tenant-declared group roster,
// pinned to its original TenantBundle UID and approved private store/model.
// A removed/renamed group is NOT discarded: its prior OpenFGA membership
// tuples must first be explicitly revoked in a later guarded offboarding
// reconciler. New groups require a verified pin before this read succeeds.
func (r *TenantBundleReconciler) readOpenFGAIAMGroupPins(
	ctx context.Context, bundle *fabricv1alpha1.TenantBundle,
) (map[string]string, error) {
	names, err := checkedIAMGroupNames(bundle)
	if err != nil { return nil, err }
	if _, err := r.readOpenFGAAuthorizationBinding(ctx, bundle); err != nil {
		return nil, fmt.Errorf("untrusted Fabric group pin tenant/store/model identity: %w", err)
	}
	name, err := openFGABindingName(bundle.Spec.TenantID)
	if err != nil { return nil, err }
	var binding corev1.ConfigMap
	if err := r.Get(ctx, types.NamespacedName{
		Namespace: openFGAPlatformNamespace, Name: name,
	}, &binding); err != nil { return nil, err }
	schema, schemaExists := binding.Data[iamGroupPinsSchemaKey]
	raw, pinsExist := binding.Data[iamGroupPinsDataKey]
	pinOwnerUID, ownerExists := binding.Data[iamGroupPinsOwnerUIDKey]
	if !schemaExists || !pinsExist || !ownerExists || schema != iamGroupPinsSchemaV1 ||
		pinOwnerUID != string(bundle.UID) {
		return nil, errors.New("incomplete or unapproved retained Fabric IAM group pin ledger")
	}
	pins, err := decodeIAMGroupPins(raw)
	if err != nil { return nil, err }
	if len(pins) != len(names) {
		return nil, errors.New("Fabric group pins do not match declared tenant groups")
	}
	for _, name := range names {
		if _, ok := pins[name]; !ok {
			return nil, errors.New("missing Fabric group pin for a declared IAM group")
		}
	}
	return pins, nil
}

// reconcileOpenFGAIAMGroupPins automatically pins the immutable Authentik
// UUIDs of exactly the TenantBundle-declared groups after the operator has
// published their Authentik blueprint and bound the private FGA store/model.
//
// The source double-reads group/membership pages. This is a guarded bootstrap,
// NOT a human enrollment step or authorization. Preflight ALL groups before
// any ConfigMap write; once pinned, reject a recreated group and never
// implicitly delete an old pin (which could leave stale OpenFGA tuples).
//
// ConfigMap Update is resourceVersion guarded. Concurrent reconciles retry
// from the freshly retained state rather than merging incompatible pins.
func (r *TenantBundleReconciler) reconcileOpenFGAIAMGroupPins(
	ctx context.Context, bundle *fabricv1alpha1.TenantBundle,
	source authoritativeIAMGroupReader,
) error {
	names, err := checkedIAMGroupNames(bundle)
	if err != nil { return err }
	if source == nil {
		return errors.New("trusted Fabric Authentik source is unavailable")
	}
	if _, err := r.readOpenFGAAuthorizationBinding(ctx, bundle); err != nil {
		return fmt.Errorf("untrusted Fabric tenant store/model for IAM group pins: %w", err)
	}
	name, err := openFGABindingName(bundle.Spec.TenantID)
	if err != nil { return err }
	var binding corev1.ConfigMap
	if err := r.Get(ctx, types.NamespacedName{
		Namespace: openFGAPlatformNamespace, Name: name,
	}, &binding); err != nil { return err }

	schema, schemaExists := binding.Data[iamGroupPinsSchemaKey]
	raw, pinsExist := binding.Data[iamGroupPinsDataKey]
	pinOwnerUID, ownerExists := binding.Data[iamGroupPinsOwnerUIDKey]
	var current map[string]string
	switch {
	case schemaExists != pinsExist || schemaExists != ownerExists:
		return errors.New("partial retained Fabric IAM group pins")
	case schemaExists:
		if schema != iamGroupPinsSchemaV1 || pinOwnerUID != string(bundle.UID) {
			return errors.New("unsupported or foreign retained Fabric IAM group pin ownership")
		}
		current, err = decodeIAMGroupPins(raw)
		if err != nil { return err }
	default:
		current = map[string]string{}
	}
	declared := make(map[string]bool, len(names))
	for _, groupName := range names { declared[groupName] = true }
	for oldName := range current {
		if !declared[oldName] {
			return errors.New("IAM group removal requires explicit tuple revocation before retiring retained group UUID")
		}
	}

	next := make(map[string]string, len(names))
	seenUUID := map[string]bool{}
	for _, groupName := range names {
		// Source checks exact names, group UUIDs, complete pages and
		// double-reads active membership. No user UUID is turned into a grant.
		snapshot, err := source.SnapshotGroupMembership(ctx, groupName)
		if err != nil {
			return fmt.Errorf("Fabric Authentik group inventory unavailable for %s: %w", groupName, err)
		}
		if snapshot.Name != groupName || !immutableAuthentikUUID.MatchString(snapshot.AuthentikGroupUUID) ||
			snapshot.ActiveAuthentikUserUUIDs == nil || seenUUID[snapshot.AuthentikGroupUUID] {
			return errors.New("invalid or ambiguous Fabric Authentik group snapshot")
		}
		seenUUID[snapshot.AuthentikGroupUUID] = true
		if oldUUID, pinned := current[groupName]; pinned && oldUUID != snapshot.AuthentikGroupUUID {
			return errors.New("Authentik group UUID changed: refuse implicit adoption of recreated group")
		}
		next[groupName] = snapshot.AuthentikGroupUUID
	}

	// No change => no write. The source is nevertheless checked every time,
	// so a replaced group or API outage is not hidden by the retained pin.
	if len(current) == len(next) {
		same := true
		for group, uuid := range next {
			if current[group] != uuid { same = false; break }
		}
		if same && schemaExists {
			_, err := r.readOpenFGAIAMGroupPins(ctx, bundle)
			return err
		}
	}

	// Re-read Authentik before committing new UUID pins, to catch a
	// deletion/recreation between initial validation and persistence.
	for _, groupName := range names {
		snapshot, err := source.SnapshotGroupMembership(ctx, groupName)
		if err != nil || snapshot.Name != groupName ||
			snapshot.AuthentikGroupUUID != next[groupName] ||
			snapshot.ActiveAuthentikUserUUIDs == nil {
			return errors.New("Fabric Authentik group identity changed during UUID pinning")
		}
	}
	encoded, err := encodeIAMGroupPins(next)
	if err != nil { return err }
	if binding.Data == nil { return errors.New("missing protected Fabric authorization binding data") }
	binding.Data[iamGroupPinsSchemaKey] = iamGroupPinsSchemaV1
	binding.Data[iamGroupPinsOwnerUIDKey] = string(bundle.UID)
	binding.Data[iamGroupPinsDataKey] = encoded
	if err := r.Update(ctx, &binding); err != nil {
		return fmt.Errorf("cannot persist trusted Fabric Authentik group pins: %w", err)
	}
	stored, err := r.readOpenFGAIAMGroupPins(ctx, bundle)
	if err != nil { return err }
	if len(stored) != len(next) { return errors.New("Fabric IAM group UUID pins not confirmed") }
	for group, uuid := range next {
		if stored[group] != uuid { return errors.New("Fabric IAM group UUID pin drift after update") }
	}
	return nil
}
