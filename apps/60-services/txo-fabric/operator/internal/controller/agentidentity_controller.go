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
// +kubebuilder:rbac:groups=fabric.truxonline.io,resources=tenantbundles;agentruntimeprofiles;integrationconnections;integrationbindings,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=namespaces;persistentvolumeclaims;configmaps;services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=networking.k8s.io,resources=networkpolicies;ingresses,verbs=get;list;watch;create;update;patch;delete

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

	toolPolicy, err := resolveToolsetPolicy(&agent, &profile)
	if err != nil {
		r.setStatus(ctx, &agent, "Failed", "CapabilityPolicyReady", metav1.ConditionFalse, "CapabilityPolicyInvalid", err.Error())
		return ctrl.Result{}, nil
	}
	if err := r.ensureManagedToolsetPolicy(ctx, &agent, &tenant, namespace, toolPolicy); err != nil {
		r.setStatus(ctx, &agent, "Degraded", "CapabilityPolicyReady", metav1.ConditionFalse, "ManagedPolicyReconcileFailed", err.Error())
		return ctrl.Result{}, err
	}
	setCondition(
		&agent.Status.Conditions,
		agent.Generation,
		"CapabilityPolicyReady",
		metav1.ConditionTrue,
		"ManagedToolsetsReady",
		fmt.Sprintf("Hermes toolset policy is reconciled (profile=%s revision=%s enabled=%v denied=%v)", profile.Name, toolPolicy.Revision, toolPolicy.Enabled, toolPolicy.Denied),
	)

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

	integrationAccess, err := r.resolveIntegrationAccess(ctx, &agent, &tenant, namespace)
	if err != nil {
		r.setStatus(ctx, &agent, "Degraded", "IntegrationAccessReady", metav1.ConditionFalse, "IntegrationResolutionFailed", err.Error())
		return ctrl.Result{}, err
	}
	if integrationAccess.Ready {
		setCondition(&agent.Status.Conditions, agent.Generation, "IntegrationAccessReady", metav1.ConditionTrue, "Reconciled", integrationAccess.Message)
	} else {
		setCondition(&agent.Status.Conditions, agent.Generation, "IntegrationAccessReady", metav1.ConditionFalse, "AuthorizationDenied", integrationAccess.Message)
	}

	humanAccess, err := resolveHumanAccess(&agent, &tenant)
	if err != nil {
		// Fail closed on policy drift: remove the externally reachable Service,
		// Ingress and ingress allow-rule even if a previous generation exposed
		// this runtime successfully.
		_ = r.ensureHumanAccessResources(ctx, &agent, &tenant, namespace, humanAccessResolution{})
		agent.Status.Runtime.HumanEndpoint = ""
		r.setStatus(ctx, &agent, "Degraded", "HumanAccessReady", metav1.ConditionFalse, "HumanAccessInvalid", err.Error())
		return ctrl.Result{}, nil
	}

	if err := r.ensureDeploymentWithIntegrations(ctx, &agent, &tenant, &profile, namespace, modelAccessSecretUID, modelAccessRevision, toolPolicy, integrationAccess); err != nil {
		r.setStatus(ctx, &agent, "Degraded", "RuntimeReady", metav1.ConditionFalse, "DeploymentReconcileFailed", err.Error())
		return ctrl.Result{}, err
	}
	if err := r.ensureEgressPolicy(ctx, &agent, &tenant, namespace, integrationAccess); err != nil {
		r.setStatus(ctx, &agent, "Degraded", "NetworkReady", metav1.ConditionFalse, "NetworkPolicyReconcileFailed", err.Error())
		return ctrl.Result{}, err
	}
	if err := r.ensureHumanAccessResources(ctx, &agent, &tenant, namespace, humanAccess); err != nil {
		r.setStatus(ctx, &agent, "Degraded", "HumanAccessReady", metav1.ConditionFalse, "HumanAccessReconcileFailed", err.Error())
		return ctrl.Result{}, err
	}
	if humanAccess.Enabled {
		setCondition(&agent.Status.Conditions, agent.Generation, "HumanAccessReady", metav1.ConditionTrue, "WebEndpointReconciled", fmt.Sprintf("authenticated human endpoint is reconciled at %s", humanAccess.PublicURL))
	} else {
		setCondition(&agent.Status.Conditions, agent.Generation, "HumanAccessReady", metav1.ConditionTrue, "NotRequested", "human entry is disabled for this AgentIdentity")
	}

	var deployment appsv1.Deployment
	if err := r.Get(ctx, types.NamespacedName{Name: runtimeName(agent.Spec.AgentKey), Namespace: namespace}, &deployment); err != nil {
		return ctrl.Result{}, err
	}
	agent.Status.ObservedGeneration = agent.Generation
	agent.Status.Namespace = namespace
	agent.Status.Runtime.DeploymentName = deployment.Name
	agent.Status.Runtime.PVCName = runtimePVCName(agent.Spec.AgentKey)
	agent.Status.Runtime.HumanEndpoint = humanAccess.PublicURL
	agent.Status.Runtime.ToolsetPolicyRevision = toolPolicy.Revision
	agent.Status.Runtime.EnabledToolsets = append([]string(nil), toolPolicy.Enabled...)
	agent.Status.Runtime.DeniedToolsets = append([]string(nil), toolPolicy.Denied...)
	agent.Status.Runtime.IntegrationPolicyRevision = integrationAccess.Revision
	agent.Status.Runtime.Integrations = append([]fabricv1alpha1.IntegrationAuthorizationStatus(nil), integrationAccess.Status...)
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
	networkMessage := "runtime egress policy is reconciled"
	if len(integrationAccess.Effective) > 0 {
		networkMessage = "runtime egress policy is reconciled; effective v0 integration binding admits temporary broad HTTP(S) POC egress"
	}
	setCondition(&agent.Status.Conditions, agent.Generation, "NetworkReady", metav1.ConditionTrue, "Reconciled", networkMessage)
	if deployment.Status.AvailableReplicas > 0 && deployment.Status.ObservedGeneration == deployment.Generation {
		setCondition(&agent.Status.Conditions, agent.Generation, "RuntimeReady", metav1.ConditionTrue, "DeploymentAvailable", "Hermes runtime Deployment is available")
		if integrationAccess.Ready {
			agent.Status.Phase = "Ready"
			setCondition(&agent.Status.Conditions, agent.Generation, "Ready", metav1.ConditionTrue, "Ready", "runtime, capability policy, scoped model access and declared integration access are ready")
		} else {
			agent.Status.Phase = "Degraded"
			setCondition(&agent.Status.Conditions, agent.Generation, "Ready", metav1.ConditionFalse, "IntegrationAccessDenied", integrationAccess.Message)
		}
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
	if integrationAccess.HasBindings {
		return ctrl.Result{RequeueAfter: integrationRefreshInterval}, nil
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
