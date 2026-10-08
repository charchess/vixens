package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// HermesRuntimeRelease is the immutable distribution artifact, not an agent
// security policy. Use two independent digests for upstream Hermes and its
// optional Hindsight provider: neither may be mutable by a tag update.
// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="a Hermes runtime release is immutable; create a new release instead"
type HermesRuntimeReleaseSpec struct {
	// Image is the exact OCI digest for the Hermes engine.
	// +kubebuilder:validation:Pattern=`^[a-zA-Z0-9][a-zA-Z0-9._:/-]*@sha256:[a-f0-9]{64}$`
	// +kubebuilder:validation:MaxLength=512
	Image string `json:"image"`

	// HindsightPluginImage optionally pins the independently packaged plugin.
	// +optional
	// +kubebuilder:validation:Pattern=`^[a-z0-9][a-z0-9._:/-]*@sha256:[a-f0-9]{64}$`
	// +kubebuilder:validation:MaxLength=512
	HindsightPluginImage string `json:"hindsightPluginImage,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster,shortName=hermesrelease,categories=txo-fabric
// +kubebuilder:printcolumn:name="Image",type=string,JSONPath=`.spec.image`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
type HermesRuntimeRelease struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec HermesRuntimeReleaseSpec `json:"spec"`
}

// +kubebuilder:object:root=true
type HermesRuntimeReleaseList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items []HermesRuntimeRelease `json:"items"`
}

func init() {
	SchemeBuilder.Register(&HermesRuntimeRelease{}, &HermesRuntimeReleaseList{})
}
