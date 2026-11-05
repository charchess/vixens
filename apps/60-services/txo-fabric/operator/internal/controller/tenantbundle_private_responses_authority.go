package controller

import (
	"context"
	"time"

	fabricv1alpha1 "github.com/charchess/vixens/apps/60-services/txo-fabric/operator/api/v1alpha1"
	"github.com/charchess/vixens/apps/60-services/txo-fabric/operator/internal/privatebridge"
	apiMeta "k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ResolvePrivateResponsesAuthority resolves only server-requested, signed
// lifecycle operations. Each operation re-reads canonical tenant/agent state;
// it never returns a Kubernetes endpoint, API credential, or policy metadata.
func (r *TenantBundleReconciler) ResolvePrivateResponsesAuthority(ctx context.Context, q privatebridge.ResponsesAuthorityRequest, now time.Time) (privatebridge.ResponsesAuthorityResponse, error) {
	if r == nil || r.Client == nil || now.IsZero() {
		return privatebridge.ResponsesAuthorityResponse{}, errBrowserAuthorityUnavailable
	}
	bundle, err := r.responsesAuthorityTenant(ctx, q.TenantKey, q.FabricTenantID, now)
	if err != nil {
		return privatebridge.ResponsesAuthorityResponse{}, err
	}

	switch q.Operation {
	case privatebridge.ResponsesAuthorityResolveTenant:
		return privatebridge.ResponsesAuthorityResponse{Tenant: &privatebridge.ResponsesTenantAuthority{
			TenantKey: bundle.Name, FabricTenantID: bundle.Spec.TenantID,
			TenantBundleUID: string(bundle.UID), Status: "Ready",
		}}, nil
	case privatebridge.ResponsesAuthorityResolveAgent:
		agent, err := r.responsesAuthorityAgent(ctx, bundle, q.AgentKey)
		if err != nil {
			return privatebridge.ResponsesAuthorityResponse{}, err
		}
		return privatebridge.ResponsesAuthorityResponse{Agent: responsesAuthorityAgent(bundle, agent)}, nil
	case privatebridge.ResponsesAuthorityAuthorizeChat:
		agent, err := r.responsesAuthorityAgent(ctx, bundle, q.AgentKey)
		if err != nil || !r.browserAgentAllowed(ctx, bundle, agent, q.FabricUserID) {
			return privatebridge.ResponsesAuthorityResponse{}, errBrowserAuthorityUnavailable
		}
		return privatebridge.ResponsesAuthorityResponse{Authorization: &privatebridge.ResponsesAuthorizationAuthority{Allowed: true}}, nil
	case privatebridge.ResponsesAuthorityVerifyRuntime:
		if q.TenantBundleUID != string(bundle.UID) || q.TenantGeneration != bundle.Generation {
			return privatebridge.ResponsesAuthorityResponse{}, errBrowserAuthorityUnavailable
		}
		agent, err := r.responsesAuthorityAgent(ctx, bundle, q.AgentKey)
		if err != nil || q.AgentIdentityUID != string(agent.UID) || q.AgentGeneration != agent.Generation {
			return privatebridge.ResponsesAuthorityResponse{}, errBrowserAuthorityUnavailable
		}
		reader := client.Reader(r.Client)
		if r.APIReader != nil {
			reader = r.APIReader
		}
		ready, err := privateHermesAPIEndpointAttested(ctx, reader, agent, bundle, tenantNamespace(bundle.Name))
		if err != nil || !ready {
			return privatebridge.ResponsesAuthorityResponse{}, errBrowserAuthorityUnavailable
		}
		return privatebridge.ResponsesAuthorityResponse{Runtime: &privatebridge.ResponsesRuntimeAuthority{Attested: true}}, nil
	default:
		return privatebridge.ResponsesAuthorityResponse{}, errBrowserAuthorityUnavailable
	}
}

func (r *TenantBundleReconciler) responsesAuthorityTenant(ctx context.Context, tenantKey, fabricTenantID string, now time.Time) (*fabricv1alpha1.TenantBundle, error) {
	reader := client.Reader(r.Client)
	if r.APIReader != nil {
		reader = r.APIReader
	}
	var bundle fabricv1alpha1.TenantBundle
	if err := reader.Get(ctx, types.NamespacedName{Name: tenantKey}, &bundle); err != nil ||
		!browserTenantCurrent(&bundle, now) || bundle.Spec.TenantID != fabricTenantID {
		return nil, errBrowserAuthorityUnavailable
	}
	return bundle.DeepCopy(), nil
}

func (r *TenantBundleReconciler) responsesAuthorityAgent(ctx context.Context, bundle *fabricv1alpha1.TenantBundle, agentKey string) (*fabricv1alpha1.AgentIdentity, error) {
	if bundle == nil {
		return nil, errBrowserAuthorityUnavailable
	}
	reader := client.Reader(r.Client)
	if r.APIReader != nil {
		reader = r.APIReader
	}
	var agents fabricv1alpha1.AgentIdentityList
	if err := reader.List(ctx, &agents); err != nil {
		return nil, errBrowserAuthorityUnavailable
	}
	var selected *fabricv1alpha1.AgentIdentity
	for i := range agents.Items {
		agent := &agents.Items[i]
		if agent.Spec.TenantRef.Name != bundle.Name || agent.Spec.AgentKey != agentKey {
			continue
		}
		reason, _ := privateHermesAPIIntent(agent, bundle)
		if selected != nil || agent.Name == "" || agent.UID == "" || agent.DeletionTimestamp != nil ||
			agent.Generation < 1 || agent.Status.ObservedGeneration != agent.Generation ||
			agent.Status.Phase != "Ready" || !agent.Spec.HumanAccess.Enabled ||
			reason != "NotReady" {
			return nil, errBrowserAuthorityUnavailable
		}
		condition := apiMeta.FindStatusCondition(agent.Status.Conditions, privateHermesAPICondition)
		if condition == nil || condition.Status != "True" || condition.ObservedGeneration != agent.Generation || condition.Reason != "EndpointAttested" {
			return nil, errBrowserAuthorityUnavailable
		}
		selected = agent.DeepCopy()
	}
	if selected == nil {
		return nil, errBrowserAuthorityUnavailable
	}
	return selected, nil
}

func responsesAuthorityAgent(bundle *fabricv1alpha1.TenantBundle, agent *fabricv1alpha1.AgentIdentity) *privatebridge.ResponsesAgentAuthority {
	return &privatebridge.ResponsesAgentAuthority{
		TenantKey: bundle.Name, FabricTenantID: bundle.Spec.TenantID,
		AgentKey: agent.Spec.AgentKey, AgentIdentityUID: string(agent.UID),
		AgentGeneration: agent.Generation, Status: "Ready",
	}
}
