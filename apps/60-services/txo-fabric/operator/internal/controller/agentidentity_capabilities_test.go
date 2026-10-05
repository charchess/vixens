package controller

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestResolveToolsetPolicyStatesAndManagedConfig(t *testing.T) {
	profile := testRuntimeProfile()
	agent := testAgentIdentity()

	policy, err := resolveToolsetPolicy(agent, profile)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := policy.Enabled, []string{"cronjob", "skills", "terminal"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("enabled toolsets = %#v, want %#v", got, want)
	}
	if got, want := policy.Denied, []string{"computer_use", "delegation"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("denied toolsets = %#v, want %#v", got, want)
	}
	baselineRevision := policy.Revision

	var managed map[string]any
	if err := json.Unmarshal([]byte(policy.Config), &managed); err != nil {
		t.Fatal(err)
	}
	security, ok := managed["security"].(map[string]any)
	if !ok || security["allow_lazy_installs"] != false {
		t.Fatalf("managed security policy = %#v", managed["security"])
	}
	model, ok := managed["model"].(map[string]any)
	if !ok {
		t.Fatalf("managed model route = %#v", managed["model"])
	}
	if got, want := model["default"], defaultAIGatewayModel; got != want {
		t.Fatalf("managed model.default = %#v, want %q", got, want)
	}
	if got, want := model["provider"], "custom"; got != want {
		t.Fatalf("managed model.provider = %#v, want %q", got, want)
	}
	if got, want := model["base_url"], defaultAIGatewayURL+"/v1"; got != want {
		t.Fatalf("managed model.base_url = %#v, want %q", got, want)
	}

	plugins, ok := managed["plugins"].(map[string]any)
	if !ok {
		t.Fatalf("managed plugins policy = %#v", managed["plugins"])
	}
	enabledPlugins, ok := plugins["enabled"].([]any)
	if !ok || len(enabledPlugins) != 0 {
		t.Fatalf("plugins.enabled = %#v, want explicit empty allow-list", plugins["enabled"])
	}
	platforms, ok := managed["platform_toolsets"].(map[string]any)
	if !ok {
		t.Fatalf("platform_toolsets = %#v", managed["platform_toolsets"])
	}
	for _, surface := range []string{"cli", "tui", "desktop", "acp", "api_server", "cron"} {
		if _, ok := platforms[surface].([]any); !ok {
			t.Fatalf("platform_toolsets.%s = %#v", surface, platforms[surface])
		}
	}
	knownPlugins, ok := managed["known_plugin_toolsets"].(map[string]any)
	if !ok {
		t.Fatalf("known_plugin_toolsets = %#v", managed["known_plugin_toolsets"])
	}
	if knownCLI, ok := knownPlugins["cli"].([]any); !ok || !jsonArrayContains(knownCLI, "delegation") {
		t.Fatalf("known_plugin_toolsets.cli = %#v", knownPlugins["cli"])
	}
	cli, ok := platforms["cli"].([]any)
	if !ok {
		t.Fatalf("platform_toolsets.cli = %#v", platforms["cli"])
	}
	for _, want := range []string{"cronjob", "skills", "terminal", "no_mcp"} {
		if !jsonArrayContains(cli, want) {
			t.Fatalf("platform_toolsets.cli missing %q: %#v", want, cli)
		}
	}
	for _, denied := range []string{"computer_use", "delegation"} {
		if jsonArrayContains(cli, denied) {
			t.Fatalf("platform_toolsets.cli unexpectedly contains denied %q: %#v", denied, cli)
		}
	}

	agent.Spec.Runtime.Capabilities.EnableToolsets = []string{"delegation"}
	enabledPolicy, err := resolveToolsetPolicy(agent, profile)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := enabledPolicy.Enabled, []string{"cronjob", "delegation", "skills", "terminal"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("enabled toolsets after opt-in = %#v, want %#v", got, want)
	}
	if got, want := enabledPolicy.Denied, []string{"computer_use"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("denied toolsets after opt-in = %#v, want %#v", got, want)
	}
	if enabledPolicy.Revision == baselineRevision {
		t.Fatal("AllowedOff opt-in did not change policy revision")
	}
}

func TestResolveToolsetPolicyRejectsEscalation(t *testing.T) {
	profile := testRuntimeProfile()
	agent := testAgentIdentity()

	agent.Spec.Runtime.Capabilities.EnableToolsets = []string{"computer_use"}
	if _, err := resolveToolsetPolicy(agent, profile); err == nil || !strings.Contains(err.Error(), "AllowedOff") {
		t.Fatalf("Off toolset request should fail closed, got %v", err)
	}

	agent.Spec.Runtime.Capabilities.EnableToolsets = []string{"unknown-toolset"}
	if _, err := resolveToolsetPolicy(agent, profile); err == nil || !strings.Contains(err.Error(), "undeclared") {
		t.Fatalf("undeclared toolset request should fail closed, got %v", err)
	}

	profile.Spec.Capabilities.Toolsets = nil
	agent.Spec.Runtime.Capabilities.EnableToolsets = nil
	if _, err := resolveToolsetPolicy(agent, profile); err == nil || !strings.Contains(err.Error(), "does not declare capabilities.toolsets") {
		t.Fatalf("policy-less Hermes profile should fail closed, got %v", err)
	}
}

func TestManagedToolsetPolicyIsMountedOutsideAgentState(t *testing.T) {
	ctx := context.Background()
	scheme := testScheme(t)
	tenant := testTenant()
	profile := testRuntimeProfile()
	agent := testAgentIdentity()
	policy, err := resolveToolsetPolicy(agent, profile)
	if err != nil {
		t.Fatal(err)
	}

	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(agent).Build()
	r := &AgentIdentityReconciler{Client: c, Scheme: scheme}

	namespace := tenantNamespace(tenant.Name)
	if err := r.ensureManagedToolsetPolicy(ctx, agent, tenant, namespace, policy); err != nil {
		t.Fatal(err)
	}

	var managed corev1.ConfigMap
	if err := c.Get(ctx, types.NamespacedName{Name: managedToolsetPolicyName(agent.Spec.AgentKey), Namespace: namespace}, &managed); err != nil {
		t.Fatal(err)
	}
	if managed.Annotations[AnnotationToolsetPolicyRevision] != policy.Revision {
		t.Fatalf("managed ConfigMap revision = %q, want %q", managed.Annotations[AnnotationToolsetPolicyRevision], policy.Revision)
	}
	if managed.Data[managedPolicyConfigKey] != policy.Config {
		t.Fatal("managed ConfigMap does not contain rendered policy")
	}

	if err := r.ensureDeployment(ctx, agent, tenant, profile, namespace, "", "", policy); err != nil {
		t.Fatal(err)
	}
	var deployment appsv1.Deployment
	if err := c.Get(ctx, types.NamespacedName{Name: runtimeName(agent.Spec.AgentKey), Namespace: namespace}, &deployment); err != nil {
		t.Fatal(err)
	}
	container := deployment.Spec.Template.Spec.Containers[0]
	if got := envValue(container.Env, "HERMES_MANAGED_DIR"); got != managedPolicyMountPath {
		t.Fatalf("HERMES_MANAGED_DIR = %q", got)
	}
	if got := envValue(container.Env, "HERMES_DISABLE_LAZY_INSTALLS"); got != "1" {
		t.Fatalf("HERMES_DISABLE_LAZY_INSTALLS = %q", got)
	}
	if got := envValue(container.Env, "TXO_TOOLSET_POLICY_REVISION"); got != policy.Revision {
		t.Fatalf("TXO_TOOLSET_POLICY_REVISION = %q, want %q", got, policy.Revision)
	}
	if got := envValue(container.Env, "HERMES_TUI_TOOLSETS"); got != "cronjob,skills,terminal" {
		t.Fatalf("HERMES_TUI_TOOLSETS = %q", got)
	}
	if deployment.Spec.Template.Annotations[AnnotationToolsetPolicyRevision] != policy.Revision {
		t.Fatalf("pod policy revision = %q", deployment.Spec.Template.Annotations[AnnotationToolsetPolicyRevision])
	}
	if !hasReadOnlyMount(container.VolumeMounts, managedPolicyVolumeName, managedPolicyMountPath) {
		t.Fatalf("managed policy mount missing/read-write: %#v", container.VolumeMounts)
	}
}

func jsonArrayContains(values []any, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func hasReadOnlyMount(mounts []corev1.VolumeMount, name, path string) bool {
	for _, mount := range mounts {
		if mount.Name == name && mount.MountPath == path && mount.ReadOnly {
			return true
		}
	}
	return false
}
