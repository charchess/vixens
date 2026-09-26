package controller

import (
	"context"
	"reflect"
	"time"

	fabricv1alpha1 "github.com/charchess/vixens/apps/60-services/txo-fabric/operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func (r *AgentIdentityReconciler) reconcileDelete(ctx context.Context, agent *fabricv1alpha1.AgentIdentity) (ctrl.Result, error) {
	if !controllerutil.ContainsFinalizer(agent, AgentFinalizer) {
		return ctrl.Result{}, nil
	}
	namespace := tenantNamespace(agent.Spec.TenantRef.Name)
	objects := []client.Object{
		&appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: runtimeName(agent.Name), Namespace: namespace}},
		&networkingv1.NetworkPolicy{ObjectMeta: metav1.ObjectMeta{Name: runtimeName(agent.Name) + "-egress", Namespace: namespace}},
		&corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{Name: runtimePVCName(agent.Name), Namespace: namespace}},
	}
	pending := false
	for _, obj := range objects {
		var current client.Object
		switch obj.(type) {
		case *appsv1.Deployment:
			current = &appsv1.Deployment{}
		case *networkingv1.NetworkPolicy:
			current = &networkingv1.NetworkPolicy{}
		case *corev1.PersistentVolumeClaim:
			current = &corev1.PersistentVolumeClaim{}
		}
		key := client.ObjectKeyFromObject(obj)
		if err := r.Get(ctx, key, current); err == nil {
			pending = true
			if current.GetDeletionTimestamp().IsZero() {
				if err := r.Delete(ctx, current); err != nil && !apierrors.IsNotFound(err) {
					return ctrl.Result{}, err
				}
			}
		} else if !apierrors.IsNotFound(err) {
			return ctrl.Result{}, err
		}
	}
	if pending {
		return ctrl.Result{RequeueAfter: 2 * time.Second}, nil
	}
	controllerutil.RemoveFinalizer(agent, AgentFinalizer)
	if err := r.Update(ctx, agent); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (r *AgentIdentityReconciler) setStatus(ctx context.Context, agent *fabricv1alpha1.AgentIdentity, phase, conditionType string, status metav1.ConditionStatus, reason, message string) {
	previousStatus := agent.DeepCopy().Status
	agent.Status.ObservedGeneration = agent.Generation
	agent.Status.Phase = phase
	setCondition(&agent.Status.Conditions, agent.Generation, conditionType, status, reason, message)
	setCondition(&agent.Status.Conditions, agent.Generation, "Ready", metav1.ConditionFalse, reason, message)
	if !reflect.DeepEqual(previousStatus, agent.Status) {
		_ = r.Status().Update(ctx, agent)
	}
}

func (r *AgentIdentityReconciler) requestsForProfile(ctx context.Context, obj client.Object) []reconcile.Request {
	profile, ok := obj.(*fabricv1alpha1.AgentRuntimeProfile)
	if !ok {
		return nil
	}
	var agents fabricv1alpha1.AgentIdentityList
	if err := r.List(ctx, &agents); err != nil {
		return nil
	}
	requests := make([]reconcile.Request, 0)
	for i := range agents.Items {
		if normalizedProfileRef(&agents.Items[i]) == profile.Name {
			requests = append(requests, reconcile.Request{NamespacedName: types.NamespacedName{Name: agents.Items[i].Name}})
		}
	}
	return requests
}

func (r *AgentIdentityReconciler) requestsForTenant(ctx context.Context, obj client.Object) []reconcile.Request {
	tenant, ok := obj.(*fabricv1alpha1.TenantBundle)
	if !ok {
		return nil
	}
	var agents fabricv1alpha1.AgentIdentityList
	if err := r.List(ctx, &agents); err != nil {
		return nil
	}
	requests := make([]reconcile.Request, 0)
	for i := range agents.Items {
		if agents.Items[i].Spec.TenantRef.Name == tenant.Name {
			requests = append(requests, reconcile.Request{NamespacedName: types.NamespacedName{Name: agents.Items[i].Name}})
		}
	}
	return requests
}

func (r *AgentIdentityReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&fabricv1alpha1.AgentIdentity{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.PersistentVolumeClaim{}).
		Owns(&networkingv1.NetworkPolicy{}).
		Watches(&fabricv1alpha1.AgentRuntimeProfile{}, handler.EnqueueRequestsFromMapFunc(r.requestsForProfile)).
		Watches(&fabricv1alpha1.TenantBundle{}, handler.EnqueueRequestsFromMapFunc(r.requestsForTenant)).
		Complete(r)
}
