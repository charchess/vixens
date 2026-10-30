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

	for _, scope := range desiredWorkspaceScopes(&bundle) {
		if migration := bundle.Spec.Workspace.Migration; migration != nil && migration.Cutover {
			if err := r.ensureWorkspaceMigrationTargetPVC(ctx, &bundle, &profile, scope); err != nil {
				return ctrl.Result{}, err
			}
			continue
		}
		if err := r.ensureWorkspacePVC(ctx, &bundle, &profile, scope); err != nil {
			return ctrl.Result{}, err
		}
		if err := r.ensureWorkspaceMigrationTargetPVC(ctx, &bundle, &profile, scope); err != nil {
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

func workspaceMigrationTargetPVCName(scope workspaceScope, suffix string) string {
	return workspacePVCName(scope) + "-" + suffix
}

func (r *SharedWorkspaceReconciler) ensureWorkspaceMigrationTargetPVC(ctx context.Context, bundle *fabricv1alpha1.TenantBundle, profile *fabricv1alpha1.SharedWorkspaceProfile, scope workspaceScope) error {
	migration := bundle.Spec.Workspace.Migration
	if migration == nil {
		return nil
	}
	sourceName := workspacePVCName(scope)
	var source corev1.PersistentVolumeClaim
	sourceErr := r.Get(ctx, client.ObjectKey{Namespace: tenantNamespace(bundle.Name), Name: sourceName}, &source)
	if sourceErr != nil && !apierrors.IsNotFound(sourceErr) {
		return fmt.Errorf("read workspace migration source PVC %s/%s: %w", tenantNamespace(bundle.Name), sourceName, sourceErr)
	}
	// A Pending source during cutover is a historical controller-created empty
	// replacement, not migration evidence. Treat it as retired so the Bound
	// target remains the active workspace and the historical source is never
	// recreated or re-adopted.
	retiredSource := apierrors.IsNotFound(sourceErr) || (sourceErr == nil && source.Status.Phase == corev1.ClaimPending)
	targetName := workspaceMigrationTargetPVCName(scope, migration.TargetClaimSuffix)
	key := client.ObjectKey{Namespace: tenantNamespace(bundle.Name), Name: targetName}
	var target corev1.PersistentVolumeClaim
	err := r.Get(ctx, key, &target)
	if apierrors.IsNotFound(err) {
		if sourceErr != nil {
			return fmt.Errorf("workspace target PVC %s/%s is absent and source is retired", key.Namespace, targetName)
		}
		labels := workspaceLabels(bundle, scope)
		labels["fabric.truxonline.io/storage-migration-source-uid"] = string(source.UID)
		labels["fabric.truxonline.io/storage-migration-target"] = "true"
		target = corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{Name: targetName, Namespace: key.Namespace, Labels: labels}, Spec: corev1.PersistentVolumeClaimSpec{AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteMany}, StorageClassName: stringPtr(migration.TargetStorageClassName), Resources: corev1.VolumeResourceRequirements{Requests: corev1.ResourceList{corev1.ResourceStorage: profile.Spec.Size}}}}
		return r.Create(ctx, &target)
	}
	if err != nil {
		return err
	}
	if valueOrEmpty(target.Spec.StorageClassName) != migration.TargetStorageClassName {
		return fmt.Errorf("workspace migration target PVC %s/%s uses immutable storageClass %q, expected %q", key.Namespace, targetName, valueOrEmpty(target.Spec.StorageClassName), migration.TargetStorageClassName)
	}
	if migration.Cutover && retiredSource {
		if target.Status.Phase != corev1.ClaimBound {
			return fmt.Errorf("finalized workspace target PVC %s/%s is not Bound", key.Namespace, targetName)
		}
		for label, expected := range workspaceLabels(bundle, scope) {
			if target.Labels[label] != expected {
				return fmt.Errorf("finalized workspace target PVC %s/%s has mismatched %s label", key.Namespace, targetName, label)
			}
		}
		if target.Labels["fabric.truxonline.io/storage-migration-source-uid"] == "" {
			return fmt.Errorf("finalized workspace target PVC %s/%s has no historical source UID", key.Namespace, targetName)
		}
		return nil
	}
	if sourceErr != nil {
		return fmt.Errorf("read workspace migration source PVC %s/%s: %w", key.Namespace, sourceName, sourceErr)
	}
	if target.Labels["fabric.truxonline.io/storage-migration-source-uid"] != string(source.UID) {
		return fmt.Errorf("workspace migration target PVC %s/%s is not bound to source PVC UID %q", key.Namespace, targetName, source.UID)
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
