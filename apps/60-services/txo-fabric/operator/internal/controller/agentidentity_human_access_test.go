package controller

import (
	"context"
	"strings"
	"testing"

	fabricv1alpha1 "github.com/charchess/vixens/apps/60-services/txo-fabric/operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestHumanAccessReconcilesStableAuthenticatedDashboard(t *testing.T) {
	ctx := context.Background()
	scheme := testScheme(t)
	tenant := humanAccessTestTenant()
	profile := testRuntimeProfile()
	agent := humanAccessTestAgent()

	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(agent).Build()
	r := &AgentIdentityReconciler{Client: c, Scheme: scheme}
	namespace := tenantNamespace(tenant.Name)

	access, err := resolveHumanAccess(agent, tenant)
	if err != nil {
		t.Fatal(err)
	}
	if !access.Enabled {
		t.Fatal("human access unexpectedly disabled")
	}
	if access.Host != "tesla-hairem-sandbox.truxonline.com" {
		t.Fatalf("host=%q", access.Host)
	}
	if access.FilesRoot != "/workspace/shared/users/client0/collaborative" {
		t.Fatalf("files root=%q", access.FilesRoot)
	}

	if err := r.ensureDeployment(ctx, agent, tenant, profile, namespace, "model-secret-uid", "baseline"); err != nil {
		t.Fatal(err)
	}
	if err := r.ensureHumanAccessResources(ctx, agent, tenant, namespace, access); err != nil {
		t.Fatal(err)
	}
	if err := r.ensureEgressPolicy(ctx, agent, tenant, namespace); err != nil {
		t.Fatal(err)
	}

	var deployment appsv1.Deployment
	if err := c.Get(ctx, types.NamespacedName{Name: runtimeName(agent.Spec.AgentKey), Namespace: namespace}, &deployment); err != nil {
		t.Fatal(err)
	}
	container := deployment.Spec.Template.Spec.Containers[0]
	for key, want := range map[string]string{
		"HERMES_DASHBOARD":                "1",
		"HERMES_DASHBOARD_HOST":           "0.0.0.0",
		"HERMES_DASHBOARD_PORT":           "9119",
		"HERMES_DASHBOARD_PUBLIC_URL":     "https://tesla-hairem-sandbox.truxonline.com",
		"HERMES_DASHBOARD_OIDC_ISSUER":    "https://authentik.truxonline.com/application/o/txo-fabric-hairem/",
		"HERMES_DASHBOARD_OIDC_CLIENT_ID": "txo-fabric-hairem",
		"HERMES_DASHBOARD_OIDC_SCOPES":    "openid profile email",
		"HERMES_DASHBOARD_FILES_ROOT":     "/workspace/shared/users/client0/collaborative",
	} {
		if got := envValue(container.Env, key); got != want {
			t.Fatalf("%s=%q want %q", key, got, want)
		}
	}
	if len(container.Ports) != 1 || container.Ports[0].ContainerPort != hermesDashboardPort {
		t.Fatalf("dashboard ports=%#v", container.Ports)
	}

	name := humanAccessResourceName(agent.Spec.AgentKey)
	var service corev1.Service
	if err := c.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, &service); err != nil {
		t.Fatal(err)
	}
	if len(service.Spec.Ports) != 1 || service.Spec.Ports[0].Port != hermesDashboardPort {
		t.Fatalf("service ports=%#v", service.Spec.Ports)
	}

	var ingress networkingv1.Ingress
	if err := c.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, &ingress); err != nil {
		t.Fatal(err)
	}
	if len(ingress.Spec.Rules) != 1 || ingress.Spec.Rules[0].Host != access.Host {
		t.Fatalf("ingress rules=%#v", ingress.Spec.Rules)
	}
	if ingress.Annotations["cert-manager.io/cluster-issuer"] != "letsencrypt-prod" {
		t.Fatalf("cluster issuer=%q", ingress.Annotations["cert-manager.io/cluster-issuer"])
	}
	if ingress.Annotations["external-dns.alpha.kubernetes.io/public"] != "true" {
		t.Fatalf("public DNS annotation=%q", ingress.Annotations["external-dns.alpha.kubernetes.io/public"])
	}
	if ingress.Annotations["external-dns.alpha.kubernetes.io/target"] != "truxonline.com" {
		t.Fatalf("DNS target annotation=%q", ingress.Annotations["external-dns.alpha.kubernetes.io/target"])
	}

	var ingressPolicy networkingv1.NetworkPolicy
	if err := c.Get(ctx, types.NamespacedName{Name: name + "-ingress", Namespace: namespace}, &ingressPolicy); err != nil {
		t.Fatal(err)
	}
	if len(ingressPolicy.Spec.Ingress) != 1 {
		t.Fatalf("dashboard ingress rules=%d", len(ingressPolicy.Spec.Ingress))
	}

	tenant.Spec.HumanAccess.Web.PublicDNS = false
	tenant.Spec.HumanAccess.Web.DNSTarget = ""
	internalOnly, err := resolveHumanAccess(agent, tenant)
	if err != nil {
		t.Fatal(err)
	}
	if err := r.ensureHumanAccessResources(ctx, agent, tenant, namespace, internalOnly); err != nil {
		t.Fatal(err)
	}
	if err := c.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, &ingress); err != nil {
		t.Fatal(err)
	}
	if _, ok := ingress.Annotations["external-dns.alpha.kubernetes.io/public"]; ok {
		t.Fatalf("stale public DNS annotation remains: %#v", ingress.Annotations)
	}
	if _, ok := ingress.Annotations["external-dns.alpha.kubernetes.io/target"]; ok {
		t.Fatalf("stale DNS target annotation remains: %#v", ingress.Annotations)
	}

	var egress networkingv1.NetworkPolicy
	if err := c.Get(ctx, types.NamespacedName{Name: runtimeName(agent.Spec.AgentKey) + "-egress", Namespace: namespace}, &egress); err != nil {
		t.Fatal(err)
	}
	if len(egress.Spec.Egress) != 2 {
		t.Fatalf("egress rules=%d want DNS + AI gateway only", len(egress.Spec.Egress))
	}

	oidcPolicy := humanOIDCEgressPolicyObject(agent.Spec.AgentKey, namespace)
	if err := c.Get(ctx, types.NamespacedName{Name: oidcPolicy.GetName(), Namespace: namespace}, oidcPolicy); err != nil {
		t.Fatal(err)
	}
	ciliumEgress, found, err := unstructured.NestedSlice(oidcPolicy.Object, "spec", "egress")
	if err != nil || !found || len(ciliumEgress) != 3 {
		t.Fatalf("Cilium OIDC egress rules=%#v found=%v err=%v", ciliumEgress, found, err)
	}
	fqdnRule, ok := ciliumEgress[1].(map[string]any)
	if !ok {
		t.Fatalf("FQDN rule has unexpected type: %#v", ciliumEgress[1])
	}
	toFQDNs, ok := fqdnRule["toFQDNs"].([]any)
	if !ok || len(toFQDNs) != 1 {
		t.Fatalf("toFQDNs=%#v", fqdnRule["toFQDNs"])
	}
	fqdn, ok := toFQDNs[0].(map[string]any)
	if !ok || fqdn["matchName"] != "authentik.truxonline.com" {
		t.Fatalf("OIDC FQDN=%#v", toFQDNs[0])
	}
	serviceRule, ok := ciliumEgress[2].(map[string]any)
	if !ok {
		t.Fatalf("service rule has unexpected type: %#v", ciliumEgress[2])
	}
	toServices, ok := serviceRule["toServices"].([]any)
	if !ok || len(toServices) != 1 {
		t.Fatalf("toServices=%#v", serviceRule["toServices"])
	}
	serviceEntry, ok := toServices[0].(map[string]any)
	if !ok {
		t.Fatalf("service entry=%#v", toServices[0])
	}
	k8sService, ok := serviceEntry["k8sService"].(map[string]any)
	if !ok || k8sService["serviceName"] != "traefik" || k8sService["namespace"] != "traefik" {
		t.Fatalf("OIDC Traefik service=%#v", serviceEntry["k8sService"])
	}

	agent.Spec.HumanAccess.Enabled = false
	disabled, err := resolveHumanAccess(agent, tenant)
	if err != nil {
		t.Fatal(err)
	}
	if err := r.ensureHumanAccessResources(ctx, agent, tenant, namespace, disabled); err != nil {
		t.Fatal(err)
	}
	if err := r.ensureDeployment(ctx, agent, tenant, profile, namespace, "model-secret-uid", "baseline"); err != nil {
		t.Fatal(err)
	}

	for _, obj := range []struct {
		name string
		get  func() error
	}{
		{"service", func() error { return c.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, &corev1.Service{}) }},
		{"ingress", func() error { return c.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, &networkingv1.Ingress{}) }},
		{"ingress-policy", func() error { return c.Get(ctx, types.NamespacedName{Name: name + "-ingress", Namespace: namespace}, &networkingv1.NetworkPolicy{}) }},
		{"oidc-egress-policy", func() error {
			obj := humanOIDCEgressPolicyObject(agent.Spec.AgentKey, namespace)
			return c.Get(ctx, types.NamespacedName{Name: obj.GetName(), Namespace: namespace}, obj)
		}},
	} {
		if err := obj.get(); !apierrors.IsNotFound(err) {
			t.Fatalf("%s remains after disable: %v", obj.name, err)
		}
	}
	if err := c.Get(ctx, types.NamespacedName{Name: runtimeName(agent.Spec.AgentKey), Namespace: namespace}, &deployment); err != nil {
		t.Fatal(err)
	}
	if got := envValue(deployment.Spec.Template.Spec.Containers[0].Env, "HERMES_DASHBOARD"); got != "" {
		t.Fatalf("dashboard env remains after disable: %q", got)
	}
}

func TestResolveHumanAccessValidationMatrix(t *testing.T) {
	tests := []struct {
		name      string
		mutate    func(*fabricv1alpha1.TenantBundle, *fabricv1alpha1.AgentIdentity)
		wantError string
	}{
		{
			name: "disabled agent ignores incomplete tenant human policy",
			mutate: func(tenant *fabricv1alpha1.TenantBundle, agent *fabricv1alpha1.AgentIdentity) {
				agent.Spec.HumanAccess.Enabled = false
				tenant.Spec.HumanAccess = nil
			},
		},
		{
			name: "tenant web policy required",
			mutate: func(tenant *fabricv1alpha1.TenantBundle, agent *fabricv1alpha1.AgentIdentity) {
				tenant.Spec.HumanAccess = nil
			},
			wantError: "has no humanAccess.web policy",
		},
		{
			name: "https issuer required",
			mutate: func(tenant *fabricv1alpha1.TenantBundle, agent *fabricv1alpha1.AgentIdentity) {
				tenant.Spec.HumanAccess.Web.OIDC.Issuer = "http://authentik.truxonline.com/application/o/txo-fabric-hairem/"
			},
			wantError: "absolute https URL",
		},
		{
			name: "user ref required",
			mutate: func(tenant *fabricv1alpha1.TenantBundle, agent *fabricv1alpha1.AgentIdentity) {
				agent.Spec.Access.UserRef = ""
			},
			wantError: "requires access.userRef",
		},
		{
			name: "workspace required",
			mutate: func(tenant *fabricv1alpha1.TenantBundle, agent *fabricv1alpha1.AgentIdentity) {
				tenant.Spec.Workspace = nil
			},
			wantError: "requires TenantBundle",
		},
		{
			name: "workspace user must exist",
			mutate: func(tenant *fabricv1alpha1.TenantBundle, agent *fabricv1alpha1.AgentIdentity) {
				agent.Spec.Access.UserRef = "missing-user"
			},
			wantError: "requires collaborative workspace user",
		},
		{
			name: "workspace user must be collaborative",
			mutate: func(tenant *fabricv1alpha1.TenantBundle, agent *fabricv1alpha1.AgentIdentity) {
				tenant.Spec.Workspace.Users[0].Collaborative = false
			},
			wantError: "requires collaborative workspace user",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tenant := humanAccessTestTenant()
			agent := humanAccessTestAgent()
			if test.mutate != nil {
				test.mutate(tenant, agent)
			}
			access, err := resolveHumanAccess(agent, tenant)
			if test.wantError == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if agent.Spec.HumanAccess.Enabled == false && access.Enabled {
					t.Fatal("disabled human access unexpectedly resolved as enabled")
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), test.wantError) {
				t.Fatalf("resolve error=%v, want substring %q", err, test.wantError)
			}
		})
	}
}

func TestResolveHumanAccessDefaultsAreStable(t *testing.T) {
	tenant := humanAccessTestTenant()
	tenant.Spec.HumanAccess.Web.IngressClassName = ""
	tenant.Spec.HumanAccess.Web.OIDC.Scopes = ""
	agent := humanAccessTestAgent()

	access, err := resolveHumanAccess(agent, tenant)
	if err != nil {
		t.Fatal(err)
	}
	if access.IngressClass != "traefik" {
		t.Fatalf("default ingress class=%q, want traefik", access.IngressClass)
	}
	if access.OIDCScopes != "openid profile email" {
		t.Fatalf("default OIDC scopes=%q", access.OIDCScopes)
	}
}

func TestHumanAccessRequiresCollaborativeUserWorkspace(t *testing.T) {
	tenant := humanAccessTestTenant()
	agent := humanAccessTestAgent()
	tenant.Spec.Workspace.Users[0].Collaborative = false
	if _, err := resolveHumanAccess(agent, tenant); err == nil {
		t.Fatal("human access without collaborative user workspace must fail closed")
	}
}

func humanAccessTestTenant() *fabricv1alpha1.TenantBundle {
	return &fabricv1alpha1.TenantBundle{
		ObjectMeta: metav1.ObjectMeta{Name: "hairem-sandbox"},
		Spec: fabricv1alpha1.TenantBundleSpec{
			TenantID:    "TEN90001",
			DisplayName: "hAIrem Sandbox",
			Workspace: &fabricv1alpha1.TenantWorkspaceSpec{
				ProfileRef: "shared-nfs",
				Users: []fabricv1alpha1.NamedWorkspaceScopeSpec{{
					Name:          "client0",
					Collaborative: true,
				}},
			},
			HumanAccess: &fabricv1alpha1.TenantHumanAccessSpec{
				Web: &fabricv1alpha1.HumanWebAccessSpec{
					DomainSuffix:     "truxonline.com",
					IngressClassName:  "traefik",
					TLSClusterIssuer:  "letsencrypt-prod",
					PublicDNS:         true,
					DNSTarget:         "truxonline.com",
					OIDC: fabricv1alpha1.HumanAccessOIDCSpec{
						Issuer:   "https://authentik.truxonline.com/application/o/txo-fabric-hairem/",
						ClientID: "txo-fabric-hairem",
					},
				},
			},
		},
	}
}

func humanAccessTestAgent() *fabricv1alpha1.AgentIdentity {
	return &fabricv1alpha1.AgentIdentity{
		ObjectMeta: metav1.ObjectMeta{Name: "hairem-sandbox-tesla"},
		Spec: fabricv1alpha1.AgentIdentitySpec{
			TenantRef:   fabricv1alpha1.ObjectReference{Name: "hairem-sandbox"},
			AgentKey:    "tesla",
			DisplayName: "Tesla",
			Access:      fabricv1alpha1.AgentAccessSpec{UserRef: "client0"},
			HumanAccess: fabricv1alpha1.AgentHumanAccessBinding{Enabled: true},
		},
	}
}
