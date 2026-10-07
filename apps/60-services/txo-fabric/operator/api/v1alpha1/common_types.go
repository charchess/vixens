package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

type ObjectReference struct {
	// Name is the metadata.name of the referenced cluster-scoped Fabric object.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=253
	Name string `json:"name"`
}

type TenantLifecycleSpec struct {
	// Mode controls the lifecycle of the mandatory tenant AI plane.
	// Active is the default and receives the canonical tenant AI plane.
	// Parked preserves the recovery shell and durable AI-plane state without
	// running tenant LiteLLM/CPA compute. Optional capabilities keep their own
	// declaration/lifecycle contracts.
	// +optional
	// +kubebuilder:default=Active
	// +kubebuilder:validation:Enum=Active;Parked
	Mode string `json:"mode,omitempty"`
}

type TenantIsolationSpec struct {
	// Profile selects the isolation policy for the tenant cell.
	// +kubebuilder:validation:Enum=Shared;Dedicated
	// +kubebuilder:default=Shared
	Profile string `json:"profile,omitempty"`
}

type PostgreSQLSpec struct {
	// Mode selects logical resources on the platform shared cluster or a tenant-dedicated cluster.
	// +kubebuilder:validation:Enum=Shared;Dedicated
	// +kubebuilder:default=Shared
	Mode string `json:"mode,omitempty"`

	// ProfileRef selects a platform-owned database profile. The profile implementation is intentionally
	// outside TenantBundle so tenants do not embed infrastructure credentials or topology.
	// +optional
	// +kubebuilder:validation:MaxLength=128
	ProfileRef string `json:"profileRef,omitempty"`
}

type TenantPersistenceSpec struct {
	// PostgreSQL requests PostgreSQL capability for this tenant. Nil means no PostgreSQL dependency.
	// +optional
	PostgreSQL *PostgreSQLSpec `json:"postgresql,omitempty"`
}

type HindsightMemorySpec struct {
	// ProfileRef selects the Hindsight service profile owned by TXO Fabric.
	// +optional
	// +kubebuilder:validation:MaxLength=128
	ProfileRef string `json:"profileRef,omitempty"`

	// HumanAccess opts this tenant into the authenticated Hindsight Control Plane
	// WebUI using TenantBundle.spec.humanAccess.web for DNS/TLS/ingress policy.
	// +optional
	// +kubebuilder:default=false
	HumanAccess bool `json:"humanAccess,omitempty"`
}

type TenantMemorySpec struct {
	// Hindsight enables tenant-scoped durable agent memory through Hindsight.
	// +optional
	Hindsight *HindsightMemorySpec `json:"hindsight,omitempty"`
}

type TenantAIGatewaySpec struct {
	// ProfileRef selects the tenant-facing LiteLLM gateway profile.
	// +optional
	// +kubebuilder:default=litellm-standard
	// +kubebuilder:validation:MaxLength=128
	ProfileRef string `json:"profileRef,omitempty"`
}

type TenantAICredentialBrokerSpec struct {
	// ProfileRef selects the tenant-local OAuth/subscription credential broker profile.
	// +optional
	// +kubebuilder:default=cliproxyapi-standard
	// +kubebuilder:validation:MaxLength=128
	ProfileRef string `json:"profileRef,omitempty"`
}

type TenantModuleSpec struct {
	// Name is the stable Fabric module key (for example paperclip or valkey).
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=63
	// +kubebuilder:validation:Pattern=`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`
	Name string `json:"name"`

	// Enabled defaults to true when the module entry is present.
	// +kubebuilder:default=true
	Enabled bool `json:"enabled,omitempty"`

	// ProfileRef selects a platform-owned implementation profile for the module.
	// +optional
	// +kubebuilder:validation:MaxLength=128
	ProfileRef string `json:"profileRef,omitempty"`
}

type ComponentStatus struct {
	// Phase is a concise machine-readable lifecycle state.
	// +optional
	Phase string `json:"phase,omitempty"`

	// Endpoint is the resolved in-cluster endpoint, when the component exposes one.
	// +optional
	Endpoint string `json:"endpoint,omitempty"`

	// HumanEndpoint is the stable authenticated human-facing endpoint when the
	// component exposes an operator/admin WebUI.
	// +optional
	HumanEndpoint string `json:"humanEndpoint,omitempty"`

	// Message contains a human-readable explanation for non-ready states.
	// +optional
	Message string `json:"message,omitempty"`
}

type TenantPersistenceStatus struct {
	// +optional
	PostgreSQL *ComponentStatus `json:"postgresql,omitempty"`
}

type TenantMemoryStatus struct {
	// +optional
	Hindsight *ComponentStatus `json:"hindsight,omitempty"`
}

type TenantModuleStatus struct {
	Name            string `json:"name"`
	ComponentStatus `json:",inline"`
}

type IntegrationAuthorizationStatus struct {
	// BindingName is the Fabric authorization object evaluated for this runtime.
	BindingName string `json:"bindingName"`

	// ConnectionName is the logical integration connection referenced by the binding.
	ConnectionName string `json:"connectionName"`

	// Phase is Effective, Revoked or Denied. It never carries credential material.
	Phase string `json:"phase"`

	// Reason is a stable non-secret diagnostic explaining the phase.
	Reason string `json:"reason,omitempty"`

	// Revision changes when binding, connection or credential metadata changes.
	Revision string `json:"revision,omitempty"`

	// Operations are the effective generic operations requested by the binding.
	// +optional
	Operations []string `json:"operations,omitempty"`

	// Scopes are the effective connection-defined scope labels.
	// +optional
	Scopes []string `json:"scopes,omitempty"`
}

type RuntimeStatus struct {
	// DeploymentName is the generated Hermes Deployment.
	// +optional
	DeploymentName string `json:"deploymentName,omitempty"`

	// PVCName is the generated runtime workspace PVC.
	// +optional
	PVCName string `json:"pvcName,omitempty"`

	// HumanEndpoint is the stable authenticated human-facing endpoint when the
	// AgentIdentity is opted into tenant human access.
	// +optional
	HumanEndpoint string `json:"humanEndpoint,omitempty"`

	// ToolsetPolicyRevision identifies the effective platform-owned Hermes toolset policy.
	// +optional
	ToolsetPolicyRevision string `json:"toolsetPolicyRevision,omitempty"`

	// EnabledToolsets lists toolsets admitted by the effective platform policy.
	// +optional
	EnabledToolsets []string `json:"enabledToolsets,omitempty"`

	// DeniedToolsets lists profile-declared toolsets denied by the effective platform policy.
	// +optional
	DeniedToolsets []string `json:"deniedToolsets,omitempty"`

	// IntegrationPolicyRevision identifies the effective non-secret integration projection.
	// +optional
	IntegrationPolicyRevision string `json:"integrationPolicyRevision,omitempty"`

	// Integrations exposes effective/revoked/denied Fabric authorization without secret values.
	// +optional
	Integrations []IntegrationAuthorizationStatus `json:"integrations,omitempty"`
}

type MemoryBindingStatus struct {
	// BankID is the resolved logical memory-bank identity.
	// +optional
	BankID string `json:"bankId,omitempty"`

	// Phase describes memory provisioning for this identity.
	// +optional
	Phase string `json:"phase,omitempty"`
}

func conditionSliceCopy(in []metav1.Condition) []metav1.Condition {
	if in == nil {
		return nil
	}
	out := make([]metav1.Condition, len(in))
	copy(out, in)
	return out
}
