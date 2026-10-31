package controller

import (
	"context"
	"fmt"

	fabricv1alpha1 "github.com/charchess/vixens/apps/60-services/txo-fabric/operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// activeWorkspaceBindings requires an exact, complete mapping from each
// declared logical scope to its pinned physical PVC. No wildcard or fallback
// exists, and the map never grants an agent membership in a new scope.
func activeWorkspaceBindings(tenant *fabricv1alpha1.TenantBundle) (map[string]*fabricv1alpha1.ActiveVolumeBinding, error) {
	workspace := tenant.Spec.Workspace
	if workspace == nil || len(workspace.ActiveVolumes) == 0 {
		return nil, nil
	}
	if migration := workspace.Migration; migration != nil && !migration.Cutover {
		return nil, fmt.Errorf("TenantBundle %q cannot finalize RWX scopes while copy migration is in preparation", tenant.Name)
	}
	desired := make(map[string]struct{})
	for _, scope := range desiredWorkspaceScopes(tenant) {
		name := workspacePVCName(scope)
		if _, exists := desired[name]; exists {
			return nil, fmt.Errorf("TenantBundle %q declares duplicate logical workspace %q", tenant.Name, name)
		}
		desired[name] = struct{}{}
	}
	bindings := make(map[string]*fabricv1alpha1.ActiveVolumeBinding, len(workspace.ActiveVolumes))
	physicalNames := make(map[string]struct{})
	for i := range workspace.ActiveVolumes {
		b := &workspace.ActiveVolumes[i]
		if _, known := desired[b.ScopePVCName]; !known {
			return nil, fmt.Errorf("TenantBundle %q binds undeclared workspace scope %q", tenant.Name, b.ScopePVCName)
		}
		if _, duplicate := bindings[b.ScopePVCName]; duplicate {
			return nil, fmt.Errorf("TenantBundle %q has duplicate active workspace binding for %q", tenant.Name, b.ScopePVCName)
		}
		if _, duplicate := physicalNames[b.Volume.ClaimName]; duplicate {
			return nil, fmt.Errorf("TenantBundle %q reuses an active PVC across independent workspace scopes", tenant.Name)
		}
		physicalNames[b.Volume.ClaimName] = struct{}{}
		bindings[b.ScopePVCName] = &b.Volume
	}
	if len(bindings) != len(desired) {
		return nil, fmt.Errorf("TenantBundle %q has incomplete active workspace bindings: pinned %d of %d scopes", tenant.Name, len(bindings), len(desired))
	}
	return bindings, nil
}

func validateWorkspaceActiveVolumes(ctx context.Context, reader client.Reader, tenant *fabricv1alpha1.TenantBundle) error {
	bindings, err := activeWorkspaceBindings(tenant)
	if err != nil || bindings == nil {
		return err
	}
	namespace := tenantNamespace(tenant.Name)
	for _, scope := range desiredWorkspaceScopes(tenant) {
		binding := bindings[workspacePVCName(scope)]
		if err := validateActiveVolumeBinding(ctx, reader, namespace, binding,
			workspaceLabels(tenant, scope), corev1.ReadWriteMany,
			workspaceRetentionPolicy(tenant.Spec.Workspace), LabelWorkspaceRetention); err != nil {
			return fmt.Errorf("workspace %q: %w", workspacePVCName(scope), err)
		}
	}
	return nil
}
