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
			AIGateway:   &fabricv1alpha1.TenantAIGatewaySpec{ProfileRef: defaultAIGatewayProfileName},
		},
	}
}

func TestReconcileAIGatewayRequiresDedicatedPostgreSQLProfile(t *testing.T) {
	ctx := context.Background()
	scheme := postgresqlTestScheme(t)
	tenant := aiGatewayTestTenant("fabric-smoke", "TEN90002")
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
	tenant := aiGatewayTestTenant("fabric-smoke", "TEN90002")
	gatewayProfile := aiGatewayTestProfile()
	postgresqlProfile := aiGatewayPostgreSQLTestProfile()
	cluster := cnpgObject(cnpgClusterGVK, "databases", "postgresql-shared")
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(tenant, gatewayProfile, postgresqlProfile, cluster).Build()
	r := &TenantBundleReconciler{Client: c, Scheme: scheme}

	result, err := r.reconcileAIGateway(ctx, tenant)
	if err != nil {
		t.Fatal(err)
	}
	if result.Reason != "PostgreSQLRolePending" || result.Status == nil || result.Status.Phase != "Provisioning" {
		t.Fatalf("first gateway reconcile = %#v", result)
	}

	names := resolveAIGatewayPostgreSQLNames(tenant, postgresqlProfile)
	if names.Database != "txo_ten90002_ai_gateway" || names.Role != "txo_ten90002_ai_gateway" {
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
	if err := c.Get(ctx, types.NamespacedName{Namespace: "databases", Name: "txo-ten90002-postgresql"}, &genericSecret); !apierrors.IsNotFound(err) {
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

	var gatewayPolicy networkingv1.NetworkPolicy
	if err := c.Get(ctx, types.NamespacedName{Namespace: tenantNamespace(tenant.Name), Name: tenantAIGatewayNetworkPolicy}, &gatewayPolicy); err != nil {
		t.Fatalf("tenant LiteLLM NetworkPolicy missing: %v", err)
	}
	if len(gatewayPolicy.Spec.Egress) != 2 {
		t.Fatalf("gateway without CPA must only have DNS + PostgreSQL egress, got %d rules", len(gatewayPolicy.Spec.Egress))
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

func TestAIGatewayPostgreSQLIsIsolatedAcrossTenants(t *testing.T) {
	ctx := context.Background()
	scheme := postgresqlTestScheme(t)
	first := aiGatewayTestTenant("fabric-smoke-a", "TEN90002")
	second := aiGatewayTestTenant("fabric-smoke-b", "TEN90003")
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

func TestCleanupAIGatewayRetainsDatabaseCredentialWhenProfileIsRetain(t *testing.T) {
	ctx := context.Background()
	scheme := postgresqlTestScheme(t)
	tenant := aiGatewayTestTenant("fabric-smoke", "TEN90002")
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
	tenant := aiGatewayTestTenant("fabric-smoke", "TEN90002")
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
