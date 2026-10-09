package controller

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"strings"
	"time"

	fabricv1alpha1 "github.com/charchess/vixens/apps/60-services/txo-fabric/operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
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
	APIReader client.Reader
	Scheme    *runtime.Scheme
	// Optional only in unit tests: runtime reads the service key from OpenBao-synced Secret.
	OpenFGAStoreClient openFGAStoreProvisioner
	// Optional test seam; runtime model calls use the same OpenBao-synced key.
	OpenFGAModelClient openFGAModelProvisioner
}

// +kubebuilder:rbac:groups=fabric.truxonline.io,resources=tenantbundles,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=fabric.truxonline.io,resources=tenantbundles/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=fabric.truxonline.io,resources=tenantbundles/finalizers,verbs=update
// +kubebuilder:rbac:groups=fabric.truxonline.io,resources=agentidentities,verbs=get;list;watch
// +kubebuilder:rbac:groups=fabric.truxonline.io,resources=postgresqlprofiles;hindsightprofiles;aigatewayprofiles;aicredentialbrokerprofiles,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=networking.k8s.io,resources=networkpolicies,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=postgresql.cnpg.io,resources=clusters,verbs=get
// +kubebuilder:rbac:groups=postgresql.cnpg.io,resources=databases;databaseroles,verbs=get;list;watch;create;update;patch;delete

func (r *TenantBundleReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var bundle fabricv1alpha1.TenantBundle
	if err := r.Get(ctx, req.NamespacedName, &bundle); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if !bundle.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, &bundle)
	}

	previousStatus := bundle.DeepCopy().Status
	hadPostgreSQL := previousStatus.Persistence.PostgreSQL != nil
	hadHindsight := previousStatus.Memory.Hindsight != nil

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
	bundle.Status.AIGateway = nil
	bundle.Status.AICredentialBroker = nil
	bundle.Status.Modules = nil
	setCondition(&bundle.Status.Conditions, bundle.Generation, "NamespaceReady", metav1.ConditionTrue, "Reconciled", "tenant namespace is reconciled")
	setCondition(&bundle.Status.Conditions, bundle.Generation, "NetworkReady", metav1.ConditionTrue, "DefaultDenyReconciled", "tenant default-deny policy is reconciled")

	if validationErr := validateTenantIAM(&bundle); validationErr != nil {
		message := validationErr.Error()
		blueprintErr := r.reconcileAuthentikBlueprint(ctx)
		if blueprintErr != nil {
			message += fmt.Sprintf("; failed to withdraw invalid IAM desired state: %v", blueprintErr)
		}
		bundle.Status.Phase = "Degraded"
		setCondition(&bundle.Status.Conditions, bundle.Generation, "IAMDesiredStateReady", metav1.ConditionFalse, "InvalidHumanAccessIAM", message)
		setCondition(&bundle.Status.Conditions, bundle.Generation, "Ready", metav1.ConditionFalse, "IAMConfigurationInvalid", message)
		if !reflect.DeepEqual(previousStatus, bundle.Status) {
			_ = r.Status().Update(ctx, &bundle)
		}
		if blueprintErr != nil {
			return ctrl.Result{}, blueprintErr
		}
		return ctrl.Result{}, nil
	}
	if err := r.reconcileAuthentikBlueprint(ctx); err != nil {
		bundle.Status.Phase = "Degraded"
		setCondition(&bundle.Status.Conditions, bundle.Generation, "IAMDesiredStateReady", metav1.ConditionFalse, "BlueprintReconcileFailed", err.Error())
		setCondition(&bundle.Status.Conditions, bundle.Generation, "Ready", metav1.ConditionFalse, "IAMReconcileFailed", err.Error())
		if !reflect.DeepEqual(previousStatus, bundle.Status) {
			_ = r.Status().Update(ctx, &bundle)
		}
		return ctrl.Result{}, err
	}
	if bundle.Spec.HumanAccess != nil && bundle.Spec.HumanAccess.Web != nil {
		groups := make([]string, 0, len(bundle.Spec.HumanAccess.Web.IAMGroups))
		for _, group := range bundle.Spec.HumanAccess.Web.IAMGroups {
			groups = append(groups, authentikGroupName(bundle.Name, group))
		}
		setCondition(&bundle.Status.Conditions, bundle.Generation, "IAMDesiredStateReady", metav1.ConditionTrue, "BlueprintPublished", fmt.Sprintf("Authentik desired state published for application %q and groups %v", authentikApplicationName(bundle.Name), groups))
	} else {
		setCondition(&bundle.Status.Conditions, bundle.Generation, "IAMDesiredStateReady", metav1.ConditionTrue, "NotRequested", "tenant does not request human IAM access")
	}

	// This gate remains OFF in the GitOps Deployment until tuple sourcing,
	// revocation and live cross-tenant checks pass acceptance. Store/model
	// readiness alone must never imply a usable authorization graph.
	openFGARolloutEnabled := os.Getenv("TXO_FABRIC_OPENFGA_STORES_ENABLED") == "true"
	if openFGARolloutEnabled {
		store, err := r.reconcileOpenFGAStore(ctx, &bundle)
		if err != nil {
			bundle.Status.Phase = "Degraded"
			setCondition(&bundle.Status.Conditions, bundle.Generation, "OpenFGAStoreReady", metav1.ConditionFalse, "StoreReconcileFailed", err.Error())
			setCondition(&bundle.Status.Conditions, bundle.Generation, "OpenFGAModelReady", metav1.ConditionFalse, "StoreNotReady", "tenant model cannot be reconciled without a verified store")
			setCondition(&bundle.Status.Conditions, bundle.Generation, "OpenFGAAuthorizationReady", metav1.ConditionFalse, "StoreNotReady", "tenant authorization is unavailable")
			setCondition(&bundle.Status.Conditions, bundle.Generation, "Ready", metav1.ConditionFalse, "OpenFGAStoreReconcileFailed", "private tenant authorization store is not reconciled")
			if !reflect.DeepEqual(previousStatus, bundle.Status) {
				_ = r.Status().Update(ctx, &bundle)
			}
			return ctrl.Result{}, err
		}
		setCondition(&bundle.Status.Conditions, bundle.Generation, "OpenFGAStoreReady", metav1.ConditionTrue, "StoreBound", "private tenant authorization store is reconciled")
		if _, err := r.reconcileOpenFGAModel(ctx, &bundle, store); err != nil {
			bundle.Status.Phase = "Degraded"
			setCondition(&bundle.Status.Conditions, bundle.Generation, "OpenFGAModelReady", metav1.ConditionFalse, "ModelReconcileFailed", err.Error())
			setCondition(&bundle.Status.Conditions, bundle.Generation, "OpenFGAAuthorizationReady", metav1.ConditionFalse, "ModelNotReady", "tenant authorization is unavailable")
			setCondition(&bundle.Status.Conditions, bundle.Generation, "Ready", metav1.ConditionFalse, "OpenFGAModelReconcileFailed", "private tenant authorization model is not reconciled")
			if !reflect.DeepEqual(previousStatus, bundle.Status) {
				_ = r.Status().Update(ctx, &bundle)
			}
			return ctrl.Result{}, err
		}
		setCondition(&bundle.Status.Conditions, bundle.Generation, "OpenFGAModelReady", metav1.ConditionTrue, "ModelBound", "approved model ID and fingerprint are durably bound to tenant store")
		// Only the operator initializes the empty, retained identity ledger.
		// Live enrollment and tuple reconciliation remain separate gates.
		if err := r.reconcileOpenFGAHumanIdentityRegistry(ctx, &bundle); err != nil {
			bundle.Status.Phase = "Degraded"
			setCondition(&bundle.Status.Conditions, bundle.Generation, "OpenFGAAuthorizationReady", metav1.ConditionFalse, "IdentityLedgerInvalid", "tenant canonical human identity ledger is missing, conflicting or invalid")
			setCondition(&bundle.Status.Conditions, bundle.Generation, "Ready", metav1.ConditionFalse, "OpenFGAIdentityReconcileFailed", "tenant canonical human identity ledger is not reconciled")
			if !reflect.DeepEqual(previousStatus, bundle.Status) {
				_ = r.Status().Update(ctx, &bundle)
			}
			return ctrl.Result{}, err
		}
		setCondition(&bundle.Status.Conditions, bundle.Generation, "OpenFGAAuthorizationReady", metav1.ConditionFalse, "TupleSyncNotImplemented", "Authentik membership and Fabric grant reconciliation are not yet implemented")
	} else {
		setCondition(&bundle.Status.Conditions, bundle.Generation, "OpenFGAStoreReady", metav1.ConditionFalse, "RolloutDisabled", "private tenant authorization store reconciliation is not enabled")
		setCondition(&bundle.Status.Conditions, bundle.Generation, "OpenFGAModelReady", metav1.ConditionFalse, "RolloutDisabled", "private tenant authorization model reconciliation is not enabled")
		setCondition(&bundle.Status.Conditions, bundle.Generation, "OpenFGAAuthorizationReady", metav1.ConditionFalse, "RolloutDisabled", "tenant authorization rollout is not enabled")
	}

	waiting := false
	degraded := false
	requeueAfter := time.Duration(0)

	runAIPlane := tenantRunsAIPlane(&bundle)
	if runAIPlane {
		setCondition(&bundle.Status.Conditions, bundle.Generation, "LifecycleReady", metav1.ConditionTrue, "Active", "tenant lifecycle is Active")
	} else {
		var agents fabricv1alpha1.AgentIdentityList
		if err := r.List(ctx, &agents); err != nil {
			return ctrl.Result{}, err
		}
		remainingAgents := 0
		for i := range agents.Items {
			if agents.Items[i].Spec.TenantRef.Name == bundle.Name && agents.Items[i].DeletionTimestamp.IsZero() {
				remainingAgents++
			}
		}
		if remainingAgents > 0 {
			// Never cut the tenant AI plane out from under live Hermes runtimes.
			// Parking is reversible and must first drain/remove AgentIdentity intent.
			runAIPlane = true
			degraded = true
			setCondition(
				&bundle.Status.Conditions,
				bundle.Generation,
				"LifecycleReady",
				metav1.ConditionFalse,
				"ParkBlockedByAgents",
				fmt.Sprintf("cannot park tenant AI plane while %d AgentIdentity resources remain", remainingAgents),
			)
			requeueAfter = 5 * time.Second
		} else {
			setCondition(&bundle.Status.Conditions, bundle.Generation, "LifecycleReady", metav1.ConditionTrue, "Parked", "tenant lifecycle is Parked")
		}
	}
	if bundle.Spec.Persistence.PostgreSQL != nil {
		result, err := r.reconcilePostgreSQL(ctx, &bundle)
		if err != nil {
			bundle.Status.Persistence.PostgreSQL = &fabricv1alpha1.ComponentStatus{Phase: "Blocked", Message: err.Error()}
			setCondition(&bundle.Status.Conditions, bundle.Generation, "PersistenceReady", metav1.ConditionFalse, "ReconcileError", err.Error())
			bundle.Status.Phase = "Degraded"
			setCondition(&bundle.Status.Conditions, bundle.Generation, "Ready", metav1.ConditionFalse, "PersistenceReconcileFailed", err.Error())
			if !reflect.DeepEqual(previousStatus, bundle.Status) {
				_ = r.Status().Update(ctx, &bundle)
			}
			return ctrl.Result{}, err
		}
		bundle.Status.Persistence.PostgreSQL = result.Status
		if result.Ready {
			setCondition(&bundle.Status.Conditions, bundle.Generation, "PersistenceReady", metav1.ConditionTrue, result.Reason, result.Message)
		} else {
			waiting = true
			if result.Status != nil && result.Status.Phase == "Blocked" {
				degraded = true
			}
			setCondition(&bundle.Status.Conditions, bundle.Generation, "PersistenceReady", metav1.ConditionFalse, result.Reason, result.Message)
			if result.RequeueAfter > 0 && (requeueAfter == 0 || result.RequeueAfter < requeueAfter) {
				requeueAfter = result.RequeueAfter
			}
		}
	} else if hadPostgreSQL {
		pending, err := r.cleanupPostgreSQL(ctx, &bundle)
		if err != nil {
			bundle.Status.Persistence.PostgreSQL = &fabricv1alpha1.ComponentStatus{Phase: "Blocked", Message: err.Error()}
			setCondition(&bundle.Status.Conditions, bundle.Generation, "PersistenceReady", metav1.ConditionFalse, "ReclaimError", err.Error())
			bundle.Status.Phase = "Degraded"
			setCondition(&bundle.Status.Conditions, bundle.Generation, "Ready", metav1.ConditionFalse, "PersistenceReclaimFailed", err.Error())
			if !reflect.DeepEqual(previousStatus, bundle.Status) {
				_ = r.Status().Update(ctx, &bundle)
			}
			return ctrl.Result{}, err
		}
		if pending {
			waiting = true
			bundle.Status.Persistence.PostgreSQL = &fabricv1alpha1.ComponentStatus{Phase: "Reclaiming", Message: "removing Fabric-owned CloudNativePG resources according to their reclaim policies"}
			setCondition(&bundle.Status.Conditions, bundle.Generation, "PersistenceReady", metav1.ConditionFalse, "Reclaiming", "PostgreSQL capability is being released")
			if requeueAfter == 0 || 2*time.Second < requeueAfter {
				requeueAfter = 2 * time.Second
			}
		} else {
			setCondition(&bundle.Status.Conditions, bundle.Generation, "PersistenceReady", metav1.ConditionTrue, "NotRequested", "tenant does not request PostgreSQL capability")
		}
	} else {
		setCondition(&bundle.Status.Conditions, bundle.Generation, "PersistenceReady", metav1.ConditionTrue, "NotRequested", "tenant does not request PostgreSQL capability")
	}

	if bundle.Spec.Memory.Hindsight != nil {
		result, err := r.reconcileHindsight(ctx, &bundle)
		if err != nil {
			bundle.Status.Memory.Hindsight = &fabricv1alpha1.ComponentStatus{Phase: "Blocked", Message: err.Error()}
			setCondition(&bundle.Status.Conditions, bundle.Generation, "MemoryReady", metav1.ConditionFalse, "ReconcileError", err.Error())
			bundle.Status.Phase = "Degraded"
			setCondition(&bundle.Status.Conditions, bundle.Generation, "Ready", metav1.ConditionFalse, "MemoryReconcileFailed", err.Error())
			if !reflect.DeepEqual(previousStatus, bundle.Status) {
				_ = r.Status().Update(ctx, &bundle)
			}
			return ctrl.Result{}, err
		}
		bundle.Status.Memory.Hindsight = result.Status
		if result.Ready {
			setCondition(&bundle.Status.Conditions, bundle.Generation, "MemoryReady", metav1.ConditionTrue, result.Reason, result.Message)
		} else {
			waiting = true
			if result.Status != nil && result.Status.Phase == "Blocked" {
				degraded = true
			}
			setCondition(&bundle.Status.Conditions, bundle.Generation, "MemoryReady", metav1.ConditionFalse, result.Reason, result.Message)
			if result.RequeueAfter > 0 && (requeueAfter == 0 || result.RequeueAfter < requeueAfter) {
				requeueAfter = result.RequeueAfter
			}
		}
	} else if hadHindsight {
		pending, err := r.cleanupHindsight(ctx, &bundle)
		if err != nil {
			bundle.Status.Memory.Hindsight = &fabricv1alpha1.ComponentStatus{Phase: "Blocked", Message: err.Error()}
			setCondition(&bundle.Status.Conditions, bundle.Generation, "MemoryReady", metav1.ConditionFalse, "ReclaimError", err.Error())
			bundle.Status.Phase = "Degraded"
			setCondition(&bundle.Status.Conditions, bundle.Generation, "Ready", metav1.ConditionFalse, "MemoryReclaimFailed", err.Error())
			if !reflect.DeepEqual(previousStatus, bundle.Status) {
				_ = r.Status().Update(ctx, &bundle)
			}
			return ctrl.Result{}, err
		}
		if pending {
			waiting = true
			bundle.Status.Memory.Hindsight = &fabricv1alpha1.ComponentStatus{Phase: "Reclaiming", Message: "removing Fabric-owned tenant Hindsight resources"}
			setCondition(&bundle.Status.Conditions, bundle.Generation, "MemoryReady", metav1.ConditionFalse, "Reclaiming", "Hindsight capability is being released")
			if requeueAfter == 0 || 2*time.Second < requeueAfter {
				requeueAfter = 2 * time.Second
			}
		} else {
			setCondition(&bundle.Status.Conditions, bundle.Generation, "MemoryReady", metav1.ConditionTrue, "NotRequested", "tenant does not request Hindsight memory")
		}
	} else {
		setCondition(&bundle.Status.Conditions, bundle.Generation, "MemoryReady", metav1.ConditionTrue, "NotRequested", "tenant does not request Hindsight memory")
	}


	if runAIPlane {
		// The tenant-local credential broker is part of the mandatory Active
		// tenant AI plane and must be reconciled before LiteLLM resolves it as
		// the txo-agent backend.
		result, err := r.reconcileAICredentialBroker(ctx, &bundle)
		if err != nil {
			bundle.Status.AICredentialBroker = &fabricv1alpha1.ComponentStatus{Phase: "Blocked", Message: err.Error()}
			setCondition(&bundle.Status.Conditions, bundle.Generation, "AICredentialBrokerReady", metav1.ConditionFalse, "ReconcileError", err.Error())
			bundle.Status.Phase = "Degraded"
			setCondition(&bundle.Status.Conditions, bundle.Generation, "Ready", metav1.ConditionFalse, "AICredentialBrokerReconcileFailed", err.Error())
			if !reflect.DeepEqual(previousStatus, bundle.Status) {
				_ = r.Status().Update(ctx, &bundle)
			}
			return ctrl.Result{}, err
		}
		bundle.Status.AICredentialBroker = result.Status
		if result.Ready {
			setCondition(&bundle.Status.Conditions, bundle.Generation, "AICredentialBrokerReady", metav1.ConditionTrue, result.Reason, result.Message)
		} else {
			waiting = true
			if result.Status != nil && result.Status.Phase == "Blocked" {
				degraded = true
			}
			setCondition(&bundle.Status.Conditions, bundle.Generation, "AICredentialBrokerReady", metav1.ConditionFalse, result.Reason, result.Message)
		}

		gatewayResult, err := r.reconcileAIGateway(ctx, &bundle)
		if err != nil {
			bundle.Status.AIGateway = &fabricv1alpha1.ComponentStatus{Phase: "Blocked", Message: err.Error()}
			setCondition(&bundle.Status.Conditions, bundle.Generation, "AIGatewayReady", metav1.ConditionFalse, "ReconcileError", err.Error())
			bundle.Status.Phase = "Degraded"
			setCondition(&bundle.Status.Conditions, bundle.Generation, "Ready", metav1.ConditionFalse, "AIGatewayReconcileFailed", err.Error())
			if !reflect.DeepEqual(previousStatus, bundle.Status) {
				_ = r.Status().Update(ctx, &bundle)
			}
			return ctrl.Result{}, err
		}
		bundle.Status.AIGateway = gatewayResult.Status
		if gatewayResult.RequeueAfter > 0 && (requeueAfter == 0 || gatewayResult.RequeueAfter < requeueAfter) {
			requeueAfter = gatewayResult.RequeueAfter
		}
		if gatewayResult.Ready {
			setCondition(&bundle.Status.Conditions, bundle.Generation, "AIGatewayReady", metav1.ConditionTrue, gatewayResult.Reason, gatewayResult.Message)
		} else {
			waiting = true
			if gatewayResult.Status != nil && gatewayResult.Status.Phase == "Blocked" {
				degraded = true
			}
			setCondition(&bundle.Status.Conditions, bundle.Generation, "AIGatewayReady", metav1.ConditionFalse, gatewayResult.Reason, gatewayResult.Message)
		}
	} else {
		// Park the stateless/runnable portion of the mandatory AI plane while
		// preserving OAuth state, LiteLLM database state and runtime Secrets for
		// a reversible recovery-shell transition.
		gatewayPending, err := r.parkAIGatewayCompute(ctx, &bundle)
		if err != nil {
			bundle.Status.AIGateway = &fabricv1alpha1.ComponentStatus{Phase: "Blocked", Message: err.Error()}
			setCondition(&bundle.Status.Conditions, bundle.Generation, "AIGatewayReady", metav1.ConditionFalse, "ParkError", err.Error())
			bundle.Status.Phase = "Degraded"
			setCondition(&bundle.Status.Conditions, bundle.Generation, "Ready", metav1.ConditionFalse, "AIGatewayParkFailed", err.Error())
			if !reflect.DeepEqual(previousStatus, bundle.Status) {
				_ = r.Status().Update(ctx, &bundle)
			}
			return ctrl.Result{}, err
		}
		brokerPending, err := r.parkAICredentialBrokerCompute(ctx, &bundle)
		if err != nil {
			bundle.Status.AICredentialBroker = &fabricv1alpha1.ComponentStatus{Phase: "Blocked", Message: err.Error()}
			setCondition(&bundle.Status.Conditions, bundle.Generation, "AICredentialBrokerReady", metav1.ConditionFalse, "ParkError", err.Error())
			bundle.Status.Phase = "Degraded"
			setCondition(&bundle.Status.Conditions, bundle.Generation, "Ready", metav1.ConditionFalse, "AICredentialBrokerParkFailed", err.Error())
			if !reflect.DeepEqual(previousStatus, bundle.Status) {
				_ = r.Status().Update(ctx, &bundle)
			}
			return ctrl.Result{}, err
		}

		if gatewayPending {
			waiting = true
			bundle.Status.AIGateway = &fabricv1alpha1.ComponentStatus{Phase: "Parking", Message: "stopping tenant LiteLLM compute while preserving durable AI-plane state"}
			setCondition(&bundle.Status.Conditions, bundle.Generation, "AIGatewayReady", metav1.ConditionFalse, "Parking", "tenant LiteLLM compute is being stopped")
		} else {
			bundle.Status.AIGateway = &fabricv1alpha1.ComponentStatus{Phase: "Parked", Message: "tenant LiteLLM compute is parked; durable AI-plane state is preserved"}
			setCondition(&bundle.Status.Conditions, bundle.Generation, "AIGatewayReady", metav1.ConditionTrue, "Parked", "tenant LiteLLM compute is parked")
		}
		if brokerPending {
			waiting = true
			bundle.Status.AICredentialBroker = &fabricv1alpha1.ComponentStatus{Phase: "Parking", Message: "stopping tenant CPA compute while preserving OAuth state"}
			setCondition(&bundle.Status.Conditions, bundle.Generation, "AICredentialBrokerReady", metav1.ConditionFalse, "Parking", "tenant CPA compute is being stopped")
		} else {
			bundle.Status.AICredentialBroker = &fabricv1alpha1.ComponentStatus{Phase: "Parked", Message: "tenant CPA compute is parked; OAuth state is preserved"}
			setCondition(&bundle.Status.Conditions, bundle.Generation, "AICredentialBrokerReady", metav1.ConditionTrue, "Parked", "tenant CPA compute is parked")
		}
		if (gatewayPending || brokerPending) && (requeueAfter == 0 || 2*time.Second < requeueAfter) {
			requeueAfter = 2 * time.Second
		}
	}

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

	if openFGARolloutEnabled {
		// Staging store/model publication cannot advertise business readiness
		// without authoritative tuple convergence and revocation.
		bundle.Status.Phase = "Degraded"
		setCondition(&bundle.Status.Conditions, bundle.Generation, "Ready", metav1.ConditionFalse, "AuthorizationTupleSyncPending", "tenant OpenFGA tuple and revocation reconciliation is not implemented")
	} else if degraded {
		bundle.Status.Phase = "Degraded"
		setCondition(&bundle.Status.Conditions, bundle.Generation, "Ready", metav1.ConditionFalse, "CapabilityBlocked", "one or more declared capabilities are blocked")
	} else if waiting {
		bundle.Status.Phase = "Provisioning"
		setCondition(&bundle.Status.Conditions, bundle.Generation, "Ready", metav1.ConditionFalse, "WaitingForCapabilities", "cell baseline is ready; declared persistence, memory, modules, or lifecycle transitions are still pending")
	} else if !tenantRunsAIPlane(&bundle) {
		bundle.Status.Phase = "Parked"
		setCondition(&bundle.Status.Conditions, bundle.Generation, "Ready", metav1.ConditionTrue, "Parked", "tenant recovery shell is reconciled with the mandatory AI-plane compute stopped")
	} else {
		bundle.Status.Phase = "Ready"
		setCondition(&bundle.Status.Conditions, bundle.Generation, "Ready", metav1.ConditionTrue, "Reconciled", "tenant cell baseline and mandatory AI plane are ready")
	}

	if !reflect.DeepEqual(previousStatus, bundle.Status) {
		if err := r.Status().Update(ctx, &bundle); err != nil {
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{RequeueAfter: requeueAfter}, nil
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

	retainedPVCs, err := r.retainedAgentPVCCount(ctx, bundle)
	if err != nil {
		return ctrl.Result{}, err
	}
	if retainedPVCs > 0 {
		bundle.Status.Phase = "Deleting"
		bundle.Status.ObservedGeneration = bundle.Generation
		setCondition(&bundle.Status.Conditions, bundle.Generation, "Ready", metav1.ConditionFalse, "RetainedAgentStorage", fmt.Sprintf("%d retained agent PVCs block tenant namespace deletion; delete them explicitly or recreate the AgentIdentity with runtime.storage.retentionPolicy=Delete before retrying", retainedPVCs))
		_ = r.Status().Update(ctx, bundle)
		return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
	}

	if err := r.reconcileAuthentikBlueprint(ctx); err != nil {
		return ctrl.Result{}, err
	}

	if bundle.Spec.AICredentialBroker != nil || bundle.Status.AICredentialBroker != nil {
		pending, err := r.cleanupAICredentialBroker(ctx, bundle)
		if err != nil {
			return ctrl.Result{}, err
		}
		if pending {
			bundle.Status.Phase = "Deleting"
			bundle.Status.AICredentialBroker = &fabricv1alpha1.ComponentStatus{Phase: "Reclaiming", Message: "removing Fabric-owned tenant AI credential broker resources"}
			setCondition(&bundle.Status.Conditions, bundle.Generation, "Ready", metav1.ConditionFalse, "AICredentialBrokerReclaiming", "tenant AI credential broker resources are being released")
			_ = r.Status().Update(ctx, bundle)
			return ctrl.Result{RequeueAfter: 2 * time.Second}, nil
		}
	}

	if bundle.Spec.Memory.Hindsight != nil || bundle.Status.Memory.Hindsight != nil {
		pending, err := r.cleanupHindsight(ctx, bundle)
		if err != nil {
			return ctrl.Result{}, err
		}
		if pending {
			bundle.Status.Phase = "Deleting"
			bundle.Status.Memory.Hindsight = &fabricv1alpha1.ComponentStatus{Phase: "Reclaiming", Message: "removing Fabric-owned tenant Hindsight resources"}
			setCondition(&bundle.Status.Conditions, bundle.Generation, "Ready", metav1.ConditionFalse, "HindsightReclaiming", "tenant Hindsight resources are being released before persistence")
			_ = r.Status().Update(ctx, bundle)
			return ctrl.Result{RequeueAfter: 2 * time.Second}, nil
		}
	}

	if bundle.Spec.Persistence.PostgreSQL != nil || bundle.Status.Persistence.PostgreSQL != nil {
		pending, err := r.cleanupPostgreSQL(ctx, bundle)
		if err != nil {
			return ctrl.Result{}, err
		}
		if pending {
			bundle.Status.Phase = "Deleting"
			bundle.Status.Persistence.PostgreSQL = &fabricv1alpha1.ComponentStatus{Phase: "Reclaiming", Message: "removing Fabric-owned CloudNativePG resources according to their reclaim policies"}
			setCondition(&bundle.Status.Conditions, bundle.Generation, "Ready", metav1.ConditionFalse, "PostgreSQLReclaiming", "tenant PostgreSQL resources are being released")
			_ = r.Status().Update(ctx, bundle)
			return ctrl.Result{RequeueAfter: 2 * time.Second}, nil
		}
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

func tenantBundleRequestsForAIProviderSecret(_ context.Context, obj client.Object) []reconcile.Request {
	if obj.GetNamespace() != tenantAIProviderSecretNamespace {
		return nil
	}
	name := strings.TrimSpace(obj.GetName())
	if !strings.HasPrefix(name, tenantAIProviderSecretPrefix) {
		return nil
	}
	tenantName := strings.TrimSpace(strings.TrimPrefix(name, tenantAIProviderSecretPrefix))
	if tenantName == "" {
		return nil
	}
	return []reconcile.Request{{NamespacedName: types.NamespacedName{Name: tenantName}}}
}

func (r *TenantBundleReconciler) tenantBundleRequestsForPostgreSQLProfile(ctx context.Context, obj client.Object) []reconcile.Request {
	profileName := obj.GetName()
	if profileName == "" {
		return nil
	}

	gatewayProfiles := map[string]struct{}{}
	var aiGatewayProfiles fabricv1alpha1.AIGatewayProfileList
	if err := r.List(ctx, &aiGatewayProfiles); err != nil {
		ctrl.LoggerFrom(ctx).Error(err, "unable to list AIGatewayProfiles for PostgreSQLProfile watch", "profile", profileName)
		return nil
	}
	for i := range aiGatewayProfiles.Items {
		postgresqlProfileRef := strings.TrimSpace(aiGatewayProfiles.Items[i].Spec.PostgreSQLProfileRef)
		if postgresqlProfileRef == "" {
			postgresqlProfileRef = defaultAIGatewayPostgreSQLProfileName
		}
		if postgresqlProfileRef == profileName {
			gatewayProfiles[aiGatewayProfiles.Items[i].Name] = struct{}{}
		}
	}

	var bundles fabricv1alpha1.TenantBundleList
	if err := r.List(ctx, &bundles); err != nil {
		ctrl.LoggerFrom(ctx).Error(err, "unable to list TenantBundles for PostgreSQLProfile watch", "profile", profileName)
		return nil
	}
	requests := make([]reconcile.Request, 0)
	seen := map[string]struct{}{}
	for i := range bundles.Items {
		bundle := &bundles.Items[i]
		postgresql := bundle.Spec.Persistence.PostgreSQL
		if postgresql != nil && postgresql.ProfileRef == profileName {
			requests = append(requests, reconcile.Request{NamespacedName: types.NamespacedName{Name: bundle.Name}})
			seen[bundle.Name] = struct{}{}
		}

		if !tenantRunsAIPlane(bundle) {
			continue
		}
		if _, matches := gatewayProfiles[tenantAIGatewayProfileName(bundle)]; matches {
			if _, duplicate := seen[bundle.Name]; !duplicate {
				requests = append(requests, reconcile.Request{NamespacedName: types.NamespacedName{Name: bundle.Name}})
				seen[bundle.Name] = struct{}{}
			}
		}
	}
	return requests
}

func (r *TenantBundleReconciler) tenantBundleRequestsForHindsightProfile(ctx context.Context, obj client.Object) []reconcile.Request {
	profileName := obj.GetName()
	if profileName == "" {
		return nil
	}
	var bundles fabricv1alpha1.TenantBundleList
	if err := r.List(ctx, &bundles); err != nil {
		ctrl.LoggerFrom(ctx).Error(err, "unable to list TenantBundles for HindsightProfile watch", "profile", profileName)
		return nil
	}
	requests := make([]reconcile.Request, 0)
	for i := range bundles.Items {
		hindsight := bundles.Items[i].Spec.Memory.Hindsight
		if hindsight != nil && hindsight.ProfileRef == profileName {
			requests = append(requests, reconcile.Request{NamespacedName: types.NamespacedName{Name: bundles.Items[i].Name}})
		}
	}
	return requests
}

func (r *TenantBundleReconciler) tenantBundleRequestsForAICredentialBrokerProfile(ctx context.Context, obj client.Object) []reconcile.Request {
	profileName := obj.GetName()
	if profileName == "" {
		return nil
	}
	var bundles fabricv1alpha1.TenantBundleList
	if err := r.List(ctx, &bundles); err != nil {
		ctrl.LoggerFrom(ctx).Error(err, "unable to list TenantBundles for AICredentialBrokerProfile watch", "profile", profileName)
		return nil
	}
	requests := make([]reconcile.Request, 0)
	for i := range bundles.Items {
		bundle := &bundles.Items[i]
		if !tenantRunsAIPlane(bundle) {
			continue
		}
		if tenantAICredentialBrokerProfileName(bundle) == profileName {
			requests = append(requests, reconcile.Request{NamespacedName: types.NamespacedName{Name: bundle.Name}})
		}
	}
	return requests
}

func (r *TenantBundleReconciler) tenantBundleRequestsForAIGatewayProfile(ctx context.Context, obj client.Object) []reconcile.Request {
	profileName := obj.GetName()
	if profileName == "" {
		return nil
	}
	var bundles fabricv1alpha1.TenantBundleList
	if err := r.List(ctx, &bundles); err != nil {
		ctrl.LoggerFrom(ctx).Error(err, "unable to list TenantBundles for AIGatewayProfile watch", "profile", profileName)
		return nil
	}
	requests := make([]reconcile.Request, 0)
	for i := range bundles.Items {
		bundle := &bundles.Items[i]
		if !tenantRunsAIPlane(bundle) {
			continue
		}
		if tenantAIGatewayProfileName(bundle) == profileName {
			requests = append(requests, reconcile.Request{NamespacedName: types.NamespacedName{Name: bundle.Name}})
		}
	}
	return requests
}

func (r *TenantBundleReconciler) SetupWithManager(mgr ctrl.Manager) error {
	AddCNPGToScheme(mgr.GetScheme())
	if r.APIReader == nil {
		r.APIReader = mgr.GetAPIReader()
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&fabricv1alpha1.TenantBundle{}).
		Watches(&fabricv1alpha1.PostgreSQLProfile{}, handler.EnqueueRequestsFromMapFunc(r.tenantBundleRequestsForPostgreSQLProfile)).
		Watches(&fabricv1alpha1.HindsightProfile{}, handler.EnqueueRequestsFromMapFunc(r.tenantBundleRequestsForHindsightProfile)).
		Watches(&fabricv1alpha1.AIGatewayProfile{}, handler.EnqueueRequestsFromMapFunc(r.tenantBundleRequestsForAIGatewayProfile)).
		Watches(&fabricv1alpha1.AICredentialBrokerProfile{}, handler.EnqueueRequestsFromMapFunc(r.tenantBundleRequestsForAICredentialBrokerProfile)).
		Watches(&corev1.Namespace{}, handler.EnqueueRequestsFromMapFunc(tenantBundleRequestsForManagedObject)).
		Watches(&corev1.Service{}, handler.EnqueueRequestsFromMapFunc(tenantBundleRequestsForManagedObject)).
		Watches(&corev1.Secret{}, handler.EnqueueRequestsFromMapFunc(tenantBundleRequestsForManagedObject)).
		Watches(&corev1.Secret{}, handler.EnqueueRequestsFromMapFunc(tenantBundleRequestsForAIProviderSecret)).
		Watches(&corev1.ConfigMap{}, handler.EnqueueRequestsFromMapFunc(r.tenantBundleRequestsForAuthentikBlueprint)).
		Watches(&appsv1.Deployment{}, handler.EnqueueRequestsFromMapFunc(tenantBundleRequestsForManagedObject)).
		Watches(&batchv1.Job{}, handler.EnqueueRequestsFromMapFunc(tenantBundleRequestsForManagedObject)).
		Watches(&networkingv1.Ingress{}, handler.EnqueueRequestsFromMapFunc(tenantBundleRequestsForManagedObject)).
		Watches(&networkingv1.NetworkPolicy{}, handler.EnqueueRequestsFromMapFunc(tenantBundleRequestsForManagedObject)).
		Watches(cnpgObject(cnpgDatabaseGVK, "", ""), handler.EnqueueRequestsFromMapFunc(tenantBundleRequestsForManagedObject)).
		Watches(cnpgObject(cnpgDatabaseRoleGVK, "", ""), handler.EnqueueRequestsFromMapFunc(tenantBundleRequestsForManagedObject)).
		Complete(r)
}
