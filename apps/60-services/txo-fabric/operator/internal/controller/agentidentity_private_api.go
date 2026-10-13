package controller

import (
	"regexp"

	fabricv1alpha1 "github.com/charchess/vixens/apps/60-services/txo-fabric/operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	privateHermesAPICondition = "PrivateHermesAPIReady"
	privateHermesAPICredentialPrefix = "hermes-api-"
)

var (
	privateHermesAPITenantID = regexp.MustCompile(`^TEN[0-9]{5,}$`)
	privateHermesAPIObjectName = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`)
)

// privateHermesAPIIntent validates *non-secret* future API intent. No
// operation in this version launches the API server or creates/mounts an API
// credential, Service, NetworkPolicy, ExternalSecret or port. "NotReady" is
// mandatory even for a completely valid request until separately approved
// physical credential, endpoint and least-privilege network checks exist.
func privateHermesAPIIntent(agent *fabricv1alpha1.AgentIdentity, tenant *fabricv1alpha1.TenantBundle) (string, string) {
	if agent == nil || tenant == nil {
		return "InvalidConfiguration", "canonical AgentIdentity and TenantBundle are required"
	}
	api := agent.Spec.Runtime.PrivateAPI
	if api == nil {
		return "Disabled", "private Hermes API is disabled"
	}
	if !api.Enabled {
		if api.TenantID != "" || api.CredentialRef != "" {
			return "InvalidConfiguration", "disabled private API cannot declare credential or tenant bindings"
		}
		return "Disabled", "private Hermes API is disabled"
	}
	// Identity must be read from reconciled Kubernetes objects. No browser
	// tenant/agent/path/endpoint/Service label is accepted as authority.
	if agent.Spec.TenantRef.Name != tenant.Name ||
		agent.Name == "" || agent.UID == "" ||
		agent.Generation < 1 || tenant.UID == "" ||
		tenant.Spec.TenantID == "" ||
		api.TenantID != tenant.Spec.TenantID ||
		len(api.TenantID) > 64 || !privateHermesAPITenantID.MatchString(api.TenantID) ||
		len(agent.Spec.AgentKey) < 1 || len(agent.Spec.AgentKey) > 63 ||
		!privateHermesAPIObjectName.MatchString(agent.Spec.AgentKey) ||
		!privateHermesAPIObjectName.MatchString(tenant.Name) ||
		tenant.DeletionTimestamp != nil || agent.DeletionTimestamp != nil {
		return "InvalidConfiguration", "private API canonical tenant or AgentIdentity binding is invalid"
	}
	expectedRef := privateHermesAPICredentialPrefix + agent.Spec.AgentKey
	if len(expectedRef) > 63 || api.CredentialRef != expectedRef ||
		!privateHermesAPIObjectName.MatchString(api.CredentialRef) {
		return "InvalidConfiguration", "private API credential reference must use the canonical agent key"
	}
	return "NotReady", "private Hermes API activation and endpoint attestation are not implemented"
}

func setPrivateHermesAPICondition(agent *fabricv1alpha1.AgentIdentity, tenant *fabricv1alpha1.TenantBundle) {
	reason, message := privateHermesAPIIntent(agent, tenant)
	setCondition(&agent.Status.Conditions, agent.Generation, privateHermesAPICondition,
		metav1.ConditionFalse, reason, message)
}

