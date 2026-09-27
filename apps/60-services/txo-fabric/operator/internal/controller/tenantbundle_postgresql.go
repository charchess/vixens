package controller

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"time"

	fabricv1alpha1 "github.com/charchess/vixens/apps/60-services/txo-fabric/operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	labelPostgreSQLProfile  = "fabric.truxonline.io/postgresql-profile"
	labelPostgreSQLResource = "fabric.truxonline.io/postgresql-resource"
	labelReclaimPolicy      = "fabric.truxonline.io/reclaim-policy"
	cnpgReloadLabel         = "cnpg.io/reload"
)

var (
	cnpgClusterGVK      = schema.GroupVersionKind{Group: "postgresql.cnpg.io", Version: "v1", Kind: "Cluster"}
	cnpgDatabaseGVK     = schema.GroupVersionKind{Group: "postgresql.cnpg.io", Version: "v1", Kind: "Database"}
	cnpgDatabaseRoleGVK = schema.GroupVersionKind{Group: "postgresql.cnpg.io", Version: "v1", Kind: "DatabaseRole"}
)

type postgresqlNames struct {
	DatabaseResource string
	RoleResource     string
	Secret           string
	Database         string
	Role             string
}

type postgresqlResult struct {
	Ready        bool
	Status       *fabricv1alpha1.ComponentStatus
	Reason       string
	Message      string
	RequeueAfter time.Duration
}

// AddCNPGToScheme registers the CloudNativePG resources consumed by TXO Fabric
// without importing CloudNativePG's Go API package. The Kubernetes API contract
// remains postgresql.cnpg.io/v1 and is reconciled through unstructured objects.
func AddCNPGToScheme(scheme *runtime.Scheme) {
	for _, gvk := range []schema.GroupVersionKind{cnpgClusterGVK, cnpgDatabaseGVK, cnpgDatabaseRoleGVK} {
		scheme.AddKnownTypeWithName(gvk, &unstructured.Unstructured{})
		scheme.AddKnownTypeWithName(schema.GroupVersionKind{Group: gvk.Group, Version: gvk.Version, Kind: gvk.Kind + "List"}, &unstructured.UnstructuredList{})
	}
}

func (r *TenantBundleReconciler) reconcilePostgreSQL(ctx context.Context, bundle *fabricv1alpha1.TenantBundle) (postgresqlResult, error) {
	request := bundle.Spec.Persistence.PostgreSQL
	if request == nil {
		return postgresqlResult{Ready: true}, nil
	}

	profileRef := strings.TrimSpace(request.ProfileRef)
	if profileRef == "" {
		return postgresqlBlocked("ProfileRefRequired", "PostgreSQL profileRef is required", 0), nil
	}
	if request.Mode != "" && request.Mode != "Shared" {
		return postgresqlBlocked("UnsupportedMode", fmt.Sprintf("PostgreSQL mode %q is not implemented; only Shared is supported", request.Mode), 0), nil
	}

	var profile fabricv1alpha1.PostgreSQLProfile
	if err := r.Get(ctx, client.ObjectKey{Name: profileRef}, &profile); err != nil {
		if apierrors.IsNotFound(err) {
			return postgresqlBlocked("ProfileNotFound", fmt.Sprintf("PostgreSQLProfile %q does not exist", profileRef), 30*time.Second), nil
		}
		return postgresqlResult{}, err
	}
	if profile.Spec.Provider != "" && profile.Spec.Provider != "CloudNativePG" {
		return postgresqlBlocked("UnsupportedProvider", fmt.Sprintf("PostgreSQL provider %q is not supported", profile.Spec.Provider), 0), nil
	}
	if profile.Spec.Topology != "SharedCluster" {
		return postgresqlBlocked("UnsupportedTopology", fmt.Sprintf("PostgreSQL topology %q is not implemented; only SharedCluster is supported", profile.Spec.Topology), 0), nil
	}
	if strings.EqualFold(profile.Spec.DatabaseReclaimPolicy, "Retain") && strings.EqualFold(profile.Spec.RoleReclaimPolicy, "Delete") {
		return postgresqlBlocked("InvalidReclaimPolicy", "roleReclaimPolicy Delete cannot be used while databaseReclaimPolicy Retain because the retained database remains owned by the role", 0), nil
	}

	clusterRef := profile.Spec.Shared.ClusterRef
	if clusterRef.Name == "" || clusterRef.Namespace == "" {
		return postgresqlBlocked("InvalidProfile", "SharedCluster profile must define clusterRef.name and clusterRef.namespace", 0), nil
	}
	cluster := cnpgObject(cnpgClusterGVK, clusterRef.Namespace, clusterRef.Name)
	if err := r.Get(ctx, client.ObjectKey{Namespace: clusterRef.Namespace, Name: clusterRef.Name}, cluster); err != nil {
		if apierrors.IsNotFound(err) {
			return postgresqlBlocked("PlatformDependencyMissing", fmt.Sprintf("CloudNativePG Cluster %s/%s does not exist", clusterRef.Namespace, clusterRef.Name), 30*time.Second), nil
		}
		return postgresqlResult{}, err
	}

	names := resolvePostgreSQLNames(bundle, &profile)
	secret, blocked, err := r.ensurePostgreSQLSecret(ctx, bundle, &profile, names)
	if err != nil {
		return postgresqlResult{}, err
	}
	if blocked != nil {
		return *blocked, nil
	}

	role, blocked, err := r.ensureDatabaseRole(ctx, bundle, &profile, names, secret.Name)
	if err != nil {
		return postgresqlResult{}, err
	}
	if blocked != nil {
		return *blocked, nil
	}
	if !cnpgAppliedForCurrentGeneration(role) {
		message, _, _ := unstructured.NestedString(role.Object, "status", "message")
		if message == "" {
			message = "CloudNativePG has not applied the current tenant DatabaseRole generation yet"
		}
		return postgresqlPending("DatabaseRolePending", message, names, 5*time.Second), nil
	}

	database, blocked, err := r.ensureDatabase(ctx, bundle, &profile, names)
	if err != nil {
		return postgresqlResult{}, err
	}
	if blocked != nil {
		return *blocked, nil
	}
	if !cnpgAppliedForCurrentGeneration(database) {
		message, _, _ := unstructured.NestedString(database.Object, "status", "message")
		if message == "" {
			message = "CloudNativePG has not applied the current tenant Database generation yet"
		}
		return postgresqlPending("DatabasePending", message, names, 5*time.Second), nil
	}
	if ready, message := requiredExtensionsReady(database, normalizedExtensions(profile.Spec.RequiredExtensions)); !ready {
		return postgresqlPending("ExtensionsPending", message, names, 5*time.Second), nil
	}

	status := postgresqlComponentStatus("Ready", names)
	return postgresqlResult{
		Ready:   true,
		Status:  status,
		Reason:  "Reconciled",
		Message: fmt.Sprintf("tenant PostgreSQL Database and DatabaseRole are reconciled; credentials are stored in Secret %s", names.Secret),
	}, nil
}

func (r *TenantBundleReconciler) cleanupPostgreSQL(ctx context.Context, bundle *fabricv1alpha1.TenantBundle) (bool, error) {
	selector := client.MatchingLabels{
		LabelManaged:    "true",
		LabelTenantID:   bundle.Spec.TenantID,
		LabelTenantName: bundle.Name,
	}
	pending := false

	databases := cnpgList(cnpgDatabaseGVK)
	if err := r.List(ctx, databases, selector); err != nil {
		return false, err
	}
	for i := range databases.Items {
		item := &databases.Items[i]
		if item.GetLabels()[labelPostgreSQLResource] != "database" {
			continue
		}
		pending = true
		if item.GetDeletionTimestamp() == nil {
			if err := r.Delete(ctx, item); err != nil && !apierrors.IsNotFound(err) {
				return false, err
			}
		}
	}

	roles := cnpgList(cnpgDatabaseRoleGVK)
	if err := r.List(ctx, roles, selector); err != nil {
		return false, err
	}
	for i := range roles.Items {
		item := &roles.Items[i]
		if item.GetLabels()[labelPostgreSQLResource] != "role" {
			continue
		}
		pending = true
		reclaim, _, _ := unstructured.NestedString(item.Object, "spec", "databaseRoleReclaimPolicy")
		secretName, _, _ := unstructured.NestedString(item.Object, "spec", "passwordSecret", "name")
		if item.GetDeletionTimestamp() == nil {
			if err := r.Delete(ctx, item); err != nil && !apierrors.IsNotFound(err) {
				return false, err
			}
		}
		if strings.EqualFold(reclaim, "delete") && secretName != "" {
			secret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: secretName, Namespace: item.GetNamespace()}}
			if err := r.Delete(ctx, secret); err != nil && !apierrors.IsNotFound(err) {
				return false, err
			}
		}
	}

	return pending, nil
}

func (r *TenantBundleReconciler) ensurePostgreSQLSecret(ctx context.Context, bundle *fabricv1alpha1.TenantBundle, profile *fabricv1alpha1.PostgreSQLProfile, names postgresqlNames) (*corev1.Secret, *postgresqlResult, error) {
	namespace := profile.Spec.Shared.ClusterRef.Namespace
	key := client.ObjectKey{Namespace: namespace, Name: names.Secret}
	var secret corev1.Secret
	if err := r.Get(ctx, key, &secret); err != nil {
		if !apierrors.IsNotFound(err) {
			return nil, nil, err
		}
		password, err := randomPassword()
		if err != nil {
			return nil, nil, err
		}
		secret = corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: names.Secret, Namespace: namespace, Labels: postgresqlLabels(bundle, profile, "credentials", profile.Spec.RoleReclaimPolicy)},
			Type:       corev1.SecretTypeBasicAuth,
			Data: map[string][]byte{
				corev1.BasicAuthUsernameKey: []byte(names.Role),
				corev1.BasicAuthPasswordKey: []byte(password),
				"host":                    []byte(profile.Spec.Shared.ClusterRef.Name + "-rw." + namespace + ".svc"),
				"port":                    []byte("5432"),
				"dbname":                  []byte(names.Database),
			},
		}
		secret.Labels[cnpgReloadLabel] = "true"
		if err := r.Create(ctx, &secret); err != nil {
			return nil, nil, err
		}
		return &secret, nil, nil
	}

	if !postgresqlOwnedBy(&secret, bundle) {
		blocked := postgresqlBlocked("OwnershipConflict", fmt.Sprintf("Secret %s/%s exists but is not owned by this Fabric tenant", namespace, names.Secret), 0)
		return nil, &blocked, nil
	}
	if secret.Type != corev1.SecretTypeBasicAuth {
		blocked := postgresqlBlocked("OwnershipConflict", fmt.Sprintf("Secret %s/%s has type %q, expected kubernetes.io/basic-auth", namespace, names.Secret, secret.Type), 0)
		return nil, &blocked, nil
	}
	if string(secret.Data[corev1.BasicAuthUsernameKey]) != names.Role || len(secret.Data[corev1.BasicAuthPasswordKey]) == 0 {
		blocked := postgresqlBlocked("OwnershipConflict", fmt.Sprintf("Secret %s/%s does not contain the expected tenant role credentials", namespace, names.Secret), 0)
		return nil, &blocked, nil
	}

	desiredLabels := postgresqlLabels(bundle, profile, "credentials", profile.Spec.RoleReclaimPolicy)
	desiredLabels[cnpgReloadLabel] = "true"
	mergedLabels := mergeStringMap(copyStringMap(secret.Labels), desiredLabels)
	desiredData := map[string][]byte{
		corev1.BasicAuthUsernameKey: secret.Data[corev1.BasicAuthUsernameKey],
		corev1.BasicAuthPasswordKey: secret.Data[corev1.BasicAuthPasswordKey],
		"host":                    []byte(profile.Spec.Shared.ClusterRef.Name + "-rw." + namespace + ".svc"),
		"port":                    []byte("5432"),
		"dbname":                  []byte(names.Database),
	}
	if !reflect.DeepEqual(secret.Labels, mergedLabels) || !reflect.DeepEqual(secret.Data, desiredData) {
		secret.Labels = mergedLabels
		secret.Data = desiredData
		if err := r.Update(ctx, &secret); err != nil {
			return nil, nil, err
		}
	}
	return &secret, nil, nil
}

func (r *TenantBundleReconciler) ensureDatabaseRole(ctx context.Context, bundle *fabricv1alpha1.TenantBundle, profile *fabricv1alpha1.PostgreSQLProfile, names postgresqlNames, secretName string) (*unstructured.Unstructured, *postgresqlResult, error) {
	namespace := profile.Spec.Shared.ClusterRef.Namespace
	key := client.ObjectKey{Namespace: namespace, Name: names.RoleResource}
	role := cnpgObject(cnpgDatabaseRoleGVK, namespace, names.RoleResource)
	desiredSpec := map[string]interface{}{
		"cluster":                   map[string]interface{}{"name": profile.Spec.Shared.ClusterRef.Name},
		"name":                      names.Role,
		"comment":                   fmt.Sprintf("TXO Fabric tenant %s (%s)", bundle.Name, bundle.Spec.TenantID),
		"login":                     true,
		"inherit":                   true,
		"superuser":                 false,
		"createdb":                  false,
		"createrole":                false,
		"replication":               false,
		"bypassrls":                 false,
		"connectionLimit":           int64(-1),
		"ensure":                    "present",
		"passwordSecret":            map[string]interface{}{"name": secretName},
		"databaseRoleReclaimPolicy": cnpgReclaimPolicy(profile.Spec.RoleReclaimPolicy),
	}
	if err := r.Get(ctx, key, role); err != nil {
		if !apierrors.IsNotFound(err) {
			return nil, nil, err
		}
		role = cnpgObject(cnpgDatabaseRoleGVK, namespace, names.RoleResource)
		role.SetLabels(postgresqlLabels(bundle, profile, "role", profile.Spec.RoleReclaimPolicy))
		role.Object["spec"] = desiredSpec
		if err := r.Create(ctx, role); err != nil {
			return nil, nil, err
		}
		return role, nil, nil
	}
	if !postgresqlOwnedBy(role, bundle) {
		blocked := postgresqlBlocked("OwnershipConflict", fmt.Sprintf("DatabaseRole %s/%s exists but is not owned by this Fabric tenant", namespace, names.RoleResource), 0)
		return nil, &blocked, nil
	}
	if !immutableCNPGIdentityMatches(role, profile.Spec.Shared.ClusterRef.Name, names.Role) {
		blocked := postgresqlBlocked("OwnershipConflict", fmt.Sprintf("DatabaseRole %s/%s has an incompatible immutable cluster/name identity", namespace, names.RoleResource), 0)
		return nil, &blocked, nil
	}
	desiredLabels := mergeStringMap(copyStringMap(role.GetLabels()), postgresqlLabels(bundle, profile, "role", profile.Spec.RoleReclaimPolicy))
	currentSpec, _, _ := unstructured.NestedMap(role.Object, "spec")
	if !reflect.DeepEqual(role.GetLabels(), desiredLabels) || !reflect.DeepEqual(currentSpec, desiredSpec) {
		role.SetLabels(desiredLabels)
		role.Object["spec"] = desiredSpec
		if err := r.Update(ctx, role); err != nil {
			return nil, nil, err
		}
	}
	return role, nil, nil
}

func (r *TenantBundleReconciler) ensureDatabase(ctx context.Context, bundle *fabricv1alpha1.TenantBundle, profile *fabricv1alpha1.PostgreSQLProfile, names postgresqlNames) (*unstructured.Unstructured, *postgresqlResult, error) {
	namespace := profile.Spec.Shared.ClusterRef.Namespace
	key := client.ObjectKey{Namespace: namespace, Name: names.DatabaseResource}
	database := cnpgObject(cnpgDatabaseGVK, namespace, names.DatabaseResource)
	extensions := make([]interface{}, 0, len(profile.Spec.RequiredExtensions))
	for _, extension := range normalizedExtensions(profile.Spec.RequiredExtensions) {
		extensions = append(extensions, map[string]interface{}{"name": extension, "ensure": "present"})
	}
	desiredSpec := map[string]interface{}{
		"cluster":               map[string]interface{}{"name": profile.Spec.Shared.ClusterRef.Name},
		"name":                  names.Database,
		"owner":                 names.Role,
		"ensure":                "present",
		"databaseReclaimPolicy": cnpgReclaimPolicy(profile.Spec.DatabaseReclaimPolicy),
		"extensions":            extensions,
	}
	if err := r.Get(ctx, key, database); err != nil {
		if !apierrors.IsNotFound(err) {
			return nil, nil, err
		}
		database = cnpgObject(cnpgDatabaseGVK, namespace, names.DatabaseResource)
		database.SetLabels(postgresqlLabels(bundle, profile, "database", profile.Spec.DatabaseReclaimPolicy))
		database.Object["spec"] = desiredSpec
		if err := r.Create(ctx, database); err != nil {
			return nil, nil, err
		}
		return database, nil, nil
	}
	if !postgresqlOwnedBy(database, bundle) {
		blocked := postgresqlBlocked("OwnershipConflict", fmt.Sprintf("Database %s/%s exists but is not owned by this Fabric tenant", namespace, names.DatabaseResource), 0)
		return nil, &blocked, nil
	}
	if !immutableCNPGIdentityMatches(database, profile.Spec.Shared.ClusterRef.Name, names.Database) {
		blocked := postgresqlBlocked("OwnershipConflict", fmt.Sprintf("Database %s/%s has an incompatible immutable cluster/name identity", namespace, names.DatabaseResource), 0)
		return nil, &blocked, nil
	}
	desiredLabels := mergeStringMap(copyStringMap(database.GetLabels()), postgresqlLabels(bundle, profile, "database", profile.Spec.DatabaseReclaimPolicy))
	currentSpec, _, _ := unstructured.NestedMap(database.Object, "spec")
	if !reflect.DeepEqual(database.GetLabels(), desiredLabels) || !reflect.DeepEqual(currentSpec, desiredSpec) {
		database.SetLabels(desiredLabels)
		database.Object["spec"] = desiredSpec
		if err := r.Update(ctx, database); err != nil {
			return nil, nil, err
		}
	}
	return database, nil, nil
}

func postgresqlBlocked(reason, message string, requeueAfter time.Duration) postgresqlResult {
	return postgresqlResult{
		Ready:        false,
		Status:       &fabricv1alpha1.ComponentStatus{Phase: "Blocked", Message: message},
		Reason:       reason,
		Message:      message,
		RequeueAfter: requeueAfter,
	}
}

func postgresqlPending(reason, message string, names postgresqlNames, requeueAfter time.Duration) postgresqlResult {
	status := postgresqlComponentStatus("Pending", names)
	status.Message = message + "; " + status.Message
	return postgresqlResult{Ready: false, Status: status, Reason: reason, Message: message, RequeueAfter: requeueAfter}
}

func postgresqlComponentStatus(phase string, names postgresqlNames) *fabricv1alpha1.ComponentStatus {
	return &fabricv1alpha1.ComponentStatus{
		Phase:   phase,
		Message: fmt.Sprintf("databaseResource=%s roleResource=%s credentialsSecret=%s", names.DatabaseResource, names.RoleResource, names.Secret),
	}
}

func resolvePostgreSQLNames(bundle *fabricv1alpha1.TenantBundle, profile *fabricv1alpha1.PostgreSQLProfile) postgresqlNames {
	tenantToken := strings.ToLower(bundle.Spec.TenantID)
	databasePrefix := profile.Spec.Shared.DatabaseNamePrefix
	if databasePrefix == "" {
		databasePrefix = "txo_"
	}
	rolePrefix := profile.Spec.Shared.RoleNamePrefix
	if rolePrefix == "" {
		rolePrefix = "txo_"
	}
	return postgresqlNames{
		DatabaseResource: boundedDNSName("txo-" + tenantToken + "-database"),
		RoleResource:     boundedDNSName("txo-" + tenantToken + "-role"),
		Secret:           boundedDNSName("txo-" + tenantToken + "-postgresql"),
		Database:         boundedPostgresIdentifier(databasePrefix + tenantToken),
		Role:             boundedPostgresIdentifier(rolePrefix + tenantToken),
	}
}

func postgresqlLabels(bundle *fabricv1alpha1.TenantBundle, profile *fabricv1alpha1.PostgreSQLProfile, resourceKind, reclaimPolicy string) map[string]string {
	labels := tenantLabels(bundle)
	labels[LabelName] = "tenant-postgresql"
	labels[LabelInstance] = strings.ToLower(bundle.Spec.TenantID)
	labels["app.kubernetes.io/component"] = "tenant-persistence"
	labels[labelPostgreSQLProfile] = profile.Name
	labels[labelPostgreSQLResource] = resourceKind
	labels[labelReclaimPolicy] = cnpgReclaimPolicy(reclaimPolicy)
	return labels
}

func postgresqlOwnedBy(obj metav1.Object, bundle *fabricv1alpha1.TenantBundle) bool {
	labels := obj.GetLabels()
	return labels[LabelManaged] == "true" && labels[LabelTenantID] == bundle.Spec.TenantID && labels[LabelTenantName] == bundle.Name
}

func immutableCNPGIdentityMatches(obj *unstructured.Unstructured, clusterName, postgresName string) bool {
	cluster, _, _ := unstructured.NestedString(obj.Object, "spec", "cluster", "name")
	name, _, _ := unstructured.NestedString(obj.Object, "spec", "name")
	return cluster == clusterName && name == postgresName
}

func cnpgAppliedForCurrentGeneration(obj *unstructured.Unstructured) bool {
	applied, found, err := unstructured.NestedBool(obj.Object, "status", "applied")
	if err != nil || !found || !applied {
		return false
	}
	observedGeneration, found, err := unstructured.NestedInt64(obj.Object, "status", "observedGeneration")
	if err != nil {
		return false
	}
	// CloudNativePG v1.30 reports observedGeneration for Database and
	// DatabaseRole. Accept an absent field defensively for compatibility with
	// older/fake clients, but never accept a stale generation once it is reported.
	return !found || observedGeneration == obj.GetGeneration()
}

func requiredExtensionsReady(database *unstructured.Unstructured, required []string) (bool, string) {
	if len(required) == 0 {
		return true, ""
	}
	statuses, found, err := unstructured.NestedSlice(database.Object, "status", "extensions")
	if err != nil {
		return false, fmt.Sprintf("CloudNativePG extension status is invalid: %v", err)
	}
	// Database.status.applied already means the aggregate database desired state
	// is reconciled. When CNPG also publishes per-extension status, use it to
	// prevent a required extension failure from being hidden behind an old ready
	// state and to surface the concrete extension message.
	if !found {
		return true, ""
	}
	byName := make(map[string]map[string]interface{}, len(statuses))
	for _, raw := range statuses {
		entry, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		name, _ := entry["name"].(string)
		if name != "" {
			byName[name] = entry
		}
	}
	for _, name := range required {
		entry, ok := byName[name]
		if !ok {
			return false, fmt.Sprintf("required PostgreSQL extension %s has not been reported by CloudNativePG", name)
		}
		applied, _ := entry["applied"].(bool)
		if applied {
			continue
		}
		message, _ := entry["message"].(string)
		if message != "" {
			return false, fmt.Sprintf("required PostgreSQL extension %s is not ready: %s", name, message)
		}
		return false, fmt.Sprintf("required PostgreSQL extension %s has not been applied by CloudNativePG", name)
	}
	return true, ""
}

func normalizedExtensions(extensions []string) []string {
	seen := make(map[string]struct{}, len(extensions))
	result := make([]string, 0, len(extensions))
	for _, extension := range extensions {
		extension = strings.TrimSpace(extension)
		if extension == "" {
			continue
		}
		if _, exists := seen[extension]; exists {
			continue
		}
		seen[extension] = struct{}{}
		result = append(result, extension)
	}
	sort.Strings(result)
	return result
}

func cnpgReclaimPolicy(value string) string {
	if strings.EqualFold(value, "Delete") {
		return "delete"
	}
	return "retain"
}

func cnpgObject(gvk schema.GroupVersionKind, namespace, name string) *unstructured.Unstructured {
	object := &unstructured.Unstructured{}
	object.SetGroupVersionKind(gvk)
	object.SetNamespace(namespace)
	object.SetName(name)
	return object
}

func cnpgList(gvk schema.GroupVersionKind) *unstructured.UnstructuredList {
	list := &unstructured.UnstructuredList{}
	list.SetGroupVersionKind(schema.GroupVersionKind{Group: gvk.Group, Version: gvk.Version, Kind: gvk.Kind + "List"})
	return list
}

func randomPassword() (string, error) {
	buffer := make([]byte, 32)
	if _, err := rand.Read(buffer); err != nil {
		return "", fmt.Errorf("generate PostgreSQL password: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buffer), nil
}

func boundedPostgresIdentifier(value string) string {
	if len(value) <= 63 {
		return value
	}
	hash := sha256.Sum256([]byte(value))
	suffix := hex.EncodeToString(hash[:4])
	prefix := strings.TrimRight(value[:63-len(suffix)-1], "_")
	return prefix + "_" + suffix
}

func boundedDNSName(value string) string {
	value = strings.ToLower(strings.ReplaceAll(value, "_", "-"))
	if len(value) <= 63 {
		return strings.Trim(value, "-")
	}
	hash := sha256.Sum256([]byte(value))
	suffix := hex.EncodeToString(hash[:4])
	prefix := strings.TrimRight(value[:63-len(suffix)-1], "-")
	return prefix + "-" + suffix
}
