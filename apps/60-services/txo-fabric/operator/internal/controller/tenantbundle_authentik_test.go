package controller

import (
	"context"
	"strings"
	"testing"

	fabricv1alpha1 "github.com/charchess/vixens/apps/60-services/txo-fabric/operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func tenantWithIAM(name, displayName, group string, hindsightHumanAccess bool) *fabricv1alpha1.TenantBundle {
	return &fabricv1alpha1.TenantBundle{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: fabricv1alpha1.TenantBundleSpec{
			TenantID:    "TEN90001",
			DisplayName: displayName,
			Memory: fabricv1alpha1.TenantMemorySpec{
				Hindsight: &fabricv1alpha1.HindsightMemorySpec{
					ProfileRef:  "hindsight-gateway",
					HumanAccess: hindsightHumanAccess,
				},
			},
			HumanAccess: &fabricv1alpha1.TenantHumanAccessSpec{
				Web: &fabricv1alpha1.HumanWebAccessSpec{
					DomainSuffix:    "truxonline.com",
					TLSClusterIssuer: "letsencrypt-prod",
					IAMGroups:       []string{group},
					OIDC: fabricv1alpha1.HumanAccessOIDCSpec{
						Issuer:   "https://authentik.truxonline.com/application/o/txo-fabric-" + name + "/",
						ClientID: "txo-fabric-" + name,
					},
				},
			},
		},
	}
}

func TestRenderAuthentikBlueprintAggregatesTenantIAM(t *testing.T) {
	hairem := tenantWithIAM("hairem", "hAIrem", "client0", true)
	indiba := tenantWithIAM("indiba", "Indiba", "sales", false)

	blueprint := renderAuthentikBlueprint([]fabricv1alpha1.TenantBundle{*indiba, *hairem})
	for _, want := range []string{
		"name: txo-fabric-hairem-client0",
		"name: txo-fabric-indiba-sales",
		"id: txo-fabric-hairem-provider",
		"id: txo-fabric-indiba-provider",
		"slug: txo-fabric-hairem",
		"slug: txo-fabric-indiba",
		"url: '^https://[a-z0-9-]+-hairem\\.truxonline\\.com/auth/callback$'",
		"url: '^https://[a-z0-9-]+-indiba\\.truxonline\\.com/auth/callback$'",
		"id: txo-fabric-hairem-hindsight-provider",
		"- !KeyOf txo-fabric-hairem-hindsight-provider",
	} {
		if !strings.Contains(blueprint, want) {
			t.Fatalf("generated blueprint missing %q:\n%s", want, blueprint)
		}
	}
	if strings.Contains(blueprint, "txo-fabric-indiba-hindsight-provider") {
		t.Fatalf("Indiba Hindsight human access is disabled but a proxy provider was rendered:\n%s", blueprint)
	}
	if strings.Index(blueprint, "txo-fabric-hairem-client0") > strings.Index(blueprint, "txo-fabric-indiba-sales") {
		t.Fatalf("tenant blueprint output is not deterministic by tenant name:\n%s", blueprint)
	}
	if got := strings.Count(blueprint, "model: authentik_outposts.outpost"); got != 1 {
		t.Fatalf("embedded outpost declarations=%d, want 1", got)
	}
	if got := strings.Count(blueprint, "policy_engine_mode: any"); got != 3 {
		t.Fatalf("application policy-engine any declarations=%d, want 3", got)
	}
}

func TestRenderAuthentikBlueprintAggregatesAllHindsightProviders(t *testing.T) {
	hairem := tenantWithIAM("hairem", "hAIrem", "client0", true)
	indiba := tenantWithIAM("indiba", "Indiba", "sales", true)

	blueprint := renderAuthentikBlueprint([]fabricv1alpha1.TenantBundle{*hairem, *indiba})
	outpost := blueprint[strings.Index(blueprint, "model: authentik_outposts.outpost"):]
	for _, provider := range []string{
		"- !KeyOf txo-fabric-hairem-hindsight-provider",
		"- !KeyOf txo-fabric-indiba-hindsight-provider",
	} {
		if !strings.Contains(outpost, provider) {
			t.Fatalf("aggregate outpost missing %q:\n%s", provider, outpost)
		}
	}
}

func TestTenantIAMRequiresExplicitStructuralGroup(t *testing.T) {
	tenant := tenantWithIAM("indiba", "Indiba", "sales", false)
	tenant.Spec.HumanAccess.Web.IAMGroups = nil
	if err := validateTenantIAM(tenant); err == nil || !strings.Contains(err.Error(), "iamGroups") {
		t.Fatalf("expected missing iamGroups validation error, got %v", err)
	}
}

func TestReconcileAuthentikBlueprintPublishesAggregateConfigMap(t *testing.T) {
	ctx := context.Background()
	scheme := testScheme(t)
	hairem := tenantWithIAM("hairem", "hAIrem", "client0", true)
	indiba := tenantWithIAM("indiba", "Indiba", "sales", false)
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: authentikNamespace}},
		hairem,
		indiba,
	).Build()

	r := &TenantBundleReconciler{Client: c, Scheme: scheme}
	if err := r.reconcileAuthentikBlueprint(ctx); err != nil {
		t.Fatal(err)
	}

	var configMap corev1.ConfigMap
	if err := c.Get(ctx, types.NamespacedName{Name: authentikBlueprintConfigMapName, Namespace: authentikNamespace}, &configMap); err != nil {
		t.Fatal(err)
	}
	blueprint := configMap.Data[authentikBlueprintKey]
	if !strings.Contains(blueprint, "txo-fabric-hairem-client0") || !strings.Contains(blueprint, "txo-fabric-indiba-sales") {
		t.Fatalf("aggregate blueprint does not contain both tenants:\n%s", blueprint)
	}
	if configMap.Labels[LabelManaged] != "true" {
		t.Fatalf("generated blueprint ConfigMap is not Fabric-managed: %#v", configMap.Labels)
	}
}

func TestReconcileAuthentikBlueprintDoesNotRequireAuthForNoHumanAccess(t *testing.T) {
	ctx := context.Background()
	scheme := testScheme(t)
	tenant := testTenant()
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(tenant).Build()
	r := &TenantBundleReconciler{Client: c, Scheme: scheme}

	if err := r.reconcileAuthentikBlueprint(ctx); err != nil {
		t.Fatalf("tenant without human access unexpectedly depends on Authentik: %v", err)
	}
}
