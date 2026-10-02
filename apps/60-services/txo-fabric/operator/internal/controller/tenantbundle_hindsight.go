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
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	labelHindsightProfile  = "fabric.truxonline.io/hindsight-profile"
	labelHindsightResource = "fabric.truxonline.io/hindsight-resource"
)

type hindsightNames struct {
	Deployment    string
	Service       string
	Secret        string
	NetworkPolicy string
	Endpoint      string
}

type hindsightResult struct {
	Ready        bool
	Status       *fabricv1alpha1.ComponentStatus
	Reason       string
	Message      string
	RequeueAfter time.Duration
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

	var deployment appsv1.Deployment
	if err := r.Get(ctx, types.NamespacedName{Name: names.Deployment, Namespace: tenantNamespace(bundle.Name)}, &deployment); err != nil {
		return hindsightResult{}, err
	}
	if deployment.Status.ObservedGeneration != deployment.Generation || deployment.Status.AvailableReplicas < 1 {
		return hindsightPending("DeploymentProgressing", "waiting for the tenant Hindsight Deployment to become available", names, 5*time.Second), nil
	}

	status := hindsightComponentStatus("Ready", names)
	message := "tenant-scoped Hindsight API is available with secret-backed PostgreSQL and API-key authentication"
	if hindsightUsesPlatformGateway(&profile) {
		rotation := hindsightEmbeddingRotationRevision(bundle)
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
		// wedge TenantBundle deletion. The deterministic alias contains no secret.
		_ = revokeModelAccessKey(ctx, hindsightEmbeddingKeyAlias(bundle))
	}
	return pending, nil
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

	databaseURL := &url.URL{
		Scheme: "postgresql",
		User:   url.UserPassword(username, password),
		Host:   net.JoinHostPort(host, port),
		Path:   "/" + database,
	}

	gatewayEnabled := hindsightUsesPlatformGateway(profile)
	alias := hindsightEmbeddingKeyAlias(bundle)
	desiredRevision := hindsightEmbeddingRotationRevision(bundle)
	namespace := tenantNamespace(bundle.Name)
	key := client.ObjectKey{Namespace: namespace, Name: names.Secret}
	var secret corev1.Secret
	if err := r.Get(ctx, key, &secret); err != nil {
		if !apierrors.IsNotFound(err) {
			return nil, err
		}
		apiKey, err := randomPassword()
		if err != nil {
			return nil, err
		}
		embeddingKey := ""
		annotations := map[string]string{}
		if gatewayEnabled {
			if err := revokeModelAccessKeyIfExists(ctx, alias); err != nil {
				return nil, fmt.Errorf("revoke stale Hindsight embedding key alias %q: %w", alias, err)
			}
			embeddingKey, err = generateHindsightEmbeddingAccessKey(ctx, bundle)
			if err != nil {
				return nil, fmt.Errorf("provision Hindsight embedding gateway credential: %w", err)
			}
			annotations[AnnotationHindsightEmbeddingRevision] = desiredRevision
		}
		secret = corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:        names.Secret,
				Namespace:   namespace,
				Labels:      hindsightLabels(bundle, profile, "runtime-secret"),
				Annotations: annotations,
			},
			Type: corev1.SecretTypeOpaque,
			Data: desiredHindsightSecretData(databaseURL.String(), apiKey, hindsightAPIPort(profile), embeddingKey, gatewayEnabled),
		}
		if err := r.Create(ctx, &secret); err != nil {
			if embeddingKey != "" {
				_ = revokeModelAccessKeyValueIfExists(ctx, embeddingKey)
			}
			return nil, err
		}
		return nil, nil
	}
	if !hindsightOwnedBy(&secret, bundle) {
		blocked := hindsightBlocked("OwnershipConflict", fmt.Sprintf("Secret %s/%s exists but is not owned by this Fabric tenant", namespace, names.Secret), 0)
		return &blocked, nil
	}

	apiKey := string(secret.Data["HINDSIGHT_API_TENANT_API_KEY"])
	if apiKey == "" {
		var err error
		apiKey, err = randomPassword()
		if err != nil {
			return nil, err
		}
	}

	embeddingKey := string(secret.Data[hindsightEmbeddingSecretKey])
	currentRevision := secret.Annotations[AnnotationHindsightEmbeddingRevision]
	generatedEmbeddingKey := false
	if gatewayEnabled {
		needsReplacement := embeddingKey == "" || currentRevision != desiredRevision
		if needsReplacement {
			if embeddingKey != "" {
				if err := revokeModelAccessKeyValueIfExists(ctx, embeddingKey); err != nil {
					return nil, fmt.Errorf("revoke previous Hindsight embedding key: %w", err)
				}
			} else if err := revokeModelAccessKeyIfExists(ctx, alias); err != nil {
				return nil, fmt.Errorf("revoke stale Hindsight embedding key alias %q: %w", alias, err)
			}
			var err error
			embeddingKey, err = generateHindsightEmbeddingAccessKey(ctx, bundle)
			if err != nil {
				return nil, fmt.Errorf("provision Hindsight embedding gateway credential: %w", err)
			}
			generatedEmbeddingKey = true
		}
	} else if embeddingKey != "" {
		_ = revokeModelAccessKey(ctx, alias)
		embeddingKey = ""
	}

	desiredLabels := mergeStringMap(copyStringMap(secret.Labels), hindsightLabels(bundle, profile, "runtime-secret"))
	desiredAnnotations := copyStringMap(secret.Annotations)
	if gatewayEnabled {
		desiredAnnotations[AnnotationHindsightEmbeddingRevision] = desiredRevision
	} else {
		delete(desiredAnnotations, AnnotationHindsightEmbeddingRevision)
	}
	desiredData := desiredHindsightSecretData(databaseURL.String(), apiKey, hindsightAPIPort(profile), embeddingKey, gatewayEnabled)
	if secret.Type != corev1.SecretTypeOpaque || !reflect.DeepEqual(secret.Labels, desiredLabels) || !reflect.DeepEqual(secret.Annotations, desiredAnnotations) || !reflect.DeepEqual(secret.Data, desiredData) {
		secret.Type = corev1.SecretTypeOpaque
		secret.Labels = desiredLabels
		secret.Annotations = desiredAnnotations
		secret.Data = desiredData
		if err := r.Update(ctx, &secret); err != nil {
			if generatedEmbeddingKey {
				_ = revokeModelAccessKeyValueIfExists(ctx, embeddingKey)
			}
			return nil, err
		}
	}
	return nil, nil
}

func desiredHindsightSecretData(databaseURL, apiKey string, port int32, embeddingKey string, gatewayEnabled bool) map[string][]byte {
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
		data["HINDSIGHT_API_EMBEDDINGS_OPENAI_BASE_URL"] = []byte(aiGatewayURL() + "/v1")
		data["HINDSIGHT_API_EMBEDDINGS_OPENAI_MODEL"] = []byte(defaultAIEmbeddingModel)
		data["HINDSIGHT_API_EMBEDDINGS_OPENAI_DIMENSIONS"] = []byte(defaultAIEmbeddingDimension)
		data[hindsightEmbeddingSecretKey] = []byte(embeddingKey)
	}
	return data
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
			if err := r.Get(ctx, client.ObjectKey{Namespace: namespace, Name: names.Secret}, &runtimeSecret); err != nil {
				return err
			}
			podAnnotations[AnnotationHindsightEmbeddingRevision] = hindsightEmbeddingRotationRevision(bundle)
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
			egress = append(egress, networkingv1.NetworkPolicyEgressRule{
				To: []networkingv1.NetworkPolicyPeer{{
					NamespaceSelector: &metav1.LabelSelector{MatchLabels: map[string]string{"kubernetes.io/metadata.name": "txo-fabric-system"}},
					PodSelector:       &metav1.LabelSelector{MatchLabels: map[string]string{LabelName: "txo-ai-gateway"}},
				}},
				Ports: []networkingv1.NetworkPolicyPort{{Protocol: protocolPtr(corev1.ProtocolTCP), Port: intOrStringPtr(4000)}},
			})
		}
		policy.Spec = networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{MatchLabels: hindsightPodSelector(bundle)},
			PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeIngress, networkingv1.PolicyTypeEgress},
			Ingress: []networkingv1.NetworkPolicyIngressRule{{
				From:  []networkingv1.NetworkPolicyPeer{{PodSelector: &metav1.LabelSelector{MatchLabels: map[string]string{LabelName: "hermes-agent"}}}},
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

func resolveHindsightNames(bundle *fabricv1alpha1.TenantBundle, profile *fabricv1alpha1.HindsightProfile) hindsightNames {
	port := hindsightAPIPort(profile)
	namespace := tenantNamespace(bundle.Name)
	return hindsightNames{
		Deployment:    "hindsight",
		Service:       "hindsight",
		Secret:        "hindsight-runtime",
		NetworkPolicy: "hindsight-access",
		Endpoint:      fmt.Sprintf("http://hindsight.%s.svc:%d", namespace, port),
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
