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

// Existing tenant workspaces are stateful, unlike disposable agent runtimes.
// Selecting an already-created physical claim must never provision a new one.
func activeWorkspaceTestFixture() (*fabricv1alpha1.TenantBundle, *fabricv1alpha1.SharedWorkspaceProfile, *corev1.PersistentVolumeClaim, *corev1.PersistentVolume) {
	tenant := workspaceTestTenant()
	tenant.Spec.Workspace.Organization = fabricv1alpha1.WorkspaceScopeSpec{Reference: true}
	tenant.Spec.Workspace.Groups = nil
	tenant.Spec.Workspace.Users = nil
	tenant.Spec.Workspace.ClaimSuffix = "synelia"
	profile := &fabricv1alpha1.SharedWorkspaceProfile{
		ObjectMeta: metav1.ObjectMeta{Name: "shared-nfs"},
		Spec: fabricv1alpha1.SharedWorkspaceProfileSpec{StorageClassName: "storage-rwx-retain"},
	}
	scope := desiredWorkspaceScopes(tenant)[0]
	name := workspaceClaimName(scope, tenant.Spec.Workspace)
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: tenantNamespace(tenant.Name), UID: types.UID("active-rwx-uid"), Labels: workspaceLabels(tenant, scope)},
		Spec: corev1.PersistentVolumeClaimSpec{
			StorageClassName: stringPtr("storage-rwx-retain"),
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteMany},
			VolumeName: "active-rwx-pv",
		},
		Status: corev1.PersistentVolumeClaimStatus{Phase: corev1.ClaimBound},
	}
	pv := &corev1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{Name: "active-rwx-pv"},
		Spec: corev1.PersistentVolumeSpec{
			ClaimRef: &corev1.ObjectReference{Namespace: pvc.Namespace, Name: pvc.Name, UID: pvc.UID},
			PersistentVolumeReclaimPolicy: corev1.PersistentVolumeReclaimRetain,
		},
		Status: corev1.PersistentVolumeStatus{Phase: corev1.VolumeBound},
	}
	return tenant, profile, pvc, pv
}

func TestActiveWorkspaceUsesExistingClaimWithoutMigration(t *testing.T) {
	ctx := context.Background()
	tenant, profile, pvc, pv := activeWorkspaceTestFixture()
	scheme := testScheme(t)
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(tenant, profile, pvc, pv).Build()
	r := &SharedWorkspaceReconciler{Client: c, Scheme: scheme}
	for i := 0; i < 2; i++ {
		if _, err := r.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{Name: tenant.Name}}); err != nil {
			t.Fatalf("reconcile #%d: %v", i+1, err)
		}
	}
	var after corev1.PersistentVolumeClaim
	if err := c.Get(ctx, types.NamespacedName{Namespace: pvc.Namespace, Name: pvc.Name}, &after); err != nil || after.UID != pvc.UID {
		t.Fatalf("active PVC was changed: %#v, %v", after, err)
	}
	var source corev1.PersistentVolumeClaim
	if err := c.Get(ctx, types.NamespacedName{Name: "ws-org-ref", Namespace: pvc.Namespace}, &source); err == nil {
		t.Fatal("historical source PVC was created")
	}
	agent := &fabricv1alpha1.AgentIdentity{Spec: fabricv1alpha1.AgentIdentitySpec{TenantRef: fabricv1alpha1.ObjectReference{Name: tenant.Name}, AgentKey: "probe"}}
	volumes, mounts, err := resolvedWorkspaceVolumes(agent, tenant)
	if err != nil || len(volumes) != 1 || len(mounts) != 1 || volumes[0].PersistentVolumeClaim.ClaimName != pvc.Name {
		t.Fatalf("workspace mount did not choose existing PVC: %+v %+v %v", volumes, mounts, err)
	}
}

func TestActiveWorkspaceRejectsMissingOrForeignOrInvalidClaims(t *testing.T) {
	cases := []struct {
		name string
		mutate func(*corev1.PersistentVolumeClaim, *corev1.PersistentVolume)
		missing bool
	}{
		{name: "missing-claim", missing: true},
		{name: "unbound", mutate: func(p *corev1.PersistentVolumeClaim, _ *corev1.PersistentVolume) { p.Status.Phase = corev1.ClaimPending }},
		{name: "wrong-tenant", mutate: func(p *corev1.PersistentVolumeClaim, _ *corev1.PersistentVolume) { p.Labels[LabelTenantName] = "foreign-tenant" }},
		{name: "wrong-scope", mutate: func(p *corev1.PersistentVolumeClaim, _ *corev1.PersistentVolume) { p.Labels[LabelWorkspaceScope] = "user" }},
		{name: "wrong-uid", mutate: func(p *corev1.PersistentVolumeClaim, _ *corev1.PersistentVolume) { p.UID = types.UID("replaced-uid") }},
		{name: "wrong-class", mutate: func(p *corev1.PersistentVolumeClaim, _ *corev1.PersistentVolume) { p.Spec.StorageClassName = stringPtr("other") }},
		{name: "wrong-access", mutate: func(p *corev1.PersistentVolumeClaim, _ *corev1.PersistentVolume) { p.Spec.AccessModes = []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce} }},
		{name: "wrong-retention", mutate: func(_ *corev1.PersistentVolumeClaim, p *corev1.PersistentVolume) { p.Spec.PersistentVolumeReclaimPolicy = corev1.PersistentVolumeReclaimDelete }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tenant, profile, pvc, pv := activeWorkspaceTestFixture()
			if tc.mutate != nil { tc.mutate(pvc, pv) }
			scheme := testScheme(t)
			objects := []client.Object{tenant, profile, pv}
			if !tc.missing { objects = append(objects, pvc) }
			c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(objects...).Build()
			r := &SharedWorkspaceReconciler{Client: c, Scheme: scheme}
			if _, err := r.Reconcile(context.Background(), ctrl.Request{NamespacedName: types.NamespacedName{Name: tenant.Name}}); err == nil {
				t.Fatal("invalid active workspace was accepted")
			}
			if tc.missing {
				var recreated corev1.PersistentVolumeClaim
				if err := c.Get(context.Background(), types.NamespacedName{Name: pvc.Name, Namespace: pvc.Namespace}, &recreated); err == nil {
					t.Fatal("recreated missing stateful active PVC")
				}
			}
		})
	}
}
