package controller

import (
	"bytes"
	"context"
	"net/url"
	"testing"

	fabricv1alpha1 "github.com/charchess/vixens/apps/60-services/txo-fabric/operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestHindsightProfileMissingIsExplicitlyBlocked(t *testing.T) {
	ctx := context.Background()
	scheme := postgresqlTestScheme(t)
	tenant := hindsightTestTenant()
	tenant.Status.Persistence.PostgreSQL = &fabricv1alpha1.ComponentStatus{Phase: "Ready"}
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(tenant).Build()

	result, err := (&TenantBundleReconciler{Client: c, Scheme: scheme}).reconcileHindsight(ctx, tenant)
	if err != nil {
		t.Fatal(err)
	}
	if result.Ready || result.Reason != "ProfileNotFound" || result.Status == nil || result.Status.Phase != "Blocked" {
		t.Fatalf("unexpected Hindsight result: %#v", result)
	}
}

func TestHindsightWaitsForPostgreSQLBinding(t *testing.T) {
	ctx := context.Background()
	scheme := postgresqlTestScheme(t)
	tenant := hindsightTestTenant()
	profile := hindsightTestProfile()
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(tenant, profile).Build()

	result, err := (&TenantBundleReconciler{Client: c, Scheme: scheme}).reconcileHindsight(ctx, tenant)
	if err != nil {
		t.Fatal(err)
	}
	if result.Ready || result.Reason != "PostgreSQLPending" || result.Status == nil || result.Status.Phase != "Pending" {
		t.Fatalf("unexpected Hindsight result: %#v", result)
	}
}

func TestTenantScopedHindsightReconcilesSecretDeploymentServiceAndNetwork(t *testing.T) {
	ctx := context.Background()
	scheme := postgresqlTestScheme(t)
	tenant := hindsightTestTenant()
	tenant.Status.Persistence.PostgreSQL = &fabricv1alpha1.ComponentStatus{Phase: "Ready"}
	hindsightProfile := hindsightTestProfile()
	postgresqlProfile := postgresqlTestProfile()
	postgresqlSecret := hindsightPostgreSQLSecret(tenant, postgresqlProfile)

	c := fake.NewClientBuilder().WithScheme(scheme).
		WithStatusSubresource(&appsv1.Deployment{}).
		WithObjects(tenant, hindsightProfile, postgresqlProfile, postgresqlSecret).
		Build()
	r := &TenantBundleReconciler{Client: c, Scheme: scheme}

	result, err := r.reconcileHindsight(ctx, tenant)
	if err != nil {
		t.Fatal(err)
	}
	if result.Ready || result.Reason != "DeploymentProgressing" {
		t.Fatalf("unexpected initial Hindsight result: %#v", result)
	}

	namespace := tenantNamespace(tenant.Name)
	var runtimeSecret corev1.Secret
	if err := c.Get(ctx, types.NamespacedName{Namespace: namespace, Name: "hindsight-runtime"}, &runtimeSecret); err != nil {
		t.Fatalf("runtime Secret not reconciled: %v", err)
	}
	if got := string(runtimeSecret.Data["HINDSIGHT_API_LLM_PROVIDER"]); got != "none" {
		t.Fatalf("HINDSIGHT_API_LLM_PROVIDER=%q, want none", got)
	}
	if got := string(runtimeSecret.Data["HINDSIGHT_API_TENANT_EXTENSION"]); got != "hindsight_api.extensions.builtin.tenant:ApiKeyTenantExtension" {
		t.Fatalf("unexpected tenant extension %q", got)
	}
	apiKey := append([]byte(nil), runtimeSecret.Data["HINDSIGHT_API_TENANT_API_KEY"]...)
	if len(apiKey) == 0 {
		t.Fatal("Hindsight API key was not generated")
	}
	databaseURL := string(runtimeSecret.Data["HINDSIGHT_API_DATABASE_URL"])
	parsedDatabaseURL, err := url.Parse(databaseURL)
	if err != nil {
		t.Fatalf("invalid Hindsight database URL: %v", err)
	}
	password, hasPassword := parsedDatabaseURL.User.Password()
	if parsedDatabaseURL.Scheme != "postgresql" || parsedDatabaseURL.User.Username() != "txo_ten90001" || !hasPassword || password != string(postgresqlSecret.Data[corev1.BasicAuthPasswordKey]) || parsedDatabaseURL.Host != "postgresql-shared-rw.databases.svc:5432" || parsedDatabaseURL.Path != "/txo_ten90001" {
		t.Fatalf("unexpected Hindsight database URL components: scheme=%q user=%q host=%q path=%q", parsedDatabaseURL.Scheme, parsedDatabaseURL.User.Username(), parsedDatabaseURL.Host, parsedDatabaseURL.Path)
	}

	var deployment appsv1.Deployment
	if err := c.Get(ctx, types.NamespacedName{Namespace: namespace, Name: "hindsight"}, &deployment); err != nil {
		t.Fatalf("Hindsight Deployment not reconciled: %v", err)
	}
	container := deployment.Spec.Template.Spec.Containers[0]
	if container.Image != hindsightProfile.Spec.Image {
		t.Fatalf("Hindsight image=%q", container.Image)
	}
	if len(container.EnvFrom) != 1 || container.EnvFrom[0].SecretRef == nil || container.EnvFrom[0].SecretRef.Name != "hindsight-runtime" {
		t.Fatalf("Hindsight runtime Secret is not injected through envFrom: %#v", container.EnvFrom)
	}
	if container.StartupProbe == nil || container.StartupProbe.Exec == nil || container.ReadinessProbe == nil || container.ReadinessProbe.Exec == nil || container.LivenessProbe == nil || container.LivenessProbe.Exec == nil {
		t.Fatal("Hindsight exec health probes are missing")
	}
	if deployment.Spec.Template.Spec.SecurityContext == nil || deployment.Spec.Template.Spec.SecurityContext.FSGroup == nil || *deployment.Spec.Template.Spec.SecurityContext.FSGroup != 1000 {
		t.Fatalf("unexpected Hindsight pod security context: %#v", deployment.Spec.Template.Spec.SecurityContext)
	}
	if container.SecurityContext == nil || container.SecurityContext.RunAsNonRoot == nil || !*container.SecurityContext.RunAsNonRoot || container.SecurityContext.AllowPrivilegeEscalation == nil || *container.SecurityContext.AllowPrivilegeEscalation {
		t.Fatalf("unexpected Hindsight container security context: %#v", container.SecurityContext)
	}

	var service corev1.Service
	if err := c.Get(ctx, types.NamespacedName{Namespace: namespace, Name: "hindsight"}, &service); err != nil {
		t.Fatalf("Hindsight Service not reconciled: %v", err)
	}
	if len(service.Spec.Ports) != 1 || service.Spec.Ports[0].Port != 8888 {
		t.Fatalf("unexpected Hindsight Service ports: %#v", service.Spec.Ports)
	}

	var policy networkingv1.NetworkPolicy
	if err := c.Get(ctx, types.NamespacedName{Namespace: namespace, Name: "hindsight-access"}, &policy); err != nil {
		t.Fatalf("Hindsight NetworkPolicy not reconciled: %v", err)
	}
	if len(policy.Spec.Ingress) != 1 || len(policy.Spec.Egress) != 2 {
		t.Fatalf("unexpected Hindsight network contract: ingress=%d egress=%d", len(policy.Spec.Ingress), len(policy.Spec.Egress))
	}

	deployment.Status.ObservedGeneration = deployment.Generation
	deployment.Status.AvailableReplicas = 1
	if err := c.Status().Update(ctx, &deployment); err != nil {
		t.Fatal(err)
	}
	result, err = r.reconcileHindsight(ctx, tenant)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Ready || result.Reason != "Reconciled" || result.Status == nil || result.Status.Endpoint != "http://hindsight.tenant-hairem-sandbox.svc:8888" {
		t.Fatalf("unexpected ready Hindsight result: %#v", result)
	}

	// Rotating the PostgreSQL password must update the local runtime binding but
	// must never rotate the independent Hindsight API key.
	var source corev1.Secret
	if err := c.Get(ctx, client.ObjectKeyFromObject(postgresqlSecret), &source); err != nil {
		t.Fatal(err)
	}
	source.Data[corev1.BasicAuthPasswordKey] = []byte("rotated-postgresql-password")
	if err := c.Update(ctx, &source); err != nil {
		t.Fatal(err)
	}
	if _, err := r.reconcileHindsight(ctx, tenant); err != nil {
		t.Fatal(err)
	}
	var after corev1.Secret
	if err := c.Get(ctx, types.NamespacedName{Namespace: namespace, Name: "hindsight-runtime"}, &after); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(apiKey, after.Data["HINDSIGHT_API_TENANT_API_KEY"]) {
		t.Fatal("idempotent Hindsight reconciliation rotated the API key")
	}
	afterURL, err := url.Parse(string(after.Data["HINDSIGHT_API_DATABASE_URL"]))
	if err != nil {
		t.Fatalf("invalid rotated Hindsight database URL: %v", err)
	}
	afterPassword, hasPassword := afterURL.User.Password()
	if !hasPassword || afterPassword != "rotated-postgresql-password" {
		t.Fatal("Hindsight database binding did not follow PostgreSQL credential rotation")
	}
}

func TestHindsightHumanAccessRequiresExplicitMemoryOptIn(t *testing.T) {
	ctx := context.Background()
	scheme := postgresqlTestScheme(t)
	tenant := hindsightTestTenant()
	tenant.Spec.HumanAccess = &fabricv1alpha1.TenantHumanAccessSpec{Web: &fabricv1alpha1.HumanWebAccessSpec{
		DomainSuffix:     "truxonline.com",
		TLSClusterIssuer: "letsencrypt-prod",
		OIDC: fabricv1alpha1.HumanAccessOIDCSpec{
			Issuer:   "https://authentik.truxonline.com/application/o/example/",
			ClientID: "example",
		},
	}}
	tenant.Status.Persistence.PostgreSQL = &fabricv1alpha1.ComponentStatus{Phase: "Ready"}
	hindsightProfile := hindsightTestProfile()
	postgresqlProfile := postgresqlTestProfile()
	postgresqlSecret := hindsightPostgreSQLSecret(tenant, postgresqlProfile)

	c := fake.NewClientBuilder().WithScheme(scheme).
		WithStatusSubresource(&appsv1.Deployment{}).
		WithObjects(tenant, hindsightProfile, postgresqlProfile, postgresqlSecret).
		Build()
	r := &TenantBundleReconciler{Client: c, Scheme: scheme}

	if _, err := r.reconcileHindsight(ctx, tenant); err != nil {
		t.Fatal(err)
	}
	var controlPlane appsv1.Deployment
	err := c.Get(ctx, types.NamespacedName{Namespace: tenantNamespace(tenant.Name), Name: "hindsight-control-plane"}, &controlPlane)
	if !apierrors.IsNotFound(err) {
		t.Fatalf("Hindsight Control Plane must remain absent without memory.hindsight.humanAccess opt-in: %v", err)
	}
}

func TestHindsightHumanAccessReconcilesAuthenticatedControlPlane(t *testing.T) {
	ctx := context.Background()
	scheme := postgresqlTestScheme(t)
	tenant := hindsightTestTenant()
	tenant.Spec.Memory.Hindsight.HumanAccess = true
	tenant.Spec.HumanAccess = &fabricv1alpha1.TenantHumanAccessSpec{Web: &fabricv1alpha1.HumanWebAccessSpec{
		DomainSuffix:       "truxonline.com",
		IngressClassName:   "traefik",
		TLSClusterIssuer:   "letsencrypt-prod",
		PublicDNS:          true,
		DNSTarget:          "truxonline.com",
		OIDC: fabricv1alpha1.HumanAccessOIDCSpec{
			Issuer:   "https://authentik.truxonline.com/application/o/txo-fabric-hairem/",
			ClientID: "txo-fabric-hairem",
			Scopes:   "openid profile email",
		},
	}}
	tenant.Status.Persistence.PostgreSQL = &fabricv1alpha1.ComponentStatus{Phase: "Ready"}
	hindsightProfile := hindsightTestProfile()
	postgresqlProfile := postgresqlTestProfile()
	postgresqlSecret := hindsightPostgreSQLSecret(tenant, postgresqlProfile)

	c := fake.NewClientBuilder().WithScheme(scheme).
		WithStatusSubresource(&appsv1.Deployment{}).
		WithObjects(tenant, hindsightProfile, postgresqlProfile, postgresqlSecret).
		Build()
	r := &TenantBundleReconciler{Client: c, Scheme: scheme}

	result, err := r.reconcileHindsight(ctx, tenant)
	if err != nil {
		t.Fatal(err)
	}
	if result.Ready || result.Reason != "DeploymentProgressing" {
		t.Fatalf("unexpected initial Hindsight result: %#v", result)
	}

	namespace := tenantNamespace(tenant.Name)
	var controlPlane appsv1.Deployment
	if err := c.Get(ctx, types.NamespacedName{Namespace: namespace, Name: "hindsight-control-plane"}, &controlPlane); err != nil {
		t.Fatalf("Hindsight Control Plane Deployment not reconciled: %v", err)
	}
	if len(controlPlane.Spec.Template.Spec.Containers) != 1 {
		t.Fatalf("unexpected Control Plane containers: %#v", controlPlane.Spec.Template.Spec.Containers)
	}
	cp := controlPlane.Spec.Template.Spec.Containers[0]
	if cp.Image != "ghcr.io/vectorize-io/hindsight-control-plane:0.10.1" {
		t.Fatalf("Control Plane image=%q", cp.Image)
	}
	if got := envValue(cp.Env, "HINDSIGHT_CP_DATAPLANE_API_URL"); got != "http://hindsight.tenant-hairem-sandbox.svc:8888" {
		t.Fatalf("Control Plane dataplane URL=%q", got)
	}
	keyEnv := envVar(cp.Env, "HINDSIGHT_CP_DATAPLANE_API_KEY")
	if keyEnv == nil || keyEnv.ValueFrom == nil || keyEnv.ValueFrom.SecretKeyRef == nil ||
		keyEnv.ValueFrom.SecretKeyRef.Name != "hindsight-runtime" ||
		keyEnv.ValueFrom.SecretKeyRef.Key != "HINDSIGHT_API_TENANT_API_KEY" {
		t.Fatalf("Control Plane API key is not secret-backed: %#v", keyEnv)
	}

	var cpService corev1.Service
	if err := c.Get(ctx, types.NamespacedName{Namespace: namespace, Name: "hindsight-control-plane"}, &cpService); err != nil {
		t.Fatalf("Control Plane Service not reconciled: %v", err)
	}
	if len(cpService.Spec.Ports) != 1 || cpService.Spec.Ports[0].Port != 3000 {
		t.Fatalf("unexpected Control Plane Service ports: %#v", cpService.Spec.Ports)
	}

	var authentikService corev1.Service
	if err := c.Get(ctx, types.NamespacedName{Namespace: namespace, Name: "hindsight-authentik"}, &authentikService); err != nil {
		t.Fatalf("Authentik ExternalName Service not reconciled: %v", err)
	}
	if authentikService.Spec.Type != corev1.ServiceTypeExternalName || authentikService.Spec.ExternalName != "authentik.auth.svc.cluster.local" {
		t.Fatalf("unexpected Authentik Service: %#v", authentikService.Spec)
	}

	var ingress networkingv1.Ingress
	if err := c.Get(ctx, types.NamespacedName{Namespace: namespace, Name: "hindsight-control-plane"}, &ingress); err != nil {
		t.Fatalf("Control Plane Ingress not reconciled: %v", err)
	}
	if got := ingress.Spec.Rules[0].Host; got != "hindsight-hairem-sandbox.truxonline.com" {
		t.Fatalf("Control Plane host=%q", got)
	}
	if got := ingress.Annotations["traefik.ingress.kubernetes.io/router.middlewares"]; got != "traefik-redirect-https@kubernetescrd,auth-authentik-forward-auth@kubernetescrd" {
		t.Fatalf("Control Plane middleware=%q", got)
	}
	if ingress.Annotations["external-dns.alpha.kubernetes.io/public"] != "true" ||
		ingress.Annotations["external-dns.alpha.kubernetes.io/target"] != "truxonline.com" {
		t.Fatalf("Control Plane DNS annotations=%#v", ingress.Annotations)
	}

	var authIngress networkingv1.Ingress
	if err := c.Get(ctx, types.NamespacedName{Namespace: namespace, Name: "hindsight-control-plane-auth"}, &authIngress); err != nil {
		t.Fatalf("Control Plane Authentik path Ingress not reconciled: %v", err)
	}
	path := authIngress.Spec.Rules[0].HTTP.Paths[0]
	if path.Path != "/outpost.goauthentik.io" || path.Backend.Service == nil || path.Backend.Service.Name != "hindsight-authentik" {
		t.Fatalf("unexpected Authentik outpost route: %#v", path)
	}
	if _, ok := authIngress.Annotations["traefik.ingress.kubernetes.io/router.middlewares"]; ok {
		t.Fatalf("outpost route must not be protected by ForwardAuth: %#v", authIngress.Annotations)
	}

	var apiPolicy networkingv1.NetworkPolicy
	if err := c.Get(ctx, types.NamespacedName{Namespace: namespace, Name: "hindsight-access"}, &apiPolicy); err != nil {
		t.Fatal(err)
	}
	if len(apiPolicy.Spec.Ingress) != 1 || len(apiPolicy.Spec.Ingress[0].From) != 2 {
		t.Fatalf("API policy must admit Hermes + Control Plane, got %#v", apiPolicy.Spec.Ingress)
	}

	var cpPolicy networkingv1.NetworkPolicy
	if err := c.Get(ctx, types.NamespacedName{Namespace: namespace, Name: "hindsight-control-plane-access"}, &cpPolicy); err != nil {
		t.Fatalf("Control Plane NetworkPolicy not reconciled: %v", err)
	}
	if len(cpPolicy.Spec.Ingress) != 1 || len(cpPolicy.Spec.Egress) != 2 {
		t.Fatalf("unexpected Control Plane network contract: ingress=%d egress=%d", len(cpPolicy.Spec.Ingress), len(cpPolicy.Spec.Egress))
	}

	var apiDeployment appsv1.Deployment
	if err := c.Get(ctx, types.NamespacedName{Namespace: namespace, Name: "hindsight"}, &apiDeployment); err != nil {
		t.Fatal(err)
	}
	apiDeployment.Status.ObservedGeneration = apiDeployment.Generation
	apiDeployment.Status.AvailableReplicas = 1
	if err := c.Status().Update(ctx, &apiDeployment); err != nil {
		t.Fatal(err)
	}
	controlPlane.Status.ObservedGeneration = controlPlane.Generation
	controlPlane.Status.AvailableReplicas = 1
	if err := c.Status().Update(ctx, &controlPlane); err != nil {
		t.Fatal(err)
	}

	result, err = r.reconcileHindsight(ctx, tenant)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Ready || result.Status == nil || result.Status.HumanEndpoint != "https://hindsight-hairem-sandbox.truxonline.com" {
		t.Fatalf("unexpected ready Hindsight WebUI result: %#v", result)
	}
}

func TestHindsightRefusesForeignRuntimeSecretAdoption(t *testing.T) {
	ctx := context.Background()
	scheme := postgresqlTestScheme(t)
	tenant := hindsightTestTenant()
	tenant.Status.Persistence.PostgreSQL = &fabricv1alpha1.ComponentStatus{Phase: "Ready"}
	hindsightProfile := hindsightTestProfile()
	postgresqlProfile := postgresqlTestProfile()
	postgresqlSecret := hindsightPostgreSQLSecret(tenant, postgresqlProfile)
	foreign := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "hindsight-runtime", Namespace: tenantNamespace(tenant.Name)},
		Type:       corev1.SecretTypeOpaque,
		Data:       map[string][]byte{"HINDSIGHT_API_TENANT_API_KEY": []byte("do-not-touch")},
	}
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(tenant, hindsightProfile, postgresqlProfile, postgresqlSecret, foreign).Build()

	result, err := (&TenantBundleReconciler{Client: c, Scheme: scheme}).reconcileHindsight(ctx, tenant)
	if err != nil {
		t.Fatal(err)
	}
	if result.Reason != "OwnershipConflict" || result.Status == nil || result.Status.Phase != "Blocked" {
		t.Fatalf("expected OwnershipConflict, got %#v", result)
	}
	var unchanged corev1.Secret
	if err := c.Get(ctx, client.ObjectKeyFromObject(foreign), &unchanged); err != nil {
		t.Fatal(err)
	}
	if string(unchanged.Data["HINDSIGHT_API_TENANT_API_KEY"]) != "do-not-touch" {
		t.Fatal("foreign Hindsight Secret was modified")
	}
}

func TestHindsightCleanupRemovesOnlyManagedTenantResources(t *testing.T) {
	ctx := context.Background()
	scheme := postgresqlTestScheme(t)
	tenant := hindsightTestTenant()
	profile := hindsightTestProfile()
	namespace := tenantNamespace(tenant.Name)
	managedLabels := hindsightLabels(tenant, profile, "test")
	deployment := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "hindsight", Namespace: namespace, Labels: managedLabels}}
	service := &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: "hindsight", Namespace: namespace, Labels: managedLabels}}
	policy := &networkingv1.NetworkPolicy{ObjectMeta: metav1.ObjectMeta{Name: "hindsight-access", Namespace: namespace, Labels: managedLabels}}
	secret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "hindsight-runtime", Namespace: namespace, Labels: managedLabels}}
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(tenant, deployment, service, policy, secret).Build()

	r := &TenantBundleReconciler{Client: c, Scheme: scheme}
	pending, err := r.cleanupHindsight(ctx, tenant)
	if err != nil {
		t.Fatal(err)
	}
	if !pending {
		t.Fatal("cleanup should report pending when managed Hindsight resources were present")
	}
	for _, object := range []client.Object{
		&appsv1.Deployment{}, &corev1.Service{}, &networkingv1.NetworkPolicy{}, &corev1.Secret{},
	} {
		name := "hindsight"
		switch object.(type) {
		case *networkingv1.NetworkPolicy:
			name = "hindsight-access"
		case *corev1.Secret:
			name = "hindsight-runtime"
		}
		if err := c.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, object); !apierrors.IsNotFound(err) {
			t.Fatalf("managed Hindsight resource %T/%s was not deleted: %v", object, name, err)
		}
	}
}

func hindsightTestTenant() *fabricv1alpha1.TenantBundle {
	tenant := postgresqlTestTenant()
	tenant.Spec.Memory = fabricv1alpha1.TenantMemorySpec{Hindsight: &fabricv1alpha1.HindsightMemorySpec{ProfileRef: "hindsight-standard"}}
	return tenant
}

func hindsightTestProfile() *fabricv1alpha1.HindsightProfile {
	return &fabricv1alpha1.HindsightProfile{
		ObjectMeta: metav1.ObjectMeta{Name: "hindsight-standard"},
		Spec: fabricv1alpha1.HindsightProfileSpec{
			Topology:    "TenantScoped",
			Image:       "ghcr.io/vectorize-io/hindsight-api:0.10.1",
			APIPort:     8888,
			APIAuthMode: "ApiKey",
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("500m"), corev1.ResourceMemory: resource.MustParse("1Gi")},
				Limits:   corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("2"), corev1.ResourceMemory: resource.MustParse("4Gi")},
			},
			PriorityClassName: "vixens-medium",
			SizingLabel:       "V-small",
			LLMAuthMode:       "Unconfigured",
		},
	}
}

func hindsightPostgreSQLSecret(tenant *fabricv1alpha1.TenantBundle, profile *fabricv1alpha1.PostgreSQLProfile) *corev1.Secret {
	names := resolvePostgreSQLNames(tenant, profile)
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: names.Secret, Namespace: profile.Spec.Shared.ClusterRef.Namespace, Labels: postgresqlLabels(tenant, profile, "credentials", profile.Spec.RoleReclaimPolicy)},
		Type:       corev1.SecretTypeBasicAuth,
		Data: map[string][]byte{
			corev1.BasicAuthUsernameKey: []byte(names.Role),
			corev1.BasicAuthPasswordKey: []byte("postgresql-password"),
			"host":                    []byte("postgresql-shared-rw.databases.svc"),
			"port":                    []byte("5432"),
			"dbname":                  []byte(names.Database),
		},
	}
}
