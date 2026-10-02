package controller

import (
	"context"
	"fmt"
	"reflect"
	"time"

	fabricv1alpha1 "github.com/charchess/vixens/apps/60-services/txo-fabric/operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apiMeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type AgentIdentityReconciler struct {
	client.Client
	APIReader client.Reader
	Scheme    *runtime.Scheme
}

// +kubebuilder:rbac:groups=fabric.truxonline.io,resources=agentidentities,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=fabric.truxonline.io,resources=agentidentities/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=fabric.truxonline.io,resources=agentidentities/finalizers,verbs=update
// +kubebuilder:rbac:groups=fabric.truxonline.io,resources=tenantbundles;agentruntimeprofiles,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=namespaces;persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=networking.k8s.io,resources=networkpolicies,verbs=get;list;watch;create;update;patch;delete

func (r *AgentIdentityReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var agent fabricv1alpha1.AgentIdentity
	if err := r.Get(ctx, req.NamespacedName, &agent); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	if !agent.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, &agent)
	}
	previousStatus := agent.DeepCopy().Status
	if !controllerutil.ContainsFinalizer(&agent, AgentFinalizer) {
		controllerutil.AddFinalizer(&agent, AgentFinalizer)
		if err := r.Update(ctx, &agent); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	var tenant fabricv1alpha1.TenantBundle
	if err := r.Get(ctx, types.NamespacedName{Name: agent.Spec.TenantRef.Name}, &tenant); err != nil {
		if apierrors.IsNotFound(err) {
			r.setStatus(ctx, &agent, "Pending", "TenantResolved", metav1.ConditionFalse, "TenantNotFound", fmt.Sprintf("TenantBundle %q does not exist", agent.Spec.TenantRef.Name))
			return ctrl.Result{RequeueAfter: 15 * time.Second}, nil
		}
		return ctrl.Result{}, err
	}

	conflict, err := r.findAgentKeyConflict(ctx, &agent)
	if err != nil {
		return ctrl.Result{}, err
	}
	if conflict != "" {
		r.setStatus(ctx, &agent, "Failed", "IdentityUnique", metav1.ConditionFalse, "DuplicateAgentKey", fmt.Sprintf("agentKey %q is already claimed by AgentIdentity %q in tenant %q", agent.Spec.AgentKey, conflict, tenant.Name))
		return ctrl.Result{}, nil
	}
	setCondition(&agent.Status.Conditions, agent.Generation, "IdentityUnique", metav1.ConditionTrue, "Unique", fmt.Sprintf("agentKey %q is unique in tenant %q", agent.Spec.AgentKey, tenant.Name))

	namespace := tenantNamespace(tenant.Name)
	var ns corev1.Namespace
	if err := r.Get(ctx, types.NamespacedName{Name: namespace}, &ns); err != nil {
		if apierrors.IsNotFound(err) {
			r.setStatus(ctx, &agent, "Pending", "TenantResolved", metav1.ConditionFalse, "TenantNamespacePending", fmt.Sprintf("namespace %q is not ready yet", namespace))
			return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
		}
		return ctrl.Result{}, err
	}

	profileName := normalizedProfileRef(&agent)
	var profile fabricv1alpha1.AgentRuntimeProfile
	if err := r.Get(ctx, types.NamespacedName{Name: profileName}, &profile); err != nil {
		if apierrors.IsNotFound(err) {
			r.setStatus(ctx, &agent, "Pending", "RuntimeProfileResolved", metav1.ConditionFalse, "RuntimeProfileNotFound", fmt.Sprintf("AgentRuntimeProfile %q does not exist", profileName))
			return ctrl.Result{RequeueAfter: 15 * time.Second}, nil
		}
		return ctrl.Result{}, err
	}
	if profile.Spec.Engine != "" && profile.Spec.Engine != "Hermes" {
		err := fmt.Errorf("runtime engine %q is not supported by this controller version", profile.Spec.Engine)
		r.setStatus(ctx, &agent, "Failed", "RuntimeProfileResolved", metav1.ConditionFalse, "UnsupportedRuntimeEngine", err.Error())
		return ctrl.Result{}, nil
	}

	if err := r.ensurePVC(ctx, &agent, &tenant, &profile, namespace); err != nil {
		r.setStatus(ctx, &agent, "Degraded", "RuntimeReady", metav1.ConditionFalse, "PVCReconcileFailed", err.Error())
		return ctrl.Result{}, err
	}
	modelAccessSecretUID, modelAccessRevision, err := r.ensureModelAccess(ctx, &agent, &tenant, namespace)
	if err != nil {
		r.setStatus(ctx, &agent, "AuthBlocked", "ModelAccessReady", metav1.ConditionFalse, "GatewayCredentialReconcileFailed", err.Error())
		return ctrl.Result{}, err
	}
	rotation := modelAccessRevision
	if rotation == "" {
		rotation = "baseline"
	}
	setCondition(
		&agent.Status.Conditions,
		agent.Generation,
		"ModelAccessReady",
		metav1.ConditionTrue,
		"GatewayCredentialReady",
		fmt.Sprintf("scoped TXO AI gateway credential is reconciled (gateway=%s model=%s rotation=%s)", aiGatewayURL(), defaultAIGatewayModel, rotation),
	)
	if err := r.ensureDeployment(ctx, &agent, &tenant, &profile, namespace, modelAccessSecretUID, modelAccessRevision); err != nil {
		r.setStatus(ctx, &agent, "Degraded", "RuntimeReady", metav1.ConditionFalse, "DeploymentReconcileFailed", err.Error())
		return ctrl.Result{}, err
	}
	if err := r.ensureEgressPolicy(ctx, &agent, &tenant, namespace); err != nil {
		r.setStatus(ctx, &agent, "Degraded", "NetworkReady", metav1.ConditionFalse, "NetworkPolicyReconcileFailed", err.Error())
		return ctrl.Result{}, err
	}

	var deployment appsv1.Deployment
	if err := r.Get(ctx, types.NamespacedName{Name: runtimeName(agent.Spec.AgentKey), Namespace: namespace}, &deployment); err != nil {
		return ctrl.Result{}, err
	}
	agent.Status.ObservedGeneration = agent.Generation
	agent.Status.Namespace = namespace
	agent.Status.Runtime.DeploymentName = deployment.Name
	agent.Status.Runtime.PVCName = runtimePVCName(agent.Spec.AgentKey)
	agent.Status.Memory.BankID = resolvedBankID(&agent)
	if tenant.Spec.Memory.Hindsight == nil {
		agent.Status.Memory.Phase = "Unconfigured"
		setCondition(&agent.Status.Conditions, agent.Generation, "MemoryReady", metav1.ConditionFalse, "TenantMemoryUnconfigured", "TenantBundle does not declare Hindsight memory")
	} else if memoryReady := apiMeta.FindStatusCondition(tenant.Status.Conditions, "MemoryReady"); memoryReady != nil && memoryReady.Status == metav1.ConditionTrue {
		agent.Status.Memory.Phase = "Ready"
		setCondition(&agent.Status.Conditions, agent.Generation, "MemoryReady", metav1.ConditionTrue, "MemoryProviderReady", fmt.Sprintf("bank binding %q is ready on the tenant Hindsight service", agent.Status.Memory.BankID))
	} else {
		agent.Status.Memory.Phase = "Pending"
		setCondition(&agent.Status.Conditions, agent.Generation, "MemoryReady", metav1.ConditionFalse, "MemoryProviderPending", "waiting for the tenant Hindsight service to become ready")
	}
	setCondition(&agent.Status.Conditions, agent.Generation, "TenantResolved", metav1.ConditionTrue, "Resolved", fmt.Sprintf("TenantBundle %q resolved", tenant.Name))
	setCondition(&agent.Status.Conditions, agent.Generation, "RuntimeProfileResolved", metav1.ConditionTrue, "Resolved", fmt.Sprintf("AgentRuntimeProfile %q resolved", profile.Name))
	setCondition(&agent.Status.Conditions, agent.Generation, "NetworkReady", metav1.ConditionTrue, "Reconciled", "runtime egress policy is reconciled")
	if deployment.Status.AvailableReplicas > 0 && deployment.Status.ObservedGeneration == deployment.Generation {
		agent.Status.Phase = "Ready"
		setCondition(&agent.Status.Conditions, agent.Generation, "RuntimeReady", metav1.ConditionTrue, "DeploymentAvailable", "Hermes runtime Deployment is available")
		setCondition(&agent.Status.Conditions, agent.Generation, "Ready", metav1.ConditionTrue, "Ready", "runtime and scoped model access are ready")
	} else {
		agent.Status.Phase = "Provisioning"
		setCondition(&agent.Status.Conditions, agent.Generation, "RuntimeReady", metav1.ConditionFalse, "DeploymentProgressing", "waiting for the Hermes Deployment to become available")
		setCondition(&agent.Status.Conditions, agent.Generation, "Ready", metav1.ConditionFalse, "RuntimeProgressing", "runtime is still provisioning")
	}
	if !reflect.DeepEqual(previousStatus, agent.Status) {
		if err := r.Status().Update(ctx, &agent); err != nil {
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{}, nil
}

func (r *AgentIdentityReconciler) findAgentKeyConflict(ctx context.Context, agent *fabricv1alpha1.AgentIdentity) (string, error) {
	var agents fabricv1alpha1.AgentIdentityList
	if err := r.List(ctx, &agents); err != nil {
		return "", err
	}
	for i := range agents.Items {
		other := &agents.Items[i]
		if other.Name == agent.Name || !other.DeletionTimestamp.IsZero() {
			continue
		}
		if other.Spec.TenantRef.Name == agent.Spec.TenantRef.Name && other.Spec.AgentKey == agent.Spec.AgentKey {
			return other.Name, nil
		}
	}
	return "", nil
}
