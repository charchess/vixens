package controller

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	fabricv1alpha1 "github.com/charchess/vixens/apps/60-services/txo-fabric/operator/api/v1alpha1"
	"github.com/charchess/vixens/apps/60-services/txo-fabric/operator/internal/openfga"
	"github.com/charchess/vixens/apps/60-services/txo-fabric/operator/internal/privatebridge"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

var errBrowserAuthorityUnavailable = errors.New("private browser authority unavailable")

// ResolvePrivateBrowserAuthority is a read-only, server-derived view. It
// refuses stale/deleting/ambiguous tenant or agent state, and never returns
// service, namespace, endpoint, credential, group, Authentik or FGA IDs.
func (r *TenantBundleReconciler) ResolvePrivateBrowserAuthority(ctx context.Context, q privatebridge.BrowserAuthorityRequest, now time.Time) (privatebridge.BrowserAuthorityResponse, error) {
	if r == nil || r.Client == nil || now.IsZero() {
		return privatebridge.BrowserAuthorityResponse{}, errBrowserAuthorityUnavailable
	}
	switch q.Kind {
	case privatebridge.BrowserAuthorityCatalog:
		return r.browserTenantCatalog(ctx, q.AuthentikUserUUID, now)
	case privatebridge.BrowserAuthorityTenant:
		bundle, err := r.browserTenant(ctx, q.TenantKey, q.AuthentikUserUUID, now)
		if err != nil {
			return privatebridge.BrowserAuthorityResponse{}, err
		}
		user, err := r.ResolvePrivateBFFIdentity(ctx, q.TenantKey, q.AuthentikUserUUID, now)
		if err != nil || user == "" {
			return privatebridge.BrowserAuthorityResponse{}, errBrowserAuthorityUnavailable
		}
		return privatebridge.BrowserAuthorityResponse{Tenant: &privatebridge.BrowserTenantAuthority{BrowserTenant: browserTenant(bundle), FabricUserID: user, AccountActive: true, Fresh: true, CanEnter: true}}, nil
	case privatebridge.BrowserAuthorityAgent:
		bundle, err := r.browserTenant(ctx, q.TenantKey, q.AuthentikUserUUID, now)
		if err != nil {
			return privatebridge.BrowserAuthorityResponse{}, err
		}
		user, err := r.ResolvePrivateBFFIdentity(ctx, q.TenantKey, q.AuthentikUserUUID, now)
		if err != nil {
			return privatebridge.BrowserAuthorityResponse{}, err
		}
		agent, err := r.browserAgent(ctx, bundle, q.AgentKey)
		if err != nil || !r.browserAgentAllowed(ctx, bundle, agent, user) {
			return privatebridge.BrowserAuthorityResponse{}, errBrowserAuthorityUnavailable
		}
		return privatebridge.BrowserAuthorityResponse{Agent: &privatebridge.BrowserAgentAuthority{AgentKey: agent.Spec.AgentKey, DisplayName: agent.Spec.DisplayName, AgentUID: string(agent.UID), Generation: agent.Generation}}, nil
	default:
		return privatebridge.BrowserAuthorityResponse{}, errBrowserAuthorityUnavailable
	}
}

func browserTenant(b *fabricv1alpha1.TenantBundle) privatebridge.BrowserTenant {
	return privatebridge.BrowserTenant{TenantKey: b.Name, FabricTenantID: b.Spec.TenantID, DisplayName: b.Spec.DisplayName}
}

// browserTenantCatalog returns only tenants that the asserted immutable
// Authentik UUID can enter now. List output is never itself an authorization
// decision: every candidate is re-read and subjected to the same retained
// ledger, pinned-group, active-account, and freshness checks as tenant entry.
// An indeterminate candidate is omitted rather than exposed.
func (r *TenantBundleReconciler) browserTenantCatalog(ctx context.Context, uuid string, now time.Time) (privatebridge.BrowserAuthorityResponse, error) {
	reader := r.APIReader
	if reader == nil {
		reader = r.Client
	}
	var bundles fabricv1alpha1.TenantBundleList
	if err := reader.List(ctx, &bundles); err != nil || len(bundles.Items) > 500 {
		return privatebridge.BrowserAuthorityResponse{}, errBrowserAuthorityUnavailable
	}
	seenKeys, seenIDs := map[string]bool{}, map[string]bool{}
	tenants := make([]privatebridge.BrowserTenant, 0, len(bundles.Items))
	for i := range bundles.Items {
		bundle := &bundles.Items[i]
		if !browserTenantCurrent(bundle, now) {
			continue
		}
		if seenKeys[bundle.Name] || seenIDs[bundle.Spec.TenantID] {
			return privatebridge.BrowserAuthorityResponse{}, errBrowserAuthorityUnavailable
		}
		seenKeys[bundle.Name], seenIDs[bundle.Spec.TenantID] = true, true
		authorized, err := r.browserTenant(ctx, bundle.Name, uuid, now)
		if err != nil {
			continue
		}
		tenants = append(tenants, browserTenant(authorized))
	}
	sort.Slice(tenants, func(i, j int) bool { return tenants[i].TenantKey < tenants[j].TenantKey })
	return privatebridge.BrowserAuthorityResponse{Tenants: tenants}, nil
}

func (r *TenantBundleReconciler) browserTenant(ctx context.Context, tenant, uuid string, now time.Time) (*fabricv1alpha1.TenantBundle, error) {
	// ResolvePrivateBFFIdentity owns the costly pinned ledger/group/freshness
	// rechecks. Read its canonical object again so display output cannot race it.
	if _, err := r.ResolvePrivateBFFIdentity(ctx, tenant, uuid, now); err != nil {
		return nil, errBrowserAuthorityUnavailable
	}
	reader := r.APIReader
	if reader == nil {
		reader = r.Client
	}
	var bundle fabricv1alpha1.TenantBundle
	if err := reader.Get(ctx, types.NamespacedName{Name: tenant}, &bundle); err != nil || !browserTenantCurrent(&bundle, now) {
		return nil, errBrowserAuthorityUnavailable
	}
	return &bundle, nil
}

func browserTenantCurrent(bundle *fabricv1alpha1.TenantBundle, now time.Time) bool {
	return bundle != nil && bundle.Name != "" && bundle.UID != "" && bundle.Generation > 0 && bundle.DeletionTimestamp.IsZero() &&
		bundle.Spec.TenantID != "" && bundle.Spec.DisplayName != "" && len(bundle.Spec.DisplayName) <= 128 && !strings.ContainsAny(bundle.Spec.DisplayName, "\r\n\x00") &&
		bundle.Status.Phase == "Ready" && bundle.Status.ObservedGeneration == bundle.Generation && privateAuthorizationConditionFresh(bundle.Status.Conditions, bundle.Generation, now)
}

func (r *TenantBundleReconciler) browserAgent(ctx context.Context, bundle *fabricv1alpha1.TenantBundle, key string) (*fabricv1alpha1.AgentIdentity, error) {
	reader := r.APIReader
	if reader == nil {
		reader = r.Client
	}
	var agents fabricv1alpha1.AgentIdentityList
	if err := reader.List(ctx, &agents); err != nil {
		return nil, errBrowserAuthorityUnavailable
	}
	var selected *fabricv1alpha1.AgentIdentity
	for i := range agents.Items {
		agent := &agents.Items[i]
		if agent.Spec.TenantRef.Name != bundle.Name || agent.Spec.AgentKey != key {
			continue
		}
		if selected != nil || agent.Name == "" || agent.UID == "" || agent.DeletionTimestamp != nil || agent.Generation < 1 || agent.Status.ObservedGeneration != agent.Generation || agent.Status.Phase != "Ready" || !agent.Spec.HumanAccess.Enabled || agent.Spec.DisplayName == "" || len(agent.Spec.DisplayName) > 128 || strings.ContainsAny(agent.Spec.DisplayName, "\r\n\x00") {
			return nil, errBrowserAuthorityUnavailable
		}
		selected = agent.DeepCopy()
	}
	if selected == nil {
		return nil, errBrowserAuthorityUnavailable
	}
	return selected, nil
}

func (r *TenantBundleReconciler) browserAgentAllowed(ctx context.Context, bundle *fabricv1alpha1.TenantBundle, agent *fabricv1alpha1.AgentIdentity, user string) bool {
	binding, err := r.readOpenFGAAuthorizationBinding(ctx, bundle)
	if err != nil || binding.Store.ID == "" || binding.ModelID == "" {
		return false
	}
	reader := r.APIReader
	if reader == nil {
		reader = r.Client
	}
	var secret corev1.Secret
	if err := reader.Get(ctx, types.NamespacedName{Namespace: openFGAPlatformNamespace, Name: openFGACredentialSecret}, &secret); err != nil {
		return false
	}
	client, err := openfga.NewClient(string(secret.Data["presharedKeys"]), nil)
	if err != nil {
		return false
	}
	allowed, err := client.Check(ctx, openfga.CheckRequest{StoreID: binding.Store.ID, AuthorizationModelID: binding.ModelID, TupleKey: openfga.Tuple{User: "user:" + user, Relation: "can_chat", Object: "agent:" + agent.Name}})
	return err == nil && allowed
}
