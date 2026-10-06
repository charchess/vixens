package controller

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"reflect"
	"strings"

	fabricv1alpha1 "github.com/charchess/vixens/apps/60-services/txo-fabric/operator/api/v1alpha1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	defaultAIGatewayPostgreSQLProfileName = "postgresql-litellm"
	tenantAIGatewayRuntimeSecretName       = "txo-ai-gateway-runtime"
	tenantAIGatewayMigrationPolicyName     = "txo-ai-gateway-migration-egress"
	labelAIGatewayPostgreSQLResource       = "fabric.truxonline.io/ai-gateway-postgresql-resource"
	annotationAIGatewayPostgreSQLSecretUID = "fabric.truxonline.io/ai-gateway-postgresql-secret-uid"
)

type aiGatewayPostgreSQLNames struct {
	DatabaseResource string
	RoleResource     string
	Secret           string
	Database         string
	Role             string
}

type aiGatewayPostgreSQLResult struct {
	Ready        bool
	Secret       *corev1.Secret
	Names        aiGatewayPostgreSQLNames
	Reason       string
	Message      string
	RequeueAfter int64
}

func resolveAIGatewayPostgreSQLNames(bundle *fabricv1alpha1.TenantBundle, profile *fabricv1alpha1.PostgreSQLProfile) aiGatewayPostgreSQLNames {
	tenantToken := strings.ToLower(bundle.Spec.TenantID)
	databasePrefix := profile.Spec.Shared.DatabaseNamePrefix
	if databasePrefix == "" {
		databasePrefix = "txo_"
	}
	rolePrefix := profile.Spec.Shared.RoleNamePrefix
	if rolePrefix == "" {
		rolePrefix = "txo_"
	}
	return aiGatewayPostgreSQLNames{
		DatabaseResource: boundedDNSName("txo-" + tenantToken + "-ai-gateway-database"),
		RoleResource:     boundedDNSName("txo-" + tenantToken + "-ai-gateway-role"),
		Secret:           boundedDNSName("txo-" + tenantToken + "-ai-gateway-postgresql"),
		Database:         boundedPostgresIdentifier(databasePrefix + tenantToken + "_ai_gateway"),
		Role:             boundedPostgresIdentifier(rolePrefix + tenantToken + "_ai_gateway"),
	}
}

func aiGatewayPostgreSQLLabels(bundle *fabricv1alpha1.TenantBundle, profile *fabricv1alpha1.PostgreSQLProfile, resourceKind, reclaimPolicy string) map[string]string {
	labels := tenantLabels(bundle)
	labels[LabelName] = "txo-ai-gateway-postgresql"
	labels[LabelInstance] = strings.ToLower(bundle.Spec.TenantID)
	labels["app.kubernetes.io/component"] = "tenant-ai-gateway-persistence"
	labels[labelPostgreSQLProfile] = profile.Name
	labels[labelAIGatewayPostgreSQLResource] = resourceKind
	labels[labelReclaimPolicy] = cnpgReclaimPolicy(reclaimPolicy)
	return labels
}

func aiGatewayPostgreSQLOwnedBy(obj metav1.Object, bundle *fabricv1alpha1.TenantBundle) bool {
	labels := obj.GetLabels()
	return labels[LabelManaged] == "true" &&
		labels[LabelTenantID] == bundle.Spec.TenantID &&
		labels[LabelTenantName] == bundle.Name &&
		labels[labelAIGatewayPostgreSQLResource] != ""
}

func (r *TenantBundleReconciler) reconcileAIGatewayPostgreSQL(
	ctx context.Context,
	bundle *fabricv1alpha1.TenantBundle,
	gatewayProfile *fabricv1alpha1.AIGatewayProfile,
) (aiGatewayPostgreSQLResult, error) {
	profileName := strings.TrimSpace(gatewayProfile.Spec.PostgreSQLProfileRef)
	if profileName == "" {
		profileName = defaultAIGatewayPostgreSQLProfileName
	}

	var profile fabricv1alpha1.PostgreSQLProfile
	if err := r.Get(ctx, client.ObjectKey{Name: profileName}, &profile); err != nil {
		if apierrors.IsNotFound(err) {
			return aiGatewayPostgreSQLResult{Reason: "PostgreSQLProfileNotFound", Message: fmt.Sprintf("PostgreSQLProfile %q does not exist", profileName), RequeueAfter: 30}, nil
		}
		return aiGatewayPostgreSQLResult{}, err
	}
	if profile.Spec.Provider != "" && profile.Spec.Provider != "CloudNativePG" {
		return aiGatewayPostgreSQLResult{Reason: "UnsupportedPostgreSQLProvider", Message: fmt.Sprintf("PostgreSQL provider %q is not supported", profile.Spec.Provider)}, nil
	}
	if profile.Spec.Topology != "SharedCluster" {
		return aiGatewayPostgreSQLResult{Reason: "UnsupportedPostgreSQLTopology", Message: fmt.Sprintf("LiteLLM requires SharedCluster PostgreSQL, got %q", profile.Spec.Topology)}, nil
	}
	if strings.EqualFold(profile.Spec.DatabaseReclaimPolicy, "Retain") && strings.EqualFold(profile.Spec.RoleReclaimPolicy, "Delete") {
		return aiGatewayPostgreSQLResult{Reason: "InvalidPostgreSQLReclaimPolicy", Message: "roleReclaimPolicy Delete cannot be used while databaseReclaimPolicy Retain"}, nil
	}

	clusterRef := profile.Spec.Shared.ClusterRef
	if strings.TrimSpace(clusterRef.Name) == "" || strings.TrimSpace(clusterRef.Namespace) == "" {
		return aiGatewayPostgreSQLResult{Reason: "InvalidPostgreSQLProfile", Message: "LiteLLM PostgreSQL profile must declare shared.clusterRef.name and namespace"}, nil
	}
	cluster := cnpgObject(cnpgClusterGVK, clusterRef.Namespace, clusterRef.Name)
	if err := r.Get(ctx, client.ObjectKeyFromObject(cluster), cluster); err != nil {
		if apierrors.IsNotFound(err) {
			return aiGatewayPostgreSQLResult{Reason: "PostgreSQLClusterMissing", Message: fmt.Sprintf("CloudNativePG Cluster %s/%s does not exist", clusterRef.Namespace, clusterRef.Name), RequeueAfter: 30}, nil
		}
		return aiGatewayPostgreSQLResult{}, err
	}

	names := resolveAIGatewayPostgreSQLNames(bundle, &profile)
	secret, blocked, err := r.ensureAIGatewayPostgreSQLSecret(ctx, bundle, &profile, names)
	if err != nil {
		return aiGatewayPostgreSQLResult{}, err
	}
	if blocked != nil {
		blocked.Names = names
		return *blocked, nil
	}

	role, blocked, err := r.ensureAIGatewayDatabaseRole(ctx, bundle, &profile, names, secret.Name)
	if err != nil {
		return aiGatewayPostgreSQLResult{}, err
	}
	if blocked != nil {
		blocked.Names = names
		return *blocked, nil
	}
	if !cnpgAppliedForCurrentGeneration(role) {
		message, _, _ := unstructured.NestedString(role.Object, "status", "message")
		if message == "" {
			message = "waiting for CloudNativePG to apply the LiteLLM DatabaseRole"
		}
		return aiGatewayPostgreSQLResult{Names: names, Reason: "PostgreSQLRolePending", Message: message, RequeueAfter: 5}, nil
	}

	database, blocked, err := r.ensureAIGatewayDatabase(ctx, bundle, &profile, names)
	if err != nil {
		return aiGatewayPostgreSQLResult{}, err
	}
	if blocked != nil {
		blocked.Names = names
		return *blocked, nil
	}
	if !cnpgAppliedForCurrentGeneration(database) {
		message, _, _ := unstructured.NestedString(database.Object, "status", "message")
		if message == "" {
			message = "waiting for CloudNativePG to apply the LiteLLM Database"
		}
		return aiGatewayPostgreSQLResult{Names: names, Reason: "PostgreSQLDatabasePending", Message: message, RequeueAfter: 5}, nil
	}
	if ready, message := requiredExtensionsReady(database, normalizedExtensions(profile.Spec.RequiredExtensions)); !ready {
		return aiGatewayPostgreSQLResult{Names: names, Reason: "PostgreSQLExtensionsPending", Message: message, RequeueAfter: 5}, nil
	}

	return aiGatewayPostgreSQLResult{
		Ready:   true,
		Secret:  secret,
		Names:   names,
		Reason:  "PostgreSQLReady",
		Message: fmt.Sprintf("dedicated LiteLLM PostgreSQL database %s and role %s are ready", names.Database, names.Role),
	}, nil
}

func (r *TenantBundleReconciler) ensureAIGatewayPostgreSQLSecret(
	ctx context.Context,
	bundle *fabricv1alpha1.TenantBundle,
	profile *fabricv1alpha1.PostgreSQLProfile,
	names aiGatewayPostgreSQLNames,
) (*corev1.Secret, *aiGatewayPostgreSQLResult, error) {
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
			ObjectMeta: metav1.ObjectMeta{
				Name:      names.Secret,
				Namespace: namespace,
				Labels:    aiGatewayPostgreSQLLabels(bundle, profile, "credentials", profile.Spec.RoleReclaimPolicy),
			},
			Type: corev1.SecretTypeBasicAuth,
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

	if !aiGatewayPostgreSQLOwnedBy(&secret, bundle) {
		blocked := &aiGatewayPostgreSQLResult{Reason: "PostgreSQLOwnershipConflict", Message: fmt.Sprintf("Secret %s/%s exists but is not owned by this tenant AI gateway", namespace, names.Secret)}
		return nil, blocked, nil
	}
	if secret.Type != corev1.SecretTypeBasicAuth ||
		string(secret.Data[corev1.BasicAuthUsernameKey]) != names.Role ||
		len(secret.Data[corev1.BasicAuthPasswordKey]) == 0 {
		blocked := &aiGatewayPostgreSQLResult{Reason: "PostgreSQLOwnershipConflict", Message: fmt.Sprintf("Secret %s/%s does not contain the expected LiteLLM database credentials", namespace, names.Secret)}
		return nil, blocked, nil
	}

	desiredLabels := aiGatewayPostgreSQLLabels(bundle, profile, "credentials", profile.Spec.RoleReclaimPolicy)
	desiredLabels[cnpgReloadLabel] = "true"
	desiredLabels = mergeStringMap(copyStringMap(secret.Labels), desiredLabels)
	desiredData := map[string][]byte{
		corev1.BasicAuthUsernameKey: secret.Data[corev1.BasicAuthUsernameKey],
		corev1.BasicAuthPasswordKey: secret.Data[corev1.BasicAuthPasswordKey],
		"host":                    []byte(profile.Spec.Shared.ClusterRef.Name + "-rw." + namespace + ".svc"),
		"port":                    []byte("5432"),
		"dbname":                  []byte(names.Database),
	}
	if !reflect.DeepEqual(secret.Labels, desiredLabels) || !reflect.DeepEqual(secret.Data, desiredData) {
		secret.Labels = desiredLabels
		secret.Data = desiredData
		if err := r.Update(ctx, &secret); err != nil {
			return nil, nil, err
		}
	}
	return &secret, nil, nil
}

func (r *TenantBundleReconciler) ensureAIGatewayDatabaseRole(
	ctx context.Context,
	bundle *fabricv1alpha1.TenantBundle,
	profile *fabricv1alpha1.PostgreSQLProfile,
	names aiGatewayPostgreSQLNames,
	secretName string,
) (*unstructured.Unstructured, *aiGatewayPostgreSQLResult, error) {
	namespace := profile.Spec.Shared.ClusterRef.Namespace
	role := cnpgObject(cnpgDatabaseRoleGVK, namespace, names.RoleResource)
	desiredSpec := map[string]interface{}{
		"cluster":                   map[string]interface{}{"name": profile.Spec.Shared.ClusterRef.Name},
		"name":                      names.Role,
		"comment":                   fmt.Sprintf("TXO Fabric LiteLLM gateway for tenant %s (%s)", bundle.Name, bundle.Spec.TenantID),
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
	if err := r.Get(ctx, client.ObjectKeyFromObject(role), role); err != nil {
		if !apierrors.IsNotFound(err) {
			return nil, nil, err
		}
		role = cnpgObject(cnpgDatabaseRoleGVK, namespace, names.RoleResource)
		role.SetLabels(aiGatewayPostgreSQLLabels(bundle, profile, "role", profile.Spec.RoleReclaimPolicy))
		role.Object["spec"] = desiredSpec
		if err := r.Create(ctx, role); err != nil {
			return nil, nil, err
		}
		return role, nil, nil
	}
	if !aiGatewayPostgreSQLOwnedBy(role, bundle) || !immutableCNPGIdentityMatches(role, profile.Spec.Shared.ClusterRef.Name, names.Role) {
		blocked := &aiGatewayPostgreSQLResult{Reason: "PostgreSQLOwnershipConflict", Message: fmt.Sprintf("DatabaseRole %s/%s has foreign or incompatible ownership", namespace, names.RoleResource)}
		return nil, blocked, nil
	}
	desiredLabels := mergeStringMap(copyStringMap(role.GetLabels()), aiGatewayPostgreSQLLabels(bundle, profile, "role", profile.Spec.RoleReclaimPolicy))
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

func (r *TenantBundleReconciler) ensureAIGatewayDatabase(
	ctx context.Context,
	bundle *fabricv1alpha1.TenantBundle,
	profile *fabricv1alpha1.PostgreSQLProfile,
	names aiGatewayPostgreSQLNames,
) (*unstructured.Unstructured, *aiGatewayPostgreSQLResult, error) {
	namespace := profile.Spec.Shared.ClusterRef.Namespace
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
	if err := r.Get(ctx, client.ObjectKeyFromObject(database), database); err != nil {
		if !apierrors.IsNotFound(err) {
			return nil, nil, err
		}
		database = cnpgObject(cnpgDatabaseGVK, namespace, names.DatabaseResource)
		database.SetLabels(aiGatewayPostgreSQLLabels(bundle, profile, "database", profile.Spec.DatabaseReclaimPolicy))
		database.Object["spec"] = desiredSpec
		if err := r.Create(ctx, database); err != nil {
			return nil, nil, err
		}
		return database, nil, nil
	}
	if !aiGatewayPostgreSQLOwnedBy(database, bundle) || !immutableCNPGIdentityMatches(database, profile.Spec.Shared.ClusterRef.Name, names.Database) {
		blocked := &aiGatewayPostgreSQLResult{Reason: "PostgreSQLOwnershipConflict", Message: fmt.Sprintf("Database %s/%s has foreign or incompatible ownership", namespace, names.DatabaseResource)}
		return nil, blocked, nil
	}
	desiredLabels := mergeStringMap(copyStringMap(database.GetLabels()), aiGatewayPostgreSQLLabels(bundle, profile, "database", profile.Spec.DatabaseReclaimPolicy))
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

func aiGatewayRuntimeLabels(bundle *fabricv1alpha1.TenantBundle) map[string]string {
	labels := tenantLabels(bundle)
	labels[LabelName] = "txo-ai-gateway"
	labels[LabelInstance] = strings.ToLower(bundle.Spec.TenantID)
	labels["app.kubernetes.io/component"] = "tenant-ai-gateway"
	return labels
}

func aiGatewayMigrationPodSelector(bundle *fabricv1alpha1.TenantBundle) map[string]string {
	return map[string]string{
		LabelName:     "txo-ai-gateway-migration",
		LabelInstance: strings.ToLower(bundle.Spec.TenantID),
	}
}

func (r *TenantBundleReconciler) ensureAIGatewayRuntimeSecret(
	ctx context.Context,
	bundle *fabricv1alpha1.TenantBundle,
	databaseSecret *corev1.Secret,
) (*corev1.Secret, error) {
	namespace := tenantNamespace(bundle.Name)
	secret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: tenantAIGatewayRuntimeSecretName, Namespace: namespace}}
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, secret, func() error {
		secret.Labels = mergeStringMap(secret.Labels, aiGatewayRuntimeLabels(bundle))
		if err := controllerutil.SetControllerReference(bundle, secret, r.Scheme); err != nil {
			return err
		}
		secret.Type = corev1.SecretTypeOpaque
		if secret.Data == nil {
			secret.Data = map[string][]byte{}
		}
		if len(secret.Data["LITELLM_MASTER_KEY"]) == 0 {
			masterKey, err := randomPassword()
			if err != nil {
				return err
			}
			secret.Data["LITELLM_MASTER_KEY"] = []byte(masterKey)
		}
		secret.Data["DB_USERNAME"] = append([]byte(nil), databaseSecret.Data[corev1.BasicAuthUsernameKey]...)
		secret.Data["DB_PASSWORD"] = append([]byte(nil), databaseSecret.Data[corev1.BasicAuthPasswordKey]...)
		secret.Data["DB_HOST"] = append([]byte(nil), databaseSecret.Data["host"]...)
		secret.Data["DB_PORT"] = append([]byte(nil), databaseSecret.Data["port"]...)
		secret.Data["DB_NAME"] = append([]byte(nil), databaseSecret.Data["dbname"]...)
		if secret.Annotations == nil {
			secret.Annotations = map[string]string{}
		}
		secret.Annotations[annotationAIGatewayPostgreSQLSecretUID] = string(databaseSecret.UID)
		return nil
	})
	return secret, err
}

func aiGatewayMigrationName(profile *fabricv1alpha1.AIGatewayProfile, databaseSecret *corev1.Secret) string {
	sum := sha256.Sum256([]byte(profile.Spec.Image + "|" + string(databaseSecret.UID)))
	return boundedDNSName("txo-ai-gateway-migrations-" + hex.EncodeToString(sum[:4]))
}

func (r *TenantBundleReconciler) ensureAIGatewayMigrationNetworkPolicy(
	ctx context.Context,
	bundle *fabricv1alpha1.TenantBundle,
	postgresqlProfile *fabricv1alpha1.PostgreSQLProfile,
) error {
	namespace := tenantNamespace(bundle.Name)
	policy := &networkingv1.NetworkPolicy{ObjectMeta: metav1.ObjectMeta{Name: tenantAIGatewayMigrationPolicyName, Namespace: namespace}}
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, policy, func() error {
		policy.Labels = mergeStringMap(policy.Labels, aiGatewayRuntimeLabels(bundle))
		if err := controllerutil.SetControllerReference(bundle, policy, r.Scheme); err != nil {
			return err
		}
		policy.Spec = networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{MatchLabels: aiGatewayMigrationPodSelector(bundle)},
			PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeEgress},
			Egress: []networkingv1.NetworkPolicyEgressRule{
				{
					To: []networkingv1.NetworkPolicyPeer{{NamespaceSelector: &metav1.LabelSelector{MatchLabels: map[string]string{"kubernetes.io/metadata.name": "kube-system"}}}},
					Ports: []networkingv1.NetworkPolicyPort{
						{Protocol: protocolPtr(corev1.ProtocolUDP), Port: intOrStringPtr(53)},
						{Protocol: protocolPtr(corev1.ProtocolTCP), Port: intOrStringPtr(53)},
					},
				},
				{
					To: []networkingv1.NetworkPolicyPeer{{NamespaceSelector: &metav1.LabelSelector{MatchLabels: map[string]string{"kubernetes.io/metadata.name": postgresqlProfile.Spec.Shared.ClusterRef.Namespace}}}},
					Ports: []networkingv1.NetworkPolicyPort{{Protocol: protocolPtr(corev1.ProtocolTCP), Port: intOrStringPtr(5432)}},
				},
			},
		}
		return nil
	})
	return err
}

func (r *TenantBundleReconciler) ensureAIGatewayMigrations(
	ctx context.Context,
	bundle *fabricv1alpha1.TenantBundle,
	gatewayProfile *fabricv1alpha1.AIGatewayProfile,
	postgresqlProfile *fabricv1alpha1.PostgreSQLProfile,
	databaseSecret *corev1.Secret,
) (bool, string, error) {
	if err := r.ensureAIGatewayMigrationNetworkPolicy(ctx, bundle, postgresqlProfile); err != nil {
		return false, "", err
	}

	namespace := tenantNamespace(bundle.Name)
	name := aiGatewayMigrationName(gatewayProfile, databaseSecret)
	key := types.NamespacedName{Namespace: namespace, Name: name}
	var existing batchv1.Job
	if err := r.Get(ctx, key, &existing); err == nil {
		if !aiGatewayRuntimeOwnedBy(&existing, bundle) {
			return false, "migration Job exists but is not owned by this tenant AI gateway", nil
		}
		if existing.Status.Failed > 0 {
			return false, "LiteLLM schema migration Job failed", nil
		}
		if existing.Status.Succeeded > 0 {
			_ = r.cleanupObsoleteAIGatewayMigrationJobs(ctx, bundle, name)
			return true, "LiteLLM schema migrations are applied", nil
		}
		return false, "waiting for LiteLLM schema migration Job to complete", nil
	} else if !apierrors.IsNotFound(err) {
		return false, "", err
	}

	backoffLimit := int32(1)
	runAsNonRoot := true
	runAsUser := int64(65534)
	runAsGroup := int64(65534)
	allowPrivilegeEscalation := false
	labels := mergeStringMap(aiGatewayRuntimeLabels(bundle), aiGatewayMigrationPodSelector(bundle))
	labels["fabric.truxonline.io/ai-gateway-resource"] = "migration"

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace, Labels: labels},
		Spec: batchv1.JobSpec{
			BackoffLimit: &backoffLimit,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: labels},
				Spec: corev1.PodSpec{
					AutomountServiceAccountToken: boolPtr(false),
					RestartPolicy:                 corev1.RestartPolicyNever,
					PriorityClassName:             defaultString(gatewayProfile.Spec.PriorityClassName, "vixens-medium"),
					SecurityContext: &corev1.PodSecurityContext{
						RunAsNonRoot: &runAsNonRoot,
						RunAsUser:    &runAsUser,
						RunAsGroup:   &runAsGroup,
					},
					Containers: []corev1.Container{{
						Name:            "prisma-migrations",
						Image:           gatewayProfile.Spec.Image,
						ImagePullPolicy: corev1.PullIfNotPresent,
						Command:         []string{"/bin/sh", "-ec"},
						Args: []string{`
export DATABASE_URL="$(python -c 'import os, urllib.parse; u=urllib.parse.quote(os.environ["DB_USERNAME"], safe=""); p=urllib.parse.quote(os.environ["DB_PASSWORD"], safe=""); print(f"postgresql://{u}:{p}@{os.environ[\\"DB_HOST\\"]}:{os.environ.get(\\"DB_PORT\\", \\"5432\\")}/{os.environ[\\"DB_NAME\\"]}")')"
exec python litellm/proxy/prisma_migration.py
`},
						Env: []corev1.EnvVar{
							{Name: "DB_USERNAME", ValueFrom: secretEnvSource(tenantAIGatewayRuntimeSecretName, "DB_USERNAME")},
							{Name: "DB_PASSWORD", ValueFrom: secretEnvSource(tenantAIGatewayRuntimeSecretName, "DB_PASSWORD")},
							{Name: "DB_HOST", ValueFrom: secretEnvSource(tenantAIGatewayRuntimeSecretName, "DB_HOST")},
							{Name: "DB_PORT", ValueFrom: secretEnvSource(tenantAIGatewayRuntimeSecretName, "DB_PORT")},
							{Name: "DB_NAME", ValueFrom: secretEnvSource(tenantAIGatewayRuntimeSecretName, "DB_NAME")},
							{Name: "DISABLE_SCHEMA_UPDATE", Value: "false"},
							{Name: "ENFORCE_PRISMA_MIGRATION_CHECK", Value: "true"},
						},
						Resources: gatewayProfile.Spec.Resources,
						SecurityContext: &corev1.SecurityContext{
							RunAsNonRoot:             &runAsNonRoot,
							RunAsUser:                &runAsUser,
							RunAsGroup:               &runAsGroup,
							AllowPrivilegeEscalation: &allowPrivilegeEscalation,
							Capabilities:             &corev1.Capabilities{Drop: []corev1.Capability{"ALL"}},
						},
					}},
				},
			},
		},
	}
	if err := controllerutil.SetControllerReference(bundle, job, r.Scheme); err != nil {
		return false, "", err
	}
	if err := r.Create(ctx, job); err != nil {
		return false, "", err
	}
	return false, "LiteLLM schema migration Job created", nil
}

func secretEnvSource(secretName, key string) *corev1.EnvVarSource {
	return &corev1.EnvVarSource{SecretKeyRef: &corev1.SecretKeySelector{
		LocalObjectReference: corev1.LocalObjectReference{Name: secretName},
		Key:                  key,
	}}
}

func aiGatewayRuntimeOwnedBy(obj metav1.Object, bundle *fabricv1alpha1.TenantBundle) bool {
	labels := obj.GetLabels()
	return labels[LabelManaged] == "true" && labels[LabelTenantID] == bundle.Spec.TenantID && labels[LabelTenantName] == bundle.Name
}

func (r *TenantBundleReconciler) cleanupObsoleteAIGatewayMigrationJobs(ctx context.Context, bundle *fabricv1alpha1.TenantBundle, keepName string) error {
	var jobs batchv1.JobList
	if err := r.List(ctx, &jobs, client.InNamespace(tenantNamespace(bundle.Name)), client.MatchingLabels{
		LabelManaged:    "true",
		LabelTenantID:   bundle.Spec.TenantID,
		LabelTenantName: bundle.Name,
		"fabric.truxonline.io/ai-gateway-resource": "migration",
	}); err != nil {
		return err
	}
	for i := range jobs.Items {
		job := &jobs.Items[i]
		if job.Name == keepName {
			continue
		}
		if err := r.Delete(ctx, job); err != nil && !apierrors.IsNotFound(err) {
			return err
		}
	}
	return nil
}

func (r *TenantBundleReconciler) cleanupAIGatewayPostgreSQL(ctx context.Context, bundle *fabricv1alpha1.TenantBundle) (bool, error) {
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
		if item.GetLabels()[labelAIGatewayPostgreSQLResource] != "database" {
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
		if item.GetLabels()[labelAIGatewayPostgreSQLResource] != "role" {
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
