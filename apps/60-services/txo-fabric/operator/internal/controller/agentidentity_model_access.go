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

func (r *AgentIdentityReconciler) ensureModelAccess(ctx context.Context, agent *fabricv1alpha1.AgentIdentity, tenant *fabricv1alpha1.TenantBundle, namespace string) error {
	name := modelAccessSecretName(agent.Spec.AgentKey)
	var secret corev1.Secret
	err := r.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, &secret)
	if err == nil {
		if len(secret.Data[modelAccessSecretKey]) == 0 {
			return fmt.Errorf("model access Secret %s/%s is missing %s", namespace, name, modelAccessSecretKey)
		}
		_, err = controllerutil.CreateOrUpdate(ctx, r.Client, &secret, func() error {
			secret.Labels = mergeStringMap(secret.Labels, agentLabels(agent, tenant))
			secret.Type = corev1.SecretTypeOpaque
			return controllerutil.SetControllerReference(agent, &secret, r.Scheme)
		})
		return err
	}
	if !apierrors.IsNotFound(err) {
		return err
	}

	key, err := generateModelAccessKey(ctx, agent, tenant)
	if err != nil {
		return err
	}
	secret = corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace, Labels: agentLabels(agent, tenant)},
		Type:       corev1.SecretTypeOpaque,
		Data:       map[string][]byte{modelAccessSecretKey: []byte(key)},
	}
	if err := controllerutil.SetControllerReference(agent, &secret, r.Scheme); err != nil {
		return err
	}
	if err := r.Create(ctx, &secret); err != nil {
		_ = revokeModelAccessKey(ctx, modelAccessKeyAlias(agent, tenant))
		return err
	}
	return nil
}
