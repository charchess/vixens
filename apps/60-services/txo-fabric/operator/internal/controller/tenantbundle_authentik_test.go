package controller

import (
	"context"
	"strings"
	"testing"

	fabricv1alpha1 "github.com/charchess/vixens/apps/60-services/txo-fabric/operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
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



type countingClient struct {
	client.Client
	creates int
	updates int
	patches int
}

func (c *countingClient) Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
	c.creates++
	return c.Client.Create(ctx, obj, opts...)
}

func (c *countingClient) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	c.updates++
	return c.Client.Update(ctx, obj, opts...)
}

func (c *countingClient) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
	c.patches++
	return c.Client.Patch(ctx, obj, patch, opts...)
}

func TestRenderAuthentikBlueprintExcludesInvalidTenantWithoutAffectingValidTenant(t *testing.T) {
	hairem := tenantWithIAM("hairem", "hAIrem", "client0", true)
	indiba := tenantWithIAM("indiba", "Indiba", "sales", false)
	indiba.Spec.HumanAccess.Web.IAMGroups = nil

	blueprint := renderAuthentikBlueprint([]fabricv1alpha1.TenantBundle{*hairem, *indiba})
	if !strings.Contains(blueprint, "txo-fabric-hairem-client0") {
		t.Fatalf("valid tenant disappeared when another tenant became invalid:\n%s", blueprint)
	}
	if strings.Contains(blueprint, "txo-fabric-indiba") {
		t.Fatalf("invalid tenant remained in aggregate IAM desired state:\n%s", blueprint)
	}
}

func TestRenderAuthentikBlueprintExcludesDeletingTenantWithoutAffectingOthers(t *testing.T) {
	hairem := tenantWithIAM("hairem", "hAIrem", "client0", true)
	indiba := tenantWithIAM("indiba", "Indiba", "sales", false)
	now := metav1.Now()
	indiba.DeletionTimestamp = &now

	blueprint := renderAuthentikBlueprint([]fabricv1alpha1.TenantBundle{*indiba, *hairem})
	if !strings.Contains(blueprint, "txo-fabric-hairem-client0") {
		t.Fatalf("active tenant disappeared while another tenant was deleting:\n%s", blueprint)
	}
	if strings.Contains(blueprint, "txo-fabric-indiba") {
		t.Fatalf("deleting tenant remained in aggregate IAM desired state:\n%s", blueprint)
	}
}

func TestRenderAuthentikBlueprintNeverPublishesHumanUsersOrMemberships(t *testing.T) {
	tenant := tenantWithIAM("indiba", "Indiba", "sales", false)
	tenant.Spec.Workspace = &fabricv1alpha1.TenantWorkspaceSpec{
		Users: []fabricv1alpha1.NamedWorkspaceScopeSpec{{
			Name:          "edfoley",
			Collaborative: true,
		}},
	}

	blueprint := renderAuthentikBlueprint([]fabricv1alpha1.TenantBundle{*tenant})
	for _, forbidden := range []string{
		"authentik_core.user",
		"edfoley",
		"users:",
		"members:",
	} {
		if strings.Contains(blueprint, forbidden) {
			t.Fatalf("Fabric IAM blueprint leaked human IAM state %q:\n%s", forbidden, blueprint)
		}
	}
}

func TestRenderAuthentikBlueprintSortsMultipleIAMGroupsDeterministically(t *testing.T) {
	tenant := tenantWithIAM("acme", "Acme", "support", false)
	tenant.Spec.HumanAccess.Web.IAMGroups = []string{"support", "admin"}

	blueprint := renderAuthentikBlueprint([]fabricv1alpha1.TenantBundle{*tenant})
	admin := strings.Index(blueprint, "txo-fabric-acme-admin")
	support := strings.Index(blueprint, "txo-fabric-acme-support")
	if admin < 0 || support < 0 {
		t.Fatalf("multi-group blueprint missing expected groups:\n%s", blueprint)
	}
	if admin > support {
		t.Fatalf("IAM groups are not rendered deterministically:\n%s", blueprint)
	}
	if got := strings.Count(blueprint, "model: authentik_policies.policybinding"); got != 2 {
		t.Fatalf("policy bindings=%d, want 2 for two authorized IAM groups:\n%s", got, blueprint)
	}
	for _, want := range []string{"order: 0", "order: 10"} {
		if !strings.Contains(blueprint, want) {
			t.Fatalf("multi-group policy binding missing %q:\n%s", want, blueprint)
		}
	}
}

func TestReconcileAuthentikBlueprintIsNoOpWhenDesiredStateIsUnchanged(t *testing.T) {
	ctx := context.Background()
	scheme := testScheme(t)
	hairem := tenantWithIAM("hairem", "hAIrem", "client0", true)
	indiba := tenantWithIAM("indiba", "Indiba", "sales", false)

	base := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: authentikNamespace}},
		hairem,
		indiba,
	).Build()
	c := &countingClient{Client: base}
	r := &TenantBundleReconciler{Client: c, Scheme: scheme}

	if err := r.reconcileAuthentikBlueprint(ctx); err != nil {
		t.Fatal(err)
	}

	c.creates = 0
	c.updates = 0
	c.patches = 0
	if err := r.reconcileAuthentikBlueprint(ctx); err != nil {
		t.Fatal(err)
	}
	if c.creates != 0 || c.updates != 0 || c.patches != 0 {
		t.Fatalf("no-op IAM reconcile mutated API objects: creates=%d updates=%d patches=%d", c.creates, c.updates, c.patches)
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

func TestTenantIAMValidationMatrix(t *testing.T) {
	tests := []struct {
		name      string
		mutate    func(*fabricv1alpha1.TenantBundle)
		wantError string
	}{
		{
			name: "valid",
		},
		{
			name: "human access absent",
			mutate: func(tenant *fabricv1alpha1.TenantBundle) {
				tenant.Spec.HumanAccess = nil
			},
		},
		{
			name: "web access absent",
			mutate: func(tenant *fabricv1alpha1.TenantBundle) {
				tenant.Spec.HumanAccess.Web = nil
			},
		},
		{
			name: "missing iam group",
			mutate: func(tenant *fabricv1alpha1.TenantBundle) {
				tenant.Spec.HumanAccess.Web.IAMGroups = nil
			},
			wantError: "iamGroups",
		},
		{
			name: "uppercase iam group rejected",
			mutate: func(tenant *fabricv1alpha1.TenantBundle) {
				tenant.Spec.HumanAccess.Web.IAMGroups = []string{"Sales"}
			},
			wantError: "lowercase DNS-like key",
		},
		{
			name: "unsafe iam group rejected",
			mutate: func(tenant *fabricv1alpha1.TenantBundle) {
				tenant.Spec.HumanAccess.Web.IAMGroups = []string{"sales/admin"}
			},
			wantError: "lowercase DNS-like key",
		},
		{
			name: "duplicate iam group rejected",
			mutate: func(tenant *fabricv1alpha1.TenantBundle) {
				tenant.Spec.HumanAccess.Web.IAMGroups = []string{"sales", "sales"}
			},
			wantError: "repeats iamGroup",
		},
		{
			name: "invalid domain rejected",
			mutate: func(tenant *fabricv1alpha1.TenantBundle) {
				tenant.Spec.HumanAccess.Web.DomainSuffix = "truxonline..com"
			},
			wantError: "valid DNS suffix",
		},
		{
			name: "non https issuer rejected",
			mutate: func(tenant *fabricv1alpha1.TenantBundle) {
				tenant.Spec.HumanAccess.Web.OIDC.Issuer = "http://authentik.truxonline.com/application/o/txo-fabric-indiba/"
			},
			wantError: "absolute https URL",
		},
		{
			name: "issuer path must match tenant application",
			mutate: func(tenant *fabricv1alpha1.TenantBundle) {
				tenant.Spec.HumanAccess.Web.OIDC.Issuer = "https://authentik.truxonline.com/application/o/txo-fabric-other/"
			},
			wantError: "must match generated Authentik application",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tenant := tenantWithIAM("indiba", "Indiba", "sales", false)
			if test.mutate != nil {
				test.mutate(tenant)
			}
			err := validateTenantIAM(tenant)
			if test.wantError == "" {
				if err != nil {
					t.Fatalf("unexpected validation error: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), test.wantError) {
				t.Fatalf("validation error=%v, want substring %q", err, test.wantError)
			}
		})
	}
}

func TestRenderAuthentikBlueprintWithNoActiveTenantIsExplicitlyEmpty(t *testing.T) {
	blueprint := renderAuthentikBlueprint(nil)
	if blueprint != "version: 1\nmetadata:\n  name: txo-fabric-tenants-generated\nentries: []\n" {
		t.Fatalf("empty blueprint changed unexpectedly:\n%s", blueprint)
	}
}

func TestReconcileAuthentikBlueprintRepairsDrift(t *testing.T) {
	ctx := context.Background()
	scheme := testScheme(t)
	tenant := tenantWithIAM("indiba", "Indiba", "sales", false)
	stale := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      authentikBlueprintConfigMapName,
			Namespace: authentikNamespace,
			Labels:    map[string]string{"stale": "true"},
		},
		Data: map[string]string{authentikBlueprintKey: "stale"},
	}
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: authentikNamespace}},
		tenant,
		stale,
	).Build()
	r := &TenantBundleReconciler{Client: c, Scheme: scheme}

	if err := r.reconcileAuthentikBlueprint(ctx); err != nil {
		t.Fatal(err)
	}

	var current corev1.ConfigMap
	if err := c.Get(ctx, types.NamespacedName{Name: authentikBlueprintConfigMapName, Namespace: authentikNamespace}, &current); err != nil {
		t.Fatal(err)
	}
	if current.Data[authentikBlueprintKey] == "stale" || !strings.Contains(current.Data[authentikBlueprintKey], "txo-fabric-indiba-sales") {
		t.Fatalf("drifted blueprint was not repaired:\n%s", current.Data[authentikBlueprintKey])
	}
	if current.Labels[LabelManaged] != "true" || current.Labels["app.kubernetes.io/component"] != "tenant-iam" {
		t.Fatalf("managed labels were not repaired: %#v", current.Labels)
	}
	if _, ok := current.Labels["stale"]; ok {
		t.Fatalf("stale labels survived desired-state repair: %#v", current.Labels)
	}
}

func TestReconcileAuthentikBlueprintWithdrawsDeletingTenant(t *testing.T) {
	ctx := context.Background()
	scheme := testScheme(t)
	hairem := tenantWithIAM("hairem", "hAIrem", "client0", true)
	indiba := tenantWithIAM("indiba", "Indiba", "sales", false)

	initial := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: authentikBlueprintConfigMapName, Namespace: authentikNamespace},
		Data: map[string]string{
			authentikBlueprintKey: renderAuthentikBlueprint([]fabricv1alpha1.TenantBundle{*hairem, *indiba}),
		},
	}
	now := metav1.Now()
	indiba.DeletionTimestamp = &now
	indiba.Finalizers = []string{TenantFinalizer}

	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: authentikNamespace}},
		hairem,
		indiba,
		initial,
	).Build()
	r := &TenantBundleReconciler{Client: c, Scheme: scheme}

	if err := r.reconcileAuthentikBlueprint(ctx); err != nil {
		t.Fatal(err)
	}

	var current corev1.ConfigMap
	if err := c.Get(ctx, types.NamespacedName{Name: authentikBlueprintConfigMapName, Namespace: authentikNamespace}, &current); err != nil {
		t.Fatal(err)
	}
	blueprint := current.Data[authentikBlueprintKey]
	if !strings.Contains(blueprint, "txo-fabric-hairem-client0") {
		t.Fatalf("active hAIrem tenant disappeared after Indiba withdrawal:\n%s", blueprint)
	}
	if strings.Contains(blueprint, "txo-fabric-indiba") {
		t.Fatalf("deleting Indiba tenant remained in blueprint:\n%s", blueprint)
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
