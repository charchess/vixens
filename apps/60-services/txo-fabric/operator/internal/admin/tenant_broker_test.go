package admin

import (
	"context"
	"testing"

	fabricv1alpha1 "github.com/charchess/vixens/apps/60-services/txo-fabric/operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func adminTestScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	if err := fabricv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	return scheme
}

func TestResolveTenantBrokerUsesTenantScopedRuntimeSecret(t *testing.T) {
	ctx := context.Background()
	tenant := &fabricv1alpha1.TenantBundle{
		ObjectMeta: metav1.ObjectMeta{Name: "hairem"},
		Spec: fabricv1alpha1.TenantBundleSpec{
			TenantID:    "TEN00001",
			DisplayName: "hAIrem",
		},
	}
	profile := &fabricv1alpha1.AICredentialBrokerProfile{
		ObjectMeta: metav1.ObjectMeta{Name: "cliproxyapi-standard"},
		Spec: fabricv1alpha1.AICredentialBrokerProfileSpec{
			Topology: "TenantScoped",
			Implementation: "CLIProxyAPI",
			APIPort: 8317,
		},
	}
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: brokerRuntimeSecretName, Namespace: "tenant-hairem"},
		Data: map[string][]byte{"management-password": []byte("hairem-management")},
	}
	c := fake.NewClientBuilder().WithScheme(adminTestScheme(t)).WithObjects(tenant, profile, secret).Build()

	target, err := ResolveTenantBroker(ctx, c, "hairem")
	if err != nil {
		t.Fatal(err)
	}
	if target.Namespace != "tenant-hairem" || target.BaseURL != "http://txo-ai-credential-broker.tenant-hairem.svc:8317" {
		t.Fatalf("target=%#v", target)
	}
	if target.ManagementCredential != "hairem-management" {
		t.Fatal("wrong tenant management credential resolved")
	}
}

func TestResolveTenantBrokerRejectsParkedTenant(t *testing.T) {
	ctx := context.Background()
	tenant := &fabricv1alpha1.TenantBundle{
		ObjectMeta: metav1.ObjectMeta{Name: "fabric-smoke"},
		Spec: fabricv1alpha1.TenantBundleSpec{
			TenantID:    "TEN90002",
			DisplayName: "Fabric Smoke",
			Lifecycle:   fabricv1alpha1.TenantLifecycleSpec{Mode: "Parked"},
		},
	}
	c := fake.NewClientBuilder().WithScheme(adminTestScheme(t)).WithObjects(tenant).Build()

	if _, err := ResolveTenantBroker(ctx, c, "fabric-smoke"); err == nil {
		t.Fatal("parked tenant must not expose CPA OAuth administration")
	}
}

func TestResolveTenantBrokerNeverFallsBackToAnotherTenantSecret(t *testing.T) {
	ctx := context.Background()
	hairem := &fabricv1alpha1.TenantBundle{
		ObjectMeta: metav1.ObjectMeta{Name: "hairem"},
		Spec: fabricv1alpha1.TenantBundleSpec{TenantID: "TEN00001", DisplayName: "hAIrem"},
	}
	indiba := &fabricv1alpha1.TenantBundle{
		ObjectMeta: metav1.ObjectMeta{Name: "indiba"},
		Spec: fabricv1alpha1.TenantBundleSpec{TenantID: "TEN00002", DisplayName: "Indiba"},
	}
	profile := &fabricv1alpha1.AICredentialBrokerProfile{
		ObjectMeta: metav1.ObjectMeta{Name: defaultBrokerProfileName},
		Spec: fabricv1alpha1.AICredentialBrokerProfileSpec{Topology: "TenantScoped", Implementation: "CLIProxyAPI", APIPort: 8317},
	}
	hairemSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: brokerRuntimeSecretName, Namespace: "tenant-hairem"},
		Data: map[string][]byte{"management-password": []byte("hairem-management")},
	}
	c := fake.NewClientBuilder().WithScheme(adminTestScheme(t)).WithObjects(hairem, indiba, profile, hairemSecret).Build()

	if _, err := ResolveTenantBroker(ctx, c, "indiba"); err == nil {
		t.Fatal("missing Indiba management Secret must fail closed instead of using hAIrem")
	}
}
