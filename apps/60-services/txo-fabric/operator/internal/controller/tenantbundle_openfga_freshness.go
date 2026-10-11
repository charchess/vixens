package controller

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"time"

	fabricv1alpha1 "github.com/charchess/vixens/apps/60-services/txo-fabric/operator/api/v1alpha1"
	"github.com/charchess/vixens/apps/60-services/txo-fabric/operator/internal/authentik"
	"github.com/charchess/vixens/apps/60-services/txo-fabric/operator/internal/openfga"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

// A condition True is a short lease on a verified *tenant membership*
// projection, not a human login, an agent permission or an OpenFGA grant
// for another action. Consumers MUST validate observedGeneration and the
// validUntil timestamp, not merely status=True.
const (
	authorizationSnapshotMaxAge = 30 * time.Second
	authorizationLease = 60 * time.Second
	authorizationRefresh = 20 * time.Second
)

type authoritativeIAMAccountReader interface {
	SnapshotUserAccount(context.Context, string) (authentik.UserAccountSnapshot, error)
}

type authoritativeIAMSource interface {
	authoritativeIAMGroupReader
	authoritativeIAMAccountReader
}

// Narrow single-user deletion; never overwrite the entire group from a
// potentially stale human membership snapshot just to revoke one user.
type scopedIAMTupleRevoker interface {
	RevokeMemberTuple(context.Context, string, string, openfga.TupleScope, string) error
}

func (r *TenantBundleReconciler) authorizationClock() time.Time {
	if r.AuthorizationNow != nil {
		return r.AuthorizationNow()
	}
	return time.Now()
}

func withinSnapshotWindow(start, finish time.Time) bool {
	return !finish.Before(start) && finish.Sub(start) <= authorizationSnapshotMaxAge
}

// reconcileFreshTenantMembers is called ONLY under the operator's explicit
// OpenFGA gate, after store/model/identity-ledger and group-pin reconciliation.
// It reads exact retained group UUIDs, rechecks current Authentik membership
// and active accounts, and writes only the two approved membership relations
// in the tenant's immutable server-resolved store/model. It never enrolls a
// user or changes an issuer/subject binding. An empty complete group inventory
// is valid; an incomplete one is NOT an empty membership.
func (r *TenantBundleReconciler) reconcileFreshTenantMembers(
	ctx context.Context, bundle *fabricv1alpha1.TenantBundle,
	source authoritativeIAMGroupReader, writer scopedIAMTupleWriter,
) (string, string) {
	if r == nil || bundle == nil || !bundle.DeletionTimestamp.IsZero() ||
		bundle.Spec.HumanAccess == nil || bundle.Spec.HumanAccess.Web == nil {
		return "", "IdentityLedgerInvalid"
	}
	if source == nil {
		return "", "IAMCredentialMissing"
	}
	account, ok := source.(authoritativeIAMAccountReader)
	if !ok {
		return "", "IAMSourceUnavailable"
	}
	if writer == nil {
		return "", "OpenFGAUnavailable"
	}
	start := r.authorizationClock()
	pins, err := r.readOpenFGAIAMGroupPins(ctx, bundle)
	if err != nil {
		return "", "GroupPinInvalid"
	}
	registry, err := r.readOpenFGAHumanIdentityRegistry(ctx, bundle)
	if err != nil {
		return "", "IdentityLedgerInvalid"
	}
	approved, err := r.readOpenFGAAuthorizationBinding(ctx, bundle)
	if err != nil {
		return "", "OpenFGAUnavailable"
	}
	names, err := checkedIAMGroupNames(bundle)
	if err != nil || len(pins) != len(names) {
		return "", "GroupPinInvalid"
	}

	// Preflight EVERY group and EVERY active user before any OpenFGA mutation.
	// A UUID may legitimately belong to more than one approved group; a
	// duplicate inside one group is rejected by ResolveGroupMembership.
	members := map[string]bool{}
	users := map[string]bool{}
	snapshots := make(map[string]authentik.GroupMembership, len(names))
	for _, name := range names {
		snapshot, readErr := source.SnapshotGroupMembership(ctx, name)
		if readErr != nil {
			return "", "IAMSourceUnavailable"
		}
		if pins[name] == "" || snapshot.Name != name ||
			snapshot.AuthentikGroupUUID != pins[name] {
			return "", "GroupPinInvalid"
		}
		ids, resolveErr := registry.ResolveGroupMembership(snapshot)
		if resolveErr != nil {
			return "", "IdentityLedgerInvalid"
		}
		for _, id := range ids {
			members[id] = true
		}
		for _, uuid := range snapshot.ActiveAuthentikUserUUIDs {
			users[uuid] = true
		}
		snapshots[name] = snapshot
	}
	orderedUsers := make([]string, 0, len(users))
	for uuid := range users {
		orderedUsers = append(orderedUsers, uuid)
	}
	sort.Strings(orderedUsers)
	inactive := make(map[string]string)
	for _, uuid := range orderedUsers {
		user, readErr := account.SnapshotUserAccount(ctx, uuid)
		if readErr != nil || user.UUID != uuid {
			// A missing/mismatched account is NOT explicit proof of
			// deactivation. No speculative revocation on IAM faults.
			return "", "IAMSourceUnavailable"
		}
		if !user.Active {
			id, lookupErr := registry.ResolveAuthentikUserUUID(uuid)
			if lookupErr != nil { return "", "IdentityLedgerInvalid" }
			inactive[uuid] = id
		}
	}
	if !withinSnapshotWindow(start, r.authorizationClock()) {
		return "", "StaleSnapshot"
	}
	if len(inactive) > 0 {
		return "", r.revokeConfirmedInactiveMembers(ctx, bundle, source, account,
			writer, pins, snapshots, orderedUsers, inactive, approved, start)
	}

	// Existing group#member reconciliation preflights all groups, revokes
	// before granting and verifies exact FGA read-back. It rechecks Authentik
	// before return, rather than trusting the preflight projection above.
	if err := r.reconcileVerifiedIAMGroupMemberships(ctx, bundle, pins, source, writer); err != nil {
		return "", "TupleMismatch"
	}
	if !withinSnapshotWindow(start, r.authorizationClock()) {
		return "", "StaleSnapshot"
	}

	// Distinct from group#member: tenant#member is the union of verified
	// memberships, never an implication of agent.can_chat or tenant admin.
	scope := openfga.TupleScope{Object: "tenant:" + bundle.Spec.TenantID, Relation: "member"}
	ids := make([]string, 0, len(members))
	for id := range members {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	desired := make([]openfga.Tuple, 0, len(ids))
	for _, id := range ids {
		desired = append(desired, openfga.Tuple{
			User: "user:" + id, Object: scope.Object, Relation: scope.Relation,
		})
	}
	if err := writer.ReconcileTupleScope(ctx, approved.Store.ID, approved.ModelID, scope, desired); err != nil {
		return "", "TupleMismatch"
	}

	// A successful write/read-back cannot hide revocation during the
	// reconcile. Recheck exact group UUIDs and membership sets, all active
	// accounts, retained tenant identity and model before issuing a lease.
	for _, name := range names {
		latest, readErr := source.SnapshotGroupMembership(ctx, name)
		if readErr != nil {
			return "", "IAMSourceUnavailable"
		}
		if !reflect.DeepEqual(latest, snapshots[name]) {
			return "", "StaleSnapshot"
		}
	}
	postInactive := make(map[string]string)
	for _, uuid := range orderedUsers {
		user, readErr := account.SnapshotUserAccount(ctx, uuid)
		if readErr != nil || user.UUID != uuid {
			return "", "IAMSourceUnavailable"
		}
		if !user.Active {
			id, lookupErr := registry.ResolveAuthentikUserUUID(uuid)
			if lookupErr != nil { return "", "IdentityLedgerInvalid" }
			postInactive[uuid] = id
		}
	}
	if len(postInactive) > 0 {
		return "", r.revokeConfirmedInactiveMembers(ctx, bundle, source, account,
			writer, pins, snapshots, orderedUsers, postInactive, approved, start)
	}
	confirmedPins, err := r.readOpenFGAIAMGroupPins(ctx, bundle)
	if err != nil || !reflect.DeepEqual(confirmedPins, pins) {
		return "", "GroupPinInvalid"
	}
	confirmed, err := r.readOpenFGAAuthorizationBinding(ctx, bundle)
	if err != nil || confirmed != approved {
		return "", "OpenFGAUnavailable"
	}
	finish := r.authorizationClock()
	if !withinSnapshotWindow(start, finish) {
		return "", "StaleSnapshot"
	}
	if finish.IsZero() {
		return "", "StaleSnapshot"
	}
	return finish.Add(authorizationLease).UTC().Format(time.RFC3339Nano), ""
}

// revokeConfirmedInactiveMembers removes only confirmed inactive Fabric
// users from every pinned tenant-owned group membership and the direct tenant
// membership. We first repeat all authoritative IAM reads and compare with
// the complete validated snapshot, then confirm retained pins/store/model.
// On *any* unavailable/changed source, no revocation is attempted. A single
// tuple write failure is NotReady; retry can safely finish partial revocation.
func (r *TenantBundleReconciler) revokeConfirmedInactiveMembers(
	ctx context.Context, bundle *fabricv1alpha1.TenantBundle,
	source authoritativeIAMGroupReader, accounts authoritativeIAMAccountReader,
	writer scopedIAMTupleWriter, pins map[string]string,
	snapshots map[string]authentik.GroupMembership,
	orderedUsers []string, inactive map[string]string,
	approved openFGAAuthorizationBinding, started time.Time,
) string {
	revoker, ok := writer.(scopedIAMTupleRevoker)
	if !ok || revoker == nil { return "TupleMismatch" }
	names, err := checkedIAMGroupNames(bundle)
	if err != nil || len(names) != len(pins) || len(inactive) == 0 {
		return "GroupPinInvalid"
	}
	for _, name := range names {
		if pins[name] == "" { return "GroupPinInvalid" }
		current, readErr := source.SnapshotGroupMembership(ctx, name)
		if readErr != nil { return "IAMSourceUnavailable" }
		if !reflect.DeepEqual(current, snapshots[name]) ||
			current.AuthentikGroupUUID != pins[name] {
			return "StaleSnapshot"
		}
	}
	// Verify all accounts, not just the inactive one: no deletion after an
	// incomplete or inconsistent authoritative inventory.
	for _, uuid := range orderedUsers {
		user, readErr := accounts.SnapshotUserAccount(ctx, uuid)
		if readErr != nil || user.UUID != uuid { return "IAMSourceUnavailable" }
		_, wasInactive := inactive[uuid]
		if wasInactive == user.Active { return "StaleSnapshot" }
	}
	currentPins, err := r.readOpenFGAIAMGroupPins(ctx, bundle)
	if err != nil || !reflect.DeepEqual(currentPins, pins) {
		return "GroupPinInvalid"
	}
	current, err := r.readOpenFGAAuthorizationBinding(ctx, bundle)
	if err != nil || current != approved { return "OpenFGAUnavailable" }
	if !withinSnapshotWindow(started, r.authorizationClock()) {
		return "StaleSnapshot"
	}

	// Deterministic scope ordering; all are derived from pinned tenant data.
	scopes := make([]openfga.TupleScope, 0, len(names)+1)
	for _, name := range names {
		scopes = append(scopes, openfga.TupleScope{
			Object: "group:" + name, Relation: "member",
		})
	}
	scopes = append(scopes, openfga.TupleScope{
		Object: "tenant:" + bundle.Spec.TenantID, Relation: "member",
	})
	uuids := make([]string, 0, len(inactive))
	for uuid := range inactive { uuids = append(uuids, uuid) }
	sort.Strings(uuids)
	for _, scope := range scopes {
		for _, uuid := range uuids {
			if !withinSnapshotWindow(started, r.authorizationClock()) {
				return "StaleSnapshot"
			}
			if err := revoker.RevokeMemberTuple(ctx, approved.Store.ID,
				approved.ModelID, scope, inactive[uuid]); err != nil {
				return "TupleMismatch"
			}
		}
	}
	// The revoker checks exact OpenFGA read-back for each scope. Even if
	// successful, explicit inactivity never creates a True freshness lease.
	return "AccountInactive"
}

// AuthorizationLeaseFresh is the mandatory additional check for future
// private readers (#4037). A stored True condition is NOT proof of freshness
// after the lease expires, controller restart, or a generation change.
func AuthorizationLeaseFresh(validUntil string, now time.Time) bool {
	expiry, err := time.Parse(time.RFC3339Nano, validUntil)
	return err == nil && !now.IsZero() && now.Before(expiry) &&
		!now.Before(expiry.Add(-authorizationLease))
}

// The writer only receives the platform-owned preshared key from an existing
// Secret or a controller test seam; no browser/user can choose the store/model.
func (r *TenantBundleReconciler) authorizationTupleWriter(ctx context.Context) (scopedIAMTupleWriter, error) {
	if r.OpenFGATupleClient != nil {
		return r.OpenFGATupleClient, nil
	}
	var secret corev1.Secret
	if err := r.Get(ctx, types.NamespacedName{
		Namespace: openFGAPlatformNamespace, Name: openFGACredentialSecret,
	}, &secret); err != nil {
		return nil, fmt.Errorf("private OpenFGA credential unavailable: %w", err)
	}
	writer, err := openfga.NewClient(string(secret.Data["presharedKeys"]), nil)
	if err != nil {
		return nil, fmt.Errorf("private OpenFGA credential invalid")
	}
	return writer, nil
}
