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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestResolveModelAccessBackendUsesTenantLiteLLM(t *testing.T) {
	ctx := context.Background()
	scheme := testScheme(t)
	tenant := testTenant()
	tenant.Spec.AIGateway = &fabricv1alpha1.TenantAIGatewaySpec{ProfileRef: defaultAIGatewayProfileName}
	profile := aiGatewayTestProfile()
	runtimeSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      tenantAIGatewayRuntimeSecretName,
			Namespace: tenantNamespace(tenant.Name),
		},
		Data: map[string][]byte{"LITELLM_MASTER_KEY": []byte("tenant-master-key")},
	}

	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(tenant, profile, runtimeSecret).Build()
	r := &AgentIdentityReconciler{Client: c, Scheme: scheme}

	backend, err := r.resolveModelAccessBackend(ctx, tenant)
	if err != nil {
		t.Fatal(err)
	}
	wantBackendID := "tenant:" + tenant.Name + ":" + profile.Name + ":4000"
	if backend.ID != wantBackendID {
		t.Fatalf("backend ID=%q want %q", backend.ID, wantBackendID)
	}
	if backend.URL != "http://txo-ai-gateway.tenant-hairem-sandbox.svc:4000" {
		t.Fatalf("backend URL=%q", backend.URL)
	}
	if backend.AdminToken != "tenant-master-key" {
		t.Fatal("tenant LiteLLM master key was not resolved")
	}
	if backend.Model != tenantAIAgentModel {
		t.Fatalf("backend model=%q want %q", backend.Model, tenantAIAgentModel)
	}
}

func TestModelAccessBackendCutoverRevokesSourceOnlyAfterRuntimeAdoption(t *testing.T) {
	ctx := context.Background()
	scheme := testScheme(t)
	tenant := testTenant()
	agent := testAgentIdentity()
	namespace := tenantNamespace(tenant.Name)
	alias := modelAccessKeyAlias(agent, tenant)

	sharedDeletes := 0
	sharedGateway := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Header.Get("Authorization") != "Bearer shared-admin" {
			t.Fatalf("shared authorization=%q", req.Header.Get("Authorization"))
		}
		if req.URL.Path != "/key/delete" {
			t.Fatalf("unexpected shared gateway path %q", req.URL.Path)
		}
		var payload map[string]any
		if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		aliases, _ := payload["key_aliases"].([]any)
		if len(aliases) != 1 || aliases[0] != alias {
			t.Fatalf("shared delete payload=%#v", payload)
		}
		sharedDeletes++
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"deleted_keys":["old-shared-key"]}`))
	}))
	defer sharedGateway.Close()
	t.Setenv("TXO_AI_GATEWAY_URL", sharedGateway.URL)
	t.Setenv("TXO_AI_GATEWAY_ADMIN_TOKEN", "shared-admin")

	tenantDeletes := 0
	tenantGenerates := 0
	tenantGateway := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Header.Get("Authorization") != "Bearer tenant-master" {
			t.Fatalf("tenant authorization=%q", req.Header.Get("Authorization"))
		}
		var payload map[string]any
		if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		switch req.URL.Path {
		case "/key/delete":
			tenantDeletes++
			http.Error(w, "not found", http.StatusNotFound)
		case "/key/generate":
			tenantGenerates++
			if payload["key_alias"] != alias {
				t.Fatalf("tenant key alias=%#v", payload["key_alias"])
			}
			models, _ := payload["models"].([]any)
			if len(models) != 1 || models[0] != tenantAIAgentModel {
				t.Fatalf("tenant models=%#v", payload["models"])
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"key":"tenant-virtual-key"}`))
		default:
			t.Fatalf("unexpected tenant gateway path %q", req.URL.Path)
		}
	}))
	defer tenantGateway.Close()

	oldSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      modelAccessSecretName(agent.Spec.AgentKey),
			Namespace: namespace,
			UID:       types.UID("model-access-uid"),
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{modelAccessSecretKey: []byte("old-shared-key")},
	}
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(agent, oldSecret).Build()
	r := &AgentIdentityReconciler{Client: c, Scheme: scheme}
	backend := modelAccessBackend{
		ID:         "tenant:hairem-sandbox:litellm-tenant:4000",
		URL:        tenantGateway.URL,
		AdminToken: "tenant-master",
		Model:      tenantAIAgentModel,
	}

	uid, revision, err := r.ensureModelAccessWithBackend(ctx, agent, tenant, namespace, backend)
	if err != nil {
		t.Fatal(err)
	}
	if uid != "model-access-uid" || revision == "" {
		t.Fatalf("cutover result uid=%q revision=%q", uid, revision)
	}
	if sharedDeletes != 0 || tenantDeletes != 1 || tenantGenerates != 1 {
		t.Fatalf("staging calls sharedDelete=%d tenantDelete=%d tenantGenerate=%d, want 0/1/1", sharedDeletes, tenantDeletes, tenantGenerates)
	}

	var staged corev1.Secret
	if err := c.Get(ctx, types.NamespacedName{Name: oldSecret.Name, Namespace: namespace}, &staged); err != nil {
		t.Fatal(err)
	}
	if got := string(staged.Data[modelAccessSecretKey]); got != "tenant-virtual-key" {
		t.Fatalf("staged key=%q", got)
	}
	if staged.Annotations[AnnotationModelAccessBackend] != backend.ID ||
		staged.Annotations[AnnotationModelAccessBackendURL] != backend.URL ||
		staged.Annotations[AnnotationModelAccessRevision] != revision ||
		staged.Annotations[AnnotationModelAccessPendingRevokeBackend] != sharedModelAccessBackendID ||
		staged.Annotations[AnnotationModelAccessPendingRevokeURL] != sharedGateway.URL {
		t.Fatalf("staged annotations=%#v", staged.Annotations)
	}

	// Retrying reconciliation before the runtime adopts the destination must not
	// remint a key or revoke the still-working source credential.
	if _, secondRevision, err := r.ensureModelAccessWithBackend(ctx, agent, tenant, namespace, backend); err != nil {
		t.Fatal(err)
	} else if secondRevision != revision {
		t.Fatalf("idempotent revision=%q want %q", secondRevision, revision)
	}
	if sharedDeletes != 0 || tenantDeletes != 1 || tenantGenerates != 1 {
		t.Fatalf("pre-adoption retry changed credentials: sharedDelete=%d tenantDelete=%d tenantGenerate=%d", sharedDeletes, tenantDeletes, tenantGenerates)
	}

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:       runtimeName(agent.Spec.AgentKey),
			Namespace:  namespace,
			Generation: 2,
		},
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{
				AnnotationModelAccessRevision:  revision,
				AnnotationModelAccessSecretUID: uid,
			}}},
		},
		Status: appsv1.DeploymentStatus{
			ObservedGeneration: 1,
			AvailableReplicas:   1,
		},
	}

	pending, err := r.finalizeModelAccessBackendCutover(ctx, agent, tenant, namespace, backend, deployment)
	if err != nil {
		t.Fatal(err)
	}
	if !pending || sharedDeletes != 0 {
		t.Fatalf("source revoked before Deployment adopted destination: pending=%v sharedDelete=%d", pending, sharedDeletes)
	}

	deployment.Status.ObservedGeneration = deployment.Generation
	if err := c.Create(ctx, deployment); err != nil {
		t.Fatal(err)
	}

	pending, err = r.finalizeModelAccessBackendCutover(ctx, agent, tenant, namespace, backend, deployment)
	if err != nil {
		t.Fatal(err)
	}
	if pending {
		t.Fatal("cutover cleanup unexpectedly remained pending after runtime adoption")
	}
	if sharedDeletes != 1 {
		t.Fatalf("source delete calls=%d want 1 after runtime adoption", sharedDeletes)
	}

	var finalized corev1.Secret
	if err := c.Get(ctx, types.NamespacedName{Name: oldSecret.Name, Namespace: namespace}, &finalized); err != nil {
		t.Fatal(err)
	}
	if got, _ := pendingModelAccessRevokeBackend(&finalized); got != "" {
		t.Fatalf("pending source backend remained after finalization: %#v", finalized.Annotations)
	}
	if got := string(finalized.Data[modelAccessSecretKey]); got != "tenant-virtual-key" {
		t.Fatalf("finalized key=%q", got)
	}

	pending, err = r.finalizeModelAccessBackendCutover(ctx, agent, tenant, namespace, backend, deployment)
	if err != nil {
		t.Fatal(err)
	}
	if pending || sharedDeletes != 1 {
		t.Fatalf("finalization is not idempotent: pending=%v sharedDelete=%d", pending, sharedDeletes)
	}
}

func TestModelAccessBackendCutoverDestinationFailureKeepsSourceCredential(t *testing.T) {
	ctx := context.Background()
	scheme := testScheme(t)
	tenant := testTenant()
	agent := testAgentIdentity()
	namespace := tenantNamespace(tenant.Name)

	sharedDeletes := 0
	sharedGateway := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		sharedDeletes++
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"deleted_keys":["old-shared-key"]}`))
	}))
	defer sharedGateway.Close()
	t.Setenv("TXO_AI_GATEWAY_URL", sharedGateway.URL)
	t.Setenv("TXO_AI_GATEWAY_ADMIN_TOKEN", "shared-admin")

	unreachable := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	unreachableURL := unreachable.URL
	unreachable.Close()

	oldSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      modelAccessSecretName(agent.Spec.AgentKey),
			Namespace: namespace,
			UID:       types.UID("model-access-uid"),
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{modelAccessSecretKey: []byte("old-shared-key")},
	}
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(agent, oldSecret).Build()
	r := &AgentIdentityReconciler{Client: c, Scheme: scheme}
	backend := modelAccessBackend{
		ID:         "tenant:hairem-sandbox:litellm-tenant:4000",
		URL:        unreachableURL,
		AdminToken: "tenant-master",
		Model:      tenantAIAgentModel,
	}

	if _, _, err := r.ensureModelAccessWithBackend(ctx, agent, tenant, namespace, backend); err == nil {
		t.Fatal("expected unreachable destination to fail cutover preparation")
	}
	if sharedDeletes != 0 {
		t.Fatalf("source credential was revoked before destination preparation: deletes=%d", sharedDeletes)
	}

	var current corev1.Secret
	if err := c.Get(ctx, types.NamespacedName{Name: oldSecret.Name, Namespace: namespace}, &current); err != nil {
		t.Fatal(err)
	}
	if got := string(current.Data[modelAccessSecretKey]); got != "old-shared-key" {
		t.Fatalf("source key changed after destination failure: %q", got)
	}
	if normalizedAppliedModelAccessBackend(&current) != sharedModelAccessBackendID {
		t.Fatalf("source backend changed after destination failure: %#v", current.Annotations)
	}
	if pending, _ := pendingModelAccessRevokeBackend(&current); pending != "" {
		t.Fatalf("pending cutover metadata recorded despite destination failure: %#v", current.Annotations)
	}
}

func TestRecordedModelAccessBackendRejectsCrossTenantBackend(t *testing.T) {
	scheme := testScheme(t)
	tenant := testTenant()
	r := &AgentIdentityReconciler{
		Client: fake.NewClientBuilder().WithScheme(scheme).Build(),
		Scheme: scheme,
	}
	if _, err := r.recordedModelAccessBackend(
		context.Background(),
		tenant,
		"tenant:indiba:litellm-standard:4000",
		"http://txo-ai-gateway.tenant-indiba.svc:4000",
	); err == nil {
		t.Fatal("expected cross-tenant recorded backend to fail closed")
	}
}

func TestTenantModelBindingFlowsIntoHermesPolicyAndRuntime(t *testing.T) {
	ctx := context.Background()
	scheme := testScheme(t)
	tenant := testTenant()
	tenant.Spec.AIGateway = &fabricv1alpha1.TenantAIGatewaySpec{ProfileRef: defaultAIGatewayProfileName}
	agent := testAgentIdentity()
	profile := testRuntimeProfile()
	backend := modelAccessBackend{
		ID:    "tenant:hairem-sandbox:litellm-tenant:4000",
		URL:   "http://txo-ai-gateway.tenant-hairem-sandbox.svc:4000",
		Model: tenantAIAgentModel,
	}

	policy, err := resolveToolsetPolicy(agent, profile, backend.runtimeBinding())
	if err != nil {
		t.Fatal(err)
	}
	if policy.ModelDefault != tenantAIAgentModel ||
		policy.ModelBaseURL != "http://txo-ai-gateway.tenant-hairem-sandbox.svc:4000/v1" {
		t.Fatalf("resolved model binding=%#v", policy)
	}

	c := fake.NewClientBuilder().WithScheme(scheme).Build()
	r := &AgentIdentityReconciler{Client: c, Scheme: scheme}
	if err := r.ensureDeployment(ctx, agent, tenant, profile, tenantNamespace(tenant.Name), "secret-uid", "tenant-revision", policy); err != nil {
		t.Fatal(err)
	}
	var deployment appsv1.Deployment
	if err := c.Get(ctx, types.NamespacedName{Name: runtimeName(agent.Spec.AgentKey), Namespace: tenantNamespace(tenant.Name)}, &deployment); err != nil {
		t.Fatal(err)
	}
	container := deployment.Spec.Template.Spec.Containers[0]
	if got := envValue(container.Env, "OPENAI_BASE_URL"); got != policy.ModelBaseURL {
		t.Fatalf("OPENAI_BASE_URL=%q want %q", got, policy.ModelBaseURL)
	}
	if got := envValue(container.Env, "HERMES_MODEL"); got != tenantAIAgentModel {
		t.Fatalf("HERMES_MODEL=%q want %q", got, tenantAIAgentModel)
	}
}
