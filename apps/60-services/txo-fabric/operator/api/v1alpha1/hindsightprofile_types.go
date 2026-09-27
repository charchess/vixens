package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type HindsightModelCacheSpec struct {
	// Enabled provisions a persistent cache for local models/rerankers.
	// The standard upstream full API image already contains its default models;
	// enable this only for an explicitly designed runtime-download/cache path.
	// +kubebuilder:default=false
	Enabled bool `json:"enabled,omitempty"`

	// Size is the cache PVC request when enabled.
	// +kubebuilder:default="5Gi"
	Size resource.Quantity `json:"size,omitempty"`

	// StorageClassName is explicit because Vixens intentionally has no default StorageClass.
	// +optional
	// +kubebuilder:validation:MaxLength=253
	StorageClassName string `json:"storageClassName,omitempty"`
}

type HindsightProfileSpec struct {
	// Topology is tenant-scoped in the initial Fabric contract. A shared service
	// requires a separately designed multi-tenant authentication/schema boundary.
	// +kubebuilder:validation:Enum=TenantScoped
	// +kubebuilder:default=TenantScoped
	Topology string `json:"topology,omitempty"`

	// Image is an immutable, versioned Hindsight API image reference.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=512
	Image string `json:"image"`

	// APIPort is the Hindsight memory API port.
	// +kubebuilder:default=8888
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	APIPort int32 `json:"apiPort,omitempty"`

	// APIAuthMode selects the inbound authentication contract for the tenant-scoped Hindsight API.
	// ApiKey maps to the upstream ApiKeyTenantExtension; credential material is generated/injected via Secret and never stored in this CR.
	// +kubebuilder:validation:Enum=ApiKey
	// +kubebuilder:default=ApiKey
	APIAuthMode string `json:"apiAuthMode,omitempty"`

	// Resources controls the generated Hindsight API container.
	Resources corev1.ResourceRequirements `json:"resources"`

	// PriorityClassName maps Hindsight to Vixens scheduling policy.
	// +kubebuilder:default=vixens-medium
	// +kubebuilder:validation:MaxLength=253
	PriorityClassName string `json:"priorityClassName,omitempty"`

	// SizingLabel is copied to the generated workload for platform resource governance.
	// +kubebuilder:default=V-small
	// +kubebuilder:validation:MaxLength=63
	SizingLabel string `json:"sizingLabel,omitempty"`

	// ModelCache configures optional persistent cache storage for local models/rerankers.
	// It is disabled by default; production profiles should prefer models baked into the image.
	// +optional
	ModelCache HindsightModelCacheSpec `json:"modelCache,omitempty"`

	// LLMAuthMode declares how the platform will eventually supply model credentials.
	// It never contains provider credentials itself.
	// +kubebuilder:validation:Enum=Unconfigured;PlatformGateway
	// +kubebuilder:default=Unconfigured
	LLMAuthMode string `json:"llmAuthMode,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster,shortName=hsprofile,categories=txo-fabric
// +kubebuilder:printcolumn:name="Topology",type=string,JSONPath=`.spec.topology`
// +kubebuilder:printcolumn:name="Image",type=string,JSONPath=`.spec.image`
// +kubebuilder:printcolumn:name="API Auth",type=string,JSONPath=`.spec.apiAuthMode`
// +kubebuilder:printcolumn:name="LLM Auth",type=string,JSONPath=`.spec.llmAuthMode`
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
