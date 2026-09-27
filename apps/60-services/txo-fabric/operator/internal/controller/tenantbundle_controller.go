package controller

import (
	"context"
	"fmt"
	"reflect"
	"time"

	fabricv1alpha1 "github.com/charchess/vixens/apps/60-services/txo-fabric/operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type TenantBundleReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=fabric.truxonline.io,resources=tenantbundles,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=fabric.truxonline.io,resources=tenantbundles/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=fabric.truxonline.io,resources=tenantbundles/finalizers,verbs=update
// +kubebuilder:rbac:groups=fabric.truxonline.io,resources=agentidentities,verbs=get;list;watch
// +kubebuilder:rbac:groups=fabric.truxonline.io,resources=postgresqlprofiles;hindsightprofiles,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=networking.k8s.io,resources=networkpolicies,verbs=get;list;watch;create;update;patch;delete

func (r *TenantBundleReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var bundle fabricv1alpha1.TenantBundle
	if err := r.Get(ctx, req.NamespacedName, &bundle); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if !bundle.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, &bundle)
	}

	previousStatus := bundle.DeepCopy().Status

	if !controllerutil.ContainsFinalizer(&bundle, TenantFinalizer) {
		controllerutil.AddFinalizer(&bundle, TenantFinalizer)
		if err := r.Update(ctx, &bundle); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	namespaceName := tenantNamespace(bundle.Name)
	if err := r.ensureNamespace(ctx, &bundle, namespaceName); err != nil {
		r.setFailedStatus(ctx, &bundle, "NamespaceReconcileFailed", err.Error())
		return ctrl.Result{}, err
	}
	if err := r.ensureDefaultDeny(ctx, &bundle, namespaceName); err != nil {
		r.setFailedStatus(ctx, &bundle, "NetworkReconcileFailed", err.Error())
		return ctrl.Result{}, err
	}

	bundle.Status.ObservedGeneration = bundle.Generation
	bundle.Status.Namespace = namespaceName
	bundle.Status.Persistence = fabricv1alpha1.TenantPersistenceStatus{}
	bundle.Status.Memory = fabricv1alpha1.TenantMemoryStatus{}
	bundle.Status.Modules = nil
	setCondition(&bundle.Status.Conditions, bundle.Generation, "NamespaceReady", metav1.ConditionTrue, "Reconciled", "tenant namespace is reconciled")
	setCondition(&bundle.Status.Conditions, bundle.Generation, "NetworkReady", metav1.ConditionTrue, "DefaultDenyReconciled", "tenant default-deny policy is reconciled")

	waiting := false
	postgresWaiting, err := r.reconcilePostgreSQLCapability(ctx, &bundle)
	if err != nil {
		r.setFailedStatus(ctx, &bundle, "PersistenceProfileResolveFailed", err.Error())
		return ctrl.Result{}, err
	}
	waiting = waiting || postgresWaiting

	hindsightWaiting, err := r.reconcileHindsightCapability(ctx, &bundle)
	if err != nil {
		r.setFailedStatus(ctx, &bundle, "MemoryProfileResolveFailed", err.Error())
		return ctrl.Result{}, err
	}
	waiting = waiting || hindsightWaiting

	enabledModules := 0
	bundle.Status.Modules = make([]fabricv1alpha1.TenantModuleStatus, 0, len(bundle.Spec.Modules))
	for _, module := range bundle.Spec.Modules {
		if !module.Enabled {
			continue
		}
		enabledModules++
		bundle.Status.Modules = append(bundle.Status.Modules, fabricv1alpha1.TenantModuleStatus{
			Name:            module.Name,
			ComponentStatus: fabricv1alpha1.ComponentStatus{Phase: "Pending", Message: "module reconciliation is not implemented by this controller version"},
		})
	}
	if enabledModules > 0 {
		waiting = true
		setCondition(&bundle.Status.Conditions, bundle.Generation, "ModulesReady", metav1.ConditionFalse, "ControllerNotImplemented", "one or more optional modules are waiting for a reconciler")
	} else {
		bundle.Status.Modules = nil
		setCondition(&bundle.Status.Conditions, bundle.Generation, "ModulesReady", metav1.ConditionTrue, "NotRequested", "tenant does not request optional modules")
	}

	if waiting {
		bundle.Status.Phase = "Provisioning"
		setCondition(&bundle.Status.Conditions, bundle.Generation, "Ready", metav1.ConditionFalse, "WaitingForCapabilities", "cell baseline is ready; declared persistence, memory, or modules are still pending")
	} else {
		bundle.Status.Phase = "Ready"
		setCondition(&bundle.Status.Conditions, bundle.Generation, "Ready", metav1.ConditionTrue, "Reconciled", "tenant cell baseline is ready")
	}

	if !reflect.DeepEqual(previousStatus, bundle.Status) {
		if err := r.Status().Update(ctx, &bundle); err != nil {
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{}, nil
}

func (r *TenantBundleReconciler) ensureNamespace(ctx context.Context, bundle *fabricv1alpha1.TenantBundle, name string) error {
	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: name}}
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, ns, func() error {
		ns.Labels = mergeStringMap(ns.Labels, tenantLabels(bundle))
		// Deliberately do not owner-reference the Namespace. Tenant deletion is
		// ordered by TenantFinalizer: AgentIdentity resources must disappear
		// before the namespace is deleted. A controller ownerReference would let
		// Kubernetes garbage collection race that lifecycle contract.
		return nil
	})
	return err
}

func (r *TenantBundleReconciler) ensureDefaultDeny(ctx context.Context, bundle *fabricv1alpha1.TenantBundle, namespace string) error {
	np := &networkingv1.NetworkPolicy{ObjectMeta: metav1.ObjectMeta{Name: "txo-fabric-default-deny", Namespace: namespace}}
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, np, func() error {
		np.Labels = mergeStringMap(np.Labels, tenantLabels(bundle))
		np.Labels["app.kubernetes.io/component"] = "tenant-network-baseline"
		np.Spec = networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{},
			PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeIngress, networkingv1.PolicyTypeEgress},
		}
		// The baseline must remain while AgentIdentity resources keep the tenant
		// finalizer blocked. Namespace deletion eventually removes it atomically
		// with the rest of the tenant cell.
		return nil
	})
	return err
}

func (r *TenantBundleReconciler) reconcileDelete(ctx context.Context, bundle *fabricv1alpha1.TenantBundle) (ctrl.Result, error) {
	if !controllerutil.ContainsFinalizer(bundle, TenantFinalizer) {
		return ctrl.Result{}, nil
	}

	var agents fabricv1alpha1.AgentIdentityList
	if err := r.List(ctx, &agents); err != nil {
		return ctrl.Result{}, err
	}
	remaining := 0
	for i := range agents.Items {
		if agents.Items[i].Spec.TenantRef.Name == bundle.Name {
			remaining++
		}
	}
	if remaining > 0 {
		bundle.Status.Phase = "Deleting"
		bundle.Status.ObservedGeneration = bundle.Generation
		setCondition(&bundle.Status.Conditions, bundle.Generation, "Ready", metav1.ConditionFalse, "AgentIdentitiesRemain", fmt.Sprintf("%d AgentIdentity resources still reference this tenant", remaining))
		_ = r.Status().Update(ctx, bundle)
		return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
	}

	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: tenantNamespace(bundle.Name)}}
	if err := r.Delete(ctx, ns); err != nil && !apierrors.IsNotFound(err) {
		return ctrl.Result{}, err
	}
	var current corev1.Namespace
	if err := r.Get(ctx, client.ObjectKey{Name: ns.Name}, &current); err == nil {
		return ctrl.Result{RequeueAfter: 2 * time.Second}, nil
	} else if !apierrors.IsNotFound(err) {
		return ctrl.Result{}, err
	}

	controllerutil.RemoveFinalizer(bundle, TenantFinalizer)
	if err := r.Update(ctx, bundle); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (r *TenantBundleReconciler) setFailedStatus(ctx context.Context, bundle *fabricv1alpha1.TenantBundle, reason, message string) {
	previousStatus := bundle.DeepCopy().Status
	bundle.Status.ObservedGeneration = bundle.Generation
	bundle.Status.Phase = "Degraded"
	setCondition(&bundle.Status.Conditions, bundle.Generation, "Ready", metav1.ConditionFalse, reason, message)
	if !reflect.DeepEqual(previousStatus, bundle.Status) {
		_ = r.Status().Update(ctx, bundle)
	}
}

func tenantBundleRequestsForManagedObject(_ context.Context, obj client.Object) []reconcile.Request {
	labels := obj.GetLabels()
	if labels[LabelManaged] != "true" {
		return nil
	}
	tenantName := labels[LabelTenantName]
	if tenantName == "" {
		return nil
	}
	return []reconcile.Request{{NamespacedName: types.NamespacedName{Name: tenantName}}}
}

func (r *TenantBundleReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&fabricv1alpha1.TenantBundle{}).
		Watches(&corev1.Namespace{}, handler.EnqueueRequestsFromMapFunc(tenantBundleRequestsForManagedObject)).
		Watches(&networkingv1.NetworkPolicy{}, handler.EnqueueRequestsFromMapFunc(tenantBundleRequestsForManagedObject)).
		Watches(&fabricv1alpha1.PostgreSQLProfile{}, handler.EnqueueRequestsFromMapFunc(r.tenantBundleRequestsForCapabilityProfile)).
		Watches(&fabricv1alpha1.HindsightProfile{}, handler.EnqueueRequestsFromMapFunc(r.tenantBundleRequestsForCapabilityProfile)).
		Complete(r)
}
