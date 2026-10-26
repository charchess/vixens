package controller

import (
	"context"
	"fmt"
	"path"

	fabricv1alpha1 "github.com/charchess/vixens/apps/60-services/txo-fabric/operator/api/v1alpha1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

// ensureWorkspaceMigrationEvidence is the fail-closed cutover gate. The resolver
// can select a target only after each workspace mounted by this agent has a Bound
// target and a completed UID-bound, GitOps-declared copy evidence Job.
func (r *AgentIdentityReconciler) ensureWorkspaceMigrationEvidence(ctx context.Context, agent *fabricv1alpha1.AgentIdentity, tenant *fabricv1alpha1.TenantBundle) error {
	workspace := tenant.Spec.Workspace
	if workspace == nil || workspace.Migration == nil || !workspace.Migration.Cutover {
		return nil
	}
	migration := workspace.Migration
	if migration.CopyJobPrefix == "" {
		return fmt.Errorf("TenantBundle %q requests workspace cutover without copyJobPrefix", tenant.Name)
	}
	_, _, err := resolvedWorkspaceVolumes(agent, tenant)
	if err != nil {
		return err
	}
	// Reconstruct only the exact scopes this agent consumes; use the canonical
	// resolver names so proof cannot be redirected by an arbitrary Job name.
	groups := map[string]fabricv1alpha1.NamedWorkspaceScopeSpec{}
	for _, g := range workspace.Groups {
		groups[g.Name] = g
	}
	users := map[string]fabricv1alpha1.NamedWorkspaceScopeSpec{}
	for _, u := range workspace.Users {
		users[u.Name] = u
	}
	scopes := []workspaceScope{}
	appendModes := func(domain, scope, key string, ref, collab bool) {
		if ref {
			scopes = append(scopes, workspaceScope{Domain: domain, Scope: scope, Key: key, Mode: "reference", ReadOnly: true})
		}
		if collab {
			scopes = append(scopes, workspaceScope{Domain: domain, Scope: scope, Key: key, Mode: "collaborative"})
		}
	}
	appendScope := func(scope, key string, ref, collab, sref, scollab bool) {
		appendModes("shared", scope, key, ref, collab)
		appendModes("skills", scope, key, sref, scollab)
	}
	o := workspace.Organization
	appendScope("organization", "organization", o.Reference, o.Collaborative, o.SkillsReference, o.SkillsCollaborative)
	for _, name := range agent.Spec.Access.Groups {
		if g, ok := groups[name]; ok {
			appendScope("group", g.Name, g.Reference, g.Collaborative, g.SkillsReference, g.SkillsCollaborative)
		}
	}
	if u, ok := users[agent.Spec.Access.UserRef]; agent.Spec.Access.UserRef != "" && ok {
		appendScope("user", u.Name, u.Reference, u.Collaborative, u.SkillsReference, u.SkillsCollaborative)
	}
	ns := tenantNamespace(tenant.Name)
	for _, scope := range scopes {
		sourceName := workspacePVCName(scope)
		targetName := workspaceMigrationTargetPVCName(scope, migration.TargetClaimSuffix)
		var source, target corev1.PersistentVolumeClaim
		if err := r.Get(ctx, types.NamespacedName{Namespace: ns, Name: sourceName}, &source); err != nil {
			return fmt.Errorf("read workspace source PVC %s/%s: %w", ns, sourceName, err)
		}
		if err := r.Get(ctx, types.NamespacedName{Namespace: ns, Name: targetName}, &target); err != nil {
			return fmt.Errorf("read workspace target PVC %s/%s: %w", ns, targetName, err)
		}
		if target.Status.Phase != corev1.ClaimBound {
			return fmt.Errorf("workspace target PVC %s/%s is not Bound", ns, targetName)
		}
		jobName := migration.CopyJobPrefix + "-" + sourceName
		var job batchv1.Job
		if err := r.Get(ctx, types.NamespacedName{Namespace: ns, Name: jobName}, &job); err != nil {
			return fmt.Errorf("read workspace copy Job %s/%s: %w", ns, jobName, err)
		}
		if job.Labels["fabric.truxonline.io/storage-migration-source-uid"] != string(source.UID) || job.Labels["fabric.truxonline.io/storage-migration-target-uid"] != string(target.UID) {
			return fmt.Errorf("workspace copy Job %s/%s does not bind exact source/target PVC UIDs", ns, jobName)
		}
		complete := false
		for _, c := range job.Status.Conditions {
			if c.Type == batchv1.JobComplete && c.Status == corev1.ConditionTrue {
				complete = true
			}
		}
		if !complete {
			return fmt.Errorf("workspace copy Job %s/%s has not completed successfully", ns, jobName)
		}
	}
	return nil
}

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
		if workspace.Migration != nil && workspace.Migration.Cutover {
			name = workspaceMigrationTargetPVCName(scope, workspace.Migration.TargetClaimSuffix)
		}
		mountPath := workspaceMountPath(scope)
		volumes = append(volumes, corev1.Volume{
			Name:         name,
			VolumeSource: corev1.VolumeSource{PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: name}},
		})
		mounts = append(mounts, corev1.VolumeMount{Name: name, MountPath: mountPath, ReadOnly: scope.ReadOnly})
	}
	return volumes, mounts, nil
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
