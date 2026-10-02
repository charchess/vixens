package controller

import (
	"context"
	"strings"
	"testing"

	fabricv1alpha1 "github.com/charchess/vixens/apps/60-services/txo-fabric/operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestHermesHindsightDeploymentWiring(t *testing.T) {
	ctx := context.Background()
	scheme := testScheme(t)
	tenant := testTenant()
	profile := testRuntimeProfile()
	agent := testAgentIdentity()
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(agent).Build()
	r := &AgentIdentityReconciler{Client: c, Scheme: scheme}
	namespace := tenantNamespace(tenant.Name)

	if err := r.ensureDeployment(ctx, agent, tenant, profile, namespace, ""); err != nil {
		t.Fatal(err)
	}
	if err := r.ensureDeployment(ctx, agent, tenant, profile, namespace, ""); err != nil {
		t.Fatal(err)
	}

	var deployment appsv1.Deployment
	if err := c.Get(ctx, types.NamespacedName{Name: runtimeName(agent.Spec.AgentKey), Namespace: namespace}, &deployment); err != nil {
		t.Fatal(err)
	}
	if len(deployment.Spec.Template.Spec.InitContainers) != 1 || len(deployment.Spec.Template.Spec.Containers) != 1 {
		t.Fatalf("unexpected Hermes pod shape: %#v", deployment.Spec.Template.Spec)
	}

	bootstrap := deployment.Spec.Template.Spec.InitContainers[0]
	if len(bootstrap.Command) < 3 {
		t.Fatalf("bootstrap command missing: %#v", bootstrap.Command)
	}
	command := bootstrap.Command[2]
	for _, want := range []string{
		`legacy_profile="/opt/data/profiles/${AGENT_NAME}"`,
		`TXO_RUNTIME_STORAGE_RETENTION`,
		`explicit migration is required`,
		"config set skills.external_dirs '[\"/workspace/skills\"]'",
		"config set memory.provider hindsight",
		"\"mode\": \"local_external\"",
		"\"api_url\": \"http://hindsight:8888\"",
		"\"memory_mode\": \"context\"",
		"\"recall_types\": \"\"",
		"\"auto_retain\": True",
		"\"auto_recall\": True",
		"\"retain_indicator\": False",
		"\"recall_indicator\": False",
		`hermes_home / "hindsight" / "config.json"`,
	} {
		if !strings.Contains(command, want) {
			t.Fatalf("bootstrap command missing %q:\n%s", want, command)
		}
	}
	for _, forbidden := range []string{"hermes profile create", `hermes -p "${AGENT_NAME}"`, `profile_home / "hindsight"`} {
		if strings.Contains(command, forbidden) {
			t.Fatalf("bootstrap still contains named-profile behavior %q:\n%s", forbidden, command)
		}
	}
	if got := strings.Count(command, "config set memory.provider hindsight"); got != 1 {
		t.Fatalf("provider bootstrap repeated %d times after idempotent reconciliation", got)
	}
	if strings.Contains(command, "HINDSIGHT_API_KEY") || strings.Contains(command, "HINDSIGHT_API_DATABASE_URL") || strings.Contains(command, "POSTGRES") {
		t.Fatalf("bootstrap config contains a credential/database surface:\n%s", command)
	}
	if got := envValue(bootstrap.Env, "HERMES_HOME"); got != "/opt/data" {
		t.Fatalf("bootstrap HERMES_HOME = %q", got)
	}
	if got := envValue(bootstrap.Env, "HINDSIGHT_BANK_ID"); got != "hairem-sandbox-tina" {
		t.Fatalf("bootstrap HINDSIGHT_BANK_ID = %q", got)
	}
	if got := envValue(bootstrap.Env, "TXO_RUNTIME_STORAGE_RETENTION"); got != StorageRetentionRetain {
		t.Fatalf("bootstrap retention = %q", got)
	}

	hermes := deployment.Spec.Template.Spec.Containers[0]
	if got := envValue(hermes.Env, "HERMES_HOME"); got != "/opt/data" {
		t.Fatalf("runtime HERMES_HOME = %q", got)
	}
	apiKey := envVar(hermes.Env, "HINDSIGHT_API_KEY")
	if apiKey == nil || apiKey.Value != "" || apiKey.ValueFrom == nil || apiKey.ValueFrom.SecretKeyRef == nil {
		t.Fatalf("HINDSIGHT_API_KEY is not Secret-backed: %#v", apiKey)
	}
	if apiKey.ValueFrom.SecretKeyRef.Name != hermesHindsightSecretName || apiKey.ValueFrom.SecretKeyRef.Key != hermesHindsightSecretKey {
		t.Fatalf("HINDSIGHT_API_KEY Secret reference = %#v", apiKey.ValueFrom.SecretKeyRef)
	}
	if got := countEnv(hermes.Env, "HINDSIGHT_API_KEY"); got != 1 {
		t.Fatalf("HINDSIGHT_API_KEY repeated %d times after idempotent reconciliation", got)
	}
	if got := envValue(hermes.Env, "TXO_MEMORY_BANK_ID"); got != "hairem-sandbox-tina" {
		t.Fatalf("TXO_MEMORY_BANK_ID = %q", got)
	}
	for _, item := range hermes.Env {
		name := strings.ToUpper(item.Name)
		if strings.Contains(name, "DATABASE") || strings.Contains(name, "POSTGRES") || strings.Contains(name, "HINDSIGHT_API_LLM_API_KEY") {
			t.Fatalf("Hermes received forbidden Hindsight/DB credential env %q", item.Name)
		}
	}
}

func TestHermesDisposableRuntimeCanDropLegacyNamedProfile(t *testing.T) {
	ctx := context.Background()
	scheme := testScheme(t)
	tenant := testTenant()
	profile := testRuntimeProfile()
	agent := testAgentIdentity()
	agent.Spec.Runtime.Storage.RetentionPolicy = StorageRetentionDelete
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(agent).Build()
	r := &AgentIdentityReconciler{Client: c, Scheme: scheme}
	namespace := tenantNamespace(tenant.Name)

	if err := r.ensureDeployment(ctx, agent, tenant, profile, namespace, ""); err != nil {
		t.Fatal(err)
	}
	var deployment appsv1.Deployment
	if err := c.Get(ctx, types.NamespacedName{Name: runtimeName(agent.Spec.AgentKey), Namespace: namespace}, &deployment); err != nil {
		t.Fatal(err)
	}
	bootstrap := deployment.Spec.Template.Spec.InitContainers[0]
	if got := envValue(bootstrap.Env, "TXO_RUNTIME_STORAGE_RETENTION"); got != StorageRetentionDelete {
		t.Fatalf("bootstrap retention = %q", got)
	}
	if !strings.Contains(bootstrap.Command[2], `rm -rf -- "${legacy_profile}"`) {
		t.Fatalf("disposable migration cleanup missing:\n%s", bootstrap.Command[2])
	}
}

func TestHermesHindsightUsesExplicitBankID(t *testing.T) {
	ctx := context.Background()
	scheme := testScheme(t)
	tenant := testTenant()
	profile := testRuntimeProfile()
	agent := testAgentIdentity()
	agent.Spec.Memory.BankID = "customer-success-bank"
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(agent).Build()
	r := &AgentIdentityReconciler{Client: c, Scheme: scheme}
	namespace := tenantNamespace(tenant.Name)

	if err := r.ensureDeployment(ctx, agent, tenant, profile, namespace, ""); err != nil {
		t.Fatal(err)
	}
	var deployment appsv1.Deployment
	if err := c.Get(ctx, types.NamespacedName{Name: runtimeName(agent.Spec.AgentKey), Namespace: namespace}, &deployment); err != nil {
		t.Fatal(err)
	}
	if got := envValue(deployment.Spec.Template.Spec.InitContainers[0].Env, "HINDSIGHT_BANK_ID"); got != "customer-success-bank" {
		t.Fatalf("HINDSIGHT_BANK_ID = %q", got)
	}
	if got := envValue(deployment.Spec.Template.Spec.Containers[0].Env, "TXO_MEMORY_BANK_ID"); got != "customer-success-bank" {
		t.Fatalf("TXO_MEMORY_BANK_ID = %q", got)
	}
}

func TestHermesTenantWithoutHindsightRemainsUnmodified(t *testing.T) {
	ctx := context.Background()
	scheme := testScheme(t)
	tenant := testTenant()
	tenant.Spec.Memory.Hindsight = nil
	profile := testRuntimeProfile()
	agent := testAgentIdentity()
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(agent).Build()
	r := &AgentIdentityReconciler{Client: c, Scheme: scheme}
	namespace := tenantNamespace(tenant.Name)

	if err := r.ensureDeployment(ctx, agent, tenant, profile, namespace, ""); err != nil {
		t.Fatal(err)
	}
	var deployment appsv1.Deployment
	if err := c.Get(ctx, types.NamespacedName{Name: runtimeName(agent.Spec.AgentKey), Namespace: namespace}, &deployment); err != nil {
		t.Fatal(err)
	}
	bootstrap := deployment.Spec.Template.Spec.InitContainers[0]
	if strings.Contains(bootstrap.Command[2], "memory.provider hindsight") || strings.Contains(bootstrap.Command[2], `hermes_home / "hindsight"`) {
		t.Fatalf("tenant without Hindsight received provider bootstrap:\n%s", bootstrap.Command[2])
	}
	if strings.Contains(bootstrap.Command[2], "hermes profile create") || strings.Contains(bootstrap.Command[2], `hermes -p "${AGENT_NAME}"`) {
		t.Fatalf("tenant without Hindsight still uses a named profile:\n%s", bootstrap.Command[2])
	}
	if envVar(bootstrap.Env, "HINDSIGHT_BANK_ID") != nil {
		t.Fatal("tenant without Hindsight received HINDSIGHT_BANK_ID")
	}
	if envVar(deployment.Spec.Template.Spec.Containers[0].Env, "HINDSIGHT_API_KEY") != nil {
		t.Fatal("tenant without Hindsight received HINDSIGHT_API_KEY")
	}
}

func testAgentIdentity() *fabricv1alpha1.AgentIdentity {
	return &fabricv1alpha1.AgentIdentity{
		ObjectMeta: metav1.ObjectMeta{Name: "hairem-sandbox-tina"},
		Spec: fabricv1alpha1.AgentIdentitySpec{
			TenantRef:   fabricv1alpha1.ObjectReference{Name: "hairem-sandbox"},
			AgentKey:    "tina",
			DisplayName: "Tina",
			Runtime:     fabricv1alpha1.AgentRuntimeBinding{ProfileRef: "hermes-default"},
		},
	}
}

func countEnv(env []corev1.EnvVar, name string) int {
	count := 0
	for i := range env {
		if env[i].Name == name {
			count++
		}
	}
	return count
}
