package controller

import (
	"context"
	"testing"

	fabricv1alpha1 "github.com/charchess/vixens/apps/60-services/txo-fabric/operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestTenantLifecycleDefaultsToActive(t *testing.T) {
	active := &fabricv1alpha1.TenantBundle{}
	if got := tenantLifecycleMode(active); got != TenantLifecycleActive {
		t.Fatalf("default lifecycle=%q want %q", got, TenantLifecycleActive)
	}
	if !tenantRunsAIPlane(active) {
		t.Fatal("tenant without lifecycle field must run the mandatory AI plane")
	}

	parked := active.DeepCopy()
	parked.Spec.Lifecycle.Mode = TenantLifecycleParked
	if got := tenantLifecycleMode(parked); got != TenantLifecycleParked {
		t.Fatalf("parked lifecycle=%q", got)
	}
	if tenantRunsAIPlane(parked) {
		t.Fatal("parked tenant must not run steady-state AI-plane compute")
	}
}

func TestDefaultAIProfileWatchesIncludeActiveTenantAndExcludeParkedTenant(t *testing.T) {
	ctx := context.Background()
	scheme := testScheme(t)
	active := &fabricv1alpha1.TenantBundle{
		ObjectMeta: metav1.ObjectMeta{Name: "indiba"},
		Spec: fabricv1alpha1.TenantBundleSpec{
			TenantID:    "TEN00002",
			DisplayName: "Indiba",
		},
	}
	parked := &fabricv1alpha1.TenantBundle{
		ObjectMeta: metav1.ObjectMeta{Name: "fabric-smoke"},
		Spec: fabricv1alpha1.TenantBundleSpec{
			TenantID:    "TEN90002",
			DisplayName: "Fabric Smoke",
			Lifecycle:   fabricv1alpha1.TenantLifecycleSpec{Mode: TenantLifecycleParked},
		},
	}
	gatewayProfile := aiGatewayTestProfile()
	brokerProfile := aiCredentialBrokerTestProfile()
	postgresqlProfile := aiGatewayPostgreSQLTestProfile()

	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
		active,
		parked,
		gatewayProfile,
		brokerProfile,
		postgresqlProfile,
	).Build()
	r := &TenantBundleReconciler{Client: c, Scheme: scheme}

	assertOnlyTenant := func(name string, requests []reconcile.Request) {
		t.Helper()
		if len(requests) != 1 || requests[0].Name != "indiba" {
			t.Fatalf("%s watch requests=%#v want only indiba", name, requests)
		}
	}

	assertOnlyTenant("AIGatewayProfile", r.tenantBundleRequestsForAIGatewayProfile(ctx, gatewayProfile))
	assertOnlyTenant("AICredentialBrokerProfile", r.tenantBundleRequestsForAICredentialBrokerProfile(ctx, brokerProfile))
	assertOnlyTenant("PostgreSQLProfile", r.tenantBundleRequestsForPostgreSQLProfile(ctx, postgresqlProfile))
}

func TestParkedTenantWithLiveAgentKeepsAIPlaneRunning(t *testing.T) {
	ctx := context.Background()
	scheme := testScheme(t)
	tenant := &fabricv1alpha1.TenantBundle{
		ObjectMeta: metav1.ObjectMeta{Name: "hairem"},
		Spec: fabricv1alpha1.TenantBundleSpec{
			TenantID:    "TEN00001",
			DisplayName: "hAIrem",
			Lifecycle:   fabricv1alpha1.TenantLifecycleSpec{Mode: TenantLifecycleParked},
		},
	}
	agent := &fabricv1alpha1.AgentIdentity{
		ObjectMeta: metav1.ObjectMeta{Name: "ten00001-usr000001-agt00001"},
		Spec: fabricv1alpha1.AgentIdentitySpec{
			TenantRef: fabricv1alpha1.ObjectReference{Name: tenant.Name},
			AgentKey:  "usr000001-agt00001",
		},
	}
	namespace := tenantNamespace(tenant.Name)
	gatewayDeployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      tenantAIGatewayName,
			Namespace: namespace,
			Labels:    aiGatewayWorkloadLabels(tenant),
		},
	}
	brokerDeployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      tenantAICredentialBrokerName,
			Namespace: namespace,
			Labels:    aiCredentialBrokerLabels(tenant),
		},
	}

	c := fake.NewClientBuilder().WithScheme(scheme).
		WithStatusSubresource(&fabricv1alpha1.TenantBundle{}).
		WithObjects(tenant, agent, gatewayDeployment, brokerDeployment).
		Build()
	r := &TenantBundleReconciler{Client: c, Scheme: scheme}
	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: tenant.Name}}

	if _, err := r.Reconcile(ctx, req); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Reconcile(ctx, req); err != nil {
		t.Fatal(err)
	}

	var current fabricv1alpha1.TenantBundle
	if err := c.Get(ctx, types.NamespacedName{Name: tenant.Name}, &current); err != nil {
		t.Fatal(err)
	}
	if current.Status.Phase != "Degraded" {
		t.Fatalf("phase=%q want Degraded while parking is blocked by live agents", current.Status.Phase)
	}
	found := false
	for _, condition := range current.Status.Conditions {
		if condition.Type == "LifecycleReady" && condition.Status == metav1.ConditionFalse && condition.Reason == "ParkBlockedByAgents" {
			found = true
		}
	}
	if !found {
		t.Fatalf("ParkBlockedByAgents condition missing: %#v", current.Status.Conditions)
	}

	var gatewayAfter appsv1.Deployment
	if err := c.Get(ctx, types.NamespacedName{Name: tenantAIGatewayName, Namespace: namespace}, &gatewayAfter); err != nil {
		t.Fatalf("parking removed LiteLLM while an agent still exists: %v", err)
	}
	var brokerAfter appsv1.Deployment
	if err := c.Get(ctx, types.NamespacedName{Name: tenantAICredentialBrokerName, Namespace: namespace}, &brokerAfter); err != nil {
		t.Fatalf("parking removed CPA while an agent still exists: %v", err)
	}
}

func TestEffectiveAIProfileOverridesRemainBackwardCompatible(t *testing.T) {
	tenant := &fabricv1alpha1.TenantBundle{
		Spec: fabricv1alpha1.TenantBundleSpec{
			AIGateway: &fabricv1alpha1.TenantAIGatewaySpec{ProfileRef: "litellm-custom"},
			AICredentialBroker: &fabricv1alpha1.TenantAICredentialBrokerSpec{ProfileRef: "cpa-custom"},
		},
	}
	if got := tenantAIGatewayProfileName(tenant); got != "litellm-custom" {
		t.Fatalf("gateway override=%q", got)
	}
	if got := tenantAICredentialBrokerProfileName(tenant); got != "cpa-custom" {
		t.Fatalf("broker override=%q", got)
	}

	tenant.Spec.AIGateway = nil
	tenant.Spec.AICredentialBroker = nil
	if got := tenantAIGatewayProfileName(tenant); got != defaultAIGatewayProfileName {
		t.Fatalf("default gateway profile=%q", got)
	}
	if got := tenantAICredentialBrokerProfileName(tenant); got != defaultAICredentialBrokerProfileName {
		t.Fatalf("default broker profile=%q", got)
	}
}
