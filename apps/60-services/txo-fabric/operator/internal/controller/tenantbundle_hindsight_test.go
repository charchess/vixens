package controller

import (
	"bytes"
	"context"
	"strings"
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
	if !strings.HasPrefix(databaseURL, "postgresql://txo_ten90001:") || !strings.Contains(databaseURL, "@postgresql-shared-rw.databases.svc:5432/txo_ten90001") {
		t.Fatalf("unexpected Hindsight database URL %q", databaseURL)
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
	if !strings.Contains(string(after.Data["HINDSIGHT_API_DATABASE_URL"]), "rotated-postgresql-password") {
		t.Fatal("Hindsight database binding did not follow PostgreSQL credential rotation")
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
