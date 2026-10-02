package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

const (
	IntegrationProtocolHTTP           = "HTTP"
	IntegrationAuthenticationBearer   = "BearerToken"
	IntegrationBindingStateActive     = "Active"
	IntegrationBindingStateRevoked    = "Revoked"
)

// IntegrationConnectionSpec describes one logical external integration without
// embedding implementation-specific secret storage details or secret values.
type IntegrationConnectionSpec struct {
	// TenantRef identifies the tenant that owns this logical connection.
	// +kubebuilder:validation:XValidation:rule="self.name == oldSelf.name",message="tenantRef.name is immutable"
	TenantRef ObjectReference `json:"tenantRef"`

	// Protocol is the generic transport understood by the v0 runtime adapter.
	// +kubebuilder:validation:Enum=HTTP
	Protocol string `json:"protocol"`

	// Endpoint is non-secret connection metadata.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=2048
	// +kubebuilder:validation:Pattern=`^https?://`
	Endpoint string `json:"endpoint"`

	// Authentication defines how the runtime uses the referenced credential.
	// BearerToken is deliberately the only v0 canary mechanism; future backends
	// may satisfy this contract through a broker without changing the connection.
	// +kubebuilder:validation:Enum=BearerToken
	Authentication string `json:"authentication"`

	// CredentialRef is a logical platform-managed credential reference. It never
	// contains a credential value and does not expose a vault path.
	CredentialRef ObjectReference `json:"credentialRef"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster,shortName=iconn,categories=txo-fabric
// +kubebuilder:printcolumn:name="Tenant",type=string,JSONPath=`.spec.tenantRef.name`
// +kubebuilder:printcolumn:name="Protocol",type=string,JSONPath=`.spec.protocol`
// +kubebuilder:printcolumn:name="Endpoint",type=string,JSONPath=`.spec.endpoint`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
type IntegrationConnection struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec IntegrationConnectionSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true
type IntegrationConnectionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []IntegrationConnection `json:"items"`
}

// IntegrationBindingSpec grants one AgentIdentity a scoped set of operations
// against one IntegrationConnection. Authorization is derived from this Fabric
// object, never from retained agent-local state.
type IntegrationBindingSpec struct {
	// TenantRef must match both the AgentIdentity and IntegrationConnection tenant.
	// +kubebuilder:validation:XValidation:rule="self.name == oldSelf.name",message="tenantRef.name is immutable"
	TenantRef ObjectReference `json:"tenantRef"`

	// AgentRef identifies the AgentIdentity receiving the authorization.
	AgentRef ObjectReference `json:"agentRef"`

	// ConnectionRef identifies the logical IntegrationConnection.
	ConnectionRef ObjectReference `json:"connectionRef"`

	// State allows explicit revocation without deleting retained runtime storage.
	// +kubebuilder:default=Active
	// +kubebuilder:validation:Enum=Active;Revoked
	State string `json:"state,omitempty"`

	// Operations are generic operation names admitted by this binding.
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=32
	// +listType=set
	Operations []string `json:"operations"`

	// Scopes are connection-defined, non-secret authorization scope labels.
	// +optional
	// +kubebuilder:validation:MaxItems=64
	// +listType=set
	Scopes []string `json:"scopes,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster,shortName=ibind,categories=txo-fabric
// +kubebuilder:printcolumn:name="Tenant",type=string,JSONPath=`.spec.tenantRef.name`
// +kubebuilder:printcolumn:name="Agent",type=string,JSONPath=`.spec.agentRef.name`
// +kubebuilder:printcolumn:name="Connection",type=string,JSONPath=`.spec.connectionRef.name`
// +kubebuilder:printcolumn:name="State",type=string,JSONPath=`.spec.state`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
type IntegrationBinding struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec IntegrationBindingSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true
type IntegrationBindingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []IntegrationBinding `json:"items"`
}

func init() {
	SchemeBuilder.Register(
		&IntegrationConnection{}, &IntegrationConnectionList{},
		&IntegrationBinding{}, &IntegrationBindingList{},
	)
}
