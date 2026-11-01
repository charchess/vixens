package controller

import (
	"context"
	"fmt"
	"time"

	fabricv1alpha1 "github.com/charchess/vixens/apps/60-services/txo-fabric/operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	LabelWorkspaceDomain    = "fabric.truxonline.io/workspace-domain"
	LabelWorkspaceScope     = "fabric.truxonline.io/workspace-scope"
	LabelWorkspaceKey       = "fabric.truxonline.io/workspace-key"
	LabelWorkspaceMode      = "fabric.truxonline.io/workspace-mode"
	LabelWorkspaceRetention = "fabric.truxonline.io/workspace-retention"
)

type SharedWorkspaceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

type workspaceScope struct {
	Domain   string
	Scope    string
	Key      string
	Mode     string
	ReadOnly bool
}

// +kubebuilder:rbac:groups=fabric.truxonline.io,resources=tenantbundles;sharedworkspaceprofiles,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete

func (r *SharedWorkspaceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var bundle fabricv1alpha1.TenantBundle
	if err := r.Get(ctx, req.NamespacedName, &bundle); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if bundle.Spec.Workspace == nil || !bundle.DeletionTimestamp.IsZero() {
		return r.reconcileWorkspaceRelease(ctx, &bundle)
	}

	var profile fabricv1alpha1.SharedWorkspaceProfile
	if err := r.Get(ctx, client.ObjectKey{Name: bundle.Spec.Workspace.ProfileRef}, &profile); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{RequeueAfter: 15 * time.Second}, nil
		}
		return ctrl.Result{}, err
	}
	if profile.Spec.StorageClassName == "" {
		return ctrl.Result{}, fmt.Errorf("SharedWorkspaceProfile %q has empty storageClassName", profile.Name)
	}

	if err := validateWorkspaceClaimNames(&bundle); err != nil {
		return ctrl.Result{}, err
	}
	for _, scope := range desiredWorkspaceScopes(&bundle) {
		if err := r.ensureWorkspacePVC(ctx, &bundle, &profile, scope); err != nil {
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{}, nil
}

func (r *SharedWorkspaceReconciler) reconcileWorkspaceRelease(ctx context.Context, bundle *fabricv1alpha1.TenantBundle) (ctrl.Result, error) {
	var pvcs corev1.PersistentVolumeClaimList
	if err := r.List(ctx, &pvcs, client.InNamespace(tenantNamespace(bundle.Name)), client.MatchingLabels{
		LabelPartOf:     "txo-fabric",
		LabelTenantName: bundle.Name,
	}); err != nil {
		return ctrl.Result{}, err
	}

	pending := false
	for i := range pvcs.Items {
		pvc := &pvcs.Items[i]
		if pvc.Labels[LabelWorkspaceScope] == "" {
			continue
		}
		retention := pvc.Labels[LabelWorkspaceRetention]
		if retention == "" && bundle.Spec.Workspace != nil {
			retention = workspaceRetentionPolicy(bundle.Spec.Workspace)
		}
		if retention != fabricv1alpha1.WorkspaceRetentionDelete {
			continue
		}
		pending = true
		if pvc.DeletionTimestamp.IsZero() {
			if err := r.Delete(ctx, pvc); err != nil && !apierrors.IsNotFound(err) {
				return ctrl.Result{}, err
			}
		}
	}
	if pending {
		return ctrl.Result{RequeueAfter: 2 * time.Second}, nil
	}
	return ctrl.Result{}, nil
}

func (r *SharedWorkspaceReconciler) ensureWorkspacePVC(ctx context.Context, bundle *fabricv1alpha1.TenantBundle, profile *fabricv1alpha1.SharedWorkspaceProfile, scope workspaceScope) error {
	if _, existing := bundle.Spec.Workspace.ClaimNames[workspacePVCName(scope)]; existing {
		return verifyExistingWorkspacePVC(ctx, r.Client, bundle, profile, scope)
	}
	name := workspacePVCName(scope)
	key := client.ObjectKey{Namespace: tenantNamespace(bundle.Name), Name: name}
	var pvc corev1.PersistentVolumeClaim
	err := r.Get(ctx, key, &pvc)
	if apierrors.IsNotFound(err) {
		size := profile.Spec.Size
		if size.IsZero() {
			size = resource.MustParse("5Gi")
		}
		pvc = corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: key.Namespace, Labels: workspaceLabels(bundle, scope)},
			Spec: corev1.PersistentVolumeClaimSpec{
				AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteMany},
				StorageClassName: stringPtr(profile.Spec.StorageClassName),
				Resources:        corev1.VolumeResourceRequirements{Requests: corev1.ResourceList{corev1.ResourceStorage: size}},
			},
		}
		return r.Create(ctx, &pvc)
	}
	if err != nil {
		return err
	}
	if pvc.Spec.StorageClassName == nil || *pvc.Spec.StorageClassName != profile.Spec.StorageClassName {
		return fmt.Errorf("shared workspace PVC %s/%s uses storageClass %q; profile %q requires %q", key.Namespace, name, valueOrEmpty(pvc.Spec.StorageClassName), profile.Name, profile.Spec.StorageClassName)
	}
	if len(pvc.Spec.AccessModes) != 1 || pvc.Spec.AccessModes[0] != corev1.ReadWriteMany {
		return fmt.Errorf("shared workspace PVC %s/%s must use ReadWriteMany", key.Namespace, name)
	}
	_, err = controllerutil.CreateOrUpdate(ctx, r.Client, &pvc, func() error {
		pvc.Labels = mergeStringMap(pvc.Labels, workspaceLabels(bundle, scope))
		return nil
	})
	return err
}

// Existing suffixed claims are physical business data. Never create, adopt,
// re-label or overwrite them when the selected claim is missing or wrong.
func verifyExistingWorkspacePVC(ctx context.Context, reader client.Reader, bundle *fabricv1alpha1.TenantBundle, profile *fabricv1alpha1.SharedWorkspaceProfile, scope workspaceScope) error {
	name := workspaceClaimName(scope, bundle.Spec.Workspace)
	ns := tenantNamespace(bundle.Name)
	var pvc corev1.PersistentVolumeClaim
	if err := reader.Get(ctx, client.ObjectKey{Namespace: ns, Name: name}, &pvc); err != nil {
		return fmt.Errorf("read active workspace PVC %s/%s: %w", ns, name, err)
	}
	if pvc.UID == "" || pvc.Status.Phase != corev1.ClaimBound || !pvc.DeletionTimestamp.IsZero() ||
		pvc.Spec.VolumeName == "" || valueOrEmpty(pvc.Spec.StorageClassName) != profile.Spec.StorageClassName ||
		len(pvc.Spec.AccessModes) != 1 || pvc.Spec.AccessModes[0] != corev1.ReadWriteMany {
		return fmt.Errorf("active workspace PVC %s/%s is not a valid Bound RWX claim", ns, name)
	}
	for k, expected := range workspaceLabels(bundle, scope) {
		if pvc.Labels[k] != expected {
			return fmt.Errorf("active workspace PVC %s/%s has mismatched tenant/scope label %q", ns, name, k)
		}
	}
	var pv corev1.PersistentVolume
	if err := reader.Get(ctx, client.ObjectKey{Name: pvc.Spec.VolumeName}, &pv); err != nil {
		return fmt.Errorf("read active workspace PV %q: %w", pvc.Spec.VolumeName, err)
	}
	if pv.Spec.ClaimRef == nil || pv.Spec.ClaimRef.Namespace != ns || pv.Spec.ClaimRef.Name != name ||
		pv.Spec.ClaimRef.UID != pvc.UID || pv.Status.Phase != corev1.VolumeBound {
		return fmt.Errorf("active workspace PV %q has wrong claimRef or binding", pv.Name)
	}
	reclaim := corev1.PersistentVolumeReclaimRetain
	if workspaceRetentionPolicy(bundle.Spec.Workspace) == fabricv1alpha1.WorkspaceRetentionDelete {
		reclaim = corev1.PersistentVolumeReclaimDelete
	}
	if pv.Spec.PersistentVolumeReclaimPolicy != reclaim {
		return fmt.Errorf("active workspace PV %q has unexpected reclaim policy", pv.Name)
	}
	return nil
}

// ClaimNames selects a concrete PVC for an existing scope. All unspecified
// scopes use canonical names and ordinary auto-provisioning.
func workspaceClaimName(scope workspaceScope, workspace *fabricv1alpha1.TenantWorkspaceSpec) string {
	name := workspacePVCName(scope)
	if workspace != nil {
		if selected, ok := workspace.ClaimNames[name]; ok {
			return selected
		}
	}
	return name
}

// Validate references before any PVC creation or agent rollout. Deprecated
// claimSuffix must not silently fall back to an empty workspace.
func validateWorkspaceClaimNames(bundle *fabricv1alpha1.TenantBundle) error {
	workspace := bundle.Spec.Workspace
	if workspace == nil {
		return nil
	}
	if workspace.ClaimSuffix != "" && len(workspace.ClaimNames) == 0 {
		return fmt.Errorf("tenant %q declares legacy workspace claimSuffix without explicit claimNames", bundle.Name)
	}
	declared := make(map[string]struct{})
	for _, scope := range desiredWorkspaceScopes(bundle) {
		declared[workspacePVCName(scope)] = struct{}{}
	}
	selected := make(map[string]string)
	for canonical, actual := range workspace.ClaimNames {
		if _, ok := declared[canonical]; !ok {
			return fmt.Errorf("tenant %q references undeclared workspace scope %q", bundle.Name, canonical)
		}
		if problems := validation.IsDNS1123Label(actual); len(problems) != 0 {
			return fmt.Errorf("tenant %q workspace PVC %q has invalid name: %v", bundle.Name, actual, problems)
		}
		if prior, duplicate := selected[actual]; duplicate {
			return fmt.Errorf("tenant %q workspace scopes %q and %q select the same PVC %q", bundle.Name, prior, canonical, actual)
		}
		if canonical != actual {
			if _, collides := declared[actual]; collides {
				return fmt.Errorf("tenant %q workspace PVC %q collides with another declared scope", bundle.Name, actual)
			}
		}
		selected[actual] = canonical
	}
	return nil
}

func workspaceLabels(bundle *fabricv1alpha1.TenantBundle, scope workspaceScope) map[string]string {
	labels := tenantLabels(bundle)
	labels[LabelWorkspaceDomain] = scope.Domain
	labels[LabelWorkspaceScope] = scope.Scope
	labels[LabelWorkspaceKey] = scope.Key
	labels[LabelWorkspaceMode] = scope.Mode
	labels[LabelWorkspaceRetention] = workspaceRetentionPolicy(bundle.Spec.Workspace)
	return labels
}

func workspaceRetentionPolicy(spec *fabricv1alpha1.TenantWorkspaceSpec) string {
	if spec != nil && spec.RetentionPolicy == fabricv1alpha1.WorkspaceRetentionDelete {
		return fabricv1alpha1.WorkspaceRetentionDelete
	}
	return fabricv1alpha1.WorkspaceRetentionRetain
}

func desiredWorkspaceScopes(bundle *fabricv1alpha1.TenantBundle) []workspaceScope {
	if bundle.Spec.Workspace == nil {
		return nil
	}
	result := make([]workspaceScope, 0)
	appendModes := func(domain, scope, key string, reference, collaborative bool) {
		if reference {
			result = append(result, workspaceScope{Domain: domain, Scope: scope, Key: key, Mode: "reference", ReadOnly: true})
		}
		if collaborative {
			result = append(result, workspaceScope{Domain: domain, Scope: scope, Key: key, Mode: "collaborative"})
		}
	}
	organization := bundle.Spec.Workspace.Organization
	appendModes("shared", "organization", "organization", organization.Reference, organization.Collaborative)
	appendModes("skills", "organization", "organization", organization.SkillsReference, organization.SkillsCollaborative)
	for _, group := range bundle.Spec.Workspace.Groups {
		appendModes("shared", "group", group.Name, group.Reference, group.Collaborative)
		appendModes("skills", "group", group.Name, group.SkillsReference, group.SkillsCollaborative)
	}
	for _, user := range bundle.Spec.Workspace.Users {
		appendModes("shared", "user", user.Name, user.Reference, user.Collaborative)
		appendModes("skills", "user", user.Name, user.SkillsReference, user.SkillsCollaborative)
	}
	return result
}

func workspacePVCName(scope workspaceScope) string {
	root := "ws"
	if scope.Domain == "skills" {
		root = "skills"
	}
	prefix := root + "-" + scope.Scope
	if scope.Scope == "organization" {
		prefix = root + "-org"
	} else {
		prefix += "-" + scope.Key
	}
	if scope.Mode == "reference" {
		return prefix + "-ref"
	}
	return prefix + "-rw"
}

func (r *SharedWorkspaceReconciler) requestsForWorkspaceProfile(ctx context.Context, obj client.Object) []reconcile.Request {
	var bundles fabricv1alpha1.TenantBundleList
	if err := r.List(ctx, &bundles); err != nil {
		return nil
	}
	requests := make([]reconcile.Request, 0)
	for i := range bundles.Items {
		workspace := bundles.Items[i].Spec.Workspace
		if workspace != nil && workspace.ProfileRef == obj.GetName() {
			requests = append(requests, reconcile.Request{NamespacedName: client.ObjectKey{Name: bundles.Items[i].Name}})
		}
	}
	return requests
}

func workspaceRequestForPVC(_ context.Context, obj client.Object) []reconcile.Request {
	labels := obj.GetLabels()
	if labels[LabelWorkspaceScope] == "" || labels[LabelTenantName] == "" {
		return nil
	}
	return []reconcile.Request{{NamespacedName: client.ObjectKey{Name: labels[LabelTenantName]}}}
}

func (r *SharedWorkspaceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		Named("sharedworkspace").
		For(&fabricv1alpha1.TenantBundle{}).
		Watches(&fabricv1alpha1.SharedWorkspaceProfile{}, handler.EnqueueRequestsFromMapFunc(r.requestsForWorkspaceProfile)).
		Watches(&corev1.PersistentVolumeClaim{}, handler.EnqueueRequestsFromMapFunc(workspaceRequestForPVC)).
		Complete(r)
}
