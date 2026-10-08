package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ToolsetPolicyOn         = "On"
	ToolsetPolicyOff        = "Off"
	ToolsetPolicyAllowedOff = "AllowedOff"
)

type RuntimeStorageSpec struct {
	// Size is the runtime workspace PVC request.
	// +kubebuilder:default="2Gi"
	Size resource.Quantity `json:"size,omitempty"`

	// StorageClassName is explicit because Vixens intentionally has no default StorageClass.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=253
	StorageClassName string `json:"storageClassName"`
}

type RuntimeCompatibilitySpec struct {
	// S6Overlay marks images that require the documented Vixens root/capability bypass for s6-overlay.
	// +kubebuilder:default=false
	S6Overlay bool `json:"s6Overlay,omitempty"`
}

type RuntimeToolsetPolicy struct {
	// Name is the Hermes toolset name governed by this immutable runtime profile.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=128
	// +kubebuilder:validation:Pattern=`^[a-z0-9]([a-z0-9._/-]*[a-z0-9])?$`
	Name string `json:"name"`

	// State defines platform ownership for this toolset.
	// On is always enabled, Off is forbidden, AllowedOff requires an AgentIdentity opt-in.
	// +kubebuilder:validation:Enum=On;Off;AllowedOff
	State string `json:"state"`
}

type RuntimeCapabilityPolicySpec struct {
	// Toolsets freezes the Hermes executable-capability policy for this runtime profile.
	// A Hermes profile without an explicit toolset policy is rejected fail-closed.
	// +optional
	// +listType=map
	// +listMapKey=name
	Toolsets []RuntimeToolsetPolicy `json:"toolsets,omitempty"`
}

type AgentRuntimeProfileSpec struct {
	// Engine identifies the runtime implementation.
	// +kubebuilder:validation:Enum=Hermes
	// +kubebuilder:default=Hermes
	Engine string `json:"engine,omitempty"`

	// Image is an immutable, versioned runtime image reference.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=512
	Image string `json:"image"`

	Storage RuntimeStorageSpec `json:"storage"`

	// Resources controls the generated main runtime container.
	Resources corev1.ResourceRequirements `json:"resources"`

	// PriorityClassName maps the runtime to Vixens scheduling policy.
	// +kubebuilder:default=vixens-medium
	// +kubebuilder:validation:MaxLength=253
	PriorityClassName string `json:"priorityClassName,omitempty"`

	// SizingLabel is copied to vixens.io/sizing.hermes for platform resource governance.
	// +kubebuilder:default=V-small
	// +kubebuilder:validation:MaxLength=63
	SizingLabel string `json:"sizingLabel,omitempty"`

	// +optional
	Compatibility RuntimeCompatibilitySpec `json:"compatibility,omitempty"`

	// Capabilities declares platform-owned executable capability policy for this runtime profile.
	Capabilities RuntimeCapabilityPolicySpec `json:"capabilities,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster,shortName=arprofile,categories=txo-fabric
// +kubebuilder:printcolumn:name="Engine",type=string,JSONPath=`.spec.engine`
// +kubebuilder:printcolumn:name="Image",type=string,JSONPath=`.spec.image`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
type AgentRuntimeProfile struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              AgentRuntimeProfileSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true
type AgentRuntimeProfileList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AgentRuntimeProfile `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AgentRuntimeProfile{}, &AgentRuntimeProfileList{})
}
