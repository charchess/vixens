package controller

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"time"

	fabricv1alpha1 "github.com/charchess/vixens/apps/60-services/txo-fabric/operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	labelHindsightProfile       = "fabric.truxonline.io/hindsight-profile"
	labelHindsightResource      = "fabric.truxonline.io/hindsight-resource"
	hindsightControlPlanePort   = int32(3000)
	hindsightControlPlanePrefix = "ghcr.io/vectorize-io/hindsight-control-plane:"
)

type hindsightNames struct {
	Deployment                string
	Service                   string
	Secret                    string
	NetworkPolicy             string
	Endpoint                  string
	ControlPlaneDeployment    string
	ControlPlaneService       string
	ControlPlaneNetworkPolicy string
	ControlPlaneIngress       string
	ControlPlaneAuthIngress   string
	AuthentikExternalService  string
}

type hindsightResult struct {
	Ready        bool
	Status       *fabricv1alpha1.ComponentStatus
	Reason       string
	Message      string
	RequeueAfter time.Duration
}

type hindsightHumanAccessResolution struct {
	Enabled      bool
	Host         string
	PublicURL    string
	IngressClass string
	TLSIssuer    string
	PublicDNS    bool
	DNSTarget    string
}

func (r *TenantBundleReconciler) reconcileHindsight(ctx context.Context, bundle *fabricv1alpha1.TenantBundle) (hindsightResult, error) {
	request := bundle.Spec.Memory.Hindsight
	if request == nil {
		return hindsightResult{Ready: true}, nil
	}

	profileRef := strings.TrimSpace(request.ProfileRef)
	if profileRef == "" {
		return hindsightBlocked("ProfileRefRequired", "Hindsight profileRef is required", 0), nil
	}

	var profile fabricv1alpha1.HindsightProfile
	if err := r.Get(ctx, client.ObjectKey{Name: profileRef}, &profile); err != nil {
		if apierrors.IsNotFound(err) {
			return hindsightBlocked("ProfileNotFound", fmt.Sprintf("HindsightProfile %q does not exist", profileRef), 30*time.Second), nil
		}
		return hindsightResult{}, err
	}
	if profile.Spec.Topology != "" && profile.Spec.Topology != "TenantScoped" {
		return hindsightBlocked("UnsupportedTopology", fmt.Sprintf("Hindsight topology %q is not implemented; only TenantScoped is supported", profile.Spec.Topology), 0), nil
	}
	if profile.Spec.APIAuthMode != "" && profile.Spec.APIAuthMode != "ApiKey" {
		return hindsightBlocked("UnsupportedAPIAuthMode", fmt.Sprintf("Hindsight API auth mode %q is not implemented; only ApiKey is supported", profile.Spec.APIAuthMode), 0), nil
	}
	if mode := defaultString(profile.Spec.LLMAuthMode, "Unconfigured"); mode != "Unconfigured" && mode != "PlatformGateway" {
		return hindsightBlocked("UnsupportedLLMAuthMode", fmt.Sprintf("Hindsight model auth mode %q is not implemented", profile.Spec.LLMAuthMode), 0), nil
	}
	if strings.TrimSpace(profile.Spec.Image) == "" {
		return hindsightBlocked("InvalidProfile", "HindsightProfile image is required", 0), nil
	}

	humanAccess, err := resolveHindsightHumanAccess(bundle)
	if err != nil {
		return hindsightBlocked("InvalidHumanAccess", err.Error(), 0), nil
	}
	if humanAccess.Enabled {
		if _, err := hindsightControlPlaneImage(&profile); err != nil {
			return hindsightBlocked("InvalidControlPlaneImage", err.Error(), 0), nil
		}
	}

	postgresqlRequest := bundle.Spec.Persistence.PostgreSQL
	if postgresqlRequest == nil || strings.TrimSpace(postgresqlRequest.ProfileRef) == "" {
		return hindsightBlocked("PostgreSQLRequired", "tenant-scoped Hindsight requires a PostgreSQL capability with profileRef", 0), nil
	}
	if bundle.Status.Persistence.PostgreSQL == nil || bundle.Status.Persistence.PostgreSQL.Phase != "Ready" {
		return hindsightPending("PostgreSQLPending", "waiting for the tenant PostgreSQL binding to become ready", resolveHindsightNames(bundle, &profile), 5*time.Second), nil
	}

	var postgresqlProfile fabricv1alpha1.PostgreSQLProfile
	if err := r.Get(ctx, client.ObjectKey{Name: strings.TrimSpace(postgresqlRequest.ProfileRef)}, &postgresqlProfile); err != nil {
		if apierrors.IsNotFound(err) {
			return hindsightBlocked("PostgreSQLProfileNotFound", fmt.Sprintf("PostgreSQLProfile %q does not exist", postgresqlRequest.ProfileRef), 30*time.Second), nil
		}
		return hindsightResult{}, err
	}
	if postgresqlProfile.Spec.Topology != "SharedCluster" {
		return hindsightBlocked("UnsupportedPostgreSQLTopology", fmt.Sprintf("Hindsight requires the implemented SharedCluster PostgreSQL binding, got %q", postgresqlProfile.Spec.Topology), 0), nil
	}

	postgresqlNames := resolvePostgreSQLNames(bundle, &postgresqlProfile)
	postgresqlSecretKey := client.ObjectKey{Namespace: postgresqlProfile.Spec.Shared.ClusterRef.Namespace, Name: postgresqlNames.Secret}
	var postgresqlSecret corev1.Secret
	if err := r.Get(ctx, postgresqlSecretKey, &postgresqlSecret); err != nil {
		if apierrors.IsNotFound(err) {
			return hindsightPending("PostgreSQLCredentialsPending", fmt.Sprintf("tenant PostgreSQL Secret %s/%s does not exist yet", postgresqlSecretKey.Namespace, postgresqlSecretKey.Name), resolveHindsightNames(bundle, &profile), 5*time.Second), nil
		}
		return hindsightResult{}, err
	}
	if !postgresqlOwnedBy(&postgresqlSecret, bundle) {
		return hindsightBlocked("PostgreSQLOwnershipConflict", fmt.Sprintf("tenant PostgreSQL Secret %s/%s is not owned by this Fabric tenant", postgresqlSecretKey.Namespace, postgresqlSecretKey.Name), 0), nil
	}

	names := resolveHindsightNames(bundle, &profile)
	if blocked, err := r.ensureHindsightSecret(ctx, bundle, &profile, names, &postgresqlSecret); err != nil {
		return hindsightResult{}, err
	} else if blocked != nil {
		return *blocked, nil
	}
	if blocked, err := r.ensureHindsightDeployment(ctx, bundle, &profile, names); err != nil {
		return hindsightResult{}, err
	} else if blocked != nil {
		return *blocked, nil
	}
	if blocked, err := r.ensureHindsightService(ctx, bundle, &profile, names); err != nil {
		return hindsightResult{}, err
	} else if blocked != nil {
		return *blocked, nil
	}
	if blocked, err := r.ensureHindsightNetworkPolicy(ctx, bundle, &profile, &postgresqlProfile, names); err != nil {
		return hindsightResult{}, err
	} else if blocked != nil {
		return *blocked, nil
	}

	if blocked, err := r.ensureHindsightControlPlane(ctx, bundle, &profile, names, humanAccess); err != nil {
		return hindsightResult{}, err
	} else if blocked != nil {
		return *blocked, nil
	}

	var deployment appsv1.Deployment
	if err := r.Get(ctx, types.NamespacedName{Name: names.Deployment, Namespace: tenantNamespace(bundle.Name)}, &deployment); err != nil {
		return hindsightResult{}, err
	}
	if deployment.Status.ObservedGeneration != deployment.Generation || deployment.Status.AvailableReplicas < 1 {
		return hindsightPending("DeploymentProgressing", "waiting for the tenant Hindsight Deployment to become available", names, 5*time.Second), nil
	}
	if humanAccess.Enabled {
		var controlPlane appsv1.Deployment
		if err := r.Get(ctx, types.NamespacedName{Name: names.ControlPlaneDeployment, Namespace: tenantNamespace(bundle.Name)}, &controlPlane); err != nil {
			return hindsightResult{}, err
		}
		if controlPlane.Status.ObservedGeneration != controlPlane.Generation || controlPlane.Status.AvailableReplicas < 1 {
			return hindsightPending("ControlPlaneProgressing", "waiting for the tenant Hindsight WebUI to become available", names, 5*time.Second), nil
		}
	}

	status := hindsightComponentStatus("Ready", names)
	if humanAccess.Enabled {
		status.HumanEndpoint = humanAccess.PublicURL
	}
	message := "tenant-scoped Hindsight API is available with secret-backed PostgreSQL and API-key authentication"
	if humanAccess.Enabled {
		message += fmt.Sprintf("; authenticated WebUI=%s", humanAccess.PublicURL)
	}
	if hindsightUsesPlatformGateway(&profile) {
		var runtimeSecret corev1.Secret
		reader := r.APIReader
		if reader == nil {
			reader = r.Client
		}
		if err := reader.Get(ctx, client.ObjectKey{Namespace: tenantNamespace(bundle.Name), Name: names.Secret}, &runtimeSecret); err != nil {
			return hindsightResult{}, err
		}
		rotation := runtimeSecret.Annotations[AnnotationHindsightEmbeddingRevision]
		if rotation == "" {
			rotation = "baseline"
		}
		message += fmt.Sprintf("; embeddings use scoped TXO AI gateway access (gateway=%s model=%s rotation=%s)", aiGatewayURL(), defaultAIEmbeddingModel, rotation)
	}
	return hindsightResult{
		Ready:   true,
		Status:  status,
		Reason:  "Reconciled",
		Message: message,
	}, nil
}

func (r *TenantBundleReconciler) cleanupHindsight(ctx context.Context, bundle *fabricv1alpha1.TenantBundle) (bool, error) {
	namespace := tenantNamespace(bundle.Name)
	pending := false

	objects := []client.Object{
		&appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "hindsight", Namespace: namespace}},
		&corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: "hindsight", Namespace: namespace}},
		&networkingv1.NetworkPolicy{ObjectMeta: metav1.ObjectMeta{Name: "hindsight-access", Namespace: namespace}},
		&appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "hindsight-control-plane", Namespace: namespace}},
		&corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: "hindsight-control-plane", Namespace: namespace}},
		&networkingv1.NetworkPolicy{ObjectMeta: metav1.ObjectMeta{Name: "hindsight-control-plane-access", Namespace: namespace}},
		&networkingv1.Ingress{ObjectMeta: metav1.ObjectMeta{Name: "hindsight-control-plane", Namespace: namespace}},
		&networkingv1.Ingress{ObjectMeta: metav1.ObjectMeta{Name: "hindsight-control-plane-auth", Namespace: namespace}},
		&corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: "hindsight-authentik", Namespace: namespace}},
		&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "hindsight-runtime", Namespace: namespace}},
	}
	for _, object := range objects {
		if err := r.Get(ctx, client.ObjectKeyFromObject(object), object); err != nil {
			if apierrors.IsNotFound(err) {
				continue
			}
			return false, err
		}
		if !hindsightOwnedBy(object, bundle) {
			continue
		}
		pending = true
		if object.GetDeletionTimestamp() == nil {
			if err := r.Delete(ctx, object); err != nil && !apierrors.IsNotFound(err) {
				return false, err
			}
		}
	}
	if !pending {
		// Revocation is intentionally best-effort: gateway unavailability must not
		// wedge TenantBundle deletion. Revoke both the historical migration source
		// and the canonical tenant-local backend when they are reachable.
		alias := hindsightEmbeddingKeyAlias(bundle)
		_ = revokeModelAccessKeyIfExistsWithBackend(ctx, sharedModelAccessBackend(), alias)
		if backend, err := tenantModelAccessBackend(ctx, r.Client, bundle, defaultAIEmbeddingModel); err == nil {
			_ = revokeModelAccessKeyIfExistsWithBackend(ctx, backend, alias)
		}
	}
	return pending, nil
}

func normalizedHindsightEmbeddingBackend(secret *corev1.Secret) string {
	if secret == nil {
		return sharedModelAccessBackendID
	}
	backendID := strings.TrimSpace(secret.Annotations[AnnotationHindsightEmbeddingBackend])
	if backendID == "" {
		return sharedModelAccessBackendID
	}
	return backendID
}

func (r *TenantBundleReconciler) tenantOpenRouterCredentialAvailable(ctx context.Context, bundle *fabricv1alpha1.TenantBundle) (bool, error) {
	var secret corev1.Secret
	if err := r.Get(ctx, client.ObjectKey{
		Namespace: tenantAIProviderSecretNamespace,
		Name:      tenantAIProviderSecretName(bundle.Name),
	}, &secret); err != nil {
		if apierrors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return len(secret.Data[tenantAIOpenRouterSecretKey]) > 0, nil
}

func (r *TenantBundleReconciler) tenantEmbeddingRouteReady(ctx context.Context, bundle *fabricv1alpha1.TenantBundle) (bool, error) {
	namespace := tenantNamespace(bundle.Name)
	var config corev1.ConfigMap
	if err := r.Get(ctx, client.ObjectKey{Namespace: namespace, Name: tenantAIGatewayConfigMapName}, &config); err != nil {
		if apierrors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	if !strings.Contains(config.Data["config.yaml"], "model_name: "+defaultAIEmbeddingModel) {
		return false, nil
	}
	configHash := strings.TrimSpace(config.Annotations["fabric.truxonline.io/config-hash"])
	if configHash == "" {
		return false, nil
	}

	var deployment appsv1.Deployment
	if err := r.Get(ctx, client.ObjectKey{Namespace: namespace, Name: tenantAIGatewayName}, &deployment); err != nil {
		if apierrors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	if deployment.Spec.Template.Annotations["fabric.truxonline.io/config-hash"] != configHash {
		return false, nil
	}
	return deployment.Status.ObservedGeneration == deployment.Generation && deployment.Status.AvailableReplicas > 0, nil
}

func (r *TenantBundleReconciler) resolveHindsightEmbeddingBackend(
	ctx context.Context,
	bundle *fabricv1alpha1.TenantBundle,
	appliedBackendID string,
) (modelAccessBackend, string, error) {
	appliedBackendID = strings.TrimSpace(appliedBackendID)
	if appliedBackendID == "" {
		appliedBackendID = sharedModelAccessBackendID
	}
	alreadyTenant := strings.HasPrefix(appliedBackendID, "tenant:"+bundle.Name+":")
	if appliedBackendID != sharedModelAccessBackendID && !alreadyTenant {
		return modelAccessBackend{}, fmt.Sprintf("Hindsight embedding backend %q does not belong to tenant %q", appliedBackendID, bundle.Name), nil
	}

	hasProviderCredential, err := r.tenantOpenRouterCredentialAvailable(ctx, bundle)
	if err != nil {
		return modelAccessBackend{}, "", err
	}
	if !hasProviderCredential {
		if alreadyTenant {
			return modelAccessBackend{}, "tenant OpenRouter credential is unavailable; refusing fallback to the historical shared gateway", nil
		}
		return sharedModelAccessBackend(), "", nil
	}

	tenantBackend, err := tenantModelAccessBackend(ctx, r.Client, bundle, defaultAIEmbeddingModel)
	if err != nil {
		if alreadyTenant {
			return modelAccessBackend{}, "tenant LiteLLM embedding backend is unavailable: " + err.Error(), nil
		}
		return sharedModelAccessBackend(), "", nil
	}
	routeReady, err := r.tenantEmbeddingRouteReady(ctx, bundle)
	if err != nil {
		return modelAccessBackend{}, "", err
	}
	if !routeReady {
		if alreadyTenant {
			return modelAccessBackend{}, "waiting for the tenant LiteLLM txo-embedding route to become available", nil
		}
		// Existing tenants remain on the known-good shared embedding route until
		// the destination tenant LiteLLM route is fully available.
		return sharedModelAccessBackend(), "", nil
	}
	return tenantBackend, "", nil
}

func hindsightDeploymentAdoptedEmbedding(deployment *appsv1.Deployment, secret *corev1.Secret) bool {
	if deployment == nil || secret == nil {
		return false
	}
	revision := strings.TrimSpace(secret.Annotations[AnnotationHindsightEmbeddingRevision])
	if revision == "" {
		return false
	}
	if deployment.Spec.Template.Annotations[AnnotationHindsightEmbeddingRevision] != revision {
		return false
	}
	if deployment.Spec.Template.Annotations[AnnotationHindsightEmbeddingSecretUID] != string(secret.UID) {
		return false
	}
	return deployment.Status.ObservedGeneration == deployment.Generation && deployment.Status.AvailableReplicas > 0
}

func hindsightEmbeddingRevision(backend modelAccessBackend, requestedRotation, appliedRevision string, backendChanged bool) string {
	if requestedRotation != "" {
		return modelAccessBackendRevision(backend.ID, requestedRotation)
	}
	if backendChanged && backend.ID != sharedModelAccessBackendID {
		return modelAccessBackendRevision(backend.ID, "")
	}
	return appliedRevision
}

func desiredHindsightSecretData(databaseURL, apiKey string, port int32, embeddingKey string, embeddingBaseURL string, gatewayEnabled bool) map[string][]byte {
	data := map[string][]byte{
		"HINDSIGHT_API_DATABASE_URL":       []byte(databaseURL),
		"HINDSIGHT_API_DATABASE_SCHEMA":    []byte("public"),
		"HINDSIGHT_API_VECTOR_EXTENSION":   []byte("pgvector"),
		"HINDSIGHT_API_TENANT_EXTENSION":   []byte("hindsight_api.extensions.builtin.tenant:ApiKeyTenantExtension"),
		"HINDSIGHT_API_TENANT_API_KEY":     []byte(apiKey),
		"HINDSIGHT_API_LLM_PROVIDER":       []byte("none"),
		"HF_HUB_OFFLINE":                   []byte("1"),
		"TRANSFORMERS_OFFLINE":             []byte("1"),
		"HINDSIGHT_API_HOST":               []byte("0.0.0.0"),
		"HINDSIGHT_API_PORT":               []byte(strconv.Itoa(int(port))),
		"HINDSIGHT_API_LOG_LEVEL":          []byte("info"),
		"HINDSIGHT_ENABLE_API":             []byte("true"),
		"HINDSIGHT_ENABLE_CP":              []byte("false"),
	}
	if gatewayEnabled {
		data["HINDSIGHT_API_EMBEDDINGS_PROVIDER"] = []byte("openai")
		data["HINDSIGHT_API_EMBEDDINGS_OPENAI_BASE_URL"] = []byte(embeddingBaseURL)
		data["HINDSIGHT_API_EMBEDDINGS_OPENAI_MODEL"] = []byte(defaultAIEmbeddingModel)
		data["HINDSIGHT_API_EMBEDDINGS_OPENAI_DIMENSIONS"] = []byte(defaultAIEmbeddingDimension)
		data[hindsightEmbeddingSecretKey] = []byte(embeddingKey)
	}
	return data
}

func (r *TenantBundleReconciler) ensureHindsightSecret(ctx context.Context, bundle *fabricv1alpha1.TenantBundle, profile *fabricv1alpha1.HindsightProfile, names hindsightNames, postgresqlSecret *corev1.Secret) (*hindsightResult, error) {
	username := strings.TrimSpace(string(postgresqlSecret.Data[corev1.BasicAuthUsernameKey]))
	password := string(postgresqlSecret.Data[corev1.BasicAuthPasswordKey])
	host := strings.TrimSpace(string(postgresqlSecret.Data["host"]))
	port := strings.TrimSpace(string(postgresqlSecret.Data["port"]))
	database := strings.TrimSpace(string(postgresqlSecret.Data["dbname"]))
	if username == "" || password == "" || host == "" || database == "" {
		blocked := hindsightBlocked("InvalidPostgreSQLCredentials", "tenant PostgreSQL Secret is missing username, password, host, or dbname", 0)
		return &blocked, nil
	}
	if port == "" {
		port = "5432"
	}
	databaseURL := (&url.URL{
		Scheme: "postgresql",
		User:   url.UserPassword(username, password),
		Host:   net.JoinHostPort(host, port),
		Path:   "/" + database,
	}).String()

	gatewayEnabled := hindsightUsesPlatformGateway(profile)
	alias := hindsightEmbeddingKeyAlias(bundle)
	requestedRotation := hindsightEmbeddingRotationRevision(bundle)
	namespace := tenantNamespace(bundle.Name)
	key := client.ObjectKey{Namespace: namespace, Name: names.Secret}

	var secret corev1.Secret
	secretExists := true
	if err := r.Get(ctx, key, &secret); err != nil {
		if !apierrors.IsNotFound(err) {
			return nil, err
		}
		secretExists = false
	}
	if secretExists && !hindsightOwnedBy(&secret, bundle) {
		blocked := hindsightBlocked("OwnershipConflict", fmt.Sprintf("Secret %s/%s exists but is not owned by this Fabric tenant", namespace, names.Secret), 0)
		return &blocked, nil
	}

	var existingSecret *corev1.Secret
	if secretExists {
		existingSecret = &secret
	}
	appliedBackendID := normalizedHindsightEmbeddingBackend(existingSecret)

	if !gatewayEnabled {
		apiKey := ""
		if secretExists {
			apiKey = string(secret.Data["HINDSIGHT_API_TENANT_API_KEY"])
		}
		if apiKey == "" {
			var err error
			apiKey, err = randomPassword()
			if err != nil {
				return nil, err
			}
		}
		if secretExists && string(secret.Data[hindsightEmbeddingSecretKey]) != "" {
			backend := sharedModelAccessBackend()
			if strings.HasPrefix(appliedBackendID, "tenant:"+bundle.Name+":") {
				if resolved, err := tenantModelAccessBackend(ctx, r.Client, bundle, defaultAIEmbeddingModel); err == nil {
					backend = resolved
				}
			}
			if err := revokeModelAccessKeyIfExistsWithBackend(ctx, backend, alias); err != nil {
				return nil, fmt.Errorf("revoke disabled Hindsight embedding key alias %q: %w", alias, err)
			}
		}
		desired := desiredHindsightSecretData(databaseURL, apiKey, hindsightAPIPort(profile), "", "", false)
		if !secretExists {
			secret = corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Name: names.Secret, Namespace: namespace, Labels: hindsightLabels(bundle, profile, "runtime-secret")},
				Type:       corev1.SecretTypeOpaque,
				Data:       desired,
			}
			if err := r.Create(ctx, &secret); err != nil {
				return nil, err
			}
			return nil, nil
		}
		secret.Type = corev1.SecretTypeOpaque
		secret.Labels = mergeStringMap(copyStringMap(secret.Labels), hindsightLabels(bundle, profile, "runtime-secret"))
		secret.Annotations = copyStringMap(secret.Annotations)
		delete(secret.Annotations, AnnotationHindsightEmbeddingRevision)
		delete(secret.Annotations, AnnotationHindsightEmbeddingBackend)
		delete(secret.Annotations, AnnotationHindsightEmbeddingBackendURL)
		delete(secret.Annotations, AnnotationHindsightLegacyCleanupPending)
		secret.Data = desired
		if err := r.Update(ctx, &secret); err != nil {
			return nil, err
		}
		return nil, nil
	}

	backend, pendingMessage, err := r.resolveHindsightEmbeddingBackend(ctx, bundle, appliedBackendID)
	if err != nil {
		return nil, err
	}
	if pendingMessage != "" {
		pending := hindsightPending("EmbeddingBackendPending", pendingMessage, names, 5*time.Second)
		return &pending, nil
	}

	apiKey := ""
	if secretExists {
		apiKey = string(secret.Data["HINDSIGHT_API_TENANT_API_KEY"])
	}
	if apiKey == "" {
		apiKey, err = randomPassword()
		if err != nil {
			return nil, err
		}
	}

	if !secretExists {
		if err := revokeModelAccessKeyIfExistsWithBackend(ctx, backend, alias); err != nil {
			return nil, fmt.Errorf("revoke stale Hindsight embedding key alias %q on backend %q: %w", alias, backend.ID, err)
		}
		embeddingKey, err := generateHindsightEmbeddingAccessKeyWithBackend(ctx, backend, bundle)
		if err != nil {
			return nil, fmt.Errorf("provision Hindsight embedding gateway credential on backend %q: %w", backend.ID, err)
		}
		revision := hindsightEmbeddingRevision(backend, requestedRotation, "", backend.ID != sharedModelAccessBackendID)
		annotations := map[string]string{
			AnnotationHindsightEmbeddingBackend:    backend.ID,
			AnnotationHindsightEmbeddingBackendURL: backend.URL,
		}
		if revision != "" {
			annotations[AnnotationHindsightEmbeddingRevision] = revision
		}
		secret = corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:        names.Secret,
				Namespace:   namespace,
				Labels:      hindsightLabels(bundle, profile, "runtime-secret"),
				Annotations: annotations,
			},
			Type: corev1.SecretTypeOpaque,
			Data: desiredHindsightSecretData(
				databaseURL, apiKey, hindsightAPIPort(profile), embeddingKey,
				backend.runtimeBinding().BaseURL, true,
			),
		}
		if err := r.Create(ctx, &secret); err != nil {
			_ = revokeModelAccessKeyValueIfExistsWithBackend(ctx, backend, embeddingKey)
			return nil, err
		}
		return nil, nil
	}

	// A shared -> tenant cutover is staged: destination credential first, runtime
	// adoption second, source revocation last. Never migrate back to shared.
	backendChanged := appliedBackendID != backend.ID
	if backendChanged {
		if appliedBackendID != sharedModelAccessBackendID || !strings.HasPrefix(backend.ID, "tenant:"+bundle.Name+":") {
			blocked := hindsightBlocked("EmbeddingBackendTransitionRefused", fmt.Sprintf("refusing Hindsight embedding backend transition %q -> %q", appliedBackendID, backend.ID), 0)
			return &blocked, nil
		}
		if err := revokeModelAccessKeyIfExistsWithBackend(ctx, backend, alias); err != nil {
			return nil, fmt.Errorf("cleanup destination Hindsight embedding alias %q on backend %q: %w", alias, backend.ID, err)
		}
		replacementKey, err := generateHindsightEmbeddingAccessKeyWithBackend(ctx, backend, bundle)
		if err != nil {
			return nil, fmt.Errorf("prepare destination Hindsight embedding credential on backend %q: %w", backend.ID, err)
		}
		revision := hindsightEmbeddingRevision(backend, requestedRotation, secret.Annotations[AnnotationHindsightEmbeddingRevision], true)
		desiredAnnotations := copyStringMap(secret.Annotations)
		desiredAnnotations[AnnotationHindsightEmbeddingBackend] = backend.ID
		desiredAnnotations[AnnotationHindsightEmbeddingBackendURL] = backend.URL
		desiredAnnotations[AnnotationHindsightEmbeddingRevision] = revision
		desiredAnnotations[AnnotationHindsightLegacyCleanupPending] = sharedModelAccessBackendID
		secret.Type = corev1.SecretTypeOpaque
		secret.Labels = mergeStringMap(copyStringMap(secret.Labels), hindsightLabels(bundle, profile, "runtime-secret"))
		secret.Annotations = desiredAnnotations
		secret.Data = desiredHindsightSecretData(
			databaseURL, apiKey, hindsightAPIPort(profile), replacementKey,
			backend.runtimeBinding().BaseURL, true,
		)
		if err := r.Update(ctx, &secret); err != nil {
			_ = revokeModelAccessKeyValueIfExistsWithBackend(ctx, backend, replacementKey)
			return nil, err
		}
		return nil, nil
	}

	if secret.Annotations[AnnotationHindsightLegacyCleanupPending] == sharedModelAccessBackendID {
		var deployment appsv1.Deployment
		if err := r.Get(ctx, client.ObjectKey{Namespace: namespace, Name: names.Deployment}, &deployment); err != nil {
			if !apierrors.IsNotFound(err) {
				return nil, err
			}
			return nil, nil
		}
		if !hindsightDeploymentAdoptedEmbedding(&deployment, &secret) {
			return nil, nil
		}
		if err := revokeModelAccessKeyIfExistsWithBackend(ctx, sharedModelAccessBackend(), alias); err != nil {
			return nil, fmt.Errorf("finalize Hindsight shared embedding credential cleanup: %w", err)
		}
		secret.Annotations = copyStringMap(secret.Annotations)
		delete(secret.Annotations, AnnotationHindsightLegacyCleanupPending)
		if err := r.Update(ctx, &secret); err != nil {
			return nil, err
		}
	}

	appliedRevision := strings.TrimSpace(secret.Annotations[AnnotationHindsightEmbeddingRevision])
	wantedRevision := hindsightEmbeddingRevision(backend, requestedRotation, appliedRevision, false)
	embeddingKey := string(secret.Data[hindsightEmbeddingSecretKey])
	generatedEmbeddingKey := false
	if embeddingKey == "" || (requestedRotation != "" && appliedRevision != wantedRevision) {
		if err := revokeModelAccessKeyIfExistsWithBackend(ctx, backend, alias); err != nil {
			return nil, fmt.Errorf("revoke previous Hindsight embedding key alias %q on backend %q: %w", alias, backend.ID, err)
		}
		embeddingKey, err = generateHindsightEmbeddingAccessKeyWithBackend(ctx, backend, bundle)
		if err != nil {
			return nil, fmt.Errorf("provision Hindsight embedding gateway credential on backend %q: %w", backend.ID, err)
		}
		generatedEmbeddingKey = true
		appliedRevision = wantedRevision
	}

	desiredAnnotations := copyStringMap(secret.Annotations)
	desiredAnnotations[AnnotationHindsightEmbeddingBackend] = backend.ID
	desiredAnnotations[AnnotationHindsightEmbeddingBackendURL] = backend.URL
	if appliedRevision != "" {
		desiredAnnotations[AnnotationHindsightEmbeddingRevision] = appliedRevision
	}
	desiredData := desiredHindsightSecretData(
		databaseURL, apiKey, hindsightAPIPort(profile), embeddingKey,
		backend.runtimeBinding().BaseURL, true,
	)
	desiredLabels := mergeStringMap(copyStringMap(secret.Labels), hindsightLabels(bundle, profile, "runtime-secret"))
	if secret.Type != corev1.SecretTypeOpaque || !reflect.DeepEqual(secret.Labels, desiredLabels) || !reflect.DeepEqual(secret.Annotations, desiredAnnotations) || !reflect.DeepEqual(secret.Data, desiredData) {
		secret.Type = corev1.SecretTypeOpaque
		secret.Labels = desiredLabels
		secret.Annotations = desiredAnnotations
		secret.Data = desiredData
		if err := r.Update(ctx, &secret); err != nil {
			if generatedEmbeddingKey {
				_ = revokeModelAccessKeyValueIfExistsWithBackend(ctx, backend, embeddingKey)
			}
			return nil, err
		}
	}
	return nil, nil
}

func desiredHindsightEnv(bundle *fabricv1alpha1.TenantBundle, names hindsightNames) []corev1.EnvVar {
	return []corev1.EnvVar{{
		Name:  "HINDSIGHT_API_WORKER_ID",
		Value: bundle.Name + "-" + names.Deployment,
	}}
}

func (r *TenantBundleReconciler) ensureHindsightDeployment(ctx context.Context, bundle *fabricv1alpha1.TenantBundle, profile *fabricv1alpha1.HindsightProfile, names hindsightNames) (*hindsightResult, error) {
	namespace := tenantNamespace(bundle.Name)
	key := client.ObjectKey{Namespace: namespace, Name: names.Deployment}
	var existing appsv1.Deployment
	if err := r.Get(ctx, key, &existing); err == nil {
		if !hindsightOwnedBy(&existing, bundle) {
			blocked := hindsightBlocked("OwnershipConflict", fmt.Sprintf("Deployment %s/%s exists but is not owned by this Fabric tenant", namespace, names.Deployment), 0)
			return &blocked, nil
		}
	} else if !apierrors.IsNotFound(err) {
		return nil, err
	}

	deployment := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: names.Deployment, Namespace: namespace}}
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, deployment, func() error {
		labels := hindsightLabels(bundle, profile, "deployment")
		deployment.Labels = mergeStringMap(deployment.Labels, labels)
		replicas := int32(1)
		revisionHistory := int32(2)
		deployment.Spec.Replicas = &replicas
		deployment.Spec.RevisionHistoryLimit = &revisionHistory
		deployment.Spec.Selector = &metav1.LabelSelector{MatchLabels: hindsightPodSelector(bundle)}

		podLabels := mergeStringMap(copyStringMap(labels), hindsightPodSelector(bundle))
		podLabels["vixens.io/sizing.hindsight"] = defaultString(profile.Spec.SizingLabel, "V-small")
		podAnnotations := map[string]string{}
		if hindsightUsesPlatformGateway(profile) {
			var runtimeSecret corev1.Secret
			reader := r.APIReader
			if reader == nil {
				reader = r.Client
			}
			if err := reader.Get(ctx, client.ObjectKey{Namespace: namespace, Name: names.Secret}, &runtimeSecret); err != nil {
				return err
			}
			podAnnotations[AnnotationHindsightEmbeddingRevision] = runtimeSecret.Annotations[AnnotationHindsightEmbeddingRevision]
			podAnnotations[AnnotationHindsightEmbeddingSecretUID] = string(runtimeSecret.UID)
		}
		runAsNonRoot := true
		runAsUser := int64(1000)
		allowPrivilegeEscalation := false
		port := hindsightAPIPort(profile)
		probeURL := func(path string) []string {
			return []string{"python", "-c", fmt.Sprintf("import urllib.request; urllib.request.urlopen('http://127.0.0.1:%d%s', timeout=3).read()", port, path)}
		}
		deployment.Spec.Template = corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{Labels: podLabels, Annotations: podAnnotations},
			Spec: corev1.PodSpec{
				PriorityClassName: defaultString(profile.Spec.PriorityClassName, "vixens-medium"),
				SecurityContext:   &corev1.PodSecurityContext{FSGroup: int64Ptr(1000)},
				Containers: []corev1.Container{{
					Name:            "hindsight",
					Image:           profile.Spec.Image,
					ImagePullPolicy: corev1.PullIfNotPresent,
					Env:             desiredHindsightEnv(bundle, names),
					EnvFrom:         []corev1.EnvFromSource{{SecretRef: &corev1.SecretEnvSource{LocalObjectReference: corev1.LocalObjectReference{Name: names.Secret}}}},
					Ports:           []corev1.ContainerPort{{Name: "http", ContainerPort: port, Protocol: corev1.ProtocolTCP}},
					Resources:       profile.Spec.Resources,
					SecurityContext: &corev1.SecurityContext{
						RunAsNonRoot:             &runAsNonRoot,
						RunAsUser:                &runAsUser,
						AllowPrivilegeEscalation: &allowPrivilegeEscalation,
						Capabilities:             &corev1.Capabilities{Drop: []corev1.Capability{"ALL"}},
					},
					StartupProbe:   &corev1.Probe{ProbeHandler: corev1.ProbeHandler{Exec: &corev1.ExecAction{Command: probeURL("/health/live")}}, PeriodSeconds: 5, TimeoutSeconds: 3, FailureThreshold: 60},
					ReadinessProbe: &corev1.Probe{ProbeHandler: corev1.ProbeHandler{Exec: &corev1.ExecAction{Command: probeURL("/health")}}, InitialDelaySeconds: 5, PeriodSeconds: 5, TimeoutSeconds: 3, FailureThreshold: 3},
					LivenessProbe:  &corev1.Probe{ProbeHandler: corev1.ProbeHandler{Exec: &corev1.ExecAction{Command: probeURL("/health/live")}}, InitialDelaySeconds: 30, PeriodSeconds: 10, TimeoutSeconds: 3, FailureThreshold: 3},
				}},
			},
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func (r *TenantBundleReconciler) ensureHindsightService(ctx context.Context, bundle *fabricv1alpha1.TenantBundle, profile *fabricv1alpha1.HindsightProfile, names hindsightNames) (*hindsightResult, error) {
	namespace := tenantNamespace(bundle.Name)
	key := client.ObjectKey{Namespace: namespace, Name: names.Service}
	var existing corev1.Service
	if err := r.Get(ctx, key, &existing); err == nil {
		if !hindsightOwnedBy(&existing, bundle) {
			blocked := hindsightBlocked("OwnershipConflict", fmt.Sprintf("Service %s/%s exists but is not owned by this Fabric tenant", namespace, names.Service), 0)
			return &blocked, nil
		}
	} else if !apierrors.IsNotFound(err) {
		return nil, err
	}

	service := &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: names.Service, Namespace: namespace}}
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, service, func() error {
		service.Labels = mergeStringMap(service.Labels, hindsightLabels(bundle, profile, "service"))
		service.Spec.Type = corev1.ServiceTypeClusterIP
		service.Spec.Selector = hindsightPodSelector(bundle)
		port := hindsightAPIPort(profile)
		service.Spec.Ports = []corev1.ServicePort{{Name: "http", Port: port, TargetPort: intstr.FromInt32(port), Protocol: corev1.ProtocolTCP}}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func (r *TenantBundleReconciler) ensureHindsightNetworkPolicy(ctx context.Context, bundle *fabricv1alpha1.TenantBundle, profile *fabricv1alpha1.HindsightProfile, postgresqlProfile *fabricv1alpha1.PostgreSQLProfile, names hindsightNames) (*hindsightResult, error) {
	namespace := tenantNamespace(bundle.Name)
	key := client.ObjectKey{Namespace: namespace, Name: names.NetworkPolicy}
	var existing networkingv1.NetworkPolicy
	if err := r.Get(ctx, key, &existing); err == nil {
		if !hindsightOwnedBy(&existing, bundle) {
			blocked := hindsightBlocked("OwnershipConflict", fmt.Sprintf("NetworkPolicy %s/%s exists but is not owned by this Fabric tenant", namespace, names.NetworkPolicy), 0)
			return &blocked, nil
		}
	} else if !apierrors.IsNotFound(err) {
		return nil, err
	}

	policy := &networkingv1.NetworkPolicy{ObjectMeta: metav1.ObjectMeta{Name: names.NetworkPolicy, Namespace: namespace}}
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, policy, func() error {
		policy.Labels = mergeStringMap(policy.Labels, hindsightLabels(bundle, profile, "network-policy"))
		apiPort := int(hindsightAPIPort(profile))
		egress := []networkingv1.NetworkPolicyEgressRule{
			{
				To: []networkingv1.NetworkPolicyPeer{{NamespaceSelector: &metav1.LabelSelector{MatchLabels: map[string]string{"kubernetes.io/metadata.name": "kube-system"}}}},
				Ports: []networkingv1.NetworkPolicyPort{{Protocol: protocolPtr(corev1.ProtocolUDP), Port: intOrStringPtr(53)}, {Protocol: protocolPtr(corev1.ProtocolTCP), Port: intOrStringPtr(53)}},
			},
			{
				To: []networkingv1.NetworkPolicyPeer{{
					NamespaceSelector: &metav1.LabelSelector{MatchLabels: map[string]string{"kubernetes.io/metadata.name": postgresqlProfile.Spec.Shared.ClusterRef.Namespace}},
					PodSelector:       &metav1.LabelSelector{MatchLabels: map[string]string{"cnpg.io/cluster": postgresqlProfile.Spec.Shared.ClusterRef.Name}},
				}},
				Ports: []networkingv1.NetworkPolicyPort{{Protocol: protocolPtr(corev1.ProtocolTCP), Port: intOrStringPtr(5432)}},
			},
		}
		if hindsightUsesPlatformGateway(profile) {
			backendID := sharedModelAccessBackendID
			legacyCleanupPending := false
			var runtimeSecret corev1.Secret
			if err := r.Get(ctx, client.ObjectKey{Namespace: namespace, Name: names.Secret}, &runtimeSecret); err == nil {
				backendID = normalizedHindsightEmbeddingBackend(&runtimeSecret)
				legacyCleanupPending = runtimeSecret.Annotations[AnnotationHindsightLegacyCleanupPending] == sharedModelAccessBackendID
			} else if !apierrors.IsNotFound(err) {
				return err
			}

			if backendID == sharedModelAccessBackendID || legacyCleanupPending {
				egress = append(egress, networkingv1.NetworkPolicyEgressRule{
					To: []networkingv1.NetworkPolicyPeer{{
						NamespaceSelector: &metav1.LabelSelector{MatchLabels: map[string]string{"kubernetes.io/metadata.name": "txo-fabric-system"}},
						PodSelector:       &metav1.LabelSelector{MatchLabels: map[string]string{LabelName: "txo-ai-gateway"}},
					}},
					Ports: []networkingv1.NetworkPolicyPort{{Protocol: protocolPtr(corev1.ProtocolTCP), Port: intOrStringPtr(4000)}},
				})
			}
			if strings.HasPrefix(backendID, "tenant:"+bundle.Name+":") {
				egress = append(egress, networkingv1.NetworkPolicyEgressRule{
					To: []networkingv1.NetworkPolicyPeer{{
						PodSelector: &metav1.LabelSelector{MatchLabels: map[string]string{
							LabelManaged:    "true",
							LabelTenantName: bundle.Name,
							"app.kubernetes.io/component": "tenant-ai-gateway",
						}},
					}},
					Ports: []networkingv1.NetworkPolicyPort{{Protocol: protocolPtr(corev1.ProtocolTCP), Port: intOrStringPtr(4000)}},
				})
			}
		}
		ingressPeers := []networkingv1.NetworkPolicyPeer{{
			PodSelector: &metav1.LabelSelector{MatchLabels: map[string]string{LabelName: "hermes-agent"}},
		}}
		if bundle.Spec.Memory.Hindsight != nil && bundle.Spec.Memory.Hindsight.HumanAccess {
			ingressPeers = append(ingressPeers, networkingv1.NetworkPolicyPeer{
				PodSelector: &metav1.LabelSelector{MatchLabels: hindsightControlPlanePodSelector(bundle)},
			})
		}
		policy.Spec = networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{MatchLabels: hindsightPodSelector(bundle)},
			PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeIngress, networkingv1.PolicyTypeEgress},
			Ingress: []networkingv1.NetworkPolicyIngressRule{{
				From:  ingressPeers,
				Ports: []networkingv1.NetworkPolicyPort{{Protocol: protocolPtr(corev1.ProtocolTCP), Port: intOrStringPtr(apiPort)}},
			}},
			Egress: egress,
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return nil, nil
}


func resolveHindsightHumanAccess(bundle *fabricv1alpha1.TenantBundle) (hindsightHumanAccessResolution, error) {
	if bundle.Spec.Memory.Hindsight == nil || !bundle.Spec.Memory.Hindsight.HumanAccess {
		return hindsightHumanAccessResolution{}, nil
	}
	if bundle.Spec.HumanAccess == nil || bundle.Spec.HumanAccess.Web == nil {
		return hindsightHumanAccessResolution{}, fmt.Errorf("TenantBundle %q enables Hindsight human access but has no humanAccess.web policy", bundle.Name)
	}
	web := bundle.Spec.HumanAccess.Web
	domain := strings.Trim(strings.TrimSpace(web.DomainSuffix), ".")
	tlsIssuer := strings.TrimSpace(web.TLSClusterIssuer)
	if domain == "" || tlsIssuer == "" {
		return hindsightHumanAccessResolution{}, fmt.Errorf("TenantBundle %q Hindsight WebUI requires humanAccess.web domainSuffix and tlsClusterIssuer", bundle.Name)
	}
	ingressClass := strings.TrimSpace(web.IngressClassName)
	if ingressClass == "" {
		ingressClass = "traefik"
	}
	host := fmt.Sprintf("hindsight-%s.%s", bundle.Name, domain)
	if len(host) > 253 {
		return hindsightHumanAccessResolution{}, fmt.Errorf("derived Hindsight WebUI host %q exceeds DNS length", host)
	}
	return hindsightHumanAccessResolution{
		Enabled:      true,
		Host:         host,
		PublicURL:    "https://" + host,
		IngressClass: ingressClass,
		TLSIssuer:    tlsIssuer,
		PublicDNS:    web.PublicDNS,
		DNSTarget:    strings.TrimSpace(web.DNSTarget),
	}, nil
}

func hindsightControlPlaneImage(profile *fabricv1alpha1.HindsightProfile) (string, error) {
	image := strings.TrimSpace(profile.Spec.Image)
	const apiPrefix = "ghcr.io/vectorize-io/hindsight-api:"
	if !strings.HasPrefix(image, apiPrefix) || strings.Contains(image, "@") {
		return "", fmt.Errorf("Hindsight WebUI currently requires paired upstream image %s<version>; got %q", apiPrefix, image)
	}
	version := strings.TrimPrefix(image, apiPrefix)
	if version == "" {
		return "", fmt.Errorf("Hindsight API image %q has no version tag for Control Plane pairing", image)
	}
	return hindsightControlPlanePrefix + version, nil
}

func hindsightControlPlanePodSelector(bundle *fabricv1alpha1.TenantBundle) map[string]string {
	return map[string]string{
		LabelName:     "hindsight-control-plane",
		LabelInstance: strings.ToLower(bundle.Spec.TenantID),
	}
}

func (r *TenantBundleReconciler) ensureHindsightControlPlane(
	ctx context.Context,
	bundle *fabricv1alpha1.TenantBundle,
	profile *fabricv1alpha1.HindsightProfile,
	names hindsightNames,
	access hindsightHumanAccessResolution,
) (*hindsightResult, error) {
	namespace := tenantNamespace(bundle.Name)
	if !access.Enabled {
		for _, obj := range []client.Object{
			&appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: names.ControlPlaneDeployment, Namespace: namespace}},
			&corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: names.ControlPlaneService, Namespace: namespace}},
			&networkingv1.NetworkPolicy{ObjectMeta: metav1.ObjectMeta{Name: names.ControlPlaneNetworkPolicy, Namespace: namespace}},
			&networkingv1.Ingress{ObjectMeta: metav1.ObjectMeta{Name: names.ControlPlaneIngress, Namespace: namespace}},
			&networkingv1.Ingress{ObjectMeta: metav1.ObjectMeta{Name: names.ControlPlaneAuthIngress, Namespace: namespace}},
			&corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: names.AuthentikExternalService, Namespace: namespace}},
		} {
			if err := r.Delete(ctx, obj); err != nil && !apierrors.IsNotFound(err) {
				return nil, err
			}
		}
		return nil, nil
	}

	image, err := hindsightControlPlaneImage(profile)
	if err != nil {
		blocked := hindsightBlocked("InvalidControlPlaneImage", err.Error(), 0)
		return &blocked, nil
	}
	labels := hindsightLabels(bundle, profile, "control-plane")
	selector := hindsightControlPlanePodSelector(bundle)
	labels = mergeStringMap(labels, selector)

	deployment := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: names.ControlPlaneDeployment, Namespace: namespace}}
	if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, deployment, func() error {
		deployment.Labels = mergeStringMap(deployment.Labels, labels)
		replicas := int32(1)
		revisionHistory := int32(2)
		deployment.Spec.Replicas = &replicas
		deployment.Spec.RevisionHistoryLimit = &revisionHistory
		deployment.Spec.Selector = &metav1.LabelSelector{MatchLabels: selector}
		runAsNonRoot := true
		runAsUser := int64(1000)
		allowPrivilegeEscalation := false
		deployment.Spec.Template = corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{Labels: labels},
			Spec: corev1.PodSpec{
				PriorityClassName: defaultString(profile.Spec.PriorityClassName, "vixens-medium"),
				SecurityContext:   &corev1.PodSecurityContext{FSGroup: int64Ptr(1000)},
				Containers: []corev1.Container{{
					Name:            "control-plane",
					Image:           image,
					ImagePullPolicy: corev1.PullIfNotPresent,
					Ports: []corev1.ContainerPort{{Name: "http", ContainerPort: hindsightControlPlanePort, Protocol: corev1.ProtocolTCP}},
					Env: []corev1.EnvVar{
						{Name: "NODE_ENV", Value: "production"},
						{Name: "HINDSIGHT_CP_HOSTNAME", Value: "0.0.0.0"},
						{Name: "HINDSIGHT_CP_PORT", Value: strconv.Itoa(int(hindsightControlPlanePort))},
						{Name: "HINDSIGHT_CP_DATAPLANE_API_URL", Value: names.Endpoint},
						{
							Name: "HINDSIGHT_CP_DATAPLANE_API_KEY",
							ValueFrom: &corev1.EnvVarSource{SecretKeyRef: &corev1.SecretKeySelector{
								LocalObjectReference: corev1.LocalObjectReference{Name: names.Secret},
								Key:                  "HINDSIGHT_API_TENANT_API_KEY",
							}},
						},
					},
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("250m"),
							corev1.ResourceMemory: resource.MustParse("512Mi"),
						},
						Limits: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("1"),
							corev1.ResourceMemory: resource.MustParse("2Gi"),
						},
					},
					SecurityContext: &corev1.SecurityContext{
						RunAsNonRoot:             &runAsNonRoot,
						RunAsUser:                &runAsUser,
						AllowPrivilegeEscalation: &allowPrivilegeEscalation,
						Capabilities:             &corev1.Capabilities{Drop: []corev1.Capability{"ALL"}},
					},
					StartupProbe: &corev1.Probe{
						ProbeHandler: corev1.ProbeHandler{TCPSocket: &corev1.TCPSocketAction{Port: intstr.FromInt32(hindsightControlPlanePort)}},
						PeriodSeconds: 5, TimeoutSeconds: 3, FailureThreshold: 60,
					},
					ReadinessProbe: &corev1.Probe{
						ProbeHandler: corev1.ProbeHandler{TCPSocket: &corev1.TCPSocketAction{Port: intstr.FromInt32(hindsightControlPlanePort)}},
						InitialDelaySeconds: 10, PeriodSeconds: 5, TimeoutSeconds: 3, FailureThreshold: 3,
					},
					LivenessProbe: &corev1.Probe{
						ProbeHandler: corev1.ProbeHandler{TCPSocket: &corev1.TCPSocketAction{Port: intstr.FromInt32(hindsightControlPlanePort)}},
						InitialDelaySeconds: 30, PeriodSeconds: 10, TimeoutSeconds: 5, FailureThreshold: 3,
					},
				}},
			},
		}
		return nil
	}); err != nil {
		return nil, err
	}

	service := &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: names.ControlPlaneService, Namespace: namespace}}
	if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, service, func() error {
		service.Labels = mergeStringMap(service.Labels, labels)
		service.Spec.Type = corev1.ServiceTypeClusterIP
		service.Spec.Selector = selector
		service.Spec.Ports = []corev1.ServicePort{{
			Name: "http", Protocol: corev1.ProtocolTCP, Port: hindsightControlPlanePort, TargetPort: intstr.FromInt32(hindsightControlPlanePort),
		}}
		return nil
	}); err != nil {
		return nil, err
	}

	authentikService := &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: names.AuthentikExternalService, Namespace: namespace}}
	if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, authentikService, func() error {
		authentikService.Labels = mergeStringMap(authentikService.Labels, labels)
		authentikService.Spec = corev1.ServiceSpec{
			Type:         corev1.ServiceTypeExternalName,
			ExternalName: "authentik.auth.svc.cluster.local",
			Ports: []corev1.ServicePort{{
				Name: "http", Protocol: corev1.ProtocolTCP, Port: 9000, TargetPort: intstr.FromInt32(9000),
			}},
		}
		return nil
	}); err != nil {
		return nil, err
	}

	pathType := networkingv1.PathTypePrefix
	ingressClass := access.IngressClass
	ingress := &networkingv1.Ingress{ObjectMeta: metav1.ObjectMeta{Name: names.ControlPlaneIngress, Namespace: namespace}}
	if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, ingress, func() error {
		ingress.Labels = mergeStringMap(ingress.Labels, labels)
		annotations := map[string]string{
			"cert-manager.io/cluster-issuer":                    access.TLSIssuer,
			"traefik.ingress.kubernetes.io/router.entrypoints": "web, websecure",
			"traefik.ingress.kubernetes.io/router.middlewares": "traefik-redirect-https@kubernetescrd,auth-authentik-forward-auth@kubernetescrd",
		}
		if access.PublicDNS {
			annotations["external-dns.alpha.kubernetes.io/public"] = "true"
		}
		if access.DNSTarget != "" {
			annotations["external-dns.alpha.kubernetes.io/target"] = access.DNSTarget
		}
		ingress.Annotations = annotations
		ingress.Spec = networkingv1.IngressSpec{
			IngressClassName: &ingressClass,
			Rules: []networkingv1.IngressRule{{
				Host: access.Host,
				IngressRuleValue: networkingv1.IngressRuleValue{HTTP: &networkingv1.HTTPIngressRuleValue{Paths: []networkingv1.HTTPIngressPath{{
					Path: "/", PathType: &pathType,
					Backend: networkingv1.IngressBackend{Service: &networkingv1.IngressServiceBackend{
						Name: names.ControlPlaneService, Port: networkingv1.ServiceBackendPort{Number: hindsightControlPlanePort},
					}},
				}}}},
			}},
			TLS: []networkingv1.IngressTLS{{Hosts: []string{access.Host}, SecretName: names.ControlPlaneIngress + "-tls"}},
		}
		return nil
	}); err != nil {
		return nil, err
	}

	authIngress := &networkingv1.Ingress{ObjectMeta: metav1.ObjectMeta{Name: names.ControlPlaneAuthIngress, Namespace: namespace}}
	if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, authIngress, func() error {
		authIngress.Labels = mergeStringMap(authIngress.Labels, labels)
		authIngress.Annotations = map[string]string{
			"traefik.ingress.kubernetes.io/router.entrypoints": "web, websecure",
		}
		authIngress.Spec = networkingv1.IngressSpec{
			IngressClassName: &ingressClass,
			Rules: []networkingv1.IngressRule{{
				Host: access.Host,
				IngressRuleValue: networkingv1.IngressRuleValue{HTTP: &networkingv1.HTTPIngressRuleValue{Paths: []networkingv1.HTTPIngressPath{{
					Path: "/outpost.goauthentik.io", PathType: &pathType,
					Backend: networkingv1.IngressBackend{Service: &networkingv1.IngressServiceBackend{
						Name: names.AuthentikExternalService, Port: networkingv1.ServiceBackendPort{Number: 9000},
					}},
				}}}},
			}},
			TLS: []networkingv1.IngressTLS{{Hosts: []string{access.Host}, SecretName: names.ControlPlaneIngress + "-tls"}},
		}
		return nil
	}); err != nil {
		return nil, err
	}

	policy := &networkingv1.NetworkPolicy{ObjectMeta: metav1.ObjectMeta{Name: names.ControlPlaneNetworkPolicy, Namespace: namespace}}
	if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, policy, func() error {
		policy.Labels = mergeStringMap(policy.Labels, labels)
		policy.Spec = networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{MatchLabels: selector},
			PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeIngress, networkingv1.PolicyTypeEgress},
			Ingress: []networkingv1.NetworkPolicyIngressRule{{
				From: []networkingv1.NetworkPolicyPeer{{
					NamespaceSelector: &metav1.LabelSelector{MatchLabels: map[string]string{"kubernetes.io/metadata.name": "traefik"}},
				}},
				Ports: []networkingv1.NetworkPolicyPort{{Protocol: protocolPtr(corev1.ProtocolTCP), Port: intOrStringPtr(int(hindsightControlPlanePort))}},
			}},
			Egress: []networkingv1.NetworkPolicyEgressRule{
				{
					To: []networkingv1.NetworkPolicyPeer{{NamespaceSelector: &metav1.LabelSelector{MatchLabels: map[string]string{"kubernetes.io/metadata.name": "kube-system"}}}},
					Ports: []networkingv1.NetworkPolicyPort{{Protocol: protocolPtr(corev1.ProtocolUDP), Port: intOrStringPtr(53)}, {Protocol: protocolPtr(corev1.ProtocolTCP), Port: intOrStringPtr(53)}},
				},
				{
					To: []networkingv1.NetworkPolicyPeer{{PodSelector: &metav1.LabelSelector{MatchLabels: hindsightPodSelector(bundle)}}},
					Ports: []networkingv1.NetworkPolicyPort{{Protocol: protocolPtr(corev1.ProtocolTCP), Port: intOrStringPtr(int(hindsightAPIPort(profile)))}},
				},
			},
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return nil, nil
}

func resolveHindsightNames(bundle *fabricv1alpha1.TenantBundle, profile *fabricv1alpha1.HindsightProfile) hindsightNames {
	port := hindsightAPIPort(profile)
	namespace := tenantNamespace(bundle.Name)
	return hindsightNames{
		Deployment:                "hindsight",
		Service:                   "hindsight",
		Secret:                    "hindsight-runtime",
		NetworkPolicy:             "hindsight-access",
		Endpoint:                  fmt.Sprintf("http://hindsight.%s.svc:%d", namespace, port),
		ControlPlaneDeployment:    "hindsight-control-plane",
		ControlPlaneService:       "hindsight-control-plane",
		ControlPlaneNetworkPolicy: "hindsight-control-plane-access",
		ControlPlaneIngress:       "hindsight-control-plane",
		ControlPlaneAuthIngress:   "hindsight-control-plane-auth",
		AuthentikExternalService:  "hindsight-authentik",
	}
}

func hindsightAPIPort(profile *fabricv1alpha1.HindsightProfile) int32 {
	if profile.Spec.APIPort > 0 {
		return profile.Spec.APIPort
	}
	return 8888
}

func hindsightUsesPlatformGateway(profile *fabricv1alpha1.HindsightProfile) bool {
	return profile.Spec.LLMAuthMode == "PlatformGateway"
}

func hindsightPodSelector(bundle *fabricv1alpha1.TenantBundle) map[string]string {
	return map[string]string{
		LabelName:     "hindsight",
		LabelInstance: strings.ToLower(bundle.Spec.TenantID),
	}
}

func hindsightLabels(bundle *fabricv1alpha1.TenantBundle, profile *fabricv1alpha1.HindsightProfile, resourceKind string) map[string]string {
	labels := tenantLabels(bundle)
	labels[LabelName] = "hindsight"
	labels[LabelInstance] = strings.ToLower(bundle.Spec.TenantID)
	labels["app.kubernetes.io/component"] = "tenant-memory"
	labels[labelHindsightProfile] = profile.Name
	labels[labelHindsightResource] = resourceKind
	return labels
}

func hindsightOwnedBy(obj metav1.Object, bundle *fabricv1alpha1.TenantBundle) bool {
	labels := obj.GetLabels()
	return labels[LabelManaged] == "true" && labels[LabelTenantID] == bundle.Spec.TenantID && labels[LabelTenantName] == bundle.Name
}

func hindsightBlocked(reason, message string, requeueAfter time.Duration) hindsightResult {
	return hindsightResult{
		Ready:        false,
		Status:       &fabricv1alpha1.ComponentStatus{Phase: "Blocked", Message: message},
		Reason:       reason,
		Message:      message,
		RequeueAfter: requeueAfter,
	}
}

func hindsightPending(reason, message string, names hindsightNames, requeueAfter time.Duration) hindsightResult {
	status := hindsightComponentStatus("Pending", names)
	status.Message = message
	return hindsightResult{Ready: false, Status: status, Reason: reason, Message: message, RequeueAfter: requeueAfter}
}

func hindsightComponentStatus(phase string, names hindsightNames) *fabricv1alpha1.ComponentStatus {
	return &fabricv1alpha1.ComponentStatus{
		Phase:    phase,
		Endpoint: names.Endpoint,
		Message:  fmt.Sprintf("deployment=%s service=%s credentialsSecret=%s networkPolicy=%s", names.Deployment, names.Service, names.Secret, names.NetworkPolicy),
	}
}
