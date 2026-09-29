package controller

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	fabricv1alpha1 "github.com/charchess/vixens/apps/60-services/txo-fabric/operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestHindsightPlatformGatewayReconcilesScopedEmbeddingAccess(t *testing.T) {
	ctx := context.Background()
	scheme := postgresqlTestScheme(t)
	tenant := hindsightTestTenant()
	tenant.Status.Persistence.PostgreSQL = &fabricv1alpha1.ComponentStatus{Phase: "Ready"}
	hindsightProfile := hindsightTestProfile()
	hindsightProfile.Spec.LLMAuthMode = "PlatformGateway"
	postgresqlProfile := postgresqlTestProfile()
	postgresqlSecret := hindsightPostgreSQLSecret(tenant, postgresqlProfile)

	generateCalls := 0
	var keyRequest map[string]any
	gateway := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/key/generate" {
			t.Fatalf("unexpected gateway path %q", req.URL.Path)
		}
		if req.Header.Get("Authorization") != "Bearer test-admin-token" {
			t.Fatalf("unexpected gateway authorization header")
		}
		generateCalls++
		if err := json.NewDecoder(req.Body).Decode(&keyRequest); err != nil {
			t.Fatal(err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"key":"sk-hindsight-embedding-test"}`))
	}))
	defer gateway.Close()
	t.Setenv("TXO_AI_GATEWAY_URL", gateway.URL)
	t.Setenv("TXO_AI_GATEWAY_ADMIN_TOKEN", "test-admin-token")

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
	if generateCalls != 1 {
		t.Fatalf("gateway key generation calls=%d, want 1", generateCalls)
	}
	if got := keyRequest["key_alias"]; got != "txo-fabric:hairem-sandbox:hindsight-embeddings" {
		t.Fatalf("gateway key alias=%#v", got)
	}
	models, ok := keyRequest["models"].([]any)
	if !ok || len(models) != 1 || models[0] != "txo-embedding" {
		t.Fatalf("gateway models=%#v", keyRequest["models"])
	}
	metadata, ok := keyRequest["metadata"].(map[string]any)
	if !ok || metadata["tenant"] != tenant.Name || metadata["tenant_id"] != tenant.Spec.TenantID || metadata["component"] != "hindsight" || metadata["capability"] != "embeddings" {
		t.Fatalf("gateway metadata=%#v", keyRequest["metadata"])
	}

	namespace := tenantNamespace(tenant.Name)
	var runtimeSecret corev1.Secret
	if err := c.Get(ctx, types.NamespacedName{Namespace: namespace, Name: "hindsight-runtime"}, &runtimeSecret); err != nil {
		t.Fatal(err)
	}
	wantSecretValues := map[string]string{
		"HINDSIGHT_API_LLM_PROVIDER":                    "none",
		"HINDSIGHT_API_EMBEDDINGS_PROVIDER":             "openai",
		"HINDSIGHT_API_EMBEDDINGS_OPENAI_BASE_URL":      gateway.URL + "/v1",
		"HINDSIGHT_API_EMBEDDINGS_OPENAI_MODEL":         "txo-embedding",
		"HINDSIGHT_API_EMBEDDINGS_OPENAI_DIMENSIONS":    "384",
		"HINDSIGHT_API_EMBEDDINGS_OPENAI_API_KEY":       "sk-hindsight-embedding-test",
	}
	for key, want := range wantSecretValues {
		if got := string(runtimeSecret.Data[key]); got != want {
			t.Fatalf("%s=%q, want %q", key, got, want)
		}
	}

	var policy networkingv1.NetworkPolicy
	if err := c.Get(ctx, types.NamespacedName{Namespace: namespace, Name: "hindsight-access"}, &policy); err != nil {
		t.Fatal(err)
	}
	if len(policy.Spec.Egress) != 3 {
		t.Fatalf("Hindsight egress rule count=%d, want 3", len(policy.Spec.Egress))
	}
	gatewayRule := policy.Spec.Egress[2]
	if len(gatewayRule.To) != 1 || gatewayRule.To[0].NamespaceSelector == nil || gatewayRule.To[0].PodSelector == nil {
		t.Fatalf("unexpected gateway egress peer: %#v", gatewayRule.To)
	}
	if gatewayRule.To[0].NamespaceSelector.MatchLabels["kubernetes.io/metadata.name"] != "txo-fabric-system" || gatewayRule.To[0].PodSelector.MatchLabels[LabelName] != "txo-ai-gateway" {
		t.Fatalf("unexpected gateway egress selector: %#v", gatewayRule.To[0])
	}
	if len(gatewayRule.Ports) != 1 || gatewayRule.Ports[0].Port == nil || gatewayRule.Ports[0].Port.IntVal != 4000 {
		t.Fatalf("unexpected gateway egress ports: %#v", gatewayRule.Ports)
	}

	// The scoped virtual key is stable runtime state. Ordinary reconciles must reuse
	// the tenant-local Secret instead of creating another LiteLLM key.
	if _, err := r.reconcileHindsight(ctx, tenant); err != nil {
		t.Fatal(err)
	}
	if generateCalls != 1 {
		t.Fatalf("idempotent reconcile generated %d gateway keys, want 1", generateCalls)
	}
}
