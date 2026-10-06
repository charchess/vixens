package controller

import (
	"context"
	"testing"

	fabricv1alpha1 "github.com/charchess/vixens/apps/60-services/txo-fabric/operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func aiGatewayTestProfile() *fabricv1alpha1.AIGatewayProfile {
	return &fabricv1alpha1.AIGatewayProfile{
		ObjectMeta: metav1.ObjectMeta{Name: defaultAIGatewayProfileName},
		Spec: fabricv1alpha1.AIGatewayProfileSpec{
			Topology:       "TenantScoped",
			Implementation: "LiteLLM",
			Image:          "ghcr.io/berriai/litellm-non_root:v1.102.1",
			APIPort:        4000,
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("250m"),
					corev1.ResourceMemory: resource.MustParse("512Mi"),
				},
			},
			PriorityClassName: "vixens-medium",
			SizingLabel:       "V-scout",
		},
	}
}

func aiGatewayTestTenant(name, tenantID string) *fabricv1alpha1.TenantBundle {
	return &fabricv1alpha1.TenantBundle{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: fabricv1alpha1.TenantBundleSpec{
			TenantID:  tenantID,
			DisplayName: name,
			AIGateway: &fabricv1alpha1.TenantAIGatewaySpec{ProfileRef: defaultAIGatewayProfileName},
		},
	}
}

func TestReconcileAIGatewayReservesLiteLLMContractFailClosed(t *testing.T) {
	ctx := context.Background()
	scheme := testScheme(t)
	tenant := aiGatewayTestTenant("fabric-smoke", "TEN90002")
	profile := aiGatewayTestProfile()
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(tenant, profile).Build()

	result, err := (&TenantBundleReconciler{Client: c, Scheme: scheme}).reconcileAIGateway(ctx, tenant)
	if err != nil {
		t.Fatal(err)
	}
	if result.Ready || result.Status == nil || result.Status.Phase != "Blocked" || result.Reason != "ControllerNotImplemented" {
		t.Fatalf("LiteLLM gateway placeholder must fail closed until runtime reconciliation lands: %#v", result)
	}
}

func TestReconcileAIGatewayRejectsCredentialBrokerImplementation(t *testing.T) {
	ctx := context.Background()
	scheme := testScheme(t)
	tenant := aiGatewayTestTenant("fabric-smoke", "TEN90002")
	profile := aiGatewayTestProfile()
	profile.Spec.Implementation = "CLIProxyAPI"
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(tenant, profile).Build()

	result, err := (&TenantBundleReconciler{Client: c, Scheme: scheme}).reconcileAIGateway(ctx, tenant)
	if err != nil {
		t.Fatal(err)
	}
	if result.Ready || result.Status == nil || result.Status.Phase != "Blocked" || result.Reason != "UnsupportedImplementation" {
		t.Fatalf("CPA must not be accepted as tenant-facing AI gateway: %#v", result)
	}
}
