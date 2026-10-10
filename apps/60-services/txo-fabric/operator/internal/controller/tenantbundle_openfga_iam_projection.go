package controller

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sort"

	fabricv1alpha1 "github.com/charchess/vixens/apps/60-services/txo-fabric/operator/api/v1alpha1"
	"github.com/charchess/vixens/apps/60-services/txo-fabric/operator/internal/authentik"
	"github.com/charchess/vixens/apps/60-services/txo-fabric/operator/internal/openfga"
)

// authoritativeIAMGroupReader must be an operator-owned, authenticated
// Authentik reader (the implementation double-reads complete membership pages).
// A browser JWT groups claim is never an authoritative membership snapshot.
type authoritativeIAMGroupReader interface {
	SnapshotGroupMembership(context.Context, string) (authentik.GroupMembership, error)
}

// scopedIAMTupleWriter can only write the scoped relations of the exact
// server-side store/model pair. No tenant-provided store or tuple is accepted.
type scopedIAMTupleWriter interface {
	ReconcileTupleScope(context.Context, string, string, openfga.TupleScope, []openfga.Tuple) error
}

type desiredIAMGroup struct {
	name string
	groupUUID string
	membership authentik.GroupMembership
	scope openfga.TupleScope
	tuples []openfga.Tuple
}

// reconcileVerifiedIAMGroupMemberships is the operator-side source-to-FGA
// convergence boundary for *group#member* relations only. It is intentionally
// not called from the main TenantBundle loop yet: verified human enrollment,
// durable immutable Authentik group UUID pinning, credentials/NP, freshness
// policy, status/lease and runtime deny-on-stale are not yet wired.
//
// pinnedGroupUUIDs MUST come from a previously verified, durable Fabric-owned
// group registry, NEVER this request, the browser or the current snapshot.
// A missing/foreign/recreated group fails closed rather than auto-adopting it.
// This function never mints an identity or invents an agent-level grant.
func (r *TenantBundleReconciler) reconcileVerifiedIAMGroupMemberships(
	ctx context.Context,
	bundle *fabricv1alpha1.TenantBundle,
	pinnedGroupUUIDs map[string]string,
	source authoritativeIAMGroupReader,
	writer scopedIAMTupleWriter,
) error {
	if r == nil || bundle == nil || !bundle.DeletionTimestamp.IsZero() ||
		bundle.Spec.HumanAccess == nil || bundle.Spec.HumanAccess.Web == nil ||
		source == nil || writer == nil {
		return errors.New("incomplete trusted Fabric group reconciliation context")
	}
	if err := validateTenantIAM(bundle); err != nil {
		return fmt.Errorf("invalid tenant IAM group scope: %w", err)
	}
	groupKeys := append([]string(nil), bundle.Spec.HumanAccess.Web.IAMGroups...)
	sort.Strings(groupKeys)
	if len(pinnedGroupUUIDs) != len(groupKeys) {
		return errors.New("missing or stale pinned Authentik group identity set")
	}
	// The store and model are chosen exclusively by retained Fabric state,
	// including immutable TenantBundle UID, not caller-supplied IDs.
	approved, err := r.readOpenFGAAuthorizationBinding(ctx, bundle)
	if err != nil {
		return fmt.Errorf("untrusted Fabric tenant OpenFGA binding: %w", err)
	}
	registry, err := r.readOpenFGAHumanIdentityRegistry(ctx, bundle)
	if err != nil {
		return fmt.Errorf("human identity registry is not verified: %w", err)
	}

	groups := make([]desiredIAMGroup, 0, len(groupKeys))
	for _, key := range groupKeys {
		name := authentikGroupName(bundle.Name, key)
		uuid, ok := pinnedGroupUUIDs[name]
		if !ok || uuid == "" {
			return errors.New("tenant group UUID has no trusted retained pin")
		}
		// Verify ALL approved groups before any mutation. A missing member
		// association is not silently interpreted as an empty grant set.
		membership, err := source.SnapshotGroupMembership(ctx, name)
		if err != nil {
			return fmt.Errorf("complete Authentik membership snapshot unavailable for %s: %w", name, err)
		}
		if membership.Name != name || membership.AuthentikGroupUUID != uuid {
			return errors.New("recreated, renamed or foreign Authentik group identity")
		}
		users, err := registry.ResolveGroupMembership(membership)
		if err != nil {
			return fmt.Errorf("cannot project trusted Authentik group membership for %s: %w", name, err)
		}
		scope := openfga.TupleScope{Object: "group:" + name, Relation: "member"}
		tuples := make([]openfga.Tuple, 0, len(users))
		for _, user := range users {
			tuples = append(tuples, openfga.Tuple{
				User: "user:" + user, Relation: scope.Relation, Object: scope.Object,
			})
		}
		groups = append(groups, desiredIAMGroup{
			name: name, groupUUID: uuid, membership: membership,
			scope: scope, tuples: tuples,
		})
	}

	// Scoped FGA primitive revokes stale members before adding new ones,
	// then reads back exact state and revalidates the approved model.
	// A partial failure never counts as convergence; retry must recompute
	// from authoritative source and higher-level authorization MUST deny.
	for _, group := range groups {
		if err := writer.ReconcileTupleScope(ctx, approved.Store.ID, approved.ModelID,
			group.scope, group.tuples); err != nil {
			return fmt.Errorf("cannot converge Fabric group membership for %s: %w", group.name, err)
		}
	}

	// Fail if Authentik changed while OpenFGA was being updated. Authentik's
	// double-reads are not a transactional snapshot and this read-back does
	// not replace periodic refresh or runtime stale-policy enforcement.
	for _, group := range groups {
		current, err := source.SnapshotGroupMembership(ctx, group.name)
		if err != nil || !reflect.DeepEqual(current, group.membership) {
			return errors.New("Authentik membership changed or became unavailable during tuple convergence")
		}
	}
	confirmed, err := r.readOpenFGAAuthorizationBinding(ctx, bundle)
	if err != nil || confirmed != approved {
		return errors.New("Fabric tenant store/model changed during membership convergence")
	}
	return nil
}
