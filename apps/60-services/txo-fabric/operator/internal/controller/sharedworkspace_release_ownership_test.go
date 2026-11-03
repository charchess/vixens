package controller

import (
	"context"
	"testing"

	fabricv1alpha1 "github.com/charchess/vixens/apps/60-services/txo-fabric/operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// Release may run after spec.workspace has been removed. Only complete native
// tenant/scope identity, not a matching name or migration label, authorizes
// deletion of a disposable shared PVC.
func TestWorkspaceDeleteReleaseRequiresCompleteOwnership(t *testing.T) {
	scopes := []workspaceScope{
		{Domain: "shared", Scope: "organization", Key: "organization", Mode: "reference"},
		{Domain: "shared", Scope: "group", Key: "team", Mode: "collaborative"},
		{Domain: "skills", Scope: "group", Key: "team", Mode: "reference"},
		{Domain: "skills", Scope: "user", Key: "person", Mode: "collaborative"},
	}
	cases := []struct {
		name string
		mutate func(*corev1.PersistentVolumeClaim)
		wantDelete bool
	}{
		{name: "exact owner", wantDelete: true},
		{name: "other tenant ID", mutate: func(p *corev1.PersistentVolumeClaim) { p.Labels[LabelTenantID] = "TEN-foreign" }},
		{name: "missing tenant ID", mutate: func(p *corev1.PersistentVolumeClaim) { delete(p.Labels, LabelTenantID) }},
		{name: "missing tenant name", mutate: func(p *corev1.PersistentVolumeClaim) { delete(p.Labels, LabelTenantName) }},
		{name: "other tenant name", mutate: func(p *corev1.PersistentVolumeClaim) { p.Labels[LabelTenantName] = "foreign" }},
		{name: "missing part of", mutate: func(p *corev1.PersistentVolumeClaim) { delete(p.Labels, LabelPartOf) }},
		{name: "missing domain", mutate: func(p *corev1.PersistentVolumeClaim) { delete(p.Labels, LabelWorkspaceDomain) }},
		{name: "wrong domain", mutate: func(p *corev1.PersistentVolumeClaim) { p.Labels[LabelWorkspaceDomain] = "untrusted" }},
		{name: "missing scope", mutate: func(p *corev1.PersistentVolumeClaim) { delete(p.Labels, LabelWorkspaceScope) }},
		{name: "wrong scope", mutate: func(p *corev1.PersistentVolumeClaim) { p.Labels[LabelWorkspaceScope] = "untrusted" }},
		{name: "missing scope key", mutate: func(p *corev1.PersistentVolumeClaim) { delete(p.Labels, LabelWorkspaceKey) }},
		{name: "forged scope key", mutate: func(p *corev1.PersistentVolumeClaim) { p.Labels[LabelWorkspaceKey] = "other" }},
		{name: "missing mode", mutate: func(p *corev1.PersistentVolumeClaim) { delete(p.Labels, LabelWorkspaceMode) }},
		{name: "unrecognized mode", mutate: func(p *corev1.PersistentVolumeClaim) { p.Labels[LabelWorkspaceMode] = "writeall" }},
		{name: "retained volume", mutate: func(p *corev1.PersistentVolumeClaim) { p.Labels[LabelWorkspaceRetention] = fabricv1alpha1.WorkspaceRetentionRetain }},
		{name: "missing retention", mutate: func(p *corev1.PersistentVolumeClaim) { delete(p.Labels, LabelWorkspaceRetention) }},
		{name: "noncanonical name", mutate: func(p *corev1.PersistentVolumeClaim) { p.Name += "-foreign" }},
	}
	for _, tenantName := range []string{"generic-client-one", "generic-client-two"} {
		for _, scope := range scopes {
			for _, tc := range cases {
				t.Run(tenantName+"/"+workspacePVCName(scope)+"/"+tc.name, func(t *testing.T) {
					ctx := context.Background()
					tenant := workspaceTestTenant()
					tenant.Name = tenantName
					tenant.Spec.TenantID = "TEN-" + tenantName
					tenant.Spec.Workspace.RetentionPolicy = fabricv1alpha1.WorkspaceRetentionDelete
					name := workspacePVCName(scope)
					pvc := &corev1.PersistentVolumeClaim{
						ObjectMeta: metav1.ObjectMeta{
							Name: name,
							Namespace: tenantNamespace(tenant.Name),
							Labels: workspaceLabels(tenant, scope),
						},
					}
					if tc.mutate != nil {
						tc.mutate(pvc)
					}
					// Release also runs when the storage declaration is gone.
					tenant.Spec.Workspace = nil
					scheme := testScheme(t)
					c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(tenant, pvc).Build()
					r := &SharedWorkspaceReconciler{Client: c, Scheme: scheme}
					req := ctrl.Request{NamespacedName: types.NamespacedName{Name: tenant.Name}}
					for attempt := 0; attempt < 2; attempt++ {
						if _, err := r.Reconcile(ctx, req); err != nil {
							t.Fatalf("release %d: %v", attempt, err)
						}
					}
					var remaining corev1.PersistentVolumeClaim
					err := c.Get(ctx, types.NamespacedName{Name: pvc.Name, Namespace: pvc.Namespace}, &remaining)
					if tc.wantDelete && err == nil {
						t.Fatal("owned disposable workspace PVC should be deleted")
					}
					if !tc.wantDelete && err != nil {
						t.Fatalf("foreign or unverified workspace PVC was deleted: %v", err)
					}
				})
			}
		}
	}
}

func TestWorkspaceReleaseKeepsClaimsForTenantWithoutIdentity(t *testing.T) {
	tenant := workspaceTestTenant()
	tenant.Spec.TenantID = ""
	tenant.Spec.Workspace.RetentionPolicy = fabricv1alpha1.WorkspaceRetentionDelete
	scope := workspaceScope{Domain: "shared", Scope: "organization", Key: "organization", Mode: "collaborative"}
	pvc := &corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{
		Name: workspacePVCName(scope), Namespace: tenantNamespace(tenant.Name),
		Labels: workspaceLabels(tenant, scope),
	}}
	tenant.Spec.Workspace = nil
	c := fake.NewClientBuilder().WithScheme(testScheme(t)).WithObjects(tenant, pvc).Build()
	r := &SharedWorkspaceReconciler{Client: c, Scheme: testScheme(t)}
	if _, err := r.Reconcile(context.Background(), ctrl.Request{NamespacedName: types.NamespacedName{Name: tenant.Name}}); err != nil {
		t.Fatal(err)
	}
	var remaining corev1.PersistentVolumeClaim
	if err := c.Get(context.Background(), types.NamespacedName{Name: pvc.Name, Namespace: pvc.Namespace}, &remaining); err != nil {
		t.Fatalf("missing tenant identity permitted deletion: %v", err)
	}
}
