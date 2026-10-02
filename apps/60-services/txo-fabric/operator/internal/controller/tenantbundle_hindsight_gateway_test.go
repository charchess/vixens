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
	deleteCalls := 0
	var keyRequest map[string]any
	var deletePayloads []map[string]any
	gateway := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Header.Get("Authorization") != "Bearer test-admin-token" {
			t.Fatalf("unexpected gateway authorization header")
		}
		switch req.URL.Path {
		case "/key/delete":
			deleteCalls++
			var payload map[string]any
			if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
				t.Fatal(err)
			}
			deletePayloads = append(deletePayloads, payload)
			if deleteCalls == 1 {
				http.Error(w, "not found", http.StatusNotFound)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"deleted_keys":["sk-hindsight-embedding-test"]}`))
		case "/key/generate":
			generateCalls++
			if err := json.NewDecoder(req.Body).Decode(&keyRequest); err != nil {
				t.Fatal(err)
			}
			key := "sk-hindsight-embedding-test"
			if generateCalls > 1 {
				key = "sk-hindsight-embedding-rotated"
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"key":"` + key + `"}`))
		default:
			t.Fatalf("unexpected gateway path %q", req.URL.Path)
		}
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
	if generateCalls != 1 || deleteCalls != 1 {
		t.Fatalf("gateway calls generate=%d delete=%d, want 1/1", generateCalls, deleteCalls)
	}
	aliases, ok := deletePayloads[0]["key_aliases"].([]any)
	if !ok || len(aliases) != 1 || aliases[0] != "txo-fabric:hairem-sandbox:hindsight-embeddings" {
		t.Fatalf("initial stale-key cleanup payload=%#v", deletePayloads[0])
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
		"HINDSIGHT_API_EMBEDDINGS_OPENAI_DIMENSIONS":    "",
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
	if generateCalls != 1 || deleteCalls != 1 {
		t.Fatalf("idempotent reconcile repeated credential mutation: generate=%d delete=%d", generateCalls, deleteCalls)
	}

	initialAPIKey := string(runtimeSecret.Data["HINDSIGHT_API_TENANT_API_KEY"])
	initialDatabaseURL := string(runtimeSecret.Data["HINDSIGHT_API_DATABASE_URL"])

	tenant.Annotations = map[string]string{AnnotationHindsightEmbeddingRotation: "rotate-hindsight-embedding"}
	if _, err := r.reconcileHindsight(ctx, tenant); err != nil {
		t.Fatal(err)
	}
	if generateCalls != 2 || deleteCalls != 2 {
		t.Fatalf("rotation gateway calls generate=%d delete=%d, want 2/2 total", generateCalls, deleteCalls)
	}
	keys, ok := deletePayloads[1]["keys"].([]any)
	if !ok || len(keys) != 1 || keys[0] != "sk-hindsight-embedding-test" {
		t.Fatalf("rotation delete payload=%#v", deletePayloads[1])
	}
	models, ok = keyRequest["models"].([]any)
	if !ok || len(models) != 1 || models[0] != defaultAIEmbeddingModel {
		t.Fatalf("rotated embedding models=%#v", keyRequest["models"])
	}

	if err := c.Get(ctx, types.NamespacedName{Namespace: namespace, Name: "hindsight-runtime"}, &runtimeSecret); err != nil {
		t.Fatal(err)
	}
	if got := string(runtimeSecret.Data[hindsightEmbeddingSecretKey]); got != "sk-hindsight-embedding-rotated" {
		t.Fatalf("rotated embedding key=%q", got)
	}
	if got := string(runtimeSecret.Data["HINDSIGHT_API_TENANT_API_KEY"]); got != initialAPIKey {
		t.Fatal("Hindsight API key changed during embedding credential rotation")
	}
	if got := string(runtimeSecret.Data["HINDSIGHT_API_DATABASE_URL"]); got != initialDatabaseURL {
		t.Fatal("Hindsight database credential changed during embedding credential rotation")
	}
	if got := string(runtimeSecret.Data["HINDSIGHT_API_LLM_PROVIDER"]); got != "none" {
		t.Fatalf("Hindsight LLM provider changed to %q", got)
	}
	wantRevision := hindsightEmbeddingRotationRevision(tenant)
	if got := runtimeSecret.Annotations[AnnotationHindsightEmbeddingRevision]; got != wantRevision || got == "" {
		t.Fatalf("Hindsight embedding revision=%q, want %q", got, wantRevision)
	}

	var deployment appsv1.Deployment
	if err := c.Get(ctx, types.NamespacedName{Namespace: namespace, Name: "hindsight"}, &deployment); err != nil {
		t.Fatal(err)
	}
	if got := deployment.Spec.Template.Annotations[AnnotationHindsightEmbeddingRevision]; got != wantRevision {
		t.Fatalf("Hindsight pod rotation revision=%q, want %q", got, wantRevision)
	}
}
