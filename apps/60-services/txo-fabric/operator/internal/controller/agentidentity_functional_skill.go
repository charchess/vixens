package controller

import (
	"context"
	"crypto/sha256"
	"fmt"

	fabricv1alpha1 "github.com/charchess/vixens/apps/60-services/txo-fabric/operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const functionalSkillVolumeName = "functional-role-skill"

// Distinct managed name: never override a user's local or group reference skill.
func functionalSkillName(profileName string) string {
	hash := sha256.Sum256([]byte(profileName))
	return fmt.Sprintf("txo-role-%x", hash[:8])
}

func functionalSkillConfigMapName(agent *fabricv1alpha1.AgentIdentity) string {
	return "hermes-" + agent.Spec.AgentKey + "-functional"
}

func renderFunctionalSkill(role effectiveFunctionalProfile) string {
	description := fmt.Sprintf("Reference business guidance for the approved %s function, to load when relevant", role.ProfileName)
	return fmt.Sprintf("---\nname: %s\ndescription: %q\n---\n\n# Functional role: %s\n\n%s\n", functionalSkillName(role.ProfileName), description, role.ProfileName, role.Instructions)
}

func (r *AgentIdentityReconciler) ensureFunctionalSkillConfigMap(ctx context.Context, agent *fabricv1alpha1.AgentIdentity, tenant *fabricv1alpha1.TenantBundle, namespace string, role effectiveFunctionalProfile) error {
	if !role.Enabled {
		return nil
	}
	cm := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: functionalSkillConfigMapName(agent), Namespace: namespace}}
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, cm, func() error {
		cm.Labels = mergeStringMap(cm.Labels, agentLabels(agent, tenant))
		cm.Data = map[string]string{"SKILL.md": renderFunctionalSkill(role)}
		return controllerutil.SetControllerReference(agent, cm, r.Scheme)
	})
	return err
}

func (r *AgentIdentityReconciler) deleteFunctionalSkillConfigMap(ctx context.Context, agent *fabricv1alpha1.AgentIdentity, namespace string) error {
	var cm corev1.ConfigMap
	key := types.NamespacedName{Name: functionalSkillConfigMapName(agent), Namespace: namespace}
	if err := r.Get(ctx, key, &cm); err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return err
	}
	if cm.Labels[LabelInstance] != agent.Spec.AgentKey || cm.Labels[LabelTenantName] != agent.Spec.TenantRef.Name {
		return fmt.Errorf("refusing to delete unowned functional ConfigMap %s/%s", namespace, cm.Name)
	}
	return r.Delete(ctx, &cm)
}
