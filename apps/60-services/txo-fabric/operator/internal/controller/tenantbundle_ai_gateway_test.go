package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	fabricv1alpha1 "github.com/charchess/vixens/apps/60-services/txo-fabric/operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func aiGatewayTestProfile() *fabricv1alpha1.AIGatewayProfile {
	return &fabricv1alpha1.AIGatewayProfile{
		ObjectMeta: metav1.ObjectMeta{Name: defaultAIGatewayProfileName},
		Spec: fabricv1alpha1.AIGatewayProfileSpec{
			Topology:             "TenantScoped",
			Implementation:       "LiteLLM",
			Image:                "ghcr.io/berriai/litellm-non_root:v1.102.1",
			APIPort:              4000,
			PostgreSQLProfileRef: defaultAIGatewayPostgreSQLProfileName,
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("250m"),
					corev1.ResourceMemory: resource.MustParse("512Mi"),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("2"),
					corev1.ResourceMemory: resource.MustParse("8Gi"),
				},
			},
			PriorityClassName: "vixens-medium",
			SizingLabel:       "V-scout",
		},
	}
}

func aiGatewayPostgreSQLTestProfile() *fabricv1alpha1.PostgreSQLProfile {
	profile := postgresqlTestProfile()
	profile.Name = defaultAIGatewayPostgreSQLProfileName
	profile.Spec.RequiredExtensions = nil
	return profile
}

func aiGatewayTestTenant(name, tenantID string) *fabricv1alpha1.TenantBundle {
	return &fabricv1alpha1.TenantBundle{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: fabricv1alpha1.TenantBundleSpec{
			TenantID:    tenantID,
			DisplayName: name,
		},
	}
}

func aiGatewayTestBrokerBackend(tenant *fabricv1alpha1.TenantBundle) (*corev1.Service, *corev1.Secret) {
	namespace := tenantNamespace(tenant.Name)
	return &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{Name: tenantAICredentialBrokerName, Namespace: namespace},
			Spec: corev1.ServiceSpec{Ports: []corev1.ServicePort{{Port: 8317}}},
		}, &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: tenantAICredentialBrokerSecretName, Namespace: namespace},
			Data:       map[string][]byte{"bootstrap-api-key": []byte("test-cpa-bootstrap")},
		}
}

func aiGatewayTestProviderSecret(tenantName string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      tenantAIProviderSecretName(tenantName),
			Namespace: tenantAIProviderSecretNamespace,
		},
		Data: map[string][]byte{tenantAIOpenRouterSecretKey: []byte("test-openrouter-key-" + tenantName)},
	}
}

func TestReconcileAIGatewayRequiresDedicatedPostgreSQLProfile(t *testing.T) {
	ctx := context.Background()
	scheme := postgresqlTestScheme(t)
	tenant := aiGatewayTestTenant("hairem", "TEN00001")
	profile := aiGatewayTestProfile()
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(tenant, profile).Build()

	result, err := (&TenantBundleReconciler{Client: c, Scheme: scheme}).reconcileAIGateway(ctx, tenant)
	if err != nil {
		t.Fatal(err)
	}
	if result.Ready || result.Status == nil || result.Status.Phase != "Blocked" || result.Reason != "PostgreSQLProfileNotFound" {
		t.Fatalf("missing LiteLLM PostgreSQL profile must fail closed: %#v", result)
	}
	if result.RequeueAfter == 0 {
		t.Fatal("missing platform PostgreSQL dependency should be retried")
	}
}

func TestReconcileAIGatewayCreatesDedicatedCNPGStateAndMigrations(t *testing.T) {
	ctx := context.Background()
	scheme := postgresqlTestScheme(t)
	tenant := aiGatewayTestTenant("hairem", "TEN00001")
	gatewayProfile := aiGatewayTestProfile()
	postgresqlProfile := aiGatewayPostgreSQLTestProfile()
	cluster := cnpgObject(cnpgClusterGVK, "databases", "postgresql-shared")
	brokerService, brokerSecret := aiGatewayTestBrokerBackend(tenant)
	providerSecret := aiGatewayTestProviderSecret(tenant.Name)
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(tenant, gatewayProfile, postgresqlProfile, cluster, brokerService, brokerSecret, providerSecret).Build()
	r := &TenantBundleReconciler{Client: c, Scheme: scheme}

	result, err := r.reconcileAIGateway(ctx, tenant)
	if err != nil {
		t.Fatal(err)
	}
	if result.Reason != "PostgreSQLRolePending" || result.Status == nil || result.Status.Phase != "Provisioning" {
		t.Fatalf("first gateway reconcile = %#v", result)
	}

	names := resolveAIGatewayPostgreSQLNames(tenant, postgresqlProfile)
	if names.Database != "txo_ten00001_ai_gateway" || names.Role != "txo_ten00001_ai_gateway" {
		t.Fatalf("unexpected dedicated LiteLLM database identity: %#v", names)
	}

	var sourceSecret corev1.Secret
	if err := c.Get(ctx, types.NamespacedName{Namespace: "databases", Name: names.Secret}, &sourceSecret); err != nil {
		t.Fatalf("LiteLLM PostgreSQL credentials were not reconciled: %v", err)
	}
	if got := string(sourceSecret.Data[corev1.BasicAuthUsernameKey]); got != names.Role {
		t.Fatalf("database username=%q, want %q", got, names.Role)
	}
	if got := string(sourceSecret.Data["dbname"]); got != names.Database {
		t.Fatalf("database name=%q, want %q", got, names.Database)
	}
	originalDatabasePassword := append([]byte(nil), sourceSecret.Data[corev1.BasicAuthPasswordKey]...)

	// The gateway has its own database binding. It must never silently reuse the
	// generic tenant/Hindsight database identity.
	var genericSecret corev1.Secret
	if err := c.Get(ctx, types.NamespacedName{Namespace: "databases", Name: "txo-ten00001-postgresql"}, &genericSecret); !apierrors.IsNotFound(err) {
		t.Fatalf("gateway unexpectedly created/reused generic tenant PostgreSQL Secret: %v", err)
	}

	role := cnpgObject(cnpgDatabaseRoleGVK, "databases", names.RoleResource)
	if err := c.Get(ctx, client.ObjectKeyFromObject(role), role); err != nil {
		t.Fatal(err)
	}
	if got, _, _ := unstructured.NestedString(role.Object, "spec", "name"); got != names.Role {
		t.Fatalf("DatabaseRole name=%q", got)
	}
	if reclaim, _, _ := unstructured.NestedString(role.Object, "spec", "databaseRoleReclaimPolicy"); reclaim != "retain" {
		t.Fatalf("DatabaseRole reclaim=%q, want retain", reclaim)
	}

	setApplied(t, ctx, c, role)
	result, err = r.reconcileAIGateway(ctx, tenant)
	if err != nil {
		t.Fatal(err)
	}
	if result.Reason != "PostgreSQLDatabasePending" {
		t.Fatalf("gateway did not wait for Database readiness: %#v", result)
	}

	database := cnpgObject(cnpgDatabaseGVK, "databases", names.DatabaseResource)
	if err := c.Get(ctx, client.ObjectKeyFromObject(database), database); err != nil {
		t.Fatal(err)
	}
	if got, _, _ := unstructured.NestedString(database.Object, "spec", "owner"); got != names.Role {
		t.Fatalf("Database owner=%q, want %q", got, names.Role)
	}
	extensions, _, _ := unstructured.NestedSlice(database.Object, "spec", "extensions")
	if len(extensions) != 0 {
		t.Fatalf("LiteLLM profile must not inherit Hindsight/vector extensions: %#v", extensions)
	}

	setApplied(t, ctx, c, database)
	result, err = r.reconcileAIGateway(ctx, tenant)
	if err != nil {
		t.Fatal(err)
	}
	if result.Reason != "MigrationPending" || result.Status == nil || result.Status.Phase != "Provisioning" {
		t.Fatalf("gateway must wait for schema migrations: %#v", result)
	}

	var runtimeSecret corev1.Secret
	if err := c.Get(ctx, types.NamespacedName{Namespace: tenantNamespace(tenant.Name), Name: tenantAIGatewayRuntimeSecretName}, &runtimeSecret); err != nil {
		t.Fatalf("tenant-local LiteLLM runtime Secret missing: %v", err)
	}
	if len(runtimeSecret.Data["LITELLM_MASTER_KEY"]) == 0 {
		t.Fatal("LiteLLM master key was not generated")
	}
	if !bytes.Equal(runtimeSecret.Data["DB_PASSWORD"], originalDatabasePassword) {
		t.Fatal("tenant-local runtime Secret does not mirror the dedicated database credential")
	}
	if got := string(runtimeSecret.Data[tenantAIOpenRouterSecretKey]); got != "test-openrouter-key-"+tenant.Name {
		t.Fatalf("tenant LiteLLM OpenRouter credential=%q, want tenant-scoped provider value", got)
	}
	originalMasterKey := append([]byte(nil), runtimeSecret.Data["LITELLM_MASTER_KEY"]...)

	migrationName := aiGatewayMigrationName(gatewayProfile, &sourceSecret)
	var migration batchv1.Job
	if err := c.Get(ctx, types.NamespacedName{Namespace: tenantNamespace(tenant.Name), Name: migrationName}, &migration); err != nil {
		t.Fatalf("LiteLLM migration Job missing: %v", err)
	}
	container := migration.Spec.Template.Spec.Containers[0]
	if container.Image != gatewayProfile.Spec.Image || !strings.Contains(container.Args[0], "litellm/proxy/prisma_migration.py") {
		t.Fatalf("unexpected migration container: %#v", container)
	}
	if migration.Spec.Template.Spec.AutomountServiceAccountToken == nil || *migration.Spec.Template.Spec.AutomountServiceAccountToken {
		t.Fatal("migration Job must not mount a Kubernetes service-account token")
	}
	if container.SecurityContext == nil || container.SecurityContext.AllowPrivilegeEscalation == nil || *container.SecurityContext.AllowPrivilegeEscalation {
		t.Fatal("migration container must disable privilege escalation")
	}
	if got := migration.Spec.Template.Labels["vixens.io/sizing.prisma-migrations"]; got != gatewayProfile.Spec.SizingLabel {
		t.Fatalf("migration sizing label=%q want %q", got, gatewayProfile.Spec.SizingLabel)
	}
	if got := container.Resources.Limits.Memory().String(); got != "8Gi" {
		t.Fatalf("migration memory limit=%q want 8Gi from AIGatewayProfile", got)
	}

	var policy networkingv1.NetworkPolicy
	if err := c.Get(ctx, types.NamespacedName{Namespace: tenantNamespace(tenant.Name), Name: tenantAIGatewayMigrationPolicyName}, &policy); err != nil {
		t.Fatalf("migration NetworkPolicy missing: %v", err)
	}
	if len(policy.Spec.Egress) != 2 {
		t.Fatalf("migration egress must be DNS + PostgreSQL only, got %d rules", len(policy.Spec.Egress))
	}

	migration.Status.Succeeded = 1
	if err := c.Status().Update(ctx, &migration); err != nil {
		t.Fatal(err)
	}
	result, err = r.reconcileAIGateway(ctx, tenant)
	if err != nil {
		t.Fatal(err)
	}
	if result.Reason != "DeploymentProgressing" || result.Status == nil || result.Status.Phase != "Provisioning" {
		t.Fatalf("gateway should wait for LiteLLM Deployment availability: %#v", result)
	}

	var gatewayDeployment appsv1.Deployment
	if err := c.Get(ctx, types.NamespacedName{Namespace: tenantNamespace(tenant.Name), Name: tenantAIGatewayName}, &gatewayDeployment); err != nil {
		t.Fatalf("tenant LiteLLM Deployment missing: %v", err)
	}
	gatewayContainer := gatewayDeployment.Spec.Template.Spec.Containers[0]
	if gatewayContainer.Image != gatewayProfile.Spec.Image {
		t.Fatalf("tenant LiteLLM image=%q want %q", gatewayContainer.Image, gatewayProfile.Spec.Image)
	}
	if gatewayDeployment.Spec.Template.Spec.AutomountServiceAccountToken == nil || *gatewayDeployment.Spec.Template.Spec.AutomountServiceAccountToken {
		t.Fatal("tenant LiteLLM must not mount a Kubernetes service-account token")
	}
	if gatewayContainer.SecurityContext == nil || gatewayContainer.SecurityContext.AllowPrivilegeEscalation == nil || *gatewayContainer.SecurityContext.AllowPrivilegeEscalation {
		t.Fatal("tenant LiteLLM must disable privilege escalation")
	}

	var gatewayConfig corev1.ConfigMap
	if err := c.Get(ctx, types.NamespacedName{Namespace: tenantNamespace(tenant.Name), Name: tenantAIGatewayConfigMapName}, &gatewayConfig); err != nil {
		t.Fatalf("tenant LiteLLM config missing: %v", err)
	}
	if got := gatewayConfig.Data["config.yaml"]; !strings.Contains(got, "master_key: os.environ/LITELLM_MASTER_KEY") {
		t.Fatalf("tenant LiteLLM config missing master-key contract: %s", got)
	}
	if got := gatewayConfig.Data["config.yaml"]; !strings.Contains(got, "model_name: txo-agent") ||
		!strings.Contains(got, "api_base: http://txo-ai-credential-broker:8317/v1") {
		t.Fatalf("active tenant LiteLLM config is not wired to tenant CPA: %s", got)
	}
	if got := gatewayConfig.Data["config.yaml"]; !strings.Contains(got, "model_name: txo-embedding") ||
		!strings.Contains(got, "model: "+tenantAIEmbeddingProviderModel) ||
		!strings.Contains(got, "api_key: os.environ/OPENROUTER_API_KEY") {
		t.Fatalf("tenant LiteLLM config is missing tenant OpenRouter embedding route: %s", got)
	} else if strings.Contains(got, "test-openrouter-key-"+tenant.Name) {
		t.Fatal("tenant LiteLLM config leaked the OpenRouter credential value")
	}

	var gatewayPolicy networkingv1.NetworkPolicy
	if err := c.Get(ctx, types.NamespacedName{Namespace: tenantNamespace(tenant.Name), Name: tenantAIGatewayNetworkPolicy}, &gatewayPolicy); err != nil {
		t.Fatalf("tenant LiteLLM NetworkPolicy missing: %v", err)
	}
	if len(gatewayPolicy.Spec.Egress) != 4 {
		t.Fatalf("active tenant LiteLLM with OpenRouter must have DNS + PostgreSQL + tenant CPA + HTTPS egress, got %d rules", len(gatewayPolicy.Spec.Egress))
	}
	httpsEgress := false
	for _, rule := range gatewayPolicy.Spec.Egress {
		for _, port := range rule.Ports {
			if port.Port != nil && port.Port.IntValue() == 443 {
				httpsEgress = true
			}
		}
	}
	if !httpsEgress {
		t.Fatal("tenant LiteLLM OpenRouter route is missing TCP/443 egress")
	}

	gatewayDeployment.Status.ObservedGeneration = gatewayDeployment.Generation
	gatewayDeployment.Status.AvailableReplicas = 1
	if err := c.Status().Update(ctx, &gatewayDeployment); err != nil {
		t.Fatal(err)
	}
	result, err = r.reconcileAIGateway(ctx, tenant)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Ready || result.Reason != "DeploymentAvailable" || result.Status == nil || result.Status.Phase != "Ready" {
		t.Fatalf("available LiteLLM gateway did not become Ready: %#v", result)
	}

	// A second reconcile must preserve both credential layers.
	if _, err := r.reconcileAIGateway(ctx, tenant); err != nil {
		t.Fatal(err)
	}
	var sourceAfter corev1.Secret
	if err := c.Get(ctx, types.NamespacedName{Namespace: "databases", Name: names.Secret}, &sourceAfter); err != nil {
		t.Fatal(err)
	}
	var runtimeAfter corev1.Secret
	if err := c.Get(ctx, types.NamespacedName{Namespace: tenantNamespace(tenant.Name), Name: tenantAIGatewayRuntimeSecretName}, &runtimeAfter); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(sourceAfter.Data[corev1.BasicAuthPasswordKey], originalDatabasePassword) {
		t.Fatal("idempotent reconcile rotated the LiteLLM database password")
	}
	if !bytes.Equal(runtimeAfter.Data["LITELLM_MASTER_KEY"], originalMasterKey) {
		t.Fatal("idempotent reconcile rotated the LiteLLM master key")
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(resultJSON, originalDatabasePassword) || bytes.Contains(resultJSON, originalMasterKey) {
		t.Fatal("AI gateway status/result leaked database or LiteLLM secret material")
	}
}

func TestTenantLiteLLMVPAMinCPUTracksProfileRequest(t *testing.T) {
	ctx := context.Background()
	scheme := postgresqlTestScheme(t)
	tenant := aiGatewayTestTenant("hairem", "TEN00001")
	profile := aiGatewayTestProfile()
	profile.Spec.Resources.Requests[corev1.ResourceCPU] = resource.MustParse("350m")
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(tenant).Build()
	r := &TenantBundleReconciler{Client: c, Scheme: scheme}

	if _, err := r.ensureTenantAIGatewayDeployment(ctx, tenant, profile, aiGatewayBackendState{}, "test-config-hash"); err != nil {
		t.Fatal(err)
	}

	var deployment appsv1.Deployment
	if err := c.Get(ctx, types.NamespacedName{Namespace: tenantNamespace(tenant.Name), Name: tenantAIGatewayName}, &deployment); err != nil {
		t.Fatal(err)
	}
	if got := deployment.Spec.Template.Annotations["vixens.io/vpa.min-cpu"]; got != "350m" {
		t.Fatalf("tenant LiteLLM VPA CPU floor=%q want profile request 350m", got)
	}
}

func TestAIGatewayMigrationNameChangesWithSizingContract(t *testing.T) {
	profile := aiGatewayTestProfile()
	secret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{UID: types.UID("db-secret-uid")}}

	base := aiGatewayMigrationName(profile, secret)

	profileWithDifferentSizing := profile.DeepCopy()
	profileWithDifferentSizing.Spec.SizingLabel = "V-xlarge"
	if got := aiGatewayMigrationName(profileWithDifferentSizing, secret); got == base {
		t.Fatal("migration Job name must change when sizing label changes")
	}

	profileWithDifferentResources := profile.DeepCopy()
	profileWithDifferentResources.Spec.Resources.Limits[corev1.ResourceMemory] = resource.MustParse("16Gi")
	if got := aiGatewayMigrationName(profileWithDifferentResources, secret); got == base {
		t.Fatal("migration Job name must change when resource contract changes")
	}
}

func TestAIGatewayPostgreSQLIsIsolatedAcrossTenants(t *testing.T) {
	ctx := context.Background()
	scheme := postgresqlTestScheme(t)
	first := aiGatewayTestTenant("hairem", "TEN00001")
	second := aiGatewayTestTenant("indiba", "TEN00002")
	gatewayProfile := aiGatewayTestProfile()
	postgresqlProfile := aiGatewayPostgreSQLTestProfile()
	cluster := cnpgObject(cnpgClusterGVK, "databases", "postgresql-shared")
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(first, second, gatewayProfile, postgresqlProfile, cluster).Build()
	r := &TenantBundleReconciler{Client: c, Scheme: scheme}

	for _, tenant := range []*fabricv1alpha1.TenantBundle{first, second} {
		if _, err := r.reconcileAIGateway(ctx, tenant); err != nil {
			t.Fatal(err)
		}
	}

	firstNames := resolveAIGatewayPostgreSQLNames(first, postgresqlProfile)
	secondNames := resolveAIGatewayPostgreSQLNames(second, postgresqlProfile)
	if firstNames.Database == secondNames.Database || firstNames.Role == secondNames.Role || firstNames.Secret == secondNames.Secret {
		t.Fatalf("tenant LiteLLM database identities collided: first=%#v second=%#v", firstNames, secondNames)
	}
	for _, names := range []aiGatewayPostgreSQLNames{firstNames, secondNames} {
		var secret corev1.Secret
		if err := c.Get(ctx, types.NamespacedName{Namespace: "databases", Name: names.Secret}, &secret); err != nil {
			t.Fatalf("isolated tenant database Secret %s missing: %v", names.Secret, err)
		}
	}
}

func TestParkAIGatewayComputePreservesDurableState(t *testing.T) {
	ctx := context.Background()
	scheme := postgresqlTestScheme(t)
	tenant := aiGatewayTestTenant("hairem", "TEN00001")
	namespace := tenantNamespace(tenant.Name)
	labels := aiGatewayWorkloadLabels(tenant)
	deployment := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: tenantAIGatewayName, Namespace: namespace, Labels: labels}}
	runtimeSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: tenantAIGatewayRuntimeSecretName, Namespace: namespace, Labels: aiGatewayRuntimeLabels(tenant)},
		Data:       map[string][]byte{"LITELLM_MASTER_KEY": []byte("preserved")},
	}
	config := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: tenantAIGatewayConfigMapName, Namespace: namespace, Labels: labels}}
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(tenant, deployment, runtimeSecret, config).Build()
	r := &TenantBundleReconciler{Client: c, Scheme: scheme}

	pending, err := r.parkAIGatewayCompute(ctx, tenant)
	if err != nil {
		t.Fatal(err)
	}
	if !pending {
		t.Fatal("parking existing LiteLLM compute must report a pending transition")
	}
	var removed appsv1.Deployment
	if err := c.Get(ctx, types.NamespacedName{Namespace: namespace, Name: tenantAIGatewayName}, &removed); err == nil {
		t.Fatal("LiteLLM Deployment still exists after parking")
	}
	var preservedSecret corev1.Secret
	if err := c.Get(ctx, types.NamespacedName{Namespace: namespace, Name: tenantAIGatewayRuntimeSecretName}, &preservedSecret); err != nil {
		t.Fatalf("parking deleted LiteLLM runtime/database Secret: %v", err)
	}
	var preservedConfig corev1.ConfigMap
	if err := c.Get(ctx, types.NamespacedName{Namespace: namespace, Name: tenantAIGatewayConfigMapName}, &preservedConfig); err != nil {
		t.Fatalf("parking deleted LiteLLM config: %v", err)
	}

	pending, err = r.parkAIGatewayCompute(ctx, tenant)
	if err != nil {
		t.Fatal(err)
	}
	if pending {
		t.Fatal("parking must become idempotent once LiteLLM compute is absent")
	}
}

func TestCleanupAIGatewayRetainsDatabaseCredentialWhenProfileIsRetain(t *testing.T) {
	ctx := context.Background()
	scheme := postgresqlTestScheme(t)
	tenant := aiGatewayTestTenant("hairem", "TEN00001")
	gatewayProfile := aiGatewayTestProfile()
	postgresqlProfile := aiGatewayPostgreSQLTestProfile()
	names := resolveAIGatewayPostgreSQLNames(tenant, postgresqlProfile)

	database := cnpgObject(cnpgDatabaseGVK, "databases", names.DatabaseResource)
	database.SetLabels(aiGatewayPostgreSQLLabels(tenant, postgresqlProfile, "database", "Retain"))
	database.Object["spec"] = map[string]interface{}{
		"cluster":               map[string]interface{}{"name": "postgresql-shared"},
		"name":                  names.Database,
		"owner":                 names.Role,
		"databaseReclaimPolicy": "retain",
	}
	role := cnpgObject(cnpgDatabaseRoleGVK, "databases", names.RoleResource)
	role.SetLabels(aiGatewayPostgreSQLLabels(tenant, postgresqlProfile, "role", "Retain"))
	role.Object["spec"] = map[string]interface{}{
		"cluster":                   map[string]interface{}{"name": "postgresql-shared"},
		"name":                      names.Role,
		"databaseRoleReclaimPolicy": "retain",
		"passwordSecret":            map[string]interface{}{"name": names.Secret},
	}
	sourceSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: names.Secret, Namespace: "databases", Labels: aiGatewayPostgreSQLLabels(tenant, postgresqlProfile, "credentials", "Retain")},
		Type:       corev1.SecretTypeBasicAuth,
		Data: map[string][]byte{
			corev1.BasicAuthUsernameKey: []byte(names.Role),
			corev1.BasicAuthPasswordKey: []byte("retained-password"),
		},
	}
	runtimeSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: tenantAIGatewayRuntimeSecretName, Namespace: tenantNamespace(tenant.Name), Labels: aiGatewayRuntimeLabels(tenant)},
	}
	policy := &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{Name: tenantAIGatewayMigrationPolicyName, Namespace: tenantNamespace(tenant.Name), Labels: aiGatewayRuntimeLabels(tenant)},
	}
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(tenant, gatewayProfile, postgresqlProfile, database, role, sourceSecret, runtimeSecret, policy).Build()
	r := &TenantBundleReconciler{Client: c, Scheme: scheme}

	pending, err := r.cleanupAIGateway(ctx, tenant)
	if err != nil {
		t.Fatal(err)
	}
	if !pending {
		t.Fatal("cleanup should report pending when managed gateway state existed")
	}

	if err := c.Get(ctx, client.ObjectKeyFromObject(database), cnpgObject(cnpgDatabaseGVK, "databases", names.DatabaseResource)); !apierrors.IsNotFound(err) {
		t.Fatalf("gateway Database CR was not deleted: %v", err)
	}
	if err := c.Get(ctx, client.ObjectKeyFromObject(role), cnpgObject(cnpgDatabaseRoleGVK, "databases", names.RoleResource)); !apierrors.IsNotFound(err) {
		t.Fatalf("gateway DatabaseRole CR was not deleted: %v", err)
	}
	var retained corev1.Secret
	if err := c.Get(ctx, client.ObjectKeyFromObject(sourceSecret), &retained); err != nil {
		t.Fatalf("Retain cleanup must preserve database credential Secret: %v", err)
	}
	var removedRuntime corev1.Secret
	if err := c.Get(ctx, client.ObjectKeyFromObject(runtimeSecret), &removedRuntime); !apierrors.IsNotFound(err) {
		t.Fatalf("tenant-local runtime Secret should be deleted: %v", err)
	}
}

func TestReconcileAIGatewayRejectsCredentialBrokerImplementation(t *testing.T) {
	ctx := context.Background()
	scheme := postgresqlTestScheme(t)
	tenant := aiGatewayTestTenant("hairem", "TEN00001")
	profile := aiGatewayTestProfile()
	profile.Spec.Implementation = "CLIProxyAPI"
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(tenant, profile).Build()

	result, err := (&TenantBundleReconciler{Client: c, Scheme: scheme}).reconcileAIGateway(ctx, tenant)
	if err != nil {
		t.Fatal(err)
	}
	if result.Ready || result.Status == nil || result.Status.Phase != "Blocked" || result.Reason != "UnsupportedImplementation" {
		t.Fatalf("CPA must not be accepted as tenant-facing AI gateway: %#v", result)
	}
}


func TestRenderTenantLiteLLMConfigRoutesAgentThroughCPAWithoutProviderTokens(t *testing.T) {
	config := renderTenantLiteLLMConfig(aiGatewayBackendState{CPAEnabled: true, CPAPort: 8317})
	for _, want := range []string{
		"model_name: txo-agent",
		"model: openai/gpt-5.6-sol",
		"api_base: http://txo-ai-credential-broker:8317/v1",
		"api_key: os.environ/CPA_API_KEY",
		"master_key: os.environ/LITELLM_MASTER_KEY",
	} {
		if !strings.Contains(config, want) {
			t.Fatalf("tenant LiteLLM config missing %q:\n%s", want, config)
		}
	}
	for _, forbidden := range []string{"access_token", "refresh_token", "management-password", "sk-or-v1-"} {
		if strings.Contains(config, forbidden) {
			t.Fatalf("tenant LiteLLM config leaked provider/broker secret marker %q", forbidden)
		}
	}
}

func TestTenantLiteLLMNetworkPolicyHasNoDirectInternetEgress(t *testing.T) {
	ctx := context.Background()
	scheme := postgresqlTestScheme(t)
	tenant := aiGatewayTestTenant("hairem", "TEN00001")
	profile := aiGatewayTestProfile()
	postgresqlProfile := aiGatewayPostgreSQLTestProfile()
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(tenant).Build()
	r := &TenantBundleReconciler{Client: c, Scheme: scheme}

	if err := r.ensureTenantAIGatewayNetworkPolicy(ctx, tenant, profile, postgresqlProfile, aiGatewayBackendState{CPAEnabled: true, CPAPort: 8317}); err != nil {
		t.Fatal(err)
	}
	var policy networkingv1.NetworkPolicy
	if err := c.Get(ctx, types.NamespacedName{Namespace: tenantNamespace(tenant.Name), Name: tenantAIGatewayNetworkPolicy}, &policy); err != nil {
		t.Fatal(err)
	}
	if len(policy.Spec.Egress) != 3 {
		t.Fatalf("tenant LiteLLM egress rules=%d want DNS + PostgreSQL + CPA", len(policy.Spec.Egress))
	}
	for _, rule := range policy.Spec.Egress {
		for _, peer := range rule.To {
			if peer.IPBlock != nil {
				t.Fatalf("tenant LiteLLM must not receive direct Internet IPBlock egress in this slice: %#v", peer.IPBlock)
			}
		}
	}
}

func TestTenantLiteLLMNetworkPolicyAllowsSameTenantHermesIngress(t *testing.T) {
	ctx := context.Background()
	scheme := postgresqlTestScheme(t)
	tenant := aiGatewayTestTenant("hairem", "TEN00001")
	profile := aiGatewayTestProfile()
	postgresqlProfile := aiGatewayPostgreSQLTestProfile()
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(tenant).Build()
	r := &TenantBundleReconciler{Client: c, Scheme: scheme}

	if err := r.ensureTenantAIGatewayNetworkPolicy(ctx, tenant, profile, postgresqlProfile, aiGatewayBackendState{CPAEnabled: true, CPAPort: 8317}); err != nil {
		t.Fatal(err)
	}

	var policy networkingv1.NetworkPolicy
	if err := c.Get(ctx, types.NamespacedName{Namespace: tenantNamespace(tenant.Name), Name: tenantAIGatewayNetworkPolicy}, &policy); err != nil {
		t.Fatal(err)
	}
	if len(policy.Spec.Ingress) != 1 {
		t.Fatalf("tenant LiteLLM ingress rules=%d want 1", len(policy.Spec.Ingress))
	}

	managedWorkload := false
	sameTenantHermes := false
	fabricOperator := false
	for _, peer := range policy.Spec.Ingress[0].From {
		if peer.PodSelector == nil {
			continue
		}
		labels := peer.PodSelector.MatchLabels
		if peer.NamespaceSelector == nil {
			if labels[LabelManaged] == "true" {
				managedWorkload = true
			}
			if labels[LabelPartOf] == "txo-fabric" &&
				labels[LabelName] == "hermes-agent" &&
				labels[LabelTenantName] == tenant.Name {
				sameTenantHermes = true
			}
			continue
		}
		if peer.NamespaceSelector.MatchLabels["kubernetes.io/metadata.name"] == "txo-fabric-system" &&
			labels[LabelName] == "txo-fabric-operator" {
			fabricOperator = true
		}
	}
	if !managedWorkload {
		t.Fatal("tenant LiteLLM must preserve ingress from Fabric-managed workloads")
	}
	if !sameTenantHermes {
		t.Fatal("tenant LiteLLM ingress is missing the canonical same-tenant Hermes selector")
	}
	if !fabricOperator {
		t.Fatal("tenant LiteLLM ingress is missing the Fabric operator management selector")
	}

	if len(policy.Spec.Ingress[0].Ports) != 1 ||
		policy.Spec.Ingress[0].Ports[0].Port == nil ||
		policy.Spec.Ingress[0].Ports[0].Port.IntValue() != int(profile.Spec.APIPort) {
		t.Fatalf("tenant LiteLLM ingress must remain restricted to gateway port %d: %#v", profile.Spec.APIPort, policy.Spec.Ingress[0].Ports)
	}
}



func TestTenantLiteLLMProviderCredentialIsTenantIsolated(t *testing.T) {
	ctx := context.Background()
	scheme := postgresqlTestScheme(t)
	hairem := aiGatewayTestTenant("hairem", "TEN00001")
	indiba := aiGatewayTestTenant("indiba", "TEN00002")
	hairemProvider := aiGatewayTestProviderSecret(hairem.Name)

	hairemDB := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "hairem-db", Namespace: "databases"}, Data: map[string][]byte{
		corev1.BasicAuthUsernameKey: []byte("hairem"), corev1.BasicAuthPasswordKey: []byte("hairem-pass"),
		"host": []byte("postgresql-rw.databases.svc"), "port": []byte("5432"), "dbname": []byte("hairem"),
	}}
	indibaDB := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "indiba-db", Namespace: "databases"}, Data: map[string][]byte{
		corev1.BasicAuthUsernameKey: []byte("indiba"), corev1.BasicAuthPasswordKey: []byte("indiba-pass"),
		"host": []byte("postgresql-rw.databases.svc"), "port": []byte("5432"), "dbname": []byte("indiba"),
	}}

	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(hairem, indiba, hairemProvider).Build()
	r := &TenantBundleReconciler{Client: c, Scheme: scheme}
	if _, err := r.ensureAIGatewayRuntimeSecret(ctx, hairem, hairemDB); err != nil {
		t.Fatal(err)
	}
	if _, err := r.ensureAIGatewayRuntimeSecret(ctx, indiba, indibaDB); err != nil {
		t.Fatal(err)
	}

	var hairemRuntime, indibaRuntime corev1.Secret
	if err := c.Get(ctx, types.NamespacedName{Namespace: tenantNamespace(hairem.Name), Name: tenantAIGatewayRuntimeSecretName}, &hairemRuntime); err != nil {
		t.Fatal(err)
	}
	if err := c.Get(ctx, types.NamespacedName{Namespace: tenantNamespace(indiba.Name), Name: tenantAIGatewayRuntimeSecretName}, &indibaRuntime); err != nil {
		t.Fatal(err)
	}
	if got := string(hairemRuntime.Data[tenantAIOpenRouterSecretKey]); got != "test-openrouter-key-hairem" {
		t.Fatalf("hAIrem provider credential=%q", got)
	}
	if _, exists := indibaRuntime.Data[tenantAIOpenRouterSecretKey]; exists {
		t.Fatal("Indiba runtime received another tenant's OpenRouter credential")
	}
}

func TestRenderTenantLiteLLMConfigAddsEmbeddingOnlyWithProviderCredential(t *testing.T) {
	withoutProvider := renderTenantLiteLLMConfig(aiGatewayBackendState{CPAEnabled: true, CPAPort: 8317})
	if strings.Contains(withoutProvider, "model_name: txo-embedding") {
		t.Fatalf("embedding route must stay absent without a tenant provider credential:\n%s", withoutProvider)
	}

	withProvider := renderTenantLiteLLMConfig(aiGatewayBackendState{CPAEnabled: true, CPAPort: 8317, OpenRouterEnabled: true})
	for _, want := range []string{
		"model_name: txo-agent",
		"model_name: txo-embedding",
		"model: openrouter/baai/bge-m3",
		"api_key: os.environ/OPENROUTER_API_KEY",
	} {
		if !strings.Contains(withProvider, want) {
			t.Fatalf("tenant LiteLLM config missing %q:\n%s", want, withProvider)
		}
	}
}

func TestAIProviderSecretWatchTargetsOnlyMatchingTenant(t *testing.T) {
	ctx := context.Background()
	requests := tenantBundleRequestsForAIProviderSecret(ctx, aiGatewayTestProviderSecret("hairem"))
	if len(requests) != 1 || requests[0].Name != "hairem" {
		t.Fatalf("provider Secret watch requests=%#v, want hairem", requests)
	}
	foreign := aiGatewayTestProviderSecret("indiba")
	foreign.Namespace = "tenant-indiba"
	if got := tenantBundleRequestsForAIProviderSecret(ctx, foreign); len(got) != 0 {
		t.Fatalf("provider Secret outside %s unexpectedly enqueued tenant: %#v", tenantAIProviderSecretNamespace, got)
	}
}


func TestTenantLiteLLMRollsWhenOpenRouterCredentialRevisionChanges(t *testing.T) {
	ctx := context.Background()
	scheme := postgresqlTestScheme(t)
	tenant := aiGatewayTestTenant("hairem", "TEN00001")
	profile := aiGatewayTestProfile()
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(tenant).Build()
	r := &TenantBundleReconciler{Client: c, Scheme: scheme}

	first := aiGatewayBackendState{
		OpenRouterEnabled:        true,
		OpenRouterSecretRevision: "100",
	}
	if _, err := r.ensureTenantAIGatewayDeployment(ctx, tenant, profile, first, "stable-config"); err != nil {
		t.Fatal(err)
	}
	var deployment appsv1.Deployment
	if err := c.Get(ctx, types.NamespacedName{Namespace: tenantNamespace(tenant.Name), Name: tenantAIGatewayName}, &deployment); err != nil {
		t.Fatal(err)
	}
	if got := deployment.Spec.Template.Annotations["fabric.truxonline.io/openrouter-secret-revision"]; got != "100" {
		t.Fatalf("initial provider revision=%q", got)
	}

	second := first
	second.OpenRouterSecretRevision = "101"
	if _, err := r.ensureTenantAIGatewayDeployment(ctx, tenant, profile, second, "stable-config"); err != nil {
		t.Fatal(err)
	}
	if err := c.Get(ctx, types.NamespacedName{Namespace: tenantNamespace(tenant.Name), Name: tenantAIGatewayName}, &deployment); err != nil {
		t.Fatal(err)
	}
	if got := deployment.Spec.Template.Annotations["fabric.truxonline.io/openrouter-secret-revision"]; got != "101" {
		t.Fatalf("rotated provider revision=%q want 101", got)
	}
}
