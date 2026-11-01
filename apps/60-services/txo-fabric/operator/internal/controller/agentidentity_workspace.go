package controller

import (
	"context"
	"fmt"
	"path"

	fabricv1alpha1 "github.com/charchess/vixens/apps/60-services/txo-fabric/operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

func resolvedWorkspaceVolumes(agent *fabricv1alpha1.AgentIdentity, tenant *fabricv1alpha1.TenantBundle) ([]corev1.Volume, []corev1.VolumeMount, error) {
	workspace := tenant.Spec.Workspace
	if workspace == nil {
		if agent.Spec.Access.UserRef != "" || len(agent.Spec.Access.Groups) > 0 {
			return nil, nil, fmt.Errorf("AgentIdentity %q declares workspace access but TenantBundle %q has no workspace capability", agent.Name, tenant.Name)
		}
		return nil, nil, nil
	}

	groups := make(map[string]fabricv1alpha1.NamedWorkspaceScopeSpec, len(workspace.Groups))
	for _, group := range workspace.Groups {
		groups[group.Name] = group
	}
	users := make(map[string]fabricv1alpha1.NamedWorkspaceScopeSpec, len(workspace.Users))
	for _, user := range workspace.Users {
		users[user.Name] = user
	}

	scopes := make([]workspaceScope, 0)
	appendModes := func(domain, scope, key string, reference, collaborative bool) {
		if reference {
			scopes = append(scopes, workspaceScope{Domain: domain, Scope: scope, Key: key, Mode: "reference", ReadOnly: true})
		}
		if collaborative {
			scopes = append(scopes, workspaceScope{Domain: domain, Scope: scope, Key: key, Mode: "collaborative"})
		}
	}
	appendScope := func(scope, key string, reference, collaborative, skillsReference, skillsCollaborative bool) {
		appendModes("shared", scope, key, reference, collaborative)
		appendModes("skills", scope, key, skillsReference, skillsCollaborative)
	}

	organization := workspace.Organization
	appendScope("organization", "organization", organization.Reference, organization.Collaborative, organization.SkillsReference, organization.SkillsCollaborative)

	seenGroups := map[string]struct{}{}
	for _, groupName := range agent.Spec.Access.Groups {
		if _, duplicate := seenGroups[groupName]; duplicate {
			continue
		}
		seenGroups[groupName] = struct{}{}
		group, ok := groups[groupName]
		if !ok {
			return nil, nil, fmt.Errorf("AgentIdentity %q requests undeclared workspace group %q in tenant %q", agent.Name, groupName, tenant.Name)
		}
		appendScope("group", group.Name, group.Reference, group.Collaborative, group.SkillsReference, group.SkillsCollaborative)
	}

	if agent.Spec.Access.UserRef != "" {
		user, ok := users[agent.Spec.Access.UserRef]
		if !ok {
			return nil, nil, fmt.Errorf("AgentIdentity %q requests undeclared workspace user %q in tenant %q", agent.Name, agent.Spec.Access.UserRef, tenant.Name)
		}
		appendScope("user", user.Name, user.Reference, user.Collaborative, user.SkillsReference, user.SkillsCollaborative)
	}

	volumes := make([]corev1.Volume, 0, len(scopes))
	mounts := make([]corev1.VolumeMount, 0, len(scopes))
	for _, scope := range scopes {
		name := workspacePVCName(scope)
		mountPath := workspaceMountPath(scope)
		volumes = append(volumes, corev1.Volume{
			Name:         name,
			VolumeSource: corev1.VolumeSource{PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: name}},
		})
		mounts = append(mounts, corev1.VolumeMount{Name: name, MountPath: mountPath, ReadOnly: scope.ReadOnly})
	}
	return volumes, mounts, nil
}


// ensureNativeWorkspaceClaimsReady checks the actual PVC/PV behind every
// authorized mount. Deterministic names alone never authorize access to
// preexisting business data. SharedWorkspaceReconciler owns provisioning;
// this gate never creates/adopts/patches or deletes PVCs.
func (r *AgentIdentityReconciler) ensureNativeWorkspaceClaimsReady(
	ctx context.Context, agent *fabricv1alpha1.AgentIdentity, tenant *fabricv1alpha1.TenantBundle,
) error {
	volumes, _, err := resolvedWorkspaceVolumes(agent, tenant)
	if err != nil {
		return err
	}
	if len(volumes) == 0 {
		return nil
	}
	var profile fabricv1alpha1.SharedWorkspaceProfile
	if err := r.Get(ctx, types.NamespacedName{Name: tenant.Spec.Workspace.ProfileRef}, &profile); err != nil {
		return fmt.Errorf("workspace profile for tenant %q unavailable: %w", tenant.Name, err)
	}
	if profile.Spec.StorageClassName == "" {
		return fmt.Errorf("workspace profile %q does not declare a storage class", profile.Name)
	}
	scopes := make(map[string]workspaceScope)
	for _, scope := range desiredWorkspaceScopes(tenant) {
		scopes[workspacePVCName(scope)] = scope
	}
	namespace := tenantNamespace(tenant.Name)
	for _, volume := range volumes {
		if volume.PersistentVolumeClaim == nil {
			return fmt.Errorf("workspace volume %q has no PVC reference", volume.Name)
		}
		name := volume.PersistentVolumeClaim.ClaimName
		scope, ok := scopes[name]
		if !ok {
			return fmt.Errorf("workspace claim %q is not declared for tenant %q", name, tenant.Name)
		}
		var pvc corev1.PersistentVolumeClaim
		if err := r.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, &pvc); err != nil {
			return fmt.Errorf("workspace PVC %s/%s not ready: %w", namespace, name, err)
		}
		if pvc.UID == "" || pvc.Status.Phase != corev1.ClaimBound || pvc.Spec.VolumeName == "" {
			return fmt.Errorf("workspace PVC %s/%s lacks a bound UID/PV identity", namespace, name)
		}
		if pvc.Spec.StorageClassName == nil || *pvc.Spec.StorageClassName != profile.Spec.StorageClassName {
			return fmt.Errorf("workspace PVC %s/%s has an unexpected storage class", namespace, name)
		}
		if len(pvc.Spec.AccessModes) != 1 || pvc.Spec.AccessModes[0] != corev1.ReadWriteMany {
			return fmt.Errorf("workspace PVC %s/%s is not exclusively ReadWriteMany", namespace, name)
		}
		for label, expected := range workspaceLabels(tenant, scope) {
			if pvc.Labels[label] != expected {
				return fmt.Errorf("workspace PVC %s/%s lacks exact ownership label %q", namespace, name, label)
			}
		}
		var pv corev1.PersistentVolume
		if err := r.Get(ctx, types.NamespacedName{Name: pvc.Spec.VolumeName}, &pv); err != nil {
			return fmt.Errorf("workspace PV for %s/%s unavailable: %w", namespace, name, err)
		}
		ref := pv.Spec.ClaimRef
		if ref == nil || ref.Namespace != namespace || ref.Name != name || ref.UID != pvc.UID ||
			pv.Spec.StorageClassName != profile.Spec.StorageClassName {
			return fmt.Errorf("workspace PV %q is not UID-bound to tenant claim %s/%s", pv.Name, namespace, name)
		}
		expectedReclaim := corev1.PersistentVolumeReclaimRetain
		if workspaceRetentionPolicy(tenant.Spec.Workspace) == fabricv1alpha1.WorkspaceRetentionDelete {
			expectedReclaim = corev1.PersistentVolumeReclaimDelete
		}
		if pv.Spec.PersistentVolumeReclaimPolicy != expectedReclaim {
			return fmt.Errorf("workspace PV %q has unexpected reclaim policy", pv.Name)
		}
	}
	return nil
}

func workspaceMountPath(scope workspaceScope) string {
	root := "/workspace/shared"
	if scope.Domain == "skills" {
		root = "/workspace/skills"
	}
	switch scope.Scope {
	case "organization":
		return path.Join(root, "organization", scope.Mode)
	case "group":
		return path.Join(root, "groups", scope.Key, scope.Mode)
	case "user":
		return path.Join(root, "users", scope.Key, scope.Mode)
	default:
		return path.Join(root, scope.Scope, scope.Key, scope.Mode)
	}
}
