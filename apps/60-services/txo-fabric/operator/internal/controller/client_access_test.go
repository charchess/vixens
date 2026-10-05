package controller

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestTenantBundleGetUsesAPIReaderForSecrets(t *testing.T) {
	ctx := context.Background()
	scheme := testScheme(t)
	key := types.NamespacedName{Namespace: "tenant-hairem", Name: "runtime-secret"}

	cached := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: key.Name, Namespace: key.Namespace},
			Data:       map[string][]byte{"source": []byte("cache")},
		},
	).Build()
	direct := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: key.Name, Namespace: key.Namespace},
			Data:       map[string][]byte{"source": []byte("api-reader")},
		},
	).Build()

	r := &TenantBundleReconciler{Client: cached, APIReader: direct, Scheme: scheme}
	var secret corev1.Secret
	if err := r.Get(ctx, key, &secret); err != nil {
		t.Fatal(err)
	}
	if got := string(secret.Data["source"]); got != "api-reader" {
		t.Fatalf("TenantBundle Secret Get used %q source, want APIReader", got)
	}
}

func TestTenantBundleGetKeepsOrdinaryObjectsOnCachedClient(t *testing.T) {
	ctx := context.Background()
	scheme := testScheme(t)
	key := types.NamespacedName{Namespace: "tenant-hairem", Name: "policy"}

	cached := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: key.Name, Namespace: key.Namespace},
			Data:       map[string]string{"source": "cache"},
		},
	).Build()
	direct := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: key.Name, Namespace: key.Namespace},
			Data:       map[string]string{"source": "api-reader"},
		},
	).Build()

	r := &TenantBundleReconciler{Client: cached, APIReader: direct, Scheme: scheme}
	var configMap corev1.ConfigMap
	if err := r.Get(ctx, key, &configMap); err != nil {
		t.Fatal(err)
	}
	if got := configMap.Data["source"]; got != "cache" {
		t.Fatalf("ordinary ConfigMap Get used %q source, want cached client", got)
	}
}

func TestAgentIdentityGetUsesAPIReaderForSecrets(t *testing.T) {
	ctx := context.Background()
	scheme := testScheme(t)
	key := types.NamespacedName{Namespace: "tenant-hairem", Name: "hermes-agent-model-access"}

	cached := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: key.Name, Namespace: key.Namespace},
			Data:       map[string][]byte{"source": []byte("cache")},
		},
	).Build()
	direct := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: key.Name, Namespace: key.Namespace},
			Data:       map[string][]byte{"source": []byte("api-reader")},
		},
	).Build()

	r := &AgentIdentityReconciler{Client: cached, APIReader: direct, Scheme: scheme}
	var secret corev1.Secret
	if err := r.Get(ctx, key, &secret); err != nil {
		t.Fatal(err)
	}
	if got := string(secret.Data["source"]); got != "api-reader" {
		t.Fatalf("AgentIdentity Secret Get used %q source, want APIReader", got)
	}
}

func TestReconcilersFallBackToCachedClientWithoutManagerAPIReader(t *testing.T) {
	ctx := context.Background()
	scheme := testScheme(t)
	key := types.NamespacedName{Namespace: "tenant-hairem", Name: "runtime-secret"}
	cached := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: key.Name, Namespace: key.Namespace},
			Data:       map[string][]byte{"source": []byte("cache")},
		},
	).Build()

	for name, get := range map[string]func(*corev1.Secret) error{
		"tenant": func(secret *corev1.Secret) error {
			r := &TenantBundleReconciler{Client: cached, Scheme: scheme}
			return r.Get(ctx, key, secret)
		},
		"agent": func(secret *corev1.Secret) error {
			r := &AgentIdentityReconciler{Client: cached, Scheme: scheme}
			return r.Get(ctx, key, secret)
		},
	} {
		t.Run(name, func(t *testing.T) {
			var secret corev1.Secret
			if err := get(&secret); err != nil {
				t.Fatal(err)
			}
			if got := string(secret.Data["source"]); got != "cache" {
				t.Fatalf("fallback Secret Get used %q source, want cache", got)
			}
		})
	}
}
