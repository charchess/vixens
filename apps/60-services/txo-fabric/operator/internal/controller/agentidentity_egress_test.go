package controller

import (
	"context"
	"testing"

	fabricv1alpha1 "github.com/charchess/vixens/apps/60-services/txo-fabric/operator/api/v1alpha1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestEnsureEgressPolicyAddsTenantLocalGatewayForOptInTenant(t *testing.T) {
	ctx := context.Background()
	scheme := testScheme(t)
	tenant := testTenant()
	tenant.Spec.AIGateway = &fabricv1alpha1.TenantAIGatewaySpec{ProfileRef: defaultAIGatewayProfileName}
	agent := testAgentIdentity()
	namespace := tenantNamespace(tenant.Name)

	c := fake.NewClientBuilder().WithScheme(scheme).Build()
	reconciler := &AgentIdentityReconciler{Client: c, Scheme: scheme}
	if err := reconciler.ensureEgressPolicy(ctx, agent, tenant, namespace); err != nil {
		t.Fatal(err)
	}

	var policy networkingv1.NetworkPolicy
	if err := c.Get(ctx, types.NamespacedName{Name: runtimeName(agent.Spec.AgentKey) + "-egress", Namespace: namespace}, &policy); err != nil {
		t.Fatal(err)
	}
	if len(policy.Spec.Egress) != 4 {
		t.Fatalf("expected DNS + shared gateway + tenant gateway + Hindsight egress rules, got %d", len(policy.Spec.Egress))
	}

	sharedGateway := false
	tenantGateway := false
	for _, rule := range policy.Spec.Egress {
		hasPort4000 := false
		for _, port := range rule.Ports {
			if port.Port != nil && port.Port.IntValue() == 4000 {
				hasPort4000 = true
				break
			}
		}
		if !hasPort4000 {
			continue
		}
		for _, peer := range rule.To {
			if peer.NamespaceSelector != nil &&
				peer.NamespaceSelector.MatchLabels["kubernetes.io/metadata.name"] == "txo-fabric-system" &&
				peer.PodSelector != nil &&
				peer.PodSelector.MatchLabels[LabelName] == "txo-ai-gateway" {
				sharedGateway = true
			}
			if peer.NamespaceSelector == nil &&
				peer.PodSelector != nil &&
				peer.PodSelector.MatchLabels[LabelManaged] == "true" &&
				peer.PodSelector.MatchLabels[LabelTenantName] == tenant.Name &&
				peer.PodSelector.MatchLabels["app.kubernetes.io/component"] == "tenant-ai-gateway" {
				tenantGateway = true
			}
		}
	}
	if !sharedGateway {
		t.Fatal("shared AI gateway egress must remain available during staged migration")
	}
	if !tenantGateway {
		t.Fatal("tenant-local AI gateway egress is missing for an aiGateway-enabled tenant")
	}
}
