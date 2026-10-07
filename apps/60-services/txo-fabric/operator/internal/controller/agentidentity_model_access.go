package controller

import (
	"context"
	"fmt"
	"strings"

	fabricv1alpha1 "github.com/charchess/vixens/apps/60-services/txo-fabric/operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (r *AgentIdentityReconciler) resolveModelAccessBackend(ctx context.Context, tenant *fabricv1alpha1.TenantBundle) (modelAccessBackend, error) {
	if tenant.Spec.AIGateway == nil {
		return sharedModelAccessBackend(), nil
	}

	profileName := strings.TrimSpace(tenant.Spec.AIGateway.ProfileRef)
	if profileName == "" {
		profileName = defaultAIGatewayProfileName
	}
	var profile fabricv1alpha1.AIGatewayProfile
	if err := r.Get(ctx, types.NamespacedName{Name: profileName}, &profile); err != nil {
		return modelAccessBackend{}, fmt.Errorf("resolve tenant AI gateway profile %q: %w", profileName, err)
	}

	namespace := tenantNamespace(tenant.Name)
	var runtimeSecret corev1.Secret
	if err := r.Get(ctx, types.NamespacedName{Name: tenantAIGatewayRuntimeSecretName, Namespace: namespace}, &runtimeSecret); err != nil {
		return modelAccessBackend{}, fmt.Errorf("read tenant AI gateway runtime Secret %s/%s: %w", namespace, tenantAIGatewayRuntimeSecretName, err)
	}
	adminToken := strings.TrimSpace(string(runtimeSecret.Data["LITELLM_MASTER_KEY"]))
	if adminToken == "" {
		return modelAccessBackend{}, fmt.Errorf("tenant AI gateway runtime Secret %s/%s is missing LITELLM_MASTER_KEY", namespace, tenantAIGatewayRuntimeSecretName)
	}

	port := aiGatewayPort(&profile)
	return modelAccessBackend{
		ID:         fmt.Sprintf("tenant:%s:%s:%d", tenant.Name, profile.Name, port),
		URL:        fmt.Sprintf("http://%s.%s.svc:%d", tenantAIGatewayName, namespace, port),
		AdminToken: adminToken,
		Model:      tenantAIAgentModel,
	}, nil
}

func (r *AgentIdentityReconciler) recordedModelAccessBackend(
	ctx context.Context,
	tenant *fabricv1alpha1.TenantBundle,
	backendID, backendURL string,
) (modelAccessBackend, error) {
	backendID = strings.TrimSpace(backendID)
	if backendID == "" || backendID == sharedModelAccessBackendID {
		return sharedModelAccessBackend(), nil
	}
	if !strings.HasPrefix(backendID, "tenant:"+tenant.Name+":") {
		return modelAccessBackend{}, fmt.Errorf("recorded model access backend %q does not belong to tenant %q", backendID, tenant.Name)
	}

	backendURL = strings.TrimSpace(backendURL)
	if backendURL == "" {
		return modelAccessBackend{}, fmt.Errorf("recorded model access backend %q has no endpoint", backendID)
	}
	namespace := tenantNamespace(tenant.Name)
	var runtimeSecret corev1.Secret
	if err := r.Get(ctx, types.NamespacedName{Name: tenantAIGatewayRuntimeSecretName, Namespace: namespace}, &runtimeSecret); err != nil {
		return modelAccessBackend{}, fmt.Errorf("read tenant AI gateway runtime Secret %s/%s for recorded backend %q: %w", namespace, tenantAIGatewayRuntimeSecretName, backendID, err)
	}
	adminToken := strings.TrimSpace(string(runtimeSecret.Data["LITELLM_MASTER_KEY"]))
	if adminToken == "" {
		return modelAccessBackend{}, fmt.Errorf("tenant AI gateway runtime Secret %s/%s is missing LITELLM_MASTER_KEY for recorded backend %q", namespace, tenantAIGatewayRuntimeSecretName, backendID)
	}
	return modelAccessBackend{
		ID:         backendID,
		URL:        backendURL,
		AdminToken: adminToken,
		Model:      tenantAIAgentModel,
	}, nil
}

func (r *AgentIdentityReconciler) previousModelAccessBackend(
	ctx context.Context,
	tenant *fabricv1alpha1.TenantBundle,
	secret *corev1.Secret,
) (modelAccessBackend, error) {
	backendID := normalizedAppliedModelAccessBackend(secret)
	backendURL := strings.TrimSpace(secret.Annotations[AnnotationModelAccessBackendURL])
	if backendID == sharedModelAccessBackendID && backendURL == "" {
		backendURL = sharedModelAccessBackend().URL
	}
	backend, err := r.recordedModelAccessBackend(ctx, tenant, backendID, backendURL)
	if err != nil {
		return modelAccessBackend{}, fmt.Errorf("model access Secret %s/%s: %w", secret.Namespace, secret.Name, err)
	}
	return backend, nil
}

func normalizedAppliedModelAccessBackend(secret *corev1.Secret) string {
	backendID := strings.TrimSpace(secret.Annotations[AnnotationModelAccessBackend])
	if backendID == "" {
		return sharedModelAccessBackendID
	}
	return backendID
}

func modelAccessRevisionForBackend(backend modelAccessBackend, requestedRotation string) string {
	return modelAccessBackendRevision(backend.ID, requestedRotation)
}

func pendingModelAccessRevokeBackend(secret *corev1.Secret) (string, string) {
	if secret == nil {
		return "", ""
	}
	return strings.TrimSpace(secret.Annotations[AnnotationModelAccessPendingRevokeBackend]),
		strings.TrimSpace(secret.Annotations[AnnotationModelAccessPendingRevokeURL])
}

func deploymentAdoptedModelAccess(deployment *appsv1.Deployment, secret *corev1.Secret) bool {
	if deployment == nil || secret == nil {
		return false
	}
	revision := strings.TrimSpace(secret.Annotations[AnnotationModelAccessRevision])
	if revision == "" {
		return false
	}
	if deployment.Spec.Template.Annotations[AnnotationModelAccessRevision] != revision {
		return false
	}
	if deployment.Spec.Template.Annotations[AnnotationModelAccessSecretUID] != string(secret.UID) {
		return false
	}
	return deployment.Status.ObservedGeneration == deployment.Generation && deployment.Status.AvailableReplicas > 0
}

func (r *AgentIdentityReconciler) finalizeModelAccessBackendCutover(
	ctx context.Context,
	agent *fabricv1alpha1.AgentIdentity,
	tenant *fabricv1alpha1.TenantBundle,
	namespace string,
	backend modelAccessBackend,
	deployment *appsv1.Deployment,
) (bool, error) {
	name := modelAccessSecretName(agent.Spec.AgentKey)
	var secret corev1.Secret
	if err := r.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, &secret); err != nil {
		return false, err
	}

	pendingBackendID, pendingBackendURL := pendingModelAccessRevokeBackend(&secret)
	if pendingBackendID == "" {
		return false, nil
	}
	if normalizedAppliedModelAccessBackend(&secret) != backend.ID {
		return true, fmt.Errorf(
			"model access Secret %s/%s has pending source backend %q while applied backend is %q and desired backend is %q",
			namespace,
			name,
			pendingBackendID,
			normalizedAppliedModelAccessBackend(&secret),
			backend.ID,
		)
	}
	if !deploymentAdoptedModelAccess(deployment, &secret) {
		return true, nil
	}

	sourceBackend, err := r.recordedModelAccessBackend(ctx, tenant, pendingBackendID, pendingBackendURL)
	if err != nil {
		return true, fmt.Errorf("resolve pending source model access backend: %w", err)
	}
	if sourceBackend.ID == backend.ID {
		return true, fmt.Errorf("pending source model access backend %q equals desired backend", sourceBackend.ID)
	}
	if err := revokeModelAccessKeyIfExistsWithBackend(ctx, sourceBackend, modelAccessKeyAlias(agent, tenant)); err != nil {
		return true, fmt.Errorf("revoke superseded scoped model key alias from backend %q: %w", sourceBackend.ID, err)
	}

	delete(secret.Annotations, AnnotationModelAccessPendingRevokeBackend)
	delete(secret.Annotations, AnnotationModelAccessPendingRevokeURL)
	if err := r.Update(ctx, &secret); err != nil {
		return true, fmt.Errorf("clear pending model access source backend: %w", err)
	}
	return false, nil
}

func (r *AgentIdentityReconciler) ensureModelAccess(ctx context.Context, agent *fabricv1alpha1.AgentIdentity, tenant *fabricv1alpha1.TenantBundle, namespace string) (string, string, error) {
	backend, err := r.resolveModelAccessBackend(ctx, tenant)
	if err != nil {
		return "", "", err
	}
	return r.ensureModelAccessWithBackend(ctx, agent, tenant, namespace, backend)
}

func (r *AgentIdentityReconciler) ensureModelAccessWithBackend(
	ctx context.Context,
	agent *fabricv1alpha1.AgentIdentity,
	tenant *fabricv1alpha1.TenantBundle,
	namespace string,
	backend modelAccessBackend,
) (string, string, error) {
	name := modelAccessSecretName(agent.Spec.AgentKey)
	alias := modelAccessKeyAlias(agent, tenant)
	requestedRotation := modelAccessRotationRevision(agent)

	var secret corev1.Secret
	err := r.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, &secret)
	if err == nil {
		currentKey := string(secret.Data[modelAccessSecretKey])
		if currentKey == "" {
			return "", "", fmt.Errorf("model access Secret %s/%s is missing %s", namespace, name, modelAccessSecretKey)
		}

		appliedBackendID := normalizedAppliedModelAccessBackend(&secret)
		appliedRevision := secret.Annotations[AnnotationModelAccessRevision]
		appliedRotation := secret.Annotations[AnnotationModelAccessRotationApplied]
		backendChanged := appliedBackendID != backend.ID

		rotationRequested := false
		if requestedRotation != "" {
			if backend.ID == sharedModelAccessBackendID {
				// Preserve the historical shared-gateway rotation contract.
				rotationRequested = appliedRevision != requestedRotation
			} else {
				rotationRequested = appliedRotation != requestedRotation
			}
		}

		if backendChanged || rotationRequested {
			if backendChanged {
				if pendingBackendID, _ := pendingModelAccessRevokeBackend(&secret); pendingBackendID != "" {
					return "", "", fmt.Errorf("model access backend cutover from %q is still pending cleanup; refusing to start another backend transition", pendingBackendID)
				}

				previousBackendID := appliedBackendID
				previousBackendURL := strings.TrimSpace(secret.Annotations[AnnotationModelAccessBackendURL])
				if previousBackendID == sharedModelAccessBackendID && previousBackendURL == "" {
					previousBackendURL = sharedModelAccessBackend().URL
				}

				// Prepare the destination while the currently working source
				// credential remains untouched. A failed destination cleanup or
				// mint therefore cannot invalidate the running Hermes pod.
				if err := revokeModelAccessKeyIfExistsWithBackend(ctx, backend, alias); err != nil {
					return "", "", fmt.Errorf("clear destination scoped model key alias %q on backend %q: %w", alias, backend.ID, err)
				}
				replacementKey, err := generateModelAccessKey(ctx, backend, agent, tenant)
				if err != nil {
					return "", "", fmt.Errorf("generate destination scoped model key: %w", err)
				}
				revision := modelAccessRevisionForBackend(backend, requestedRotation)
				secret.Data[modelAccessSecretKey] = []byte(replacementKey)
				secret.Annotations = mergeStringMap(secret.Annotations, map[string]string{
					AnnotationModelAccessBackend:              backend.ID,
					AnnotationModelAccessBackendURL:           backend.URL,
					AnnotationModelAccessPendingRevokeBackend: previousBackendID,
					AnnotationModelAccessPendingRevokeURL:     previousBackendURL,
				})
				if revision != "" {
					secret.Annotations[AnnotationModelAccessRevision] = revision
				} else {
					delete(secret.Annotations, AnnotationModelAccessRevision)
				}
				if requestedRotation != "" {
					secret.Annotations[AnnotationModelAccessRotationApplied] = requestedRotation
				}
				secret.Labels = mergeStringMap(secret.Labels, agentLabels(agent, tenant))
				secret.Type = corev1.SecretTypeOpaque
				if err := controllerutil.SetControllerReference(agent, &secret, r.Scheme); err != nil {
					_ = revokeModelAccessKeyValueIfExistsWithBackend(ctx, backend, replacementKey)
					return "", "", err
				}
				if err := r.Update(ctx, &secret); err != nil {
					_ = revokeModelAccessKeyValueIfExistsWithBackend(ctx, backend, replacementKey)
					return "", "", err
				}
				return string(secret.UID), revision, nil
			}

			// Same-backend rotation still requires removing the deterministic
			// alias before replacing it; backend cutovers use the staged path
			// above so the source remains valid until runtime adoption.
			if err := revokeModelAccessKeyIfExistsWithBackend(ctx, backend, alias); err != nil {
				return "", "", fmt.Errorf("revoke previous scoped model key alias %q from backend %q: %w", alias, backend.ID, err)
			}
			replacementKey, err := generateModelAccessKey(ctx, backend, agent, tenant)
			if err != nil {
				return "", "", fmt.Errorf("generate replacement scoped model key: %w", err)
			}
			revision := modelAccessRevisionForBackend(backend, requestedRotation)
			secret.Data[modelAccessSecretKey] = []byte(replacementKey)
			secret.Annotations = mergeStringMap(secret.Annotations, map[string]string{
				AnnotationModelAccessBackend:    backend.ID,
				AnnotationModelAccessBackendURL: backend.URL,
			})
			if revision != "" {
				secret.Annotations[AnnotationModelAccessRevision] = revision
			} else {
				delete(secret.Annotations, AnnotationModelAccessRevision)
			}
			if requestedRotation != "" {
				secret.Annotations[AnnotationModelAccessRotationApplied] = requestedRotation
			}
			secret.Labels = mergeStringMap(secret.Labels, agentLabels(agent, tenant))
			secret.Type = corev1.SecretTypeOpaque
			if err := controllerutil.SetControllerReference(agent, &secret, r.Scheme); err != nil {
				_ = revokeModelAccessKeyValueIfExistsWithBackend(ctx, backend, replacementKey)
				return "", "", err
			}
			if err := r.Update(ctx, &secret); err != nil {
				_ = revokeModelAccessKeyValueIfExistsWithBackend(ctx, backend, replacementKey)
				return "", "", err
			}
			return string(secret.UID), revision, nil
		}

		secret.Annotations = mergeStringMap(secret.Annotations, map[string]string{
			AnnotationModelAccessBackend:    backend.ID,
			AnnotationModelAccessBackendURL: backend.URL,
		})
		secret.Labels = mergeStringMap(secret.Labels, agentLabels(agent, tenant))
		secret.Type = corev1.SecretTypeOpaque
		if err := controllerutil.SetControllerReference(agent, &secret, r.Scheme); err != nil {
			return "", "", err
		}
		if err := r.Update(ctx, &secret); err != nil {
			return "", "", err
		}
		return string(secret.UID), appliedRevision, nil
	}
	if !apierrors.IsNotFound(err) {
		return "", "", err
	}

	// Secret loss/recreation is also a credential replacement event. Clear any
	// key left behind under the deterministic alias on the selected backend
	// before minting the replacement.
	if err := revokeModelAccessKeyIfExistsWithBackend(ctx, backend, alias); err != nil {
		return "", "", fmt.Errorf("revoke stale scoped model key alias %q on backend %q: %w", alias, backend.ID, err)
	}
	key, err := generateModelAccessKey(ctx, backend, agent, tenant)
	if err != nil {
		return "", "", err
	}
	revision := modelAccessRevisionForBackend(backend, requestedRotation)
	annotations := map[string]string{
		AnnotationModelAccessBackend:    backend.ID,
		AnnotationModelAccessBackendURL: backend.URL,
	}
	if revision != "" {
		annotations[AnnotationModelAccessRevision] = revision
	}
	if requestedRotation != "" {
		annotations[AnnotationModelAccessRotationApplied] = requestedRotation
	}
	secret = corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Labels:      agentLabels(agent, tenant),
			Annotations: annotations,
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{modelAccessSecretKey: []byte(key)},
	}
	if err := controllerutil.SetControllerReference(agent, &secret, r.Scheme); err != nil {
		_ = revokeModelAccessKeyValueIfExistsWithBackend(ctx, backend, key)
		return "", "", err
	}
	if err := r.Create(ctx, &secret); err != nil {
		_ = revokeModelAccessKeyValueIfExistsWithBackend(ctx, backend, key)
		return "", "", err
	}
	return string(secret.UID), revision, nil
}
