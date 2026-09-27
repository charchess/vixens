package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	fabricv1alpha1 "github.com/charchess/vixens/apps/60-services/txo-fabric/operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestPostgreSQLProfileMissingIsExplicitlyBlocked(t *testing.T) {
	ctx := context.Background()
	scheme := postgresqlTestScheme(t)
	tenant := postgresqlTestTenant()
	c := fake.NewClientBuilder().WithScheme(scheme).
		WithStatusSubresource(&fabricv1alpha1.TenantBundle{}).
		WithObjects(tenant).
		Build()

	reconcilePostgreSQLTenant(t, ctx, c, scheme, tenant.Name, 2)
	current := getTenant(t, ctx, c, tenant.Name)
	condition := conditionByType(current.Status.Conditions, "PersistenceReady")
	if condition == nil || condition.Status != metav1.ConditionFalse || condition.Reason != "ProfileNotFound" {
		t.Fatalf("unexpected PersistenceReady condition: %#v", condition)
	}
	if current.Status.Persistence.PostgreSQL == nil || current.Status.Persistence.PostgreSQL.Phase != "Blocked" {
		t.Fatalf("unexpected PostgreSQL status: %#v", current.Status.Persistence.PostgreSQL)
	}
}

func TestPostgreSQLClusterMissingIsExplicitlyBlocked(t *testing.T) {
	ctx := context.Background()
	scheme := postgresqlTestScheme(t)
	tenant := postgresqlTestTenant()
	profile := postgresqlTestProfile()
	c := fake.NewClientBuilder().WithScheme(scheme).
		WithStatusSubresource(&fabricv1alpha1.TenantBundle{}).
		WithObjects(tenant, profile).
		Build()

	reconcilePostgreSQLTenant(t, ctx, c, scheme, tenant.Name, 2)
	current := getTenant(t, ctx, c, tenant.Name)
	condition := conditionByType(current.Status.Conditions, "PersistenceReady")
	if condition == nil || condition.Status != metav1.ConditionFalse || condition.Reason != "PlatformDependencyMissing" {
		t.Fatalf("unexpected PersistenceReady condition: %#v", condition)
	}
}

func TestSharedPostgreSQLReconcilesSecretRoleDatabaseAndVector(t *testing.T) {
	ctx := context.Background()
	scheme := postgresqlTestScheme(t)
	tenant := postgresqlTestTenant()
	profile := postgresqlTestProfile()
	cluster := cnpgObject(cnpgClusterGVK, "databases", "postgresql-shared")
	c := fake.NewClientBuilder().WithScheme(scheme).
		WithStatusSubresource(&fabricv1alpha1.TenantBundle{}).
		WithObjects(tenant, profile, cluster).
		Build()

	// First pass adds the TenantBundle finalizer. Second pass creates the Secret
	// and DatabaseRole, but deliberately does not create the Database until CNPG
	// reports the role as applied.
	reconcilePostgreSQLTenant(t, ctx, c, scheme, tenant.Name, 2)

	var secret corev1.Secret
	if err := c.Get(ctx, types.NamespacedName{Namespace: "databases", Name: "txo-ten90001-postgresql"}, &secret); err != nil {
		t.Fatalf("credential Secret not reconciled: %v", err)
	}
	if secret.Type != corev1.SecretTypeBasicAuth || secret.Labels[cnpgReloadLabel] != "true" {
		t.Fatalf("unexpected credential Secret contract: type=%q labels=%#v", secret.Type, secret.Labels)
	}
	if string(secret.Data[corev1.BasicAuthUsernameKey]) != "txo_ten90001" || len(secret.Data[corev1.BasicAuthPasswordKey]) == 0 {
		t.Fatalf("credential Secret is missing expected username/password keys")
	}
	originalPassword := append([]byte(nil), secret.Data[corev1.BasicAuthPasswordKey]...)

	role := cnpgObject(cnpgDatabaseRoleGVK, "databases", "txo-ten90001-role")
	if err := c.Get(ctx, client.ObjectKeyFromObject(role), role); err != nil {
		t.Fatalf("DatabaseRole not reconciled: %v", err)
	}
	if got, _, _ := unstructured.NestedString(role.Object, "spec", "cluster", "name"); got != "postgresql-shared" {
		t.Fatalf("DatabaseRole cluster=%q", got)
	}
	if got, _, _ := unstructured.NestedString(role.Object, "spec", "name"); got != "txo_ten90001" {
		t.Fatalf("DatabaseRole PostgreSQL name=%q", got)
	}
	if login, _, _ := unstructured.NestedBool(role.Object, "spec", "login"); !login {
		t.Fatal("DatabaseRole must be login-enabled")
	}
	if reclaim, _, _ := unstructured.NestedString(role.Object, "spec", "databaseRoleReclaimPolicy"); reclaim != "retain" {
		t.Fatalf("DatabaseRole reclaim=%q, want retain", reclaim)
	}

	database := cnpgObject(cnpgDatabaseGVK, "databases", "txo-ten90001-database")
	if err := c.Get(ctx, client.ObjectKeyFromObject(database), database); !apierrors.IsNotFound(err) {
		t.Fatalf("Database must wait for DatabaseRole readiness, got err=%v", err)
	}

	setApplied(t, ctx, c, role)
	reconcilePostgreSQLTenant(t, ctx, c, scheme, tenant.Name, 1)
	if err := c.Get(ctx, client.ObjectKeyFromObject(database), database); err != nil {
		t.Fatalf("Database not reconciled after role became ready: %v", err)
	}
	if got, _, _ := unstructured.NestedString(database.Object, "spec", "name"); got != "txo_ten90001" {
		t.Fatalf("Database PostgreSQL name=%q", got)
	}
	if owner, _, _ := unstructured.NestedString(database.Object, "spec", "owner"); owner != "txo_ten90001" {
		t.Fatalf("Database owner=%q", owner)
	}
	if reclaim, _, _ := unstructured.NestedString(database.Object, "spec", "databaseReclaimPolicy"); reclaim != "retain" {
		t.Fatalf("Database reclaim=%q, want retain", reclaim)
	}
	extensions, _, _ := unstructured.NestedSlice(database.Object, "spec", "extensions")
	if len(extensions) != 1 {
		t.Fatalf("expected exactly one required extension, got %#v", extensions)
	}
	extension := extensions[0].(map[string]interface{})
	if extension["name"] != "vector" || extension["ensure"] != "present" {
		t.Fatalf("unexpected extension contract: %#v", extension)
	}

	setApplied(t, ctx, c, database)
	reconcilePostgreSQLTenant(t, ctx, c, scheme, tenant.Name, 1)
	current := getTenant(t, ctx, c, tenant.Name)
	condition := conditionByType(current.Status.Conditions, "PersistenceReady")
	if condition == nil || condition.Status != metav1.ConditionTrue || condition.Reason != "Reconciled" {
		t.Fatalf("unexpected PersistenceReady condition: %#v", condition)
	}
	if current.Status.Persistence.PostgreSQL == nil || current.Status.Persistence.PostgreSQL.Phase != "Ready" {
		t.Fatalf("unexpected PostgreSQL status: %#v", current.Status.Persistence.PostgreSQL)
	}

	statusJSON, err := json.Marshal(current.Status)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(statusJSON, originalPassword) {
		t.Fatal("TenantBundle status must never contain PostgreSQL password material")
	}

	// A fresh reconciler instance must reuse the same resources and password.
	reconcilePostgreSQLTenant(t, ctx, c, scheme, tenant.Name, 1)
	var after corev1.Secret
	if err := c.Get(ctx, types.NamespacedName{Namespace: "databases", Name: secret.Name}, &after); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(originalPassword, after.Data[corev1.BasicAuthPasswordKey]) {
		t.Fatal("idempotent reconciliation rotated the tenant password")
	}
}

func TestPostgreSQLRefusesForeignSecretAdoption(t *testing.T) {
	ctx := context.Background()
	scheme := postgresqlTestScheme(t)
	tenant := postgresqlTestTenant()
	profile := postgresqlTestProfile()
	cluster := cnpgObject(cnpgClusterGVK, "databases", "postgresql-shared")
	foreign := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "txo-ten90001-postgresql", Namespace: "databases"},
		Type:       corev1.SecretTypeBasicAuth,
		Data: map[string][]byte{
			corev1.BasicAuthUsernameKey: []byte("someone_else"),
			corev1.BasicAuthPasswordKey: []byte("do-not-touch"),
		},
	}
	c := fake.NewClientBuilder().WithScheme(scheme).
		WithStatusSubresource(&fabricv1alpha1.TenantBundle{}).
		WithObjects(tenant, profile, cluster, foreign).
		Build()

	reconcilePostgreSQLTenant(t, ctx, c, scheme, tenant.Name, 2)
	current := getTenant(t, ctx, c, tenant.Name)
	condition := conditionByType(current.Status.Conditions, "PersistenceReady")
	if condition == nil || condition.Reason != "OwnershipConflict" {
		t.Fatalf("expected OwnershipConflict, got %#v", condition)
	}
	var unchanged corev1.Secret
	if err := c.Get(ctx, client.ObjectKeyFromObject(foreign), &unchanged); err != nil {
		t.Fatal(err)
	}
	if string(unchanged.Data[corev1.BasicAuthPasswordKey]) != "do-not-touch" {
		t.Fatal("foreign Secret was modified")
	}
}

func TestPostgreSQLRetainCleanupDeletesCRsButKeepsCredentials(t *testing.T) {
	ctx := context.Background()
	scheme := postgresqlTestScheme(t)
	tenant := postgresqlTestTenant()
	profile := postgresqlTestProfile()
	names := resolvePostgreSQLNames(tenant, profile)
	labels := postgresqlLabels(tenant, profile, "database", "Retain")
	database := cnpgObject(cnpgDatabaseGVK, "databases", names.DatabaseResource)
	database.SetLabels(labels)
	database.Object["spec"] = map[string]interface{}{
		"cluster":               map[string]interface{}{"name": "postgresql-shared"},
		"name":                  names.Database,
		"owner":                 names.Role,
		"databaseReclaimPolicy": "retain",
	}
	role := cnpgObject(cnpgDatabaseRoleGVK, "databases", names.RoleResource)
	role.SetLabels(postgresqlLabels(tenant, profile, "role", "Retain"))
	role.Object["spec"] = map[string]interface{}{
		"cluster":                   map[string]interface{}{"name": "postgresql-shared"},
		"name":                      names.Role,
		"databaseRoleReclaimPolicy": "retain",
		"passwordSecret":            map[string]interface{}{"name": names.Secret},
	}
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: names.Secret, Namespace: "databases", Labels: postgresqlLabels(tenant, profile, "credentials", "Retain")},
		Type:       corev1.SecretTypeBasicAuth,
		Data: map[string][]byte{
			corev1.BasicAuthUsernameKey: []byte(names.Role),
			corev1.BasicAuthPasswordKey: []byte("retained-password"),
		},
	}
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(tenant, profile, database, role, secret).Build()
	r := &TenantBundleReconciler{Client: c, Scheme: scheme}

	pending, err := r.cleanupPostgreSQL(ctx, tenant)
	if err != nil {
		t.Fatal(err)
	}
	if !pending {
		t.Fatal("cleanup should report pending when managed CRs were present")
	}
	if err := c.Get(ctx, client.ObjectKeyFromObject(database), cnpgObject(cnpgDatabaseGVK, "databases", names.DatabaseResource)); !apierrors.IsNotFound(err) {
		t.Fatalf("Database CR was not deleted: %v", err)
	}
	if err := c.Get(ctx, client.ObjectKeyFromObject(role), cnpgObject(cnpgDatabaseRoleGVK, "databases", names.RoleResource)); !apierrors.IsNotFound(err) {
		t.Fatalf("DatabaseRole CR was not deleted: %v", err)
	}
	var retained corev1.Secret
	if err := c.Get(ctx, client.ObjectKeyFromObject(secret), &retained); err != nil {
		t.Fatalf("Retain cleanup must keep the credential Secret: %v", err)
	}
}

func postgresqlTestScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	scheme := testScheme(t)
	AddCNPGToScheme(scheme)
	return scheme
}

func postgresqlTestTenant() *fabricv1alpha1.TenantBundle {
	return &fabricv1alpha1.TenantBundle{
		ObjectMeta: metav1.ObjectMeta{Name: "hairem-sandbox"},
		Spec: fabricv1alpha1.TenantBundleSpec{
			TenantID:    "TEN90001",
			DisplayName: "hAIrem Sandbox",
			Persistence: fabricv1alpha1.TenantPersistenceSpec{PostgreSQL: &fabricv1alpha1.PostgreSQLSpec{Mode: "Shared", ProfileRef: "postgresql-shared"}},
		},
	}
}

func postgresqlTestProfile() *fabricv1alpha1.PostgreSQLProfile {
	return &fabricv1alpha1.PostgreSQLProfile{
		ObjectMeta: metav1.ObjectMeta{Name: "postgresql-shared"},
		Spec: fabricv1alpha1.PostgreSQLProfileSpec{
			Provider:           "CloudNativePG",
			Topology:           "SharedCluster",
			RequiredExtensions: []string{"vector"},
			Shared: fabricv1alpha1.SharedPostgreSQLProfileSpec{
				ClusterRef:         fabricv1alpha1.PostgreSQLClusterReference{Name: "postgresql-shared", Namespace: "databases"},
				DatabaseNamePrefix: "txo_",
				RoleNamePrefix:     "txo_",
			},
			DatabaseReclaimPolicy: "Retain",
			RoleReclaimPolicy:     "Retain",
		},
	}
}

func reconcilePostgreSQLTenant(t *testing.T, ctx context.Context, c client.Client, scheme *runtime.Scheme, name string, count int) {
	t.Helper()
	r := &TenantBundleReconciler{Client: c, Scheme: scheme}
	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: name}}
	for i := 0; i < count; i++ {
		if _, err := r.Reconcile(ctx, req); err != nil {
			t.Fatal(err)
		}
	}
}

func setApplied(t *testing.T, ctx context.Context, c client.Client, object *unstructured.Unstructured) {
	t.Helper()
	current := cnpgObject(object.GroupVersionKind(), object.GetNamespace(), object.GetName())
	if err := c.Get(ctx, client.ObjectKeyFromObject(object), current); err != nil {
		t.Fatal(err)
	}
	if err := unstructured.SetNestedField(current.Object, true, "status", "applied"); err != nil {
		t.Fatal(err)
	}
	if err := c.Update(ctx, current); err != nil {
		t.Fatal(err)
	}
}

func getTenant(t *testing.T, ctx context.Context, c client.Client, name string) *fabricv1alpha1.TenantBundle {
	t.Helper()
	var current fabricv1alpha1.TenantBundle
	if err := c.Get(ctx, types.NamespacedName{Name: name}, &current); err != nil {
		t.Fatal(err)
	}
	return &current
}

func conditionByType(conditions []metav1.Condition, conditionType string) *metav1.Condition {
	for i := range conditions {
		if conditions[i].Type == conditionType {
			return &conditions[i]
		}
	}
	return nil
}
