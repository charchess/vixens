package controller

import (
	"fmt"
	"path"

	fabricv1alpha1 "github.com/charchess/vixens/apps/60-services/txo-fabric/operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
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
	appendModes := func(scope, key string, reference, collaborative bool) {
		if reference {
			scopes = append(scopes, workspaceScope{Scope: scope, Key: key, Mode: "reference", ReadOnly: true})
		}
		if collaborative {
			scopes = append(scopes, workspaceScope{Scope: scope, Key: key, Mode: "collaborative"})
		}
	}
	appendModes("organization", "organization", workspace.Organization.Reference, workspace.Organization.Collaborative)

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
		appendModes("group", group.Name, group.Reference, group.Collaborative)
	}

	if agent.Spec.Access.UserRef != "" {
		user, ok := users[agent.Spec.Access.UserRef]
		if !ok {
			return nil, nil, fmt.Errorf("AgentIdentity %q requests undeclared workspace user %q in tenant %q", agent.Name, agent.Spec.Access.UserRef, tenant.Name)
		}
		appendModes("user", user.Name, user.Reference, user.Collaborative)
	}

	volumes := make([]corev1.Volume, 0, len(scopes))
	mounts := make([]corev1.VolumeMount, 0, len(scopes))
	for _, scope := range scopes {
		name := workspacePVCName(scope)
		mountPath := workspaceMountPath(scope)
		volumes = append(volumes, corev1.Volume{
			Name: name,
			VolumeSource: corev1.VolumeSource{PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: name}},
		})
		mounts = append(mounts, corev1.VolumeMount{Name: name, MountPath: mountPath, ReadOnly: scope.ReadOnly})
	}
	return volumes, mounts, nil
}

func workspaceMountPath(scope workspaceScope) string {
	switch scope.Scope {
	case "organization":
		return path.Join("/workspace/shared/organization", scope.Mode)
	case "group":
		return path.Join("/workspace/shared/groups", scope.Key, scope.Mode)
	case "user":
		return path.Join("/workspace/shared/users", scope.Key, scope.Mode)
	default:
		return path.Join("/workspace/shared", scope.Scope, scope.Key, scope.Mode)
	}
}
