package controller

import (
	"context"
	"strings"
	"testing"

	fabricv1alpha1 "github.com/charchess/vixens/apps/60-services/txo-fabric/operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestSharedWorkspaceReconcilesIsolatedRWXPVCs(t *testing.T) {
	ctx := context.Background()
	scheme := testScheme(t)
	tenant := workspaceTestTenant()
	profile := &fabricv1alpha1.SharedWorkspaceProfile{
		ObjectMeta: metav1.ObjectMeta{Name: "shared-nfs"},
		Spec: fabricv1alpha1.SharedWorkspaceProfileSpec{
			StorageClassName: "storage-rwx-retain",
			Size:             resource.MustParse("5Gi"),
		},
	}
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(tenant, profile).Build()

	r := &SharedWorkspaceReconciler{Client: c, Scheme: scheme}
	if _, err := r.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{Name: tenant.Name}}); err != nil {
		t.Fatal(err)
	}

	checks := map[string]struct {
		domain string
		scope  string
		key    string
		mode   string
	}{
		"ws-org-ref":            {domain: "shared", scope: "organization", key: "organization", mode: "reference"},
		"ws-org-rw":             {domain: "shared", scope: "organization", key: "organization", mode: "collaborative"},
		"ws-group-sales-ref":    {domain: "shared", scope: "group", key: "sales", mode: "reference"},
		"ws-group-sales-rw":     {domain: "shared", scope: "group", key: "sales", mode: "collaborative"},
		"ws-user-bertrand-rw":   {domain: "shared", scope: "user", key: "bertrand", mode: "collaborative"},
		"skills-org-ref":        {domain: "skills", scope: "organization", key: "organization", mode: "reference"},
		"skills-group-sales-rw": {domain: "skills", scope: "group", key: "sales", mode: "collaborative"},
	}
	for name, want := range checks {
		var pvc corev1.PersistentVolumeClaim
		if err := c.Get(ctx, types.NamespacedName{Name: name, Namespace: "tenant-fabric-smoke"}, &pvc); err != nil {
			t.Fatalf("workspace PVC %s missing: %v", name, err)
		}
		if len(pvc.Spec.AccessModes) != 1 || pvc.Spec.AccessModes[0] != corev1.ReadWriteMany {
			t.Fatalf("PVC %s accessModes = %#v, want RWX", name, pvc.Spec.AccessModes)
		}
		if pvc.Spec.StorageClassName == nil || *pvc.Spec.StorageClassName != "storage-rwx-retain" {
			t.Fatalf("PVC %s storageClass = %#v", name, pvc.Spec.StorageClassName)
		}
		if pvc.Labels[LabelWorkspaceDomain] != want.domain || pvc.Labels[LabelWorkspaceScope] != want.scope || pvc.Labels[LabelWorkspaceKey] != want.key || pvc.Labels[LabelWorkspaceMode] != want.mode {
			t.Fatalf("PVC %s workspace labels = %#v", name, pvc.Labels)
		}
		if pvc.Labels[LabelWorkspaceRetention] != fabricv1alpha1.WorkspaceRetentionRetain {
			t.Fatalf("PVC %s retention = %q", name, pvc.Labels[LabelWorkspaceRetention])
		}
		if len(pvc.OwnerReferences) != 0 {
			t.Fatalf("shared PVC %s must not be owned by an AgentIdentity/TenantBundle: %#v", name, pvc.OwnerReferences)
		}
	}
}

func TestResolvedWorkspaceVolumesEnforcesScopesAndReadOnly(t *testing.T) {
	tenant := workspaceTestTenant()
	agent := &fabricv1alpha1.AgentIdentity{
		ObjectMeta: metav1.ObjectMeta{Name: "fabric-smoke-sales-probe"},
		Spec: fabricv1alpha1.AgentIdentitySpec{
			TenantRef:   fabricv1alpha1.ObjectReference{Name: tenant.Name},
			AgentKey:    "sales-probe",
			DisplayName: "Sales Probe",
			Access: fabricv1alpha1.AgentAccessSpec{
				UserRef: "bertrand",
				Groups:  []string{"sales"},
			},
		},
	}

	volumes, mounts, err := resolvedWorkspaceVolumes(agent, tenant)
	if err != nil {
		t.Fatal(err)
	}
	if len(volumes) != 7 || len(mounts) != 7 {
		t.Fatalf("resolved volumes=%d mounts=%d, want 7/7", len(volumes), len(mounts))
	}

	want := map[string]bool{
		"/workspace/shared/organization/reference":       true,
		"/workspace/shared/organization/collaborative":   false,
		"/workspace/shared/groups/sales/reference":       true,
		"/workspace/shared/groups/sales/collaborative":   false,
		"/workspace/shared/users/bertrand/collaborative": false,
		"/workspace/skills/organization/reference":       true,
		"/workspace/skills/groups/sales/collaborative":   false,
	}
	for _, mount := range mounts {
		readOnly, ok := want[mount.MountPath]
		if !ok {
			t.Fatalf("unexpected workspace mount %q", mount.MountPath)
		}
		if mount.ReadOnly != readOnly {
			t.Fatalf("mount %s readOnly=%v, want %v", mount.MountPath, mount.ReadOnly, readOnly)
		}
		delete(want, mount.MountPath)
	}
	if len(want) != 0 {
		t.Fatalf("missing mounts: %#v", want)
	}
}

func TestResolvedWorkspaceVolumesRejectsUndeclaredGroup(t *testing.T) {
	tenant := workspaceTestTenant()
	agent := &fabricv1alpha1.AgentIdentity{
		ObjectMeta: metav1.ObjectMeta{Name: "fabric-smoke-tech-probe"},
		Spec: fabricv1alpha1.AgentIdentitySpec{
			TenantRef: fabricv1alpha1.ObjectReference{Name: tenant.Name}, AgentKey: "tech-probe", DisplayName: "Tech Probe",
			Access: fabricv1alpha1.AgentAccessSpec{Groups: []string{"finance"}},
		},
	}
	if _, _, err := resolvedWorkspaceVolumes(agent, tenant); err == nil {
		t.Fatal("undeclared workspace group must be rejected")
	}
}

func TestHermesDeploymentMountsOnlyAuthorizedWorkspaceScopes(t *testing.T) {
	ctx := context.Background()
	scheme := testScheme(t)
	tenant := workspaceTestTenant()
	profile := testRuntimeProfile()
	agent := &fabricv1alpha1.AgentIdentity{
		ObjectMeta: metav1.ObjectMeta{Name: "fabric-smoke-sales-probe"},
		Spec: fabricv1alpha1.AgentIdentitySpec{
			TenantRef: fabricv1alpha1.ObjectReference{Name: tenant.Name}, AgentKey: "sales-probe", DisplayName: "Sales Probe",
			Access: fabricv1alpha1.AgentAccessSpec{Groups: []string{"sales"}},
		},
	}
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(tenant, profile, agent).Build()
	r := &AgentIdentityReconciler{Client: c, Scheme: scheme}
	if err := r.ensureDeployment(ctx, agent, tenant, profile, "tenant-fabric-smoke", "", ""); err != nil {
		t.Fatal(err)
	}
	var deployment appsv1.Deployment
	if err := c.Get(ctx, types.NamespacedName{Name: "hermes-sales-probe", Namespace: "tenant-fabric-smoke"}, &deployment); err != nil {
		t.Fatal(err)
	}
	mounts := deployment.Spec.Template.Spec.Containers[0].VolumeMounts
	if !hasMount(mounts, "/workspace/shared/groups/sales/reference", true) {
		t.Fatal("sales reference mount missing or not read-only")
	}
	if !hasMount(mounts, "/workspace/shared/groups/sales/collaborative", false) {
		t.Fatal("sales collaborative mount missing or unexpectedly read-only")
	}
	if !hasMount(mounts, "/workspace/skills/organization/reference", true) {
		t.Fatal("organization skill reference mount missing or not read-only")
	}
	if !hasMount(mounts, "/workspace/skills/groups/sales/collaborative", false) {
		t.Fatal("sales collaborative skill mount missing or unexpectedly read-only")
	}
	for _, mount := range mounts {
		if strings.Contains(mount.MountPath, "/groups/tech/") {
			t.Fatalf("sales agent must not receive tech mount: %#v", mount)
		}
	}

	bootstrap := deployment.Spec.Template.Spec.InitContainers[0]
	if len(bootstrap.Command) != 3 || !strings.Contains(bootstrap.Command[2], `config set skills.external_dirs '["/workspace/skills"]'`) {
		t.Fatalf("bootstrap command does not configure Hermes shared skill root as a list: %#v", bootstrap.Command)
	}
}

func hasMount(mounts []corev1.VolumeMount, path string, readOnly bool) bool {
	for _, mount := range mounts {
		if mount.MountPath == path && mount.ReadOnly == readOnly {
			return true
		}
	}
	return false
}

func workspaceTestTenant() *fabricv1alpha1.TenantBundle {
	return &fabricv1alpha1.TenantBundle{
		ObjectMeta: metav1.ObjectMeta{Name: "fabric-smoke"},
		Spec: fabricv1alpha1.TenantBundleSpec{
			TenantID:    "TEN90002",
			DisplayName: "Fabric Smoke",
			Workspace: &fabricv1alpha1.TenantWorkspaceSpec{
				ProfileRef:      "shared-nfs",
				RetentionPolicy: fabricv1alpha1.WorkspaceRetentionRetain,
				Organization: fabricv1alpha1.WorkspaceScopeSpec{
					Reference: true, Collaborative: true, SkillsReference: true,
				},
				Groups: []fabricv1alpha1.NamedWorkspaceScopeSpec{
					{Name: "sales", Reference: true, Collaborative: true, SkillsCollaborative: true},
					{Name: "tech", Reference: true},
				},
				Users: []fabricv1alpha1.NamedWorkspaceScopeSpec{
					{Name: "bertrand", Collaborative: true},
				},
			},
		},
	}
}

// A TenantBundle workspace declaration is sufficient to provision canonical
// RWX claims on a completely empty cluster; no import or manual PVC is needed.
func TestNativeWorkspaceProvisionsCanonicalClaimsForNewTenants(t *testing.T) {
	for _, name := range []string{"fabric-smoke", "third-tenant", "hairem", "indiba"} {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			tenant := workspaceTestTenant()
			tenant.Name = name
			profile := &fabricv1alpha1.SharedWorkspaceProfile{
				ObjectMeta: metav1.ObjectMeta{Name: "shared-nfs"},
				Spec: fabricv1alpha1.SharedWorkspaceProfileSpec{
					StorageClassName: "storage-rwx-retain",
					Size: resource.MustParse("5Gi"),
				},
			}
			scheme := testScheme(t)
			c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(tenant, profile).Build()
			r := &SharedWorkspaceReconciler{Client: c, Scheme: scheme}
			for i := 0; i < 2; i++ {
				if _, err := r.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{Name: tenant.Name}}); err != nil {
					t.Fatalf("native reconcile %d: %v", i+1, err)
				}
			}
			for _, scope := range desiredWorkspaceScopes(tenant) {
				claimName := workspacePVCName(scope)
				var pvc corev1.PersistentVolumeClaim
				if err := c.Get(ctx, client.ObjectKey{Name: claimName, Namespace: tenantNamespace(tenant.Name)}, &pvc); err != nil {
					t.Fatalf("canonical workspace %q not created: %v", claimName, err)
				}
				if pvc.Labels[LabelTenantName] != tenant.Name ||
					pvc.Spec.StorageClassName == nil || *pvc.Spec.StorageClassName != profile.Spec.StorageClassName {
					t.Fatalf("workspace %q not tenant-owned with canonical class: %#v", claimName, pvc)
				}
			}
			agent := &fabricv1alpha1.AgentIdentity{Spec: fabricv1alpha1.AgentIdentitySpec{
				TenantRef: fabricv1alpha1.ObjectReference{Name: tenant.Name},
				AgentKey: "probe",
				Access: fabricv1alpha1.AgentAccessSpec{Groups: []string{"sales"}, UserRef: "bertrand"},
			}}
			volumes, _, err := resolvedWorkspaceVolumes(agent, tenant)
			if err != nil {
				t.Fatal(err)
			}
			for _, volume := range volumes {
				if strings.Contains(volume.PersistentVolumeClaim.ClaimName, "synelia") {
					t.Fatal("native workspace generated historical CSI claim name")
				}
			}
		})
	}
}

// A newly declared scope must be created by the existing PVC provisioner
// without replacing claims from a previous successful reconcile.
func TestNativeWorkspaceAddsNewScopeWithoutReplacingOldClaim(t *testing.T) {
	ctx := context.Background()
	scheme := testScheme(t)
	tenant := workspaceTestTenant()
	tenant.Spec.Workspace.Groups = nil
	tenant.Spec.Workspace.Users = nil
	profile := &fabricv1alpha1.SharedWorkspaceProfile{
		ObjectMeta: metav1.ObjectMeta{Name: "shared-nfs"},
		Spec: fabricv1alpha1.SharedWorkspaceProfileSpec{StorageClassName: "storage-rwx-retain"},
	}
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(tenant, profile).Build()
	r := &SharedWorkspaceReconciler{Client: c, Scheme: scheme}
	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: tenant.Name}}
	if _, err := r.Reconcile(ctx, req); err != nil {
		t.Fatal(err)
	}
	var existing corev1.PersistentVolumeClaim
	key := client.ObjectKey{Name: "ws-org-rw", Namespace: tenantNamespace(tenant.Name)}
	if err := c.Get(ctx, key, &existing); err != nil {
		t.Fatal(err)
	}
	existing.UID = types.UID("original-claim-uid")
	if err := c.Update(ctx, &existing); err != nil {
		t.Fatal(err)
	}
	tenant.Spec.Workspace.Groups = []fabricv1alpha1.NamedWorkspaceScopeSpec{
		{Name: "newteam", Collaborative: true},
	}
	if err := c.Update(ctx, tenant); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if _, err := r.Reconcile(ctx, req); err != nil {
			t.Fatalf("reconcile after new group: %v", err)
		}
	}
	var oldAfter corev1.PersistentVolumeClaim
	if err := c.Get(ctx, key, &oldAfter); err != nil || oldAfter.UID != existing.UID {
		t.Fatalf("previous claim was replaced: %v, %#v", err, oldAfter)
	}
	var newPVC corev1.PersistentVolumeClaim
	if err := c.Get(ctx, client.ObjectKey{Name: "ws-group-newteam-rw", Namespace: tenantNamespace(tenant.Name)}, &newPVC); err != nil {
		t.Fatalf("new scope should be provisioned automatically: %v", err)
	}
}

func TestNativeWorkspaceRejectsInvalidExistingCanonicalPVC(t *testing.T) {
	cases := []struct {
		name string
		mutate func(*corev1.PersistentVolumeClaim)
	}{
		{name: "foreign-tenant", mutate: func(p *corev1.PersistentVolumeClaim) { p.Labels[LabelTenantName] = "other-tenant" }},
		{name: "foreign-scope", mutate: func(p *corev1.PersistentVolumeClaim) { p.Labels[LabelWorkspaceScope] = "group" }},
		{name: "foreign-retention", mutate: func(p *corev1.PersistentVolumeClaim) { p.Labels[LabelWorkspaceRetention] = fabricv1alpha1.WorkspaceRetentionDelete }},
		{name: "wrong-class", mutate: func(p *corev1.PersistentVolumeClaim) { p.Spec.StorageClassName = stringPtr("foreign-class") }},
		{name: "not-rwx", mutate: func(p *corev1.PersistentVolumeClaim) { p.Spec.AccessModes = []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce} }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tenant := workspaceTestTenant()
			tenant.Spec.Workspace.Organization = fabricv1alpha1.WorkspaceScopeSpec{Reference: true}
			tenant.Spec.Workspace.Groups = nil
			tenant.Spec.Workspace.Users = nil
			scope := desiredWorkspaceScopes(tenant)[0]
			pvc := &corev1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{Name: workspacePVCName(scope), Namespace: tenantNamespace(tenant.Name), Labels: workspaceLabels(tenant, scope)},
				Spec: corev1.PersistentVolumeClaimSpec{
					StorageClassName: stringPtr("storage-rwx-retain"),
					AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteMany},
				},
			}
			tc.mutate(pvc)
			profile := &fabricv1alpha1.SharedWorkspaceProfile{ObjectMeta: metav1.ObjectMeta{Name: "shared-nfs"}, Spec: fabricv1alpha1.SharedWorkspaceProfileSpec{StorageClassName: "storage-rwx-retain"}}
			scheme := testScheme(t)
			c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(tenant, profile, pvc).Build()
			r := &SharedWorkspaceReconciler{Client: c, Scheme: scheme}
			if _, err := r.Reconcile(context.Background(), ctrl.Request{NamespacedName: types.NamespacedName{Name: tenant.Name}}); err == nil {
				t.Fatal("invalid canonical PVC was silently adopted")
			}
			var after corev1.PersistentVolumeClaim
			if err := c.Get(context.Background(), client.ObjectKey{Name: pvc.Name, Namespace: pvc.Namespace}, &after); err != nil {
				t.Fatal(err)
			}
			if after.Spec.StorageClassName == nil || *after.Spec.StorageClassName != *pvc.Spec.StorageClassName ||
				after.Labels[LabelTenantName] != pvc.Labels[LabelTenantName] {
				t.Fatal("invalid canonical PVC was relabeled or mutated")
			}
		})
	}
}
