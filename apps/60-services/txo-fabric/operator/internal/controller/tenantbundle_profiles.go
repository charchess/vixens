package controller

import (
	"context"
	"fmt"

	fabricv1alpha1 "github.com/charchess/vixens/apps/60-services/txo-fabric/operator/api/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func (r *TenantBundleReconciler) reconcilePostgreSQLCapability(ctx context.Context, bundle *fabricv1alpha1.TenantBundle) (bool, error) {
	request := bundle.Spec.Persistence.PostgreSQL
	if request == nil {
		setCondition(&bundle.Status.Conditions, bundle.Generation, "PersistenceProfileResolved", metav1.ConditionTrue, "NotRequested", "tenant does not request PostgreSQL capability")
		setCondition(&bundle.Status.Conditions, bundle.Generation, "PersistenceReady", metav1.ConditionTrue, "NotRequested", "tenant does not request PostgreSQL capability")
		return false, nil
	}

	profileName := request.ProfileRef.Name
	var profile fabricv1alpha1.PostgreSQLProfile
	if err := r.Get(ctx, types.NamespacedName{Name: profileName}, &profile); err != nil {
		if apierrors.IsNotFound(err) {
			message := fmt.Sprintf("PostgreSQLProfile %q does not exist", profileName)
			bundle.Status.Persistence.PostgreSQL = &fabricv1alpha1.ComponentStatus{Phase: "Pending", Message: message}
			setCondition(&bundle.Status.Conditions, bundle.Generation, "PersistenceProfileResolved", metav1.ConditionFalse, "ProfileNotFound", message)
			setCondition(&bundle.Status.Conditions, bundle.Generation, "PersistenceReady", metav1.ConditionFalse, "ProfileNotFound", message)
			return true, nil
		}
		return true, err
	}
	if profile.Spec.Mode != request.Mode {
		message := fmt.Sprintf("PostgreSQLProfile %q mode %q does not satisfy requested mode %q", profile.Name, profile.Spec.Mode, request.Mode)
		bundle.Status.Persistence.PostgreSQL = &fabricv1alpha1.ComponentStatus{Phase: "Pending", Message: message}
		setCondition(&bundle.Status.Conditions, bundle.Generation, "PersistenceProfileResolved", metav1.ConditionFalse, "ProfileModeMismatch", message)
		setCondition(&bundle.Status.Conditions, bundle.Generation, "PersistenceReady", metav1.ConditionFalse, "ProfileModeMismatch", message)
		return true, nil
	}

	setCondition(&bundle.Status.Conditions, bundle.Generation, "PersistenceProfileResolved", metav1.ConditionTrue, "Resolved", fmt.Sprintf("PostgreSQLProfile %q resolved", profile.Name))
	bundle.Status.Persistence.PostgreSQL = &fabricv1alpha1.ComponentStatus{Phase: "Pending", Message: fmt.Sprintf("PostgreSQLProfile %q resolved; database reconciliation is not implemented yet", profile.Name)}
	setCondition(&bundle.Status.Conditions, bundle.Generation, "PersistenceReady", metav1.ConditionFalse, "ControllerNotImplemented", "PostgreSQL profile is resolved but database lifecycle is not implemented by this controller version")
	return true, nil
}

func (r *TenantBundleReconciler) reconcileHindsightCapability(ctx context.Context, bundle *fabricv1alpha1.TenantBundle) (bool, error) {
	request := bundle.Spec.Memory.Hindsight
	if request == nil {
		setCondition(&bundle.Status.Conditions, bundle.Generation, "MemoryProfileResolved", metav1.ConditionTrue, "NotRequested", "tenant does not request Hindsight memory")
		setCondition(&bundle.Status.Conditions, bundle.Generation, "MemoryReady", metav1.ConditionTrue, "NotRequested", "tenant does not request Hindsight memory")
		return false, nil
	}

	profileName := request.ProfileRef.Name
	var profile fabricv1alpha1.HindsightProfile
	if err := r.Get(ctx, types.NamespacedName{Name: profileName}, &profile); err != nil {
		if apierrors.IsNotFound(err) {
			message := fmt.Sprintf("HindsightProfile %q does not exist", profileName)
			bundle.Status.Memory.Hindsight = &fabricv1alpha1.ComponentStatus{Phase: "Pending", Message: message}
			setCondition(&bundle.Status.Conditions, bundle.Generation, "MemoryProfileResolved", metav1.ConditionFalse, "ProfileNotFound", message)
			setCondition(&bundle.Status.Conditions, bundle.Generation, "MemoryReady", metav1.ConditionFalse, "ProfileNotFound", message)
			return true, nil
		}
		return true, err
	}

	setCondition(&bundle.Status.Conditions, bundle.Generation, "MemoryProfileResolved", metav1.ConditionTrue, "Resolved", fmt.Sprintf("HindsightProfile %q resolved", profile.Name))
	bundle.Status.Memory.Hindsight = &fabricv1alpha1.ComponentStatus{Phase: "Pending", Message: fmt.Sprintf("HindsightProfile %q resolved; memory service reconciliation is not implemented yet", profile.Name)}
	setCondition(&bundle.Status.Conditions, bundle.Generation, "MemoryReady", metav1.ConditionFalse, "ControllerNotImplemented", "Hindsight profile is resolved but memory-service lifecycle is not implemented by this controller version")
	return true, nil
}

func (r *TenantBundleReconciler) tenantBundleRequestsForCapabilityProfile(ctx context.Context, obj client.Object) []reconcile.Request {
	var bundles fabricv1alpha1.TenantBundleList
	if err := r.List(ctx, &bundles); err != nil {
		return nil
	}

	requests := make([]reconcile.Request, 0)
	for i := range bundles.Items {
		bundle := &bundles.Items[i]
		matches := false
		switch obj.(type) {
		case *fabricv1alpha1.PostgreSQLProfile:
			matches = bundle.Spec.Persistence.PostgreSQL != nil && bundle.Spec.Persistence.PostgreSQL.ProfileRef.Name == obj.GetName()
		case *fabricv1alpha1.HindsightProfile:
			matches = bundle.Spec.Memory.Hindsight != nil && bundle.Spec.Memory.Hindsight.ProfileRef.Name == obj.GetName()
		}
		if matches {
			requests = append(requests, reconcile.Request{NamespacedName: types.NamespacedName{Name: bundle.Name}})
		}
	}
	return requests
}
