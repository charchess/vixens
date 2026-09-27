package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type HindsightProfileSpec struct {
	// Image is the immutable Hindsight runtime image used for tenant memory services.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=512
	Image string `json:"image"`

	// Port is the Hindsight API port exposed inside the tenant namespace.
	// +kubebuilder:default=8888
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	Port int32 `json:"port,omitempty"`

	// APIAuthMode selects the inbound authentication contract for the Hindsight API.
	// +kubebuilder:validation:Enum=ApiKey
	// +kubebuilder:default=ApiKey
	APIAuthMode string `json:"apiAuthMode,omitempty"`

	// LLMAuthMode describes how Hindsight obtains model access. Unconfigured is valid while
	// the TXO Fabric LLM broker is not deployed; memory operations requiring an LLM remain blocked.
	// +kubebuilder:validation:Enum=Unconfigured;Gateway
	// +kubebuilder:default=Unconfigured
	LLMAuthMode string `json:"llmAuthMode,omitempty"`

	// FileStorageType selects Hindsight attachment storage. PostgreSQL keeps the initial
	// implementation self-contained; S3 is reserved for the tenant object-storage capability.
	// +kubebuilder:validation:Enum=PostgreSQL;S3
	// +kubebuilder:default=PostgreSQL
	FileStorageType string `json:"fileStorageType,omitempty"`

	// ControlPlaneEnabled exposes the optional Hindsight web control plane for this tenant profile.
	// +kubebuilder:default=false
	ControlPlaneEnabled bool `json:"controlPlaneEnabled,omitempty"`

	// Resources controls the generated Hindsight API container.
	Resources corev1.ResourceRequirements `json:"resources"`

	// PriorityClassName maps the workload to Vixens scheduling policy.
	// +kubebuilder:default=vixens-medium
	// +kubebuilder:validation:MaxLength=253
	PriorityClassName string `json:"priorityClassName,omitempty"`

	// SizingLabel is copied to the Vixens sizing label on generated Hindsight pods.
	// +kubebuilder:default=V-small
	// +kubebuilder:validation:MaxLength=63
	SizingLabel string `json:"sizingLabel,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster,shortName=hprofile,categories=txo-fabric
// +kubebuilder:printcolumn:name="Image",type=string,JSONPath=`.spec.image`
// +kubebuilder:printcolumn:name="LLMAuth",type=string,JSONPath=`.spec.llmAuthMode`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
type HindsightProfile struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              HindsightProfileSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true
type HindsightProfileList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []HindsightProfile `json:"items"`
}

func init() {
	SchemeBuilder.Register(&HindsightProfile{}, &HindsightProfileList{})
}
