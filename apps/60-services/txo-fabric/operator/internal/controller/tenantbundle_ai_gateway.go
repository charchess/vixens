package controller

import (
	"context"
	"fmt"
	"strings"

	fabricv1alpha1 "github.com/charchess/vixens/apps/60-services/txo-fabric/operator/api/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
)

const defaultAIGatewayProfileName = "litellm-standard"

type aiGatewayResult struct {
	Ready   bool
	Status  *fabricv1alpha1.ComponentStatus
	Reason  string
	Message string
}

func (r *TenantBundleReconciler) reconcileAIGateway(ctx context.Context, bundle *fabricv1alpha1.TenantBundle) (aiGatewayResult, error) {
	request := bundle.Spec.AIGateway
	if request == nil {
		return aiGatewayResult{Ready: true, Reason: "NotRequested", Message: "tenant does not request an AI gateway"}, nil
	}

	profileName := strings.TrimSpace(request.ProfileRef)
	if profileName == "" {
		profileName = defaultAIGatewayProfileName
	}
	var profile fabricv1alpha1.AIGatewayProfile
	if err := r.Get(ctx, types.NamespacedName{Name: profileName}, &profile); err != nil {
		if apierrors.IsNotFound(err) {
			status := &fabricv1alpha1.ComponentStatus{Phase: "Blocked", Message: fmt.Sprintf("AIGatewayProfile %q does not exist", profileName)}
			return aiGatewayResult{Status: status, Reason: "ProfileNotFound", Message: status.Message}, nil
		}
		return aiGatewayResult{}, err
	}
	if profile.Spec.Topology != "" && profile.Spec.Topology != "TenantScoped" {
		status := &fabricv1alpha1.ComponentStatus{Phase: "Blocked", Message: fmt.Sprintf("AIGatewayProfile %q topology %q is not supported", profileName, profile.Spec.Topology)}
		return aiGatewayResult{Status: status, Reason: "UnsupportedTopology", Message: status.Message}, nil
	}
	if profile.Spec.Implementation != "" && profile.Spec.Implementation != "LiteLLM" {
		status := &fabricv1alpha1.ComponentStatus{Phase: "Blocked", Message: fmt.Sprintf("AIGatewayProfile %q implementation %q is not supported", profileName, profile.Spec.Implementation)}
		return aiGatewayResult{Status: status, Reason: "UnsupportedImplementation", Message: status.Message}, nil
	}

	status := &fabricv1alpha1.ComponentStatus{
		Phase:   "Blocked",
		Message: "tenant LiteLLM gateway runtime reconciliation is not implemented in this slice",
	}
	return aiGatewayResult{Status: status, Reason: "ControllerNotImplemented", Message: status.Message}, nil
}

func (r *TenantBundleReconciler) cleanupAIGateway(_ context.Context, _ *fabricv1alpha1.TenantBundle) (bool, error) {
	// This slice reserves the gateway API for LiteLLM but does not create gateway
	// runtime objects yet, so there is nothing persistent to reclaim.
	return false, nil
}
