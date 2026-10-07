package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +kubebuilder:validation:XValidation:rule="!has(self.adoptLegacyProfile) || !self.adoptLegacyProfile || self.retentionPolicy == 'Retain'",message="adoptLegacyProfile requires retentionPolicy Retain"
// +kubebuilder:validation:XValidation:rule="!has(oldSelf.adoptLegacyProfile) || (has(self.adoptLegacyProfile) && self.adoptLegacyProfile == oldSelf.adoptLegacyProfile)",message="adoptLegacyProfile cannot be changed once declared"
type AgentRuntimeStorageBinding struct {
	// RetentionPolicy controls what happens to the Hermes runtime PVC when the
	// AgentIdentity is deleted. Retain is the safe default for durable customer
	// workspaces; Delete is an explicit opt-in for disposable identities.
	// +kubebuilder:default=Retain
	// +kubebuilder:validation:Enum=Retain;Delete
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="retentionPolicy is immutable"
	RetentionPolicy string `json:"retentionPolicy,omitempty"`

	// AdoptLegacyProfile non-destructively presents an existing retained
	// profiles/<agentKey> directory as the runtime's single /opt/data home.
	// It is a brownfield cutover switch: no profile copy or flattening occurs.
	// +optional
	AdoptLegacyProfile bool `json:"adoptLegacyProfile,omitempty"`
}

type AgentRuntimeCapabilityBinding struct {
	// EnableToolsets activates profile entries whose state is AllowedOff.
	// On entries require no opt-in; Off or undeclared entries cannot be requested.
	// +optional
	// +kubebuilder:validation:MaxItems=64
	// +listType=set
	EnableToolsets []string `json:"enableToolsets,omitempty"`
}

type AgentRuntimeBinding struct {
	// ProfileRef references an AgentRuntimeProfile by metadata.name.
	// +kubebuilder:default=hermes-default
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=128
	ProfileRef string `json:"profileRef,omitempty"`

	// Storage defines lifecycle intent for the identity's runtime workspace.
	// +optional
	Storage AgentRuntimeStorageBinding `json:"storage,omitempty"`

	// Capabilities selects only profile-declared AllowedOff capabilities.
	// +optional
	Capabilities AgentRuntimeCapabilityBinding `json:"capabilities,omitempty"`
}

// AgentFunctionalBinding optionally selects a tenant-owned reusable business baseline.
// It does not grant workspace, model, integration or executable-tool permissions.
type AgentFunctionalBinding struct {
	// +optional
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=253
	ProfileRef string `json:"profileRef,omitempty"`
}

type AgentMemoryBinding struct {
	// BankID optionally pins a logical bank identifier. When omitted the operator resolves
	// <tenant-name>-<agent-key> and records the result in status.memory.bankId.
	// +optional
	// +kubebuilder:validation:MinLength=2
	// +kubebuilder:validation:MaxLength=128
	// +kubebuilder:validation:Pattern=`^[a-z0-9]([a-z0-9._-]*[a-z0-9])?$`
	BankID string `json:"bankId,omitempty"`
}

type AgentHumanAccessBinding struct {
	// Enabled opts this stable AgentIdentity into the tenant's configured human
	// entry transport. Transport/OIDC details stay tenant-owned so AgentIdentity
	// remains channel-neutral.
	// +optional
	Enabled bool `json:"enabled,omitempty"`
}

type AgentIdentitySpec struct {
	// TenantRef references the owning TenantBundle by metadata.name.
	// +kubebuilder:validation:XValidation:rule="self.name == oldSelf.name",message="tenantRef.name is immutable"
	TenantRef ObjectReference `json:"tenantRef"`

	// AgentKey is the immutable tenant-local machine identifier. AgentIdentity is cluster-scoped,
	// so metadata.name is globally unique (for example hairem-sandbox-tina), while agentKey stays
	// stable and concise inside the tenant cell (for example tina). Runtime resource names and the
	// default memory bank ID are derived from this key, not from the global CR name.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=63
	// +kubebuilder:validation:Pattern=`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="agentKey is immutable"
	AgentKey string `json:"agentKey"`

	// DisplayName is the user-facing identity name.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=128
	DisplayName string `json:"displayName"`

	// Runtime selects how this identity is executed. Infrastructure details remain in the profile.
	// +optional
	Runtime AgentRuntimeBinding `json:"runtime,omitempty"`

	// Functional selects a reusable business baseline without changing identity, private state or runtime ceiling.
	// +optional
	Functional AgentFunctionalBinding `json:"functional,omitempty"`

	// Memory binds the identity to its logical tenant memory bank.
	// +optional
	Memory AgentMemoryBinding `json:"memory,omitempty"`

	// Access selects tenant-declared user/group shared workspace scopes.
	// +optional
	Access AgentAccessSpec `json:"access,omitempty"`

	// HumanAccess opts this identity into the tenant-scoped authenticated human
	// entry boundary. The tenant owns transport, public-domain and IAM settings.
	// +optional
	HumanAccess AgentHumanAccessBinding `json:"humanAccess,omitempty"`
}

type AgentIdentityStatus struct {
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// +kubebuilder:validation:Enum=Pending;Provisioning;Ready;AuthBlocked;Degraded;Failed;Deleting
	// +optional
	Phase string `json:"phase,omitempty"`

	// +optional
	// +kubebuilder:validation:MaxLength=63
	Namespace string `json:"namespace,omitempty"`

	// +optional
	Runtime RuntimeStatus `json:"runtime,omitempty"`

	// +optional
	Memory MemoryBindingStatus `json:"memory,omitempty"`

	// +optional
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,shortName=agentid,categories=txo-fabric
// +kubebuilder:printcolumn:name="Tenant",type=string,JSONPath=`.spec.tenantRef.name`
// +kubebuilder:printcolumn:name="Key",type=string,JSONPath=`.spec.agentKey`
// +kubebuilder:printcolumn:name="Agent",type=string,JSONPath=`.spec.displayName`
// +kubebuilder:printcolumn:name="Bank",type=string,JSONPath=`.status.memory.bankId`
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
type AgentIdentity struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AgentIdentitySpec   `json:"spec,omitempty"`
	Status AgentIdentityStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type AgentIdentityList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AgentIdentity `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AgentIdentity{}, &AgentIdentityList{})
}
