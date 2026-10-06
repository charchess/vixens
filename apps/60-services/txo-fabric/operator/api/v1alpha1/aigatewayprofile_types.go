package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type AIGatewayAuthStorageSpec struct {
	// Size is the persistent request for mutable OAuth credential state.
	// +kubebuilder:default="1Gi"
	Size resource.Quantity `json:"size,omitempty"`

	// StorageClassName is explicit because Vixens intentionally has no default StorageClass.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=253
	StorageClassName string `json:"storageClassName"`
}

type AIGatewayProfileSpec struct {
	// Topology is tenant-scoped for v0.1. Cross-tenant credential pools are deliberately unsupported.
	// +kubebuilder:validation:Enum=TenantScoped
	// +kubebuilder:default=TenantScoped
	Topology string `json:"topology,omitempty"`

	// Implementation selects the platform-owned gateway implementation.
	// +kubebuilder:validation:Enum=CLIProxyAPI
	// +kubebuilder:default=CLIProxyAPI
	Implementation string `json:"implementation,omitempty"`

	// Image is an immutable/versioned CLIProxyAPI image reference.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=512
	Image string `json:"image"`

	// APIPort is the tenant-local OpenAI-compatible API and authenticated management port.
	// +kubebuilder:default=8317
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	APIPort int32 `json:"apiPort,omitempty"`

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

	// AuthStorage stores mutable OAuth access/refresh state outside every AgentIdentity PVC.
	AuthStorage AIGatewayAuthStorageSpec `json:"authStorage"`

	// UsageQueueRetentionSeconds controls CPA's in-memory management usage queue.
	// TXO collectors must poll within this bounded window; durable accounting belongs outside CPA.
	// +kubebuilder:default=900
	// +kubebuilder:validation:Minimum=60
	// +kubebuilder:validation:Maximum=3600
	UsageQueueRetentionSeconds int32 `json:"usageQueueRetentionSeconds,omitempty"`
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
