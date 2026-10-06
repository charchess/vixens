package controller

import (
	"context"
	"strings"
	"testing"

	fabricv1alpha1 "github.com/charchess/vixens/apps/60-services/txo-fabric/operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func aiGatewayTestProfile() *fabricv1alpha1.AIGatewayProfile {
	return &fabricv1alpha1.AIGatewayProfile{
		ObjectMeta: metav1.ObjectMeta{Name: defaultAIGatewayProfileName},
		Spec: fabricv1alpha1.AIGatewayProfileSpec{
			Topology:       "TenantScoped",
			Implementation: "CLIProxyAPI",
			Image:          "eceasy/cli-proxy-api:v8.0.16",
			APIPort:        8317,
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("100m"), corev1.ResourceMemory: resource.MustParse("128Mi")},
			},
			PriorityClassName: "vixens-medium",
			SizingLabel:       "V-small",
			AuthStorage: fabricv1alpha1.AIGatewayAuthStorageSpec{
				Size:             resource.MustParse("1Gi"),
				StorageClassName: "truenas-iscsi-delete",
			},
			UsageQueueRetentionSeconds: 900,
		},
	}
}

func aiGatewayTestTenant(name, tenantID string) *fabricv1alpha1.TenantBundle {
	return &fabricv1alpha1.TenantBundle{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: fabricv1alpha1.TenantBundleSpec{
			TenantID:    tenantID,
			DisplayName: name,
			AIGateway:   &fabricv1alpha1.TenantAIGatewaySpec{ProfileRef: defaultAIGatewayProfileName},
		},
	}
}

func TestReconcileAIGatewayCreatesTenantScopedCPAWithoutProviderSecrets(t *testing.T) {
	ctx := context.Background()
	scheme := testScheme(t)
	tenant := aiGatewayTestTenant("hairem", "TEN00001")
	profile := aiGatewayTestProfile()
	namespace := tenantNamespace(tenant.Name)
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
		tenant,
		profile,
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}},
	).Build()
	r := &TenantBundleReconciler{Client: c, Scheme: scheme}

	result, err := r.reconcileAIGateway(ctx, tenant)
	if err != nil {
		t.Fatal(err)
	}
	if result.Ready {
		t.Fatal("new gateway must remain provisioning until its Deployment is available")
	}

	var secret corev1.Secret
	if err := c.Get(ctx, types.NamespacedName{Namespace: namespace, Name: tenantAIGatewaySecretName}, &secret); err != nil {
		t.Fatal(err)
	}
	firstBootstrapKey := string(secret.Data["bootstrap-api-key"])
	firstManagementPassword := string(secret.Data["management-password"])
	if firstBootstrapKey == "" || firstManagementPassword == "" || firstBootstrapKey == firstManagementPassword {
		t.Fatal("unexpected generated gateway credentials")
	}
	config := string(secret.Data["config.yaml"])
	for _, want := range []string{
		"config-version: 8",
		"force-model-prefix: true",
		"strategy: \"round-robin\"",
		"usage-statistics-enabled: true",
		"redis-usage-queue-retention-seconds: 900",
		"auth-dir: \"/data/auth\"",
		firstBootstrapKey,
	} {
		if !strings.Contains(config, want) {
			t.Fatalf("CPA config missing %q:\\n%s", want, config)
		}
	}
	for _, forbidden := range []string{"access_token", "refresh_token", "OPENROUTER_API_KEY", "sk-or-v1-"} {
		if strings.Contains(config, forbidden) {
			t.Fatalf("CPA config unexpectedly contains provider secret marker %q", forbidden)
		}
	}

	var pvc corev1.PersistentVolumeClaim
	if err := c.Get(ctx, types.NamespacedName{Namespace: namespace, Name: tenantAIGatewayAuthPVCName}, &pvc); err != nil {
		t.Fatal(err)
	}
	if pvc.Spec.StorageClassName == nil || *pvc.Spec.StorageClassName != "truenas-iscsi-delete" {
		t.Fatalf("auth PVC storage class=%v", pvc.Spec.StorageClassName)
	}
	if got := pvc.Spec.Resources.Requests[corev1.ResourceStorage]; got.Cmp(resource.MustParse("1Gi")) != 0 {
		t.Fatalf("auth PVC size=%s", got.String())
	}

	var deployment appsv1.Deployment
	if err := c.Get(ctx, types.NamespacedName{Namespace: namespace, Name: tenantAIGatewayName}, &deployment); err != nil {
		t.Fatal(err)
	}
	container := deployment.Spec.Template.Spec.Containers[0]
	if container.Image != profile.Spec.Image {
		t.Fatalf("gateway image=%q want %q", container.Image, profile.Spec.Image)
	}
	if deployment.Spec.Template.Spec.SecurityContext == nil || deployment.Spec.Template.Spec.SecurityContext.RunAsNonRoot == nil || !*deployment.Spec.Template.Spec.SecurityContext.RunAsNonRoot {
		t.Fatal("gateway pod must run non-root")
	}
	if len(container.Env) != 1 || container.Env[0].Name != "MANAGEMENT_PASSWORD" || container.Env[0].ValueFrom == nil || container.Env[0].ValueFrom.SecretKeyRef == nil || container.Env[0].ValueFrom.SecretKeyRef.Name != tenantAIGatewaySecretName {
		t.Fatalf("management credential is not secret-backed: %#v", container.Env)
	}

	var service corev1.Service
	if err := c.Get(ctx, types.NamespacedName{Namespace: namespace, Name: tenantAIGatewayName}, &service); err != nil {
		t.Fatal(err)
	}
	if len(service.Spec.Ports) != 1 || service.Spec.Ports[0].Port != 8317 {
		t.Fatalf("gateway service ports=%#v", service.Spec.Ports)
	}

	var policy networkingv1.NetworkPolicy
	if err := c.Get(ctx, types.NamespacedName{Namespace: namespace, Name: tenantAIGatewayName}, &policy); err != nil {
		t.Fatal(err)
	}
	if len(policy.Spec.Egress) != 2 {
		t.Fatalf("gateway egress rules=%d want DNS + HTTPS", len(policy.Spec.Egress))
	}

	if _, err := r.reconcileAIGateway(ctx, tenant); err != nil {
		t.Fatal(err)
	}
	if err := c.Get(ctx, types.NamespacedName{Namespace: namespace, Name: tenantAIGatewaySecretName}, &secret); err != nil {
		t.Fatal(err)
	}
	if string(secret.Data["bootstrap-api-key"]) != firstBootstrapKey || string(secret.Data["management-password"]) != firstManagementPassword {
		t.Fatal("idempotent reconcile rotated gateway credentials")
	}
}

func TestReconcileAIGatewaySeparatesTenantCredentialState(t *testing.T) {
	ctx := context.Background()
	scheme := testScheme(t)
	hairem := aiGatewayTestTenant("hairem", "TEN00001")
	indiba := aiGatewayTestTenant("indiba", "TEN00002")
	profile := aiGatewayTestProfile()
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
		hairem, indiba, profile,
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: tenantNamespace(hairem.Name)}},
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: tenantNamespace(indiba.Name)}},
	).Build()
	r := &TenantBundleReconciler{Client: c, Scheme: scheme}

	if _, err := r.reconcileAIGateway(ctx, hairem); err != nil {
		t.Fatal(err)
	}
	if _, err := r.reconcileAIGateway(ctx, indiba); err != nil {
		t.Fatal(err)
	}

	var hairemSecret, indibaSecret corev1.Secret
	if err := c.Get(ctx, types.NamespacedName{Namespace: tenantNamespace(hairem.Name), Name: tenantAIGatewaySecretName}, &hairemSecret); err != nil {
		t.Fatal(err)
	}
	if err := c.Get(ctx, types.NamespacedName{Namespace: tenantNamespace(indiba.Name), Name: tenantAIGatewaySecretName}, &indibaSecret); err != nil {
		t.Fatal(err)
	}
	if string(hairemSecret.Data["bootstrap-api-key"]) == string(indibaSecret.Data["bootstrap-api-key"]) {
		t.Fatal("tenant gateway API keys collided")
	}
	if string(hairemSecret.Data["management-password"]) == string(indibaSecret.Data["management-password"]) {
		t.Fatal("tenant management credentials collided")
	}

	var hairemPVC, indibaPVC corev1.PersistentVolumeClaim
	if err := c.Get(ctx, types.NamespacedName{Namespace: tenantNamespace(hairem.Name), Name: tenantAIGatewayAuthPVCName}, &hairemPVC); err != nil {
		t.Fatal(err)
	}
	if err := c.Get(ctx, types.NamespacedName{Namespace: tenantNamespace(indiba.Name), Name: tenantAIGatewayAuthPVCName}, &indibaPVC); err != nil {
		t.Fatal(err)
	}
	if hairemPVC.Namespace == indibaPVC.Namespace {
		t.Fatal("tenant OAuth state PVCs are not namespace-isolated")
	}
}

func TestReconcileAIGatewayFailsClosedOnUnsupportedTopology(t *testing.T) {
	ctx := context.Background()
	scheme := testScheme(t)
	tenant := aiGatewayTestTenant("hairem", "TEN00001")
	profile := aiGatewayTestProfile()
	profile.Spec.Topology = "Shared"
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(tenant, profile).Build()

	result, err := (&TenantBundleReconciler{Client: c, Scheme: scheme}).reconcileAIGateway(ctx, tenant)
	if err != nil {
		t.Fatal(err)
	}
	if result.Ready || result.Status == nil || result.Status.Phase != "Blocked" || result.Reason != "UnsupportedTopology" {
		t.Fatalf("unsupported topology did not fail closed: %#v", result)
	}
}
