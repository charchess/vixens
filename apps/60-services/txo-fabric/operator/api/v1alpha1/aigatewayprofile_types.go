package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type AIGatewayProfileSpec struct {
	// Topology is tenant-scoped for v0.1. Cross-tenant inference gateways are deliberately unsupported.
	// +kubebuilder:validation:Enum=TenantScoped
	// +kubebuilder:default=TenantScoped
	Topology string `json:"topology,omitempty"`

	// Implementation selects the platform-owned tenant-facing AI gateway.
	// +kubebuilder:validation:Enum=LiteLLM
	// +kubebuilder:default=LiteLLM
	Implementation string `json:"implementation,omitempty"`

	// Image is an immutable/versioned LiteLLM image reference.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=512
	Image string `json:"image"`

	// APIPort is the tenant-local OpenAI-compatible API port.
	// +kubebuilder:default=4000
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	APIPort int32 `json:"apiPort,omitempty"`

	// PostgreSQLProfileRef selects the CloudNativePG-backed persistence profile
	// used for LiteLLM virtual keys, metering, budgets and proxy state. Fabric
	// creates a dedicated database/role for the gateway; it never reuses the
	// tenant application/Hindsight database.
	// +kubebuilder:default=postgresql-litellm
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=128
	PostgreSQLProfileRef string `json:"postgresqlProfileRef,omitempty"`

	// Resources controls the generated tenant gateway container.
	Resources corev1.ResourceRequirements `json:"resources"`

	// PriorityClassName maps the gateway to Vixens scheduling policy.
	// +kubebuilder:default=vixens-medium
	// +kubebuilder:validation:MaxLength=253
	PriorityClassName string `json:"priorityClassName,omitempty"`

	// SizingLabel is copied to the generated workload for platform resource governance.
	// +kubebuilder:default=V-small
	// +kubebuilder:validation:MaxLength=63
	SizingLabel string `json:"sizingLabel,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster,shortName=aigwprofile,categories=txo-fabric
// +kubebuilder:printcolumn:name="Implementation",type=string,JSONPath=`.spec.implementation`
// +kubebuilder:printcolumn:name="Topology",type=string,JSONPath=`.spec.topology`
// +kubebuilder:printcolumn:name="Image",type=string,JSONPath=`.spec.image`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
type AIGatewayProfile struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              AIGatewayProfileSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true
type AIGatewayProfileList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AIGatewayProfile `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AIGatewayProfile{}, &AIGatewayProfileList{})
}
