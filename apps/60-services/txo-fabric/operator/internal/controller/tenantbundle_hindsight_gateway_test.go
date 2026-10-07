package controller

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	fabricv1alpha1 "github.com/charchess/vixens/apps/60-services/txo-fabric/operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	for _, unset := range []string{"max_budget", "rpm_limit", "tpm_limit"} {
		if _, exists := keyRequest[unset]; exists {
			t.Fatalf("v0 key request unexpectedly sets platform quota %q: %#v", unset, keyRequest[unset])
		}
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
	aliases, ok = deletePayloads[1]["key_aliases"].([]any)
	if !ok || len(aliases) != 1 || aliases[0] != "txo-fabric:hairem-sandbox:hindsight-embeddings" {
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

	tenant.Annotations = nil
	if _, err := r.reconcileHindsight(ctx, tenant); err != nil {
		t.Fatal(err)
	}
	if generateCalls != 2 || deleteCalls != 2 {
		t.Fatalf("annotation removal triggered a second Hindsight rotation: generate=%d delete=%d", generateCalls, deleteCalls)
	}
	if err := c.Get(ctx, types.NamespacedName{Namespace: namespace, Name: "hindsight-runtime"}, &runtimeSecret); err != nil {
		t.Fatal(err)
	}
	if got := runtimeSecret.Annotations[AnnotationHindsightEmbeddingRevision]; got != wantRevision {
		t.Fatalf("applied Hindsight revision changed after annotation removal: %q", got)
	}
	if err := c.Get(ctx, types.NamespacedName{Namespace: namespace, Name: "hindsight"}, &deployment); err != nil {
		t.Fatal(err)
	}
	if got := deployment.Spec.Template.Annotations[AnnotationHindsightEmbeddingRevision]; got != wantRevision {
		t.Fatalf("Hindsight pod rolled back revision after annotation removal: %q", got)
	}
}


type modelAccessRoundTripFunc func(*http.Request) (*http.Response, error)

func (fn modelAccessRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func modelAccessTestResponse(req *http.Request, status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Status:     http.StatusText(status),
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
		Request:    req,
	}
}

func TestHindsightEmbeddingCutoverStagesSharedRevocationUntilRuntimeAdoption(t *testing.T) {
	ctx := context.Background()
	scheme := postgresqlTestScheme(t)
	tenant := hindsightTestTenant()
	hindsightProfile := hindsightTestProfile()
	hindsightProfile.Spec.LLMAuthMode = "PlatformGateway"
	postgresqlProfile := postgresqlTestProfile()
	postgresqlSecret := hindsightPostgreSQLSecret(tenant, postgresqlProfile)
	gatewayProfile := aiGatewayTestProfile()
	namespace := tenantNamespace(tenant.Name)

	providerSecret := aiGatewayTestProviderSecret(tenant.Name)
	gatewayRuntime := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: tenantAIGatewayRuntimeSecretName, Namespace: namespace},
		Data:       map[string][]byte{"LITELLM_MASTER_KEY": []byte("tenant-master")},
	}
	gatewayConfig := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:        tenantAIGatewayConfigMapName,
			Namespace:   namespace,
			Annotations: map[string]string{"fabric.truxonline.io/config-hash": "embedding-ready"},
		},
		Data: map[string]string{"config.yaml": "model_list:\n  - model_name: txo-embedding\n"},
	}
	gatewayDeployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: tenantAIGatewayName, Namespace: namespace, Generation: 7},
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{
				"fabric.truxonline.io/config-hash": "embedding-ready",
			}}},
		},
		Status: appsv1.DeploymentStatus{ObservedGeneration: 7, AvailableReplicas: 1},
	}

	names := resolveHindsightNames(tenant, hindsightProfile)
	oldSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      names.Secret,
			Namespace: namespace,
			UID:       types.UID("hindsight-secret-uid"),
			Labels:    hindsightLabels(tenant, hindsightProfile, "runtime-secret"),
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"HINDSIGHT_API_TENANT_API_KEY": []byte("hindsight-api-key"),
			hindsightEmbeddingSecretKey:    []byte("sk-shared-old"),
		},
	}

	c := fake.NewClientBuilder().WithScheme(scheme).
		WithStatusSubresource(&appsv1.Deployment{}).
		WithObjects(
			tenant, hindsightProfile, postgresqlProfile, postgresqlSecret,
			gatewayProfile, providerSecret, gatewayRuntime, gatewayConfig, gatewayDeployment, oldSecret,
		).
		Build()
	r := &TenantBundleReconciler{Client: c, Scheme: scheme}

	var liveGateway appsv1.Deployment
	if err := c.Get(ctx, types.NamespacedName{Namespace: namespace, Name: tenantAIGatewayName}, &liveGateway); err != nil {
		t.Fatal(err)
	}
	liveGateway.Status.ObservedGeneration = liveGateway.Generation
	liveGateway.Status.AvailableReplicas = 1
	if err := c.Status().Update(ctx, &liveGateway); err != nil {
		t.Fatal(err)
	}

	sharedDeletes := 0
	tenantDeletes := 0
	tenantGenerates := 0
	oldHTTPClient := http.DefaultClient
	http.DefaultClient = &http.Client{Transport: modelAccessRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch req.URL.Host {
		case "shared-gateway:4000":
			if req.Header.Get("Authorization") != "Bearer shared-master" {
				t.Fatalf("shared gateway auth=%q", req.Header.Get("Authorization"))
			}
			if req.URL.Path != "/key/delete" {
				t.Fatalf("unexpected shared gateway path %q", req.URL.Path)
			}
			sharedDeletes++
			return modelAccessTestResponse(req, http.StatusOK, `{"deleted_keys":["legacy"]}`), nil
		case "txo-ai-gateway." + namespace + ".svc:4000":
			if req.Header.Get("Authorization") != "Bearer tenant-master" {
				t.Fatalf("tenant gateway auth=%q", req.Header.Get("Authorization"))
			}
			switch req.URL.Path {
			case "/key/delete":
				tenantDeletes++
				return modelAccessTestResponse(req, http.StatusNotFound, "not found"), nil
			case "/key/generate":
				tenantGenerates++
				return modelAccessTestResponse(req, http.StatusOK, `{"key":"sk-tenant-embedding"}`), nil
			default:
				t.Fatalf("unexpected tenant gateway path %q", req.URL.Path)
			}
		default:
			t.Fatalf("unexpected gateway host %q", req.URL.Host)
		}
		return nil, nil
	})}
	defer func() { http.DefaultClient = oldHTTPClient }()
	t.Setenv("TXO_AI_GATEWAY_URL", "http://shared-gateway:4000")
	t.Setenv("TXO_AI_GATEWAY_ADMIN_TOKEN", "shared-master")

	blocked, err := r.ensureHindsightSecret(ctx, tenant, hindsightProfile, names, postgresqlSecret)
	if err != nil {
		t.Fatal(err)
	}
	if blocked != nil {
		t.Fatalf("unexpected cutover block: %#v", blocked)
	}
	if tenantDeletes != 1 || tenantGenerates != 1 || sharedDeletes != 0 {
		t.Fatalf("destination preparation calls tenant delete/generate=%d/%d shared delete=%d; source must remain untouched", tenantDeletes, tenantGenerates, sharedDeletes)
	}

	var migrated corev1.Secret
	if err := c.Get(ctx, types.NamespacedName{Namespace: namespace, Name: names.Secret}, &migrated); err != nil {
		t.Fatal(err)
	}
	wantBackendID := "tenant:" + tenant.Name + ":" + gatewayProfile.Name + ":4000"
	if got := migrated.Annotations[AnnotationHindsightEmbeddingBackend]; got != wantBackendID {
		t.Fatalf("migrated embedding backend=%q want %q", got, wantBackendID)
	}
	if got := migrated.Annotations[AnnotationHindsightLegacyCleanupPending]; got != sharedModelAccessBackendID {
		t.Fatalf("legacy cleanup marker=%q want shared", got)
	}
	if got := string(migrated.Data["HINDSIGHT_API_EMBEDDINGS_OPENAI_BASE_URL"]); got != "http://txo-ai-gateway."+namespace+".svc:4000/v1" {
		t.Fatalf("migrated Hindsight embedding base URL=%q", got)
	}
	if got := string(migrated.Data[hindsightEmbeddingSecretKey]); got != "sk-tenant-embedding" {
		t.Fatalf("migrated tenant embedding key=%q", got)
	}

	if blocked, err := r.ensureHindsightNetworkPolicy(ctx, tenant, hindsightProfile, postgresqlProfile, names); err != nil {
		t.Fatal(err)
	} else if blocked != nil {
		t.Fatalf("unexpected migration NetworkPolicy block: %#v", blocked)
	}
	var migrationPolicy networkingv1.NetworkPolicy
	if err := c.Get(ctx, types.NamespacedName{Namespace: namespace, Name: names.NetworkPolicy}, &migrationPolicy); err != nil {
		t.Fatal(err)
	}
	if len(migrationPolicy.Spec.Egress) != 4 {
		t.Fatalf("migration Hindsight egress=%d want DNS + PostgreSQL + shared + tenant gateway", len(migrationPolicy.Spec.Egress))
	}

	hindsightDeployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:       names.Deployment,
			Namespace:  namespace,
			Generation: 3,
			Labels:     hindsightLabels(tenant, hindsightProfile, "deployment"),
		},
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{
				AnnotationHindsightEmbeddingRevision:  migrated.Annotations[AnnotationHindsightEmbeddingRevision],
				AnnotationHindsightEmbeddingSecretUID: string(migrated.UID),
			}}},
		},
	}
	if err := c.Create(ctx, hindsightDeployment); err != nil {
		t.Fatal(err)
	}
	if err := c.Get(ctx, types.NamespacedName{Namespace: namespace, Name: names.Deployment}, hindsightDeployment); err != nil {
		t.Fatal(err)
	}
	hindsightDeployment.Status.ObservedGeneration = hindsightDeployment.Generation
	hindsightDeployment.Status.AvailableReplicas = 1
	if err := c.Status().Update(ctx, hindsightDeployment); err != nil {
		t.Fatal(err)
	}

	blocked, err = r.ensureHindsightSecret(ctx, tenant, hindsightProfile, names, postgresqlSecret)
	if err != nil {
		t.Fatal(err)
	}
	if blocked != nil {
		t.Fatalf("unexpected finalization block: %#v", blocked)
	}
	if sharedDeletes != 1 {
		t.Fatalf("shared source credential revocations=%d want 1 after runtime adoption", sharedDeletes)
	}
	if tenantGenerates != 1 {
		t.Fatalf("finalization unexpectedly reminted tenant key: %d", tenantGenerates)
	}
	if err := c.Get(ctx, types.NamespacedName{Namespace: namespace, Name: names.Secret}, &migrated); err != nil {
		t.Fatal(err)
	}
	if got := migrated.Annotations[AnnotationHindsightLegacyCleanupPending]; got != "" {
		t.Fatalf("legacy cleanup marker remained after source revocation: %q", got)
	}

	if blocked, err := r.ensureHindsightNetworkPolicy(ctx, tenant, hindsightProfile, postgresqlProfile, names); err != nil {
		t.Fatal(err)
	} else if blocked != nil {
		t.Fatalf("unexpected final NetworkPolicy block: %#v", blocked)
	}
	var finalPolicy networkingv1.NetworkPolicy
	if err := c.Get(ctx, types.NamespacedName{Namespace: namespace, Name: names.NetworkPolicy}, &finalPolicy); err != nil {
		t.Fatal(err)
	}
	if len(finalPolicy.Spec.Egress) != 3 {
		t.Fatalf("settled Hindsight egress=%d want DNS + PostgreSQL + tenant gateway only", len(finalPolicy.Spec.Egress))
	}
	for _, rule := range finalPolicy.Spec.Egress {
		for _, peer := range rule.To {
			if peer.NamespaceSelector != nil && peer.NamespaceSelector.MatchLabels["kubernetes.io/metadata.name"] == "txo-fabric-system" &&
				peer.PodSelector != nil && peer.PodSelector.MatchLabels[LabelName] == "txo-ai-gateway" {
				t.Fatal("settled Hindsight NetworkPolicy still allows the historical shared gateway")
			}
		}
	}
}

func TestHindsightTenantEmbeddingRefusesFallbackWhenProviderCredentialDisappears(t *testing.T) {
	ctx := context.Background()
	scheme := postgresqlTestScheme(t)
	tenant := hindsightTestTenant()
	profile := hindsightTestProfile()
	profile.Spec.LLMAuthMode = "PlatformGateway"
	namespace := tenantNamespace(tenant.Name)
	gatewayProfile := aiGatewayTestProfile()
	gatewayRuntime := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: tenantAIGatewayRuntimeSecretName, Namespace: namespace},
		Data:       map[string][]byte{"LITELLM_MASTER_KEY": []byte("tenant-master")},
	}
	runtime := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "hindsight-runtime",
			Namespace: namespace,
			Labels:    hindsightLabels(tenant, profile, "runtime-secret"),
			Annotations: map[string]string{
				AnnotationHindsightEmbeddingBackend:    "tenant:" + tenant.Name + ":" + gatewayProfile.Name + ":4000",
				AnnotationHindsightEmbeddingBackendURL: "http://txo-ai-gateway." + namespace + ".svc:4000",
			},
		},
		Data: map[string][]byte{
			"HINDSIGHT_API_TENANT_API_KEY": []byte("hindsight-api"),
			hindsightEmbeddingSecretKey:    []byte("tenant-key"),
		},
	}
	postgresqlProfile := postgresqlTestProfile()
	postgresqlSecret := hindsightPostgreSQLSecret(tenant, postgresqlProfile)
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
		tenant, profile, gatewayProfile, gatewayRuntime, runtime, postgresqlProfile, postgresqlSecret,
	).Build()
	r := &TenantBundleReconciler{Client: c, Scheme: scheme}

	names := resolveHindsightNames(tenant, profile)
	blocked, err := r.ensureHindsightSecret(ctx, tenant, profile, names, postgresqlSecret)
	if err != nil {
		t.Fatal(err)
	}
	if blocked == nil || blocked.Reason != "EmbeddingBackendPending" {
		t.Fatalf("missing tenant provider credential must fail closed without shared fallback: %#v", blocked)
	}
	if !strings.Contains(blocked.Message, "refusing fallback") {
		t.Fatalf("unexpected fail-closed message: %q", blocked.Message)
	}
}
