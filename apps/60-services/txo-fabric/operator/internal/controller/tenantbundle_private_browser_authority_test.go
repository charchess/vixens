package controller

import (
	"context"
	"errors"
	"testing"
	"time"

	fabricv1alpha1 "github.com/charchess/vixens/apps/60-services/txo-fabric/operator/api/v1alpha1"
	"github.com/charchess/vixens/apps/60-services/txo-fabric/operator/internal/authentik"
	"github.com/charchess/vixens/apps/60-services/txo-fabric/operator/internal/privatebridge"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func markBrowserTenantReady(t *testing.T, r *TenantBundleReconciler, b *fabricv1alpha1.TenantBundle, now time.Time) {
	t.Helper()
	markPrivateMembershipReady(t, r, b, now)
	var saved fabricv1alpha1.TenantBundle
	if err := r.Get(context.Background(), types.NamespacedName{Name: b.Name}, &saved); err != nil {
		t.Fatal(err)
	}
	saved.Status.Phase = "Ready"
	if err := r.Update(context.Background(), &saved); err != nil {
		t.Fatal(err)
	}
	*b = *saved.DeepCopy()
}

func TestBrowserTenantCatalogReturnsOnlyCurrentAuthorizedTenants(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 10, 11, 20, 0, 0, 0, time.UTC)
	a := humanIdentityTenant("hairem", "TEN00001", "uid-hairem", "client0")
	b := humanIdentityTenant("indiba", "TEN00002", "uid-indiba", "sales")
	a.Generation, b.Generation = 5, 6
	r := preparedHumanIdentityReconciler(t, a, b)
	r.AuthorizationNow = func() time.Time { return now }

	aGroup := authentikGroupName(a.Name, "client0")
	bGroup := authentikGroupName(b.Name, "sales")
	source := &freshAccountSource{stagedIAMSource: &stagedIAMSource{
		groups: map[string]authentik.GroupMembership{
			aGroup: makeIAMGroup(aGroup, identityLedgerGroupUUID, identityLedgerUserUUID),
			bGroup: makeIAMGroup(bGroup, "550e8400-e29b-41d4-a716-446655440019"),
		},
		calls: map[string]int{},
		fail:  map[string]error{},
	}}
	for _, bundle := range []*fabricv1alpha1.TenantBundle{a, b} {
		if err := r.reconcileOpenFGAHumanIdentityRegistry(ctx, bundle); err != nil {
			t.Fatal(err)
		}
		if err := r.reconcileOpenFGAIAMGroupPins(ctx, bundle, source); err != nil {
			t.Fatal(err)
		}
		markBrowserTenantReady(t, r, bundle, now)
	}
	installIAMIdentityBindings(t, r, a.Spec.TenantID, authentik.IdentityBinding{
		TenantID:          a.Spec.TenantID,
		Issuer:            a.Spec.HumanAccess.Web.OIDC.Issuer,
		OIDCSubject:       "legacy-local-sub-not-global",
		AuthentikUserUUID: identityLedgerUserUUID,
		FabricUserID:      "usr000001",
	})
	r.AuthentikMembershipClient = source

	catalog, err := r.browserTenantCatalog(ctx, identityLedgerUserUUID, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(catalog.Tenants) != 1 || catalog.Tenants[0].TenantKey != a.Name || catalog.Tenants[0].FabricTenantID != a.Spec.TenantID {
		t.Fatalf("catalog exposed an unauthorized or wrong tenant: %#v", catalog.Tenants)
	}

	foreign, err := r.browserTenantCatalog(ctx, "550e8400-e29b-41d4-a716-446655440011", now)
	if err != nil {
		t.Fatal(err)
	}
	if len(foreign.Tenants) != 0 {
		t.Fatalf("foreign UUID received tenant catalog: %#v", foreign.Tenants)
	}

	source.fail[aGroup] = errors.New("authentik unavailable")
	unavailable, err := r.browserTenantCatalog(ctx, identityLedgerUserUUID, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(unavailable.Tenants) != 0 {
		t.Fatalf("IAM outage exposed a tenant: %#v", unavailable.Tenants)
	}
}

func TestResponsesAuthorityAgentRequiresAttestedCanonicalPrivateRuntime(t *testing.T) {
	ctx := context.Background()
	agent, tenant := privateAPIFixture()
	agent.Spec.Runtime.PrivateAPI = validFuturePrivateAPI(agent, tenant)
	agent.Status.Phase = "Ready"
	agent.Status.ObservedGeneration = agent.Generation
	agent.Status.Conditions = []metav1.Condition{{
		Type: privateHermesAPICondition, Status: metav1.ConditionTrue,
		Reason: "EndpointAttested", ObservedGeneration: agent.Generation,
	}}
	c := fake.NewClientBuilder().WithScheme(testScheme(t)).WithObjects(agent).Build()
	r := &TenantBundleReconciler{Client: c}

	resolved, err := r.responsesAuthorityAgent(ctx, tenant, agent.Spec.AgentKey)
	if err != nil || resolved == nil || resolved.UID != agent.UID || resolved.Generation != agent.Generation {
		t.Fatalf("attested canonical private agent not resolved: agent=%#v err=%v", resolved, err)
	}

	resolved.Status.Conditions[0].Reason = "NotReady"
	if err := c.Update(ctx, resolved); err != nil {
		t.Fatal(err)
	}
	if resolved, err := r.responsesAuthorityAgent(ctx, tenant, agent.Spec.AgentKey); err == nil || resolved != nil {
		t.Fatalf("unattested private API must not resolve: agent=%#v err=%v", resolved, err)
	}
}

func TestResolvePrivateResponsesAuthorityReturnsOnlyCurrentCanonicalTenant(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 11, 4, 15, 45, 0, 0, time.UTC)
	bundle := humanIdentityTenant("hairem", "TEN00001", "uid-hairem", "client0")
	bundle.Generation = 7
	r := preparedHumanIdentityReconciler(t, bundle)
	r.AuthorizationNow = func() time.Time { return now }
	markBrowserTenantReady(t, r, bundle, now)

	out, err := r.ResolvePrivateResponsesAuthority(ctx, privatebridge.ResponsesAuthorityRequest{
		Operation: privatebridge.ResponsesAuthorityResolveTenant,
		TenantKey: bundle.Name, FabricTenantID: bundle.Spec.TenantID,
	}, now)
	if err != nil || out.Tenant == nil || out.Tenant.TenantKey != bundle.Name ||
		out.Tenant.FabricTenantID != bundle.Spec.TenantID || out.Tenant.TenantBundleUID != string(bundle.UID) || out.Tenant.Status != "Ready" {
		t.Fatalf("canonical ready tenant was not resolved safely: out=%#v err=%v", out, err)
	}

	wrongID, err := r.ResolvePrivateResponsesAuthority(ctx, privatebridge.ResponsesAuthorityRequest{
		Operation: privatebridge.ResponsesAuthorityResolveTenant,
		TenantKey: bundle.Name, FabricTenantID: "TEN99999",
	}, now)
	if err == nil || wrongID.Tenant != nil {
		t.Fatalf("mismatched immutable tenant ID resolved: out=%#v err=%v", wrongID, err)
	}
}
