package controller

import (
	"context"
	"fmt"
	"strings"

	fabricv1alpha1 "github.com/charchess/vixens/apps/60-services/txo-fabric/operator/api/v1alpha1"
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
		Model:      tenantAICodingModel,
	}, nil
}

func (r *AgentIdentityReconciler) previousModelAccessBackend(
	ctx context.Context,
	tenant *fabricv1alpha1.TenantBundle,
	secret *corev1.Secret,
) (modelAccessBackend, error) {
	backendID := strings.TrimSpace(secret.Annotations[AnnotationModelAccessBackend])
	if backendID == "" || backendID == sharedModelAccessBackendID {
		return sharedModelAccessBackend(), nil
	}

	backendURL := strings.TrimSpace(secret.Annotations[AnnotationModelAccessBackendURL])
	if backendURL == "" {
		return modelAccessBackend{}, fmt.Errorf("model access Secret %s/%s records backend %q without an endpoint", secret.Namespace, secret.Name, backendID)
	}
	namespace := tenantNamespace(tenant.Name)
	var runtimeSecret corev1.Secret
	if err := r.Get(ctx, types.NamespacedName{Name: tenantAIGatewayRuntimeSecretName, Namespace: namespace}, &runtimeSecret); err != nil {
		return modelAccessBackend{}, fmt.Errorf("read previous tenant AI gateway runtime Secret %s/%s: %w", namespace, tenantAIGatewayRuntimeSecretName, err)
	}
	adminToken := strings.TrimSpace(string(runtimeSecret.Data["LITELLM_MASTER_KEY"]))
	if adminToken == "" {
		return modelAccessBackend{}, fmt.Errorf("previous tenant AI gateway runtime Secret %s/%s is missing LITELLM_MASTER_KEY", namespace, tenantAIGatewayRuntimeSecretName)
	}
	return modelAccessBackend{
		ID:         backendID,
		URL:        backendURL,
		AdminToken: adminToken,
		Model:      tenantAICodingModel,
	}, nil
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
			revokeBackend := backend
			if backendChanged {
				revokeBackend, err = r.previousModelAccessBackend(ctx, tenant, &secret)
				if err != nil {
					return "", "", fmt.Errorf("resolve previous model access backend: %w", err)
				}
			}
			if err := revokeModelAccessKeyIfExistsWithBackend(ctx, revokeBackend, alias); err != nil {
				return "", "", fmt.Errorf("revoke previous scoped model key alias %q from backend %q: %w", alias, revokeBackend.ID, err)
			}
			if backendChanged {
				// A failed/aborted earlier cutover may have left the deterministic
				// alias on the destination gateway. Clear it before minting again.
				if err := revokeModelAccessKeyIfExistsWithBackend(ctx, backend, alias); err != nil {
					return "", "", fmt.Errorf("clear destination scoped model key alias %q on backend %q: %w", alias, backend.ID, err)
				}
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
