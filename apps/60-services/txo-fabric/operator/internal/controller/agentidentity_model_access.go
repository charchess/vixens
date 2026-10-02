package controller

import (
	"context"
	"fmt"

	fabricv1alpha1 "github.com/charchess/vixens/apps/60-services/txo-fabric/operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (r *AgentIdentityReconciler) ensureModelAccess(ctx context.Context, agent *fabricv1alpha1.AgentIdentity, tenant *fabricv1alpha1.TenantBundle, namespace string) (string, error) {
	name := modelAccessSecretName(agent.Spec.AgentKey)
	alias := modelAccessKeyAlias(agent, tenant)
	desiredRevision := modelAccessRotationRevision(agent)

	var secret corev1.Secret
	err := r.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, &secret)
	if err == nil {
		currentKey := string(secret.Data[modelAccessSecretKey])
		if currentKey == "" {
			return "", fmt.Errorf("model access Secret %s/%s is missing %s", namespace, name, modelAccessSecretKey)
		}

		currentRevision := secret.Annotations[AnnotationModelAccessRevision]
		if currentRevision != desiredRevision {
			if err := revokeModelAccessKeyValueIfExists(ctx, currentKey); err != nil {
				return "", fmt.Errorf("revoke previous scoped model key: %w", err)
			}
			replacementKey, err := generateModelAccessKey(ctx, agent, tenant)
			if err != nil {
				return "", fmt.Errorf("generate replacement scoped model key: %w", err)
			}
			secret.Data[modelAccessSecretKey] = []byte(replacementKey)
			secret.Annotations = mergeStringMap(secret.Annotations, map[string]string{AnnotationModelAccessRevision: desiredRevision})
			secret.Labels = mergeStringMap(secret.Labels, agentLabels(agent, tenant))
			secret.Type = corev1.SecretTypeOpaque
			if err := controllerutil.SetControllerReference(agent, &secret, r.Scheme); err != nil {
				_ = revokeModelAccessKeyValueIfExists(ctx, replacementKey)
				return "", err
			}
			if err := r.Update(ctx, &secret); err != nil {
				_ = revokeModelAccessKeyValueIfExists(ctx, replacementKey)
				return "", err
			}
			return string(secret.UID), nil
		}

		secret.Annotations = mergeStringMap(secret.Annotations, map[string]string{AnnotationModelAccessRevision: desiredRevision})
		secret.Labels = mergeStringMap(secret.Labels, agentLabels(agent, tenant))
		secret.Type = corev1.SecretTypeOpaque
		if err := controllerutil.SetControllerReference(agent, &secret, r.Scheme); err != nil {
			return "", err
		}
		if err := r.Update(ctx, &secret); err != nil {
			return "", err
		}
		return string(secret.UID), nil
	}
	if !apierrors.IsNotFound(err) {
		return "", err
	}

	// Secret loss/recreation is also a credential replacement event. Clear any
	// key left behind under the deterministic alias before minting the replacement.
	if err := revokeModelAccessKeyIfExists(ctx, alias); err != nil {
		return "", fmt.Errorf("revoke stale scoped model key alias %q: %w", alias, err)
	}
	key, err := generateModelAccessKey(ctx, agent, tenant)
	if err != nil {
		return "", err
	}
	secret = corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Labels:      agentLabels(agent, tenant),
			Annotations: map[string]string{AnnotationModelAccessRevision: desiredRevision},
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{modelAccessSecretKey: []byte(key)},
	}
	if err := controllerutil.SetControllerReference(agent, &secret, r.Scheme); err != nil {
		_ = revokeModelAccessKeyValueIfExists(ctx, key)
		return "", err
	}
	if err := r.Create(ctx, &secret); err != nil {
		_ = revokeModelAccessKeyValueIfExists(ctx, key)
		return "", err
	}
	return string(secret.UID), nil
}
