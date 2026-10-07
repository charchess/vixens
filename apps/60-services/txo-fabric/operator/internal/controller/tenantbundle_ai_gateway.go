package controller

import (
	"context"
	"fmt"
	"strings"
	"time"

	fabricv1alpha1 "github.com/charchess/vixens/apps/60-services/txo-fabric/operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const defaultAIGatewayProfileName = "litellm-standard"

type aiGatewayResult struct {
	Ready        bool
	Status       *fabricv1alpha1.ComponentStatus
	Reason       string
	Message      string
	RequeueAfter time.Duration
}

func (r *TenantBundleReconciler) reconcileAIGateway(ctx context.Context, bundle *fabricv1alpha1.TenantBundle) (aiGatewayResult, error) {
	profileName := tenantAIGatewayProfileName(bundle)
	var profile fabricv1alpha1.AIGatewayProfile
	if err := r.Get(ctx, types.NamespacedName{Name: profileName}, &profile); err != nil {
		if apierrors.IsNotFound(err) {
			status := &fabricv1alpha1.ComponentStatus{Phase: "Blocked", Message: fmt.Sprintf("AIGatewayProfile %q does not exist", profileName)}
			return aiGatewayResult{Status: status, Reason: "ProfileNotFound", Message: status.Message, RequeueAfter: 30 * time.Second}, nil
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
	if strings.TrimSpace(profile.Spec.Image) == "" {
		status := &fabricv1alpha1.ComponentStatus{Phase: "Blocked", Message: fmt.Sprintf("AIGatewayProfile %q must declare an image", profileName)}
		return aiGatewayResult{Status: status, Reason: "ImageRequired", Message: status.Message}, nil
	}

	postgresql, err := r.reconcileAIGatewayPostgreSQL(ctx, bundle, &profile)
	if err != nil {
		return aiGatewayResult{}, err
	}
	if !postgresql.Ready {
		phase := "Blocked"
		if strings.HasSuffix(postgresql.Reason, "Pending") {
			phase = "Provisioning"
		}
		status := &fabricv1alpha1.ComponentStatus{Phase: phase, Message: postgresql.Message}
		return aiGatewayResult{
			Status:       status,
			Reason:       postgresql.Reason,
			Message:      postgresql.Message,
			RequeueAfter: postgresql.RequeueAfter,
		}, nil
	}

	if _, err := r.ensureAIGatewayRuntimeSecret(ctx, bundle, postgresql.Secret); err != nil {
		return aiGatewayResult{}, err
	}

	migrationsReady, migrationReason, migrationMessage, err := r.ensureAIGatewayMigrations(ctx, bundle, &profile, postgresql.Profile, postgresql.Secret)
	if err != nil {
		return aiGatewayResult{}, err
	}
	if !migrationsReady {
		phase := "Blocked"
		requeueAfter := time.Duration(0)
		if migrationReason == "MigrationPending" {
			phase = "Provisioning"
			requeueAfter = 5 * time.Second
		}
		status := &fabricv1alpha1.ComponentStatus{Phase: phase, Message: migrationMessage}
		return aiGatewayResult{Status: status, Reason: migrationReason, Message: migrationMessage, RequeueAfter: requeueAfter}, nil
	}

	backends, backendMessage, err := r.resolveAIGatewayBackends(ctx, bundle)
	if err != nil {
		return aiGatewayResult{}, err
	}
	if backendMessage != "" {
		status := &fabricv1alpha1.ComponentStatus{Phase: "Provisioning", Message: backendMessage}
		return aiGatewayResult{Status: status, Reason: "BackendPending", Message: backendMessage, RequeueAfter: 5 * time.Second}, nil
	}

	config := renderTenantLiteLLMConfig(backends)
	_, configHash, err := r.ensureTenantAIGatewayConfig(ctx, bundle, config)
	if err != nil {
		return aiGatewayResult{}, err
	}
	if err := r.ensureTenantAIGatewayService(ctx, bundle, &profile); err != nil {
		return aiGatewayResult{}, err
	}
	if err := r.ensureTenantAIGatewayNetworkPolicy(ctx, bundle, &profile, postgresql.Profile, backends); err != nil {
		return aiGatewayResult{}, err
	}
	deployment, err := r.ensureTenantAIGatewayDeployment(ctx, bundle, &profile, backends, configHash)
	if err != nil {
		return aiGatewayResult{}, err
	}

	endpoint := fmt.Sprintf("http://%s.%s.svc:%d", tenantAIGatewayName, tenantNamespace(bundle.Name), aiGatewayPort(&profile))
	if deployment.Status.ObservedGeneration != deployment.Generation || deployment.Status.AvailableReplicas < 1 {
		status := &fabricv1alpha1.ComponentStatus{Phase: "Provisioning", Endpoint: endpoint, Message: "waiting for tenant LiteLLM Deployment to become available"}
		return aiGatewayResult{Status: status, Reason: "DeploymentProgressing", Message: status.Message, RequeueAfter: 5 * time.Second}, nil
	}
	status := &fabricv1alpha1.ComponentStatus{Phase: "Ready", Endpoint: endpoint}
	return aiGatewayResult{Ready: true, Status: status, Reason: "DeploymentAvailable", Message: "tenant LiteLLM AI gateway is available"}, nil
}

func (r *TenantBundleReconciler) parkAIGatewayCompute(ctx context.Context, bundle *fabricv1alpha1.TenantBundle) (bool, error) {
	namespace := tenantNamespace(bundle.Name)
	pending := false

	deployment := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: tenantAIGatewayName, Namespace: namespace}}
	if err := r.Get(ctx, client.ObjectKeyFromObject(deployment), deployment); err != nil {
		if !apierrors.IsNotFound(err) {
			return false, err
		}
	} else if aiGatewayRuntimeOwnedBy(deployment, bundle) {
		pending = true
		if deployment.GetDeletionTimestamp().IsZero() {
			if err := r.Delete(ctx, deployment); err != nil && !apierrors.IsNotFound(err) {
				return false, err
			}
		}
	}

	var jobs batchv1.JobList
	if err := r.List(ctx, &jobs, client.InNamespace(namespace), client.MatchingLabels{
		LabelManaged:    "true",
		LabelTenantID:   bundle.Spec.TenantID,
		LabelTenantName: bundle.Name,
		"fabric.truxonline.io/ai-gateway-resource": "migration",
	}); err != nil {
		return false, err
	}
	for i := range jobs.Items {
		pending = true
		job := &jobs.Items[i]
		if job.DeletionTimestamp == nil {
			if err := r.Delete(ctx, job); err != nil && !apierrors.IsNotFound(err) {
				return false, err
			}
		}
	}

	return pending, nil
}

func (r *TenantBundleReconciler) cleanupAIGateway(ctx context.Context, bundle *fabricv1alpha1.TenantBundle) (bool, error) {
	namespace := tenantNamespace(bundle.Name)
	pending := false

	objects := []client.Object{
		&appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: tenantAIGatewayName, Namespace: namespace}},
		&corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: tenantAIGatewayName, Namespace: namespace}},
		&corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: tenantAIGatewayConfigMapName, Namespace: namespace}},
		&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: tenantAIGatewayRuntimeSecretName, Namespace: namespace}},
		&networkingv1.NetworkPolicy{ObjectMeta: metav1.ObjectMeta{Name: tenantAIGatewayNetworkPolicy, Namespace: namespace}},
		&networkingv1.NetworkPolicy{ObjectMeta: metav1.ObjectMeta{Name: tenantAIGatewayMigrationPolicyName, Namespace: namespace}},
	}
	for _, object := range objects {
		if err := r.Get(ctx, client.ObjectKeyFromObject(object), object); err != nil {
			if apierrors.IsNotFound(err) {
				continue
			}
			return false, err
		}
		if !aiGatewayRuntimeOwnedBy(object, bundle) {
			continue
		}
		pending = true
		if object.GetDeletionTimestamp() == nil {
			if err := r.Delete(ctx, object); err != nil && !apierrors.IsNotFound(err) {
				return false, err
			}
		}
	}

	var jobs batchv1.JobList
	if err := r.List(ctx, &jobs, client.InNamespace(namespace), client.MatchingLabels{
		LabelManaged:    "true",
		LabelTenantID:   bundle.Spec.TenantID,
		LabelTenantName: bundle.Name,
		"fabric.truxonline.io/ai-gateway-resource": "migration",
	}); err != nil {
		return false, err
	}
	for i := range jobs.Items {
		pending = true
		job := &jobs.Items[i]
		if job.DeletionTimestamp == nil {
			if err := r.Delete(ctx, job); err != nil && !apierrors.IsNotFound(err) {
				return false, err
			}
		}
	}

	postgresPending, err := r.cleanupAIGatewayPostgreSQL(ctx, bundle)
	if err != nil {
		return false, err
	}
	return pending || postgresPending, nil
}
