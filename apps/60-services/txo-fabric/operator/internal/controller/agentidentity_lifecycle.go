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
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
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

	// Revocation is intentionally best-effort: a temporary gateway outage must not
	// wedge tenant cleanup forever. Resolve the backend recorded on the model-access
	// Secret so tenant-local keys are not accidentally revoked only on the legacy
	// shared gateway.
	backend := sharedModelAccessBackend()
	var tenant fabricv1alpha1.TenantBundle
	if err := r.Get(ctx, types.NamespacedName{Name: agent.Spec.TenantRef.Name}, &tenant); err == nil {
		var modelSecret corev1.Secret
		if err := r.Get(ctx, types.NamespacedName{Name: modelAccessSecretName(agent.Spec.AgentKey), Namespace: namespace}, &modelSecret); err == nil {
			if resolved, err := r.previousModelAccessBackend(ctx, &tenant, &modelSecret); err == nil {
				backend = resolved
			}
		} else if resolved, err := r.resolveModelAccessBackend(ctx, &tenant); err == nil {
			backend = resolved
		}
	}
	_ = revokeModelAccessKeyIfExistsWithBackend(ctx, backend, modelAccessKeyAliasForNames(agent.Spec.TenantRef.Name, agent.Spec.AgentKey))

	if storageRetentionPolicy(agent) == StorageRetentionRetain {
		if err := r.retainRuntimePVC(ctx, agent, namespace); err != nil {
			return ctrl.Result{}, err
		}
	}

	objects := []client.Object{
		&appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: runtimeName(agent.Spec.AgentKey), Namespace: namespace}},
		&networkingv1.NetworkPolicy{ObjectMeta: metav1.ObjectMeta{Name: runtimeName(agent.Spec.AgentKey) + "-egress", Namespace: namespace}},
		&networkingv1.NetworkPolicy{ObjectMeta: metav1.ObjectMeta{Name: humanAccessResourceName(agent.Spec.AgentKey) + "-ingress", Namespace: namespace}},
		&networkingv1.Ingress{ObjectMeta: metav1.ObjectMeta{Name: humanAccessResourceName(agent.Spec.AgentKey), Namespace: namespace}},
		&corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: humanAccessResourceName(agent.Spec.AgentKey), Namespace: namespace}},
		humanOIDCEgressPolicyObject(agent.Spec.AgentKey, namespace),
		&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: modelAccessSecretName(agent.Spec.AgentKey), Namespace: namespace}},
		&corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: managedToolsetPolicyName(agent.Spec.AgentKey), Namespace: namespace}},
	}
	if storageRetentionPolicy(agent) == StorageRetentionDelete {
		objects = append(objects, &corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{Name: runtimePVCName(agent.Spec.AgentKey), Namespace: namespace}})
	}

	pending := false
	for _, obj := range objects {
		var current client.Object
		switch obj.(type) {
		case *appsv1.Deployment:
			current = &appsv1.Deployment{}
		case *networkingv1.NetworkPolicy:
			current = &networkingv1.NetworkPolicy{}
		case *networkingv1.Ingress:
			current = &networkingv1.Ingress{}
		case *unstructured.Unstructured:
			u := &unstructured.Unstructured{}
			u.SetGroupVersionKind(obj.GetObjectKind().GroupVersionKind())
			current = u
		case *corev1.Service:
			current = &corev1.Service{}
		case *corev1.Secret:
			current = &corev1.Secret{}
		case *corev1.ConfigMap:
			current = &corev1.ConfigMap{}
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

func (r *AgentIdentityReconciler) retainRuntimePVC(ctx context.Context, agent *fabricv1alpha1.AgentIdentity, namespace string) error {
	var pvc corev1.PersistentVolumeClaim
	key := types.NamespacedName{Name: runtimePVCName(agent.Spec.AgentKey), Namespace: namespace}
	if err := r.Get(ctx, key, &pvc); err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return err
	}

	if err := r.ensureRetainedPersistentVolume(ctx, agent, &pvc); err != nil {
		return err
	}

	pvc.Labels = mergeStringMap(pvc.Labels, map[string]string{
		LabelPartOf:           "txo-fabric",
		LabelTenantName:       agent.Spec.TenantRef.Name,
		LabelAgent:            agent.Spec.AgentKey,
		LabelStorageRetention: StorageRetentionRetain,
	})
	ownerReferences := pvc.OwnerReferences[:0]
	for _, owner := range pvc.OwnerReferences {
		if owner.UID == agent.UID {
			continue
		}
		ownerReferences = append(ownerReferences, owner)
	}
	pvc.OwnerReferences = ownerReferences
	return r.Update(ctx, &pvc)
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
	if r.APIReader == nil {
		r.APIReader = mgr.GetAPIReader()
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&fabricv1alpha1.AgentIdentity{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.PersistentVolumeClaim{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.Service{}).
		Owns(&networkingv1.Ingress{}).
		Owns(&networkingv1.NetworkPolicy{}).
		Watches(&fabricv1alpha1.AgentRuntimeProfile{}, handler.EnqueueRequestsFromMapFunc(r.requestsForProfile)).
		Watches(&fabricv1alpha1.TenantBundle{}, handler.EnqueueRequestsFromMapFunc(r.requestsForTenant)).
		Watches(&fabricv1alpha1.IntegrationBinding{}, handler.EnqueueRequestsFromMapFunc(r.requestsForIntegrationBinding)).
		Watches(&fabricv1alpha1.IntegrationConnection{}, handler.EnqueueRequestsFromMapFunc(r.requestsForIntegrationConnection)).
		Complete(r)
}
