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

	if err := r.ensureDeployment(ctx, agent, tenant, profile, namespace, "", ""); err != nil {
		t.Fatal(err)
	}
	if err := r.ensureDeployment(ctx, agent, tenant, profile, namespace, "", ""); err != nil {
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

func TestHermesRetainedRuntimeCanAdoptLegacyNamedProfile(t *testing.T) {
	ctx := context.Background()
	scheme := testScheme(t)
	tenant := testTenant()
	profile := testRuntimeProfile()
	agent := testAgentIdentity()
	agent.Spec.Runtime.Storage.RetentionPolicy = StorageRetentionRetain
	agent.Spec.Runtime.Storage.AdoptLegacyProfile = true
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(agent).Build()
	r := &AgentIdentityReconciler{Client: c, Scheme: scheme}
	namespace := tenantNamespace(tenant.Name)

	if err := r.ensureDeployment(ctx, agent, tenant, profile, namespace, "", ""); err != nil {
		t.Fatal(err)
	}
	var deployment appsv1.Deployment
	if err := c.Get(ctx, types.NamespacedName{Name: runtimeName(agent.Spec.AgentKey), Namespace: namespace}, &deployment); err != nil {
		t.Fatal(err)
	}

	bootstrap := deployment.Spec.Template.Spec.InitContainers[0]
	if got := envValue(bootstrap.Env, "HERMES_HOME"); got != "/mnt/txo-data/profiles/tina" {
		t.Fatalf("bootstrap HERMES_HOME = %q", got)
	}
	if got := envValue(bootstrap.Env, "TXO_LEGACY_PROFILE_ADOPTION"); got != "true" {
		t.Fatalf("legacy adoption env = %q", got)
	}
	if !strings.Contains(bootstrap.Command[2], "requested legacy Hermes profile") {
		t.Fatalf("bootstrap does not fail closed for a missing adopted profile:\n%s", bootstrap.Command[2])
	}
	var bootstrapDataMount corev1.VolumeMount
	for _, mount := range bootstrap.VolumeMounts {
		if mount.Name == "data" {
			bootstrapDataMount = mount
			break
		}
	}
	if bootstrapDataMount.MountPath != "/mnt/txo-data" || bootstrapDataMount.SubPath != "" {
		t.Fatalf("bootstrap data mount = %#v", bootstrapDataMount)
	}

	hermes := deployment.Spec.Template.Spec.Containers[0]
	if got := envValue(hermes.Env, "HERMES_HOME"); got != "/opt/data" {
		t.Fatalf("runtime HERMES_HOME = %q", got)
	}
	var runtimeDataMount corev1.VolumeMount
	for _, mount := range hermes.VolumeMounts {
		if mount.Name == "data" {
			runtimeDataMount = mount
			break
		}
	}
	if runtimeDataMount.MountPath != "/opt/data" || runtimeDataMount.SubPath != "profiles/tina" {
		t.Fatalf("runtime data mount = %#v", runtimeDataMount)
	}
}

func TestHermesLegacyProfileAdoptionRejectsDisposableStorage(t *testing.T) {
	ctx := context.Background()
	scheme := testScheme(t)
	tenant := testTenant()
	profile := testRuntimeProfile()
	agent := testAgentIdentity()
	agent.Spec.Runtime.Storage.RetentionPolicy = StorageRetentionDelete
	agent.Spec.Runtime.Storage.AdoptLegacyProfile = true
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(agent).Build()
	r := &AgentIdentityReconciler{Client: c, Scheme: scheme}

	err := r.ensureDeployment(ctx, agent, tenant, profile, tenantNamespace(tenant.Name), "", "")
	if err == nil || !strings.Contains(err.Error(), "requires retained runtime storage") {
		t.Fatalf("expected retained-storage validation error, got %v", err)
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

	if err := r.ensureDeployment(ctx, agent, tenant, profile, namespace, "", ""); err != nil {
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

	if err := r.ensureDeployment(ctx, agent, tenant, profile, namespace, "", ""); err != nil {
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

	if err := r.ensureDeployment(ctx, agent, tenant, profile, namespace, "", ""); err != nil {
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

func TestOfficialHermesHindsightPluginBootstrapIsIsolatedAndIdempotent(t *testing.T) {
	ctx := context.Background()
	scheme := testScheme(t)
	tenant := testTenant()
	agent := testAgentIdentity()
	profile := testRuntimeProfile()
	profile.Spec.Image = "nousresearch/hermes-agent:v2026.9.24"
	profile.Spec.Bootstrap.HindsightPluginImage = "ghcr.io/charchess/txo-hermes-hindsight-plugin@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(agent).Build()
	r := &AgentIdentityReconciler{Client: c, Scheme: scheme}
	ns := tenantNamespace(tenant.Name)
	for i := 0; i < 2; i++ {
		if err := r.ensureDeployment(ctx, agent, tenant, profile, ns, "", ""); err != nil { t.Fatal(err) }
	}
	var dep appsv1.Deployment
	if err := c.Get(ctx, types.NamespacedName{Name: runtimeName(agent.Spec.AgentKey), Namespace: ns}, &dep); err != nil { t.Fatal(err) }
	spec := dep.Spec.Template.Spec
	if len(spec.InitContainers) != 2 { t.Fatalf("expected plugin + profile init containers, got %d", len(spec.InitContainers)) }
	prepare, bootstrap := spec.InitContainers[0], spec.InitContainers[1]
	if prepare.Name != "prepare-hindsight-extension" || prepare.Image != profile.Spec.Bootstrap.HindsightPluginImage {
		t.Fatalf("plugin payload init order/image mismatch: %#v", prepare)
	}
	if bootstrap.Name != "bootstrap-profile" || bootstrap.Image != profile.Spec.Image {
		t.Fatalf("profile bootstrap must use official Hermes image: %#v", bootstrap)
	}
	if len(prepare.Args) == 0 || !strings.Contains(prepare.Args[0], "/bundle/hindsight/plugin.yaml") ||
		!strings.Contains(prepare.Args[0], "cp -a /bundle/python/.") {
		t.Fatalf("plugin bundle is not staged offline: %#v", prepare.Args)
	}
	if !strings.Contains(bootstrap.Command[2], "import hindsight_client, aiohttp_retry") {
		t.Fatal("bootstrap did not verify isolated Python imports")
	}
	if strings.Contains(bootstrap.Command[2], "pip install") || strings.Contains(bootstrap.Command[2], "SOUL.md") {
		t.Fatalf("bootstrap must not pip-install or rewrite SOUL.md: %s", bootstrap.Command[2])
	}
	if len(spec.Containers) != 1 || spec.Containers[0].Image != profile.Spec.Image {
		t.Fatalf("main Hermes runtime must stay exactly upstream: %#v", spec.Containers)
	}
	main := spec.Containers[0]
	if envValue(main.Env, "PYTHONPATH") != hindsightExtensionDepsRoot {
		t.Fatalf("runtime Python dependencies are not isolated: %#v", main.Env)
	}
	if envValue(main.Env, "HERMES_HOME") != "/opt/data" {
		t.Fatalf("agent private state home changed: %#v", main.Env)
	}
	for _, entry := range []struct{ path string; sub string }{
		{hindsightExtensionPluginRoot, "hindsight"},
		{hindsightExtensionDepsRoot, "python"},
	} {
		found := false
		for _, mount := range main.VolumeMounts {
			if mount.MountPath == entry.path {
				found = true
				if !mount.ReadOnly || mount.SubPath != entry.sub || mount.Name != hindsightExtensionVolume {
					t.Fatalf("Hindsight runtime mount is not readonly/scoped: %#v", mount)
				}
			}
		}
		if !found { t.Fatalf("missing Hindsight runtime mount %s", entry.path) }
	}
	found := false
	for _, volume := range spec.Volumes {
		if volume.Name == hindsightExtensionVolume {
			found = volume.EmptyDir != nil
		}
	}
	if !found { t.Fatal("Hindsight plugin bundle must use ephemeral emptyDir, not private PVC") }
	if countEnv(main.Env, "PYTHONPATH") != 1 || countEnv(main.Env, "HINDSIGHT_API_KEY") != 1 {
		t.Fatal("reconcile duplicated plugin/Python env vars")
	}
}

func TestHermesPluginBootstrapIsOptionalAndNeverGrantsUnconfiguredHindsight(t *testing.T) {
	ctx := context.Background()
	scheme := testScheme(t)
	tenant := testTenant()
	tenant.Spec.Memory.Hindsight = nil
	profile := testRuntimeProfile()
	profile.Spec.Image = "nousresearch/hermes-agent:v2026.9.24"
	profile.Spec.Bootstrap.HindsightPluginImage = "ghcr.io/charchess/txo-hermes-hindsight-plugin@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	agent := testAgentIdentity()
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(agent).Build()
	r := &AgentIdentityReconciler{Client: c, Scheme: scheme}
	ns := tenantNamespace(tenant.Name)
	if err := r.ensureDeployment(ctx, agent, tenant, profile, ns, "", ""); err != nil { t.Fatal(err) }
	var deployment appsv1.Deployment
	if err := c.Get(ctx, types.NamespacedName{Name: runtimeName(agent.Spec.AgentKey), Namespace: ns}, &deployment); err != nil { t.Fatal(err) }
	if len(deployment.Spec.Template.Spec.InitContainers) != 1 {
		t.Fatal("plugin bundle must not be mounted for tenants without Hindsight")
	}
	if envVar(deployment.Spec.Template.Spec.Containers[0].Env, "PYTHONPATH") != nil ||
		envVar(deployment.Spec.Template.Spec.Containers[0].Env, "HINDSIGHT_API_KEY") != nil {
		t.Fatal("tenant without memory got Hindsight extension or credentials")
	}
}

func TestOfficialHermesPluginBootstrapKeepsLegacyProfileAdoption(t *testing.T) {
	ctx := context.Background()
	scheme := testScheme(t)
	tenant := testTenant()
	profile := testRuntimeProfile()
	profile.Spec.Image = "nousresearch/hermes-agent:v2026.9.24"
	profile.Spec.Bootstrap.HindsightPluginImage = "ghcr.io/charchess/txo-hermes-hindsight-plugin@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	agent := testAgentIdentity()
	agent.Spec.Runtime.Storage.AdoptLegacyProfile = true
	agent.Spec.Runtime.Storage.RetentionPolicy = StorageRetentionRetain
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(agent).Build()
	r := &AgentIdentityReconciler{Client: c, Scheme: scheme}
	ns := tenantNamespace(tenant.Name)
	if err := r.ensureDeployment(ctx, agent, tenant, profile, ns, "", ""); err != nil { t.Fatal(err) }
	var deployment appsv1.Deployment
	if err := c.Get(ctx, types.NamespacedName{Name: runtimeName(agent.Spec.AgentKey), Namespace: ns}, &deployment); err != nil { t.Fatal(err) }
	bootstrap := deployment.Spec.Template.Spec.InitContainers[1]
	if envValue(bootstrap.Env, "HERMES_HOME") != "/mnt/txo-data/profiles/tina" {
		t.Fatalf("retained legacy HERMES_HOME changed: %#v", bootstrap.Env)
	}
	if !strings.Contains(bootstrap.Command[2], "requested legacy Hermes profile") {
		t.Fatal("legacy retained profile guard disappeared")
	}
	if deployment.Spec.Template.Spec.Containers[0].VolumeMounts[0].SubPath != "profiles/tina" {
		t.Fatal("legacy retained PVC subPath changed")
	}
}

func TestOfficialHermesWithoutPluginBundleIsRejectedForHindsightTenant(t *testing.T) {
	ctx := context.Background()
	tenant := testTenant()
	profile := testRuntimeProfile()
	profile.Spec.Image = "nousresearch/hermes-agent:v2026.9.24"
	profile.Spec.Bootstrap.HindsightPluginImage = ""
	agent := testAgentIdentity()
	scheme := testScheme(t)
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(agent).Build()
	r := &AgentIdentityReconciler{Client: c, Scheme: scheme}
	err := r.ensureDeployment(ctx, agent, tenant, profile, tenantNamespace(tenant.Name), "", "")
	if err == nil || !strings.Contains(err.Error(), "requires a pinned bootstrap.hindsightPluginImage") {
		t.Fatalf("unprovisioned official Hermes must fail closed: %v", err)
	}
	tenant.Spec.Memory.Hindsight = nil
	if err := r.ensureDeployment(ctx, agent, tenant, profile, tenantNamespace(tenant.Name), "", ""); err != nil {
		t.Fatalf("no-Hindsight tenant should not require a plugin bundle: %v", err)
	}
}

func TestHindsightBundleRejectsMutableTagsAndInvalidDigests(t *testing.T) {
	ctx := context.Background()
	tenant := testTenant()
	agent := testAgentIdentity()
	for _, image := range []string{
		"ghcr.io/charchess/txo-hermes-hindsight-plugin:main",
		"ghcr.io/charchess/txo-hermes-hindsight-plugin@sha256:abc",
		"ghcr.io/charchess/txo-hermes-hindsight-plugin@sha256:" + strings.Repeat("g", 64),
	} {
		profile := testRuntimeProfile()
		profile.Spec.Image = "nousresearch/hermes-agent:v2026.9.24"
		profile.Spec.Bootstrap.HindsightPluginImage = image
		scheme := testScheme(t)
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(agent).Build()
		r := &AgentIdentityReconciler{Client: c, Scheme: scheme}
		err := r.ensureDeployment(ctx, agent, tenant, profile, tenantNamespace(tenant.Name), "", "")
		if err == nil || !strings.Contains(err.Error(), "must be pinned to immutable OCI sha256 digest") {
			t.Fatalf("invalid plugin image %q was admitted: %v", image, err)
		}
	}
}
