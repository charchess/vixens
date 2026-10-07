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

func aiCredentialBrokerTestProfile() *fabricv1alpha1.AICredentialBrokerProfile {
	return &fabricv1alpha1.AICredentialBrokerProfile{
		ObjectMeta: metav1.ObjectMeta{Name: defaultAICredentialBrokerProfileName},
		Spec: fabricv1alpha1.AICredentialBrokerProfileSpec{
			Topology:       "TenantScoped",
			Implementation: "CLIProxyAPI",
			Image:          "eceasy/cli-proxy-api:v8.0.16",
			APIPort:        8317,
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("100m"), corev1.ResourceMemory: resource.MustParse("128Mi")},
			},
			PriorityClassName: "vixens-medium",
			SizingLabel:       "V-small",
			AuthStorage: fabricv1alpha1.AICredentialBrokerAuthStorageSpec{
				Size:             resource.MustParse("1Gi"),
				StorageClassName: "truenas-iscsi-delete",
			},
			UsageQueueRetentionSeconds: 900,
		},
	}
}

func aiCredentialBrokerTestTenant(name, tenantID string) *fabricv1alpha1.TenantBundle {
	return &fabricv1alpha1.TenantBundle{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: fabricv1alpha1.TenantBundleSpec{
			TenantID:    tenantID,
			DisplayName: name,
			AICredentialBroker: &fabricv1alpha1.TenantAICredentialBrokerSpec{ProfileRef: defaultAICredentialBrokerProfileName},
		},
	}
}

func TestReconcileAICredentialBrokerCreatesTenantScopedCPAWithoutProviderSecrets(t *testing.T) {
	ctx := context.Background()
	scheme := testScheme(t)
	tenant := aiCredentialBrokerTestTenant("hairem", "TEN00001")
	profile := aiCredentialBrokerTestProfile()
	namespace := tenantNamespace(tenant.Name)
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
		tenant,
		profile,
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}},
	).Build()
	r := &TenantBundleReconciler{Client: c, Scheme: scheme}

	result, err := r.reconcileAICredentialBroker(ctx, tenant)
	if err != nil {
		t.Fatal(err)
	}
	if result.Ready {
		t.Fatal("new credential broker must remain provisioning until its Deployment is available")
	}

	var secret corev1.Secret
	if err := c.Get(ctx, types.NamespacedName{Namespace: namespace, Name: tenantAICredentialBrokerSecretName}, &secret); err != nil {
		t.Fatal(err)
	}
	firstBootstrapKey := string(secret.Data["bootstrap-api-key"])
	firstManagementPassword := string(secret.Data["management-password"])
	if firstBootstrapKey == "" || firstManagementPassword == "" || firstBootstrapKey == firstManagementPassword {
		t.Fatal("unexpected generated broker credentials")
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
	if err := c.Get(ctx, types.NamespacedName{Namespace: namespace, Name: tenantAICredentialBrokerAuthPVCName}, &pvc); err != nil {
		t.Fatal(err)
	}
	if pvc.Spec.StorageClassName == nil || *pvc.Spec.StorageClassName != "truenas-iscsi-delete" {
		t.Fatalf("auth PVC storage class=%v", pvc.Spec.StorageClassName)
	}
	if got := pvc.Spec.Resources.Requests[corev1.ResourceStorage]; got.Cmp(resource.MustParse("1Gi")) != 0 {
		t.Fatalf("auth PVC size=%s", got.String())
	}

	var deployment appsv1.Deployment
	if err := c.Get(ctx, types.NamespacedName{Namespace: namespace, Name: tenantAICredentialBrokerName}, &deployment); err != nil {
		t.Fatal(err)
	}
	container := deployment.Spec.Template.Spec.Containers[0]
	if container.Image != profile.Spec.Image {
		t.Fatalf("broker image=%q want %q", container.Image, profile.Spec.Image)
	}
	if deployment.Spec.Template.Spec.SecurityContext == nil || deployment.Spec.Template.Spec.SecurityContext.RunAsNonRoot == nil || !*deployment.Spec.Template.Spec.SecurityContext.RunAsNonRoot {
		t.Fatal("broker pod must run non-root")
	}
	if len(container.Env) != 1 || container.Env[0].Name != "MANAGEMENT_PASSWORD" || container.Env[0].ValueFrom == nil || container.Env[0].ValueFrom.SecretKeyRef == nil || container.Env[0].ValueFrom.SecretKeyRef.Name != tenantAICredentialBrokerSecretName {
		t.Fatalf("management credential is not secret-backed: %#v", container.Env)
	}

	var service corev1.Service
	if err := c.Get(ctx, types.NamespacedName{Namespace: namespace, Name: tenantAICredentialBrokerName}, &service); err != nil {
		t.Fatal(err)
	}
	if len(service.Spec.Ports) != 1 || service.Spec.Ports[0].Port != 8317 {
		t.Fatalf("broker service ports=%#v", service.Spec.Ports)
	}

	var policy networkingv1.NetworkPolicy
	if err := c.Get(ctx, types.NamespacedName{Namespace: namespace, Name: tenantAICredentialBrokerName}, &policy); err != nil {
		t.Fatal(err)
	}
	if len(policy.Spec.Egress) != 2 {
		t.Fatalf("broker egress rules=%d want DNS + HTTPS", len(policy.Spec.Egress))
	}
	// Kubernetes represents "all destinations on this port" by omitting To.
	// A peer written as `to: - {}` is rejected by the API server.
	httpsEgress := policy.Spec.Egress[1]
	if len(httpsEgress.To) != 0 {
		t.Fatalf("broker HTTPS egress must omit To (all destinations), got %#v", httpsEgress.To)
	}
	if len(httpsEgress.Ports) != 1 || httpsEgress.Ports[0].Port == nil || httpsEgress.Ports[0].Port.IntValue() != 443 {
		t.Fatalf("broker HTTPS egress must allow only TCP/443: %#v", httpsEgress.Ports)
	}
	if len(policy.Spec.Ingress) != 2 {
		t.Fatalf("broker ingress rules=%d want tenant LiteLLM + Fabric operator management", len(policy.Spec.Ingress))
	}

	var gatewayIngress, operatorIngress bool
	for _, rule := range policy.Spec.Ingress {
		if len(rule.From) != 1 || rule.From[0].PodSelector == nil {
			continue
		}
		peer := rule.From[0]
		labels := peer.PodSelector.MatchLabels
		if peer.NamespaceSelector == nil &&
			labels[LabelManaged] == "true" &&
			labels[LabelTenantName] == tenant.Name &&
			labels["app.kubernetes.io/component"] == "tenant-ai-gateway" {
			gatewayIngress = true
		}
		if peer.NamespaceSelector != nil &&
			peer.NamespaceSelector.MatchLabels["kubernetes.io/metadata.name"] == "txo-fabric-system" &&
			labels[LabelName] == "txo-fabric-operator" {
			operatorIngress = true
		}
		if len(rule.Ports) != 1 || rule.Ports[0].Port == nil || rule.Ports[0].Port.IntValue() != 8317 {
			t.Fatalf("broker ingress rule is not restricted to CPA port: %#v", rule.Ports)
		}
	}
	if !gatewayIngress {
		t.Fatal("broker ingress is missing same-tenant LiteLLM")
	}
	if !operatorIngress {
		t.Fatal("broker ingress is missing Fabric operator management path")
	}

	if _, err := r.reconcileAICredentialBroker(ctx, tenant); err != nil {
		t.Fatal(err)
	}
	if err := c.Get(ctx, types.NamespacedName{Namespace: namespace, Name: tenantAICredentialBrokerSecretName}, &secret); err != nil {
		t.Fatal(err)
	}
	if string(secret.Data["bootstrap-api-key"]) != firstBootstrapKey || string(secret.Data["management-password"]) != firstManagementPassword {
		t.Fatal("idempotent reconcile rotated broker credentials")
	}
}

func TestReconcileAICredentialBrokerSeparatesTenantCredentialState(t *testing.T) {
	ctx := context.Background()
	scheme := testScheme(t)
	hairem := aiCredentialBrokerTestTenant("hairem", "TEN00001")
	indiba := aiCredentialBrokerTestTenant("indiba", "TEN00002")
	profile := aiCredentialBrokerTestProfile()
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
		hairem, indiba, profile,
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: tenantNamespace(hairem.Name)}},
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: tenantNamespace(indiba.Name)}},
	).Build()
	r := &TenantBundleReconciler{Client: c, Scheme: scheme}

	if _, err := r.reconcileAICredentialBroker(ctx, hairem); err != nil {
		t.Fatal(err)
	}
	if _, err := r.reconcileAICredentialBroker(ctx, indiba); err != nil {
		t.Fatal(err)
	}

	var hairemSecret, indibaSecret corev1.Secret
	if err := c.Get(ctx, types.NamespacedName{Namespace: tenantNamespace(hairem.Name), Name: tenantAICredentialBrokerSecretName}, &hairemSecret); err != nil {
		t.Fatal(err)
	}
	if err := c.Get(ctx, types.NamespacedName{Namespace: tenantNamespace(indiba.Name), Name: tenantAICredentialBrokerSecretName}, &indibaSecret); err != nil {
		t.Fatal(err)
	}
	if string(hairemSecret.Data["bootstrap-api-key"]) == string(indibaSecret.Data["bootstrap-api-key"]) {
		t.Fatal("tenant broker API keys collided")
	}
	if string(hairemSecret.Data["management-password"]) == string(indibaSecret.Data["management-password"]) {
		t.Fatal("tenant management credentials collided")
	}

	var hairemPVC, indibaPVC corev1.PersistentVolumeClaim
	if err := c.Get(ctx, types.NamespacedName{Namespace: tenantNamespace(hairem.Name), Name: tenantAICredentialBrokerAuthPVCName}, &hairemPVC); err != nil {
		t.Fatal(err)
	}
	if err := c.Get(ctx, types.NamespacedName{Namespace: tenantNamespace(indiba.Name), Name: tenantAICredentialBrokerAuthPVCName}, &indibaPVC); err != nil {
		t.Fatal(err)
	}
	if hairemPVC.Namespace == indibaPVC.Namespace {
		t.Fatal("tenant OAuth state PVCs are not namespace-isolated")
	}
}

func TestReconcileAICredentialBrokerFailsClosedOnUnsupportedTopology(t *testing.T) {
	ctx := context.Background()
	scheme := testScheme(t)
	tenant := aiCredentialBrokerTestTenant("hairem", "TEN00001")
	profile := aiCredentialBrokerTestProfile()
	profile.Spec.Topology = "Shared"
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(tenant, profile).Build()

	result, err := (&TenantBundleReconciler{Client: c, Scheme: scheme}).reconcileAICredentialBroker(ctx, tenant)
	if err != nil {
		t.Fatal(err)
	}
	if result.Ready || result.Status == nil || result.Status.Phase != "Blocked" || result.Reason != "UnsupportedTopology" {
		t.Fatalf("unsupported topology did not fail closed: %#v", result)
	}
}
