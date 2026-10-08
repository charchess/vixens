package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	fabricv1alpha1 "github.com/charchess/vixens/apps/60-services/txo-fabric/operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestOperatorContractReconcilesWithTenantLocalAgentKey(t *testing.T) {
	ctx := context.Background()
	scheme := testScheme(t)
	tenant := testTenant()
	profile := testRuntimeProfile()
	agent := &fabricv1alpha1.AgentIdentity{
		ObjectMeta: metav1.ObjectMeta{Name: "hairem-sandbox-tina"},
		Spec: fabricv1alpha1.AgentIdentitySpec{
			TenantRef:   fabricv1alpha1.ObjectReference{Name: "hairem-sandbox"},
			AgentKey:    "tina",
			DisplayName: "Tina",
			Runtime:     fabricv1alpha1.AgentRuntimeBinding{ProfileRef: "hermes-default"},
		},
	}

	gatewayProfile := aiGatewayTestProfile()
	gatewayRuntimeSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: tenantAIGatewayRuntimeSecretName, Namespace: tenantNamespace(tenant.Name)},
		Data:       map[string][]byte{"LITELLM_MASTER_KEY": []byte("tenant-master")},
	}
	modelBackendID := "tenant:" + tenant.Name + ":" + gatewayProfile.Name + ":4000"
	modelRevision := modelAccessBackendRevision(modelBackendID, "")
	modelSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      modelAccessSecretName(agent.Spec.AgentKey),
			Namespace: tenantNamespace(tenant.Name),
			UID:       types.UID("tenant-model-access-uid"),
			Annotations: map[string]string{
				AnnotationModelAccessBackend:    modelBackendID,
				AnnotationModelAccessBackendURL: "http://txo-ai-gateway." + tenantNamespace(tenant.Name) + ".svc:4000",
				AnnotationModelAccessRevision:   modelRevision,
			},
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{modelAccessSecretKey: []byte("test-virtual-key")},
	}

	c := fake.NewClientBuilder().WithScheme(scheme).
		WithStatusSubresource(&fabricv1alpha1.TenantBundle{}, &fabricv1alpha1.AgentIdentity{}).
		WithObjects(tenant, profile, agent, gatewayProfile, gatewayRuntimeSecret, modelSecret).
		Build()

	reconcileTenant(t, ctx, c, scheme, tenant.Name)

	var ns corev1.Namespace
	if err := c.Get(ctx, types.NamespacedName{Name: "tenant-hairem-sandbox"}, &ns); err != nil {
		t.Fatalf("tenant namespace not reconciled: %v", err)
	}
	var defaultDeny networkingv1.NetworkPolicy
	if err := c.Get(ctx, types.NamespacedName{Name: "txo-fabric-default-deny", Namespace: ns.Name}, &defaultDeny); err != nil {
		t.Fatalf("default deny not reconciled: %v", err)
	}

	agentReconciler := &AgentIdentityReconciler{Client: c, Scheme: scheme}
	reqAgent := ctrl.Request{NamespacedName: types.NamespacedName{Name: agent.Name}}
	if _, err := agentReconciler.Reconcile(ctx, reqAgent); err != nil {
		t.Fatal(err)
	}
	if _, err := agentReconciler.Reconcile(ctx, reqAgent); err != nil {
		t.Fatal(err)
	}

	if err := c.Get(ctx, types.NamespacedName{Name: "hermes-tina-model-access", Namespace: ns.Name}, modelSecret); err != nil {
		t.Fatalf("model access Secret not reconciled: %v", err)
	}
	if string(modelSecret.Data["OPENAI_API_KEY"]) != "test-virtual-key" {
		t.Fatal("model access Secret does not contain generated virtual key")
	}

	var deployment appsv1.Deployment
	if err := c.Get(ctx, types.NamespacedName{Name: "hermes-tina", Namespace: ns.Name}, &deployment); err != nil {
		t.Fatalf("runtime deployment not reconciled: %v", err)
	}
	container := deployment.Spec.Template.Spec.Containers[0]
	if len(container.Args) != 3 || container.Args[0] != "gateway" || container.Args[1] != "run" || container.Args[2] != "--replace" {
		t.Fatalf("unexpected Hermes args: %#v", container.Args)
	}
	if container.ReadinessProbe == nil || container.ReadinessProbe.Exec == nil {
		t.Fatal("semantic Hermes readiness probe is missing")
	}
	for name, probe := range map[string]*corev1.Probe{
		"startup":   container.StartupProbe,
		"readiness": container.ReadinessProbe,
		"liveness":  container.LivenessProbe,
	} {
		if probe == nil || probe.Exec == nil {
			t.Fatalf("%s Hermes probe is missing", name)
		}
		if probe.TimeoutSeconds != 3 {
			t.Fatalf("%s Hermes probe timeoutSeconds=%d, want 3", name, probe.TimeoutSeconds)
		}
	}
	if deployment.Spec.Template.Annotations["vixens.io/explicitly-allow-root"] != "true" {
		t.Fatal("s6 compatibility annotation is missing")
	}
	if len(deployment.Spec.Template.Spec.InitContainers) != 1 {
		t.Fatalf("unexpected init container count: %d", len(deployment.Spec.Template.Spec.InitContainers))
	}
	bootstrap := deployment.Spec.Template.Spec.InitContainers[0]
	if bootstrap.Name != "bootstrap-profile" || len(bootstrap.Command) != 3 {
		t.Fatalf("unexpected bootstrap container: %#v", bootstrap)
	}
	bootstrapScript := bootstrap.Command[2]
	for _, required := range []string{
		`install -d -o hermes -g hermes -m 0750 "${HERMES_HOME}/backups" "${HERMES_HOME}/backups/config"`,
		`chown hermes:hermes "${HERMES_HOME}/config.yaml"`,
		"/command/s6-setuidgid hermes /opt/hermes/.venv/bin/hermes config set skills.external_dirs",
	} {
		if !strings.Contains(bootstrapScript, required) {
			t.Fatalf("bootstrap script missing %q: %s", required, bootstrapScript)
		}
	}
	if got := envValue(container.Env, "TXO_AGENT_ID"); got != "hairem-sandbox-tina" {
		t.Fatalf("TXO_AGENT_ID = %q, want global CR identity", got)
	}
	if got := envValue(container.Env, "TXO_AGENT_KEY"); got != "tina" {
		t.Fatalf("TXO_AGENT_KEY = %q, want tenant-local key", got)
	}
	if got := envValue(container.Env, "TXO_LLM_AUTH_MODE"); got != "gateway" {
		t.Fatalf("TXO_LLM_AUTH_MODE = %q, want gateway", got)
	}
	if got := envValue(container.Env, "OPENAI_BASE_URL"); got != "http://txo-ai-gateway.tenant-hairem-sandbox.svc:4000/v1" {
		t.Fatalf("OPENAI_BASE_URL = %q, want tenant LiteLLM", got)
	}
	if got := envValue(container.Env, "HERMES_MODEL"); got != tenantAIAgentModel {
		t.Fatalf("HERMES_MODEL = %q want %q", got, tenantAIAgentModel)
	}
	apiKey := envVar(container.Env, "OPENAI_API_KEY")
	if apiKey == nil || apiKey.ValueFrom == nil || apiKey.ValueFrom.SecretKeyRef == nil || apiKey.ValueFrom.SecretKeyRef.Name != "hermes-tina-model-access" || apiKey.ValueFrom.SecretKeyRef.Key != "OPENAI_API_KEY" {
		t.Fatalf("OPENAI_API_KEY does not reference scoped model access Secret: %#v", apiKey)
	}

	var pvc corev1.PersistentVolumeClaim
	if err := c.Get(ctx, types.NamespacedName{Name: "hermes-tina-data", Namespace: ns.Name}, &pvc); err != nil {
		t.Fatalf("runtime PVC not reconciled: %v", err)
	}
	if pvc.Spec.StorageClassName == nil || *pvc.Spec.StorageClassName != "truenas-iscsi-delete" {
		t.Fatalf("unexpected storage class: %#v", pvc.Spec.StorageClassName)
	}

	var egress networkingv1.NetworkPolicy
	if err := c.Get(ctx, types.NamespacedName{Name: "hermes-tina-egress", Namespace: ns.Name}, &egress); err != nil {
		t.Fatalf("runtime egress policy not reconciled: %v", err)
	}
	if len(egress.Spec.Egress) != 3 {
		t.Fatalf("expected DNS + AI gateway + tenant Hindsight egress rules, got %d", len(egress.Spec.Egress))
	}

	var current fabricv1alpha1.AgentIdentity
	if err := c.Get(ctx, types.NamespacedName{Name: agent.Name}, &current); err != nil {
		t.Fatal(err)
	}
	if current.Status.Memory.BankID != "hairem-sandbox-tina" {
		t.Fatalf("resolved bank = %q, want hairem-sandbox-tina", current.Status.Memory.BankID)
	}
	modelReady := false
	for _, condition := range current.Status.Conditions {
		if condition.Type == "ModelAccessReady" && condition.Status == metav1.ConditionTrue && condition.Reason == "GatewayCredentialReady" {
			modelReady = true
			if !strings.Contains(condition.Message, "model="+tenantAIAgentModel) || !strings.Contains(condition.Message, "gateway=http://txo-ai-gateway.tenant-hairem-sandbox.svc:4000") {
				t.Fatalf("ModelAccessReady does not expose effective non-secret binding: %q", condition.Message)
			}
		}
	}
	if !modelReady {
		t.Fatalf("ModelAccessReady condition missing: %#v", current.Status.Conditions)
	}
	statusJSON, err := json.Marshal(current.Status)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(statusJSON, []byte("test-virtual-key")) {
		t.Fatal("AgentIdentity status leaked the scoped AI gateway credential")
	}
}

func TestAgentModelAccessRotationRevokesOldKeyAndRollsRuntime(t *testing.T) {
	ctx := context.Background()
	scheme := testScheme(t)
	tenant := testTenant()
	profile := testRuntimeProfile()
	agent := testAgentIdentity()
	agent.Annotations = map[string]string{AnnotationModelAccessRotation: "rotate-agent-model-key"}

	deleteCalls := 0
	generateCalls := 0
	gateway := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Header.Get("Authorization") != "Bearer tenant-master" {
			t.Fatalf("unexpected gateway authorization header")
		}
		var payload map[string]any
		if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		switch req.URL.Path {
		case "/key/delete":
			deleteCalls++
			aliases, ok := payload["key_aliases"].([]any)
			if !ok || len(aliases) != 1 || aliases[0] != modelAccessKeyAlias(agent, tenant) {
				t.Fatalf("rotation delete payload = %#v", payload)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"deleted_keys":["old-agent-key"]}`))
		case "/key/generate":
			generateCalls++
			models, ok := payload["models"].([]any)
			if !ok || len(models) != 1 || models[0] != tenantAIAgentModel {
				t.Fatalf("rotation models = %#v", payload["models"])
			}
			if payload["key_alias"] != modelAccessKeyAlias(agent, tenant) {
				t.Fatalf("rotation key alias = %#v", payload["key_alias"])
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"key":"new-agent-key"}`))
		default:
			t.Fatalf("unexpected gateway path %q", req.URL.Path)
		}
	}))
	defer gateway.Close()

	namespace := tenantNamespace(tenant.Name)
	backend := modelAccessBackend{
		ID:         "tenant:hairem-sandbox:litellm-standard:4000",
		URL:        gateway.URL,
		AdminToken: "tenant-master",
		Model:      tenantAIAgentModel,
	}
	oldSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      modelAccessSecretName(agent.Spec.AgentKey),
			Namespace: namespace,
			UID:       types.UID("model-access-secret-uid"),
			Annotations: map[string]string{
				AnnotationModelAccessRevision:   "previous",
				AnnotationModelAccessBackend:    backend.ID,
				AnnotationModelAccessBackendURL: backend.URL,
			},
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{modelAccessSecretKey: []byte("old-agent-key")},
	}

	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(agent, oldSecret).Build()
	r := &AgentIdentityReconciler{Client: c, Scheme: scheme}

	secretUID, appliedRevision, err := r.ensureModelAccessWithBackend(ctx, agent, tenant, namespace, backend)
	if err != nil {
		t.Fatal(err)
	}
	if deleteCalls != 1 || generateCalls != 1 {
		t.Fatalf("rotation gateway calls delete=%d generate=%d, want 1/1", deleteCalls, generateCalls)
	}
	if secretUID != "model-access-secret-uid" {
		t.Fatalf("model access Secret UID = %q", secretUID)
	}

	var rotated corev1.Secret
	if err := c.Get(ctx, types.NamespacedName{Name: oldSecret.Name, Namespace: namespace}, &rotated); err != nil {
		t.Fatal(err)
	}
	if got := string(rotated.Data[modelAccessSecretKey]); got != "new-agent-key" {
		t.Fatalf("rotated model key = %q", got)
	}
	wantRevision := modelAccessRevisionForBackend(backend, modelAccessRotationRevision(agent))
	if got := rotated.Annotations[AnnotationModelAccessRevision]; got != wantRevision || got == "" {
		t.Fatalf("rotation revision = %q, want %q", got, wantRevision)
	}

	policy, err := resolveToolsetPolicy(agent, profile, backend.runtimeBinding())
	if err != nil {
		t.Fatal(err)
	}
	if err := r.ensureDeployment(ctx, agent, tenant, profile, namespace, secretUID, appliedRevision, policy); err != nil {
		t.Fatal(err)
	}
	var deployment appsv1.Deployment
	if err := c.Get(ctx, types.NamespacedName{Name: runtimeName(agent.Spec.AgentKey), Namespace: namespace}, &deployment); err != nil {
		t.Fatal(err)
	}
	if got := deployment.Spec.Template.Annotations[AnnotationModelAccessRevision]; got != wantRevision {
		t.Fatalf("pod model-access revision = %q, want %q", got, wantRevision)
	}
	if got := deployment.Spec.Template.Annotations[AnnotationModelAccessSecretUID]; got != secretUID {
		t.Fatalf("pod model-access Secret UID = %q, want %q", got, secretUID)
	}

	if _, _, err := r.ensureModelAccessWithBackend(ctx, agent, tenant, namespace, backend); err != nil {
		t.Fatal(err)
	}
	if deleteCalls != 1 || generateCalls != 1 {
		t.Fatalf("idempotent reconcile repeated rotation: delete=%d generate=%d", deleteCalls, generateCalls)
	}

	agent.Annotations = nil
	uidAfterRemoval, revisionAfterRemoval, err := r.ensureModelAccessWithBackend(ctx, agent, tenant, namespace, backend)
	if err != nil {
		t.Fatal(err)
	}
	if deleteCalls != 1 || generateCalls != 1 {
		t.Fatalf("annotation removal triggered a second rotation: delete=%d generate=%d", deleteCalls, generateCalls)
	}
	if uidAfterRemoval != secretUID || revisionAfterRemoval != wantRevision {
		t.Fatalf("applied credential state changed after annotation removal: uid=%q revision=%q", uidAfterRemoval, revisionAfterRemoval)
	}
}

func TestSameAgentKeyAcrossDifferentTenantsIsAllowed(t *testing.T) {
	ctx := context.Background()
	scheme := testScheme(t)
	first := &fabricv1alpha1.AgentIdentity{
		ObjectMeta: metav1.ObjectMeta{Name: "hairem-usr000001-agt00001"},
		Spec: fabricv1alpha1.AgentIdentitySpec{
			TenantRef: fabricv1alpha1.ObjectReference{Name: "hairem"},
			AgentKey:  "usr000001-agt00001",
		},
	}
	second := &fabricv1alpha1.AgentIdentity{
		ObjectMeta: metav1.ObjectMeta{Name: "indiba-usr000001-agt00001"},
		Spec: fabricv1alpha1.AgentIdentitySpec{
			TenantRef: fabricv1alpha1.ObjectReference{Name: "indiba"},
			AgentKey:  "usr000001-agt00001",
		},
	}

	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(first, second).Build()
	r := &AgentIdentityReconciler{Client: c, Scheme: scheme}

	conflict, err := r.findAgentKeyConflict(ctx, second)
	if err != nil {
		t.Fatal(err)
	}
	if conflict != "" {
		t.Fatalf("tenant-local agentKey collided across tenants with %q", conflict)
	}
}

func TestDeletingDuplicateAgentDoesNotBlockReplacement(t *testing.T) {
	ctx := context.Background()
	scheme := testScheme(t)
	now := metav1.Now()
	deleting := &fabricv1alpha1.AgentIdentity{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "hairem-old-agent",
			DeletionTimestamp: &now,
			Finalizers:        []string{AgentFinalizer},
		},
		Spec: fabricv1alpha1.AgentIdentitySpec{
			TenantRef: fabricv1alpha1.ObjectReference{Name: "hairem"},
			AgentKey:  "usr000001-agt00012",
		},
	}
	replacement := &fabricv1alpha1.AgentIdentity{
		ObjectMeta: metav1.ObjectMeta{Name: "hairem-new-agent"},
		Spec: fabricv1alpha1.AgentIdentitySpec{
			TenantRef: fabricv1alpha1.ObjectReference{Name: "hairem"},
			AgentKey:  "usr000001-agt00012",
		},
	}

	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(deleting, replacement).Build()
	r := &AgentIdentityReconciler{Client: c, Scheme: scheme}

	conflict, err := r.findAgentKeyConflict(ctx, replacement)
	if err != nil {
		t.Fatal(err)
	}
	if conflict != "" {
		t.Fatalf("deleting AgentIdentity unexpectedly blocked replacement with %q", conflict)
	}
}

func TestDuplicateAgentKeyInSameTenantIsRejected(t *testing.T) {
	ctx := context.Background()
	scheme := testScheme(t)
	tenant := testTenant()
	profile := testRuntimeProfile()
	first := &fabricv1alpha1.AgentIdentity{
		ObjectMeta: metav1.ObjectMeta{Name: "hairem-sandbox-tina"},
		Spec: fabricv1alpha1.AgentIdentitySpec{
			TenantRef: fabricv1alpha1.ObjectReference{Name: tenant.Name}, AgentKey: "tina", DisplayName: "Tina",
		},
	}
	duplicate := &fabricv1alpha1.AgentIdentity{
		ObjectMeta: metav1.ObjectMeta{Name: "hairem-sandbox-tina-copy"},
		Spec: fabricv1alpha1.AgentIdentitySpec{
			TenantRef: fabricv1alpha1.ObjectReference{Name: tenant.Name}, AgentKey: "tina", DisplayName: "Tina copy",
		},
	}

	c := fake.NewClientBuilder().WithScheme(scheme).
		WithStatusSubresource(&fabricv1alpha1.TenantBundle{}, &fabricv1alpha1.AgentIdentity{}).
		WithObjects(tenant, profile, first, duplicate).
		Build()
	reconcileTenant(t, ctx, c, scheme, tenant.Name)

	r := &AgentIdentityReconciler{Client: c, Scheme: scheme}
	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: duplicate.Name}}
	if _, err := r.Reconcile(ctx, req); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Reconcile(ctx, req); err != nil {
		t.Fatal(err)
	}

	var current fabricv1alpha1.AgentIdentity
	if err := c.Get(ctx, types.NamespacedName{Name: duplicate.Name}, &current); err != nil {
		t.Fatal(err)
	}
	if current.Status.Phase != "Failed" {
		t.Fatalf("duplicate phase = %q, want Failed", current.Status.Phase)
	}
	found := false
	for _, condition := range current.Status.Conditions {
		if condition.Type == "IdentityUnique" && condition.Status == metav1.ConditionFalse && condition.Reason == "DuplicateAgentKey" {
			found = true
		}
	}
	if !found {
		t.Fatalf("DuplicateAgentKey condition missing: %#v", current.Status.Conditions)
	}

	var deployment appsv1.Deployment
	err := c.Get(ctx, types.NamespacedName{Name: "hermes-tina", Namespace: "tenant-hairem-sandbox"}, &deployment)
	if !apierrors.IsNotFound(err) {
		t.Fatalf("duplicate identity must not create runtime, got err=%v deployment=%s", err, deployment.Name)
	}
}

func testScheme(t *testing.T) *runtime.Scheme {
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

func testTenant() *fabricv1alpha1.TenantBundle {
	return &fabricv1alpha1.TenantBundle{
		ObjectMeta: metav1.ObjectMeta{Name: "hairem-sandbox"},
		Spec: fabricv1alpha1.TenantBundleSpec{
			TenantID:    "TEN90001",
			DisplayName: "hAIrem Sandbox",
			Persistence: fabricv1alpha1.TenantPersistenceSpec{PostgreSQL: &fabricv1alpha1.PostgreSQLSpec{Mode: "Shared", ProfileRef: "shared-poc"}},
			Memory:      fabricv1alpha1.TenantMemorySpec{Hindsight: &fabricv1alpha1.HindsightMemorySpec{ProfileRef: "shared-poc"}},
		},
	}
}

func testRuntimeProfile() *fabricv1alpha1.AgentRuntimeProfile {
	return &fabricv1alpha1.AgentRuntimeProfile{
		ObjectMeta: metav1.ObjectMeta{Name: "hermes-default"},
		Spec: fabricv1alpha1.AgentRuntimeProfileSpec{
			Engine:  "Hermes",
			// Existing deployed runtime is the legacy derived Hermes image; opt-in
			// official-image bootstrap tests override this fixture explicitly.
			Image:   "ghcr.io/charchess/txo-fabric-hermes:main-ab881f643cbc",
			Storage: fabricv1alpha1.RuntimeStorageSpec{Size: resource.MustParse("2Gi"), StorageClassName: "truenas-iscsi-delete"},
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("100m"), corev1.ResourceMemory: resource.MustParse("256Mi")},
				Limits:   corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("1"), corev1.ResourceMemory: resource.MustParse("2Gi")},
			},
			Compatibility: fabricv1alpha1.RuntimeCompatibilitySpec{S6Overlay: true},
			Capabilities: fabricv1alpha1.RuntimeCapabilityPolicySpec{Toolsets: []fabricv1alpha1.RuntimeToolsetPolicy{
				{Name: "terminal", State: fabricv1alpha1.ToolsetPolicyOn},
				{Name: "skills", State: fabricv1alpha1.ToolsetPolicyOn},
				{Name: "cronjob", State: fabricv1alpha1.ToolsetPolicyOn},
				{Name: "delegation", State: fabricv1alpha1.ToolsetPolicyAllowedOff},
				{Name: "computer_use", State: fabricv1alpha1.ToolsetPolicyOff},
			}},
		},
	}
}

func reconcileTenant(t *testing.T, ctx context.Context, c client.Client, scheme *runtime.Scheme, name string) {
	t.Helper()
	r := &TenantBundleReconciler{Client: c, Scheme: scheme}
	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: name}}
	if _, err := r.Reconcile(ctx, req); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Reconcile(ctx, req); err != nil {
		t.Fatal(err)
	}
}

func envValue(env []corev1.EnvVar, name string) string {
	if item := envVar(env, name); item != nil {
		return item.Value
	}
	return ""
}

func envVar(env []corev1.EnvVar, name string) *corev1.EnvVar {
	for i := range env {
		if env[i].Name == name {
			return &env[i]
		}
	}
	return nil
}
