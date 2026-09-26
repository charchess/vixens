package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

type AgentRuntimeBinding struct {
	// ProfileRef references an AgentRuntimeProfile by metadata.name.
	// +kubebuilder:default=hermes-default
	ProfileRef string `json:"profileRef,omitempty"`
}

type AgentMemoryBinding struct {
	// BankID optionally pins a logical bank identifier. When omitted the operator resolves
	// <tenant-name>-<agent-name> and records the result in status.memory.bankId.
	// +optional
	// +kubebuilder:validation:Pattern=`^[a-z0-9]([a-z0-9._-]*[a-z0-9])?$`
	BankID string `json:"bankId,omitempty"`
}

type AgentIdentitySpec struct {
	// TenantRef references the owning TenantBundle by metadata.name.
	// +kubebuilder:validation:XValidation:rule="self.name == oldSelf.name",message="tenantRef.name is immutable"
	TenantRef ObjectReference `json:"tenantRef"`

	// DisplayName is the user-facing identity name.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=128
	DisplayName string `json:"displayName"`

	// Runtime selects how this identity is executed. Infrastructure details remain in the profile.
	// +optional
	Runtime AgentRuntimeBinding `json:"runtime,omitempty"`

	// Memory binds the identity to its logical tenant memory bank.
	// +optional
	Memory AgentMemoryBinding `json:"memory,omitempty"`
}

type AgentIdentityStatus struct {
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// +kubebuilder:validation:Enum=Pending;Provisioning;Ready;AuthBlocked;Degraded;Failed;Deleting
	// +optional
	Phase string `json:"phase,omitempty"`

	// +optional
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
// +kubebuilder:resource:scope=Cluster,shortName=agentid
// +kubebuilder:printcolumn:name="Tenant",type=string,JSONPath=`.spec.tenantRef.name`
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
