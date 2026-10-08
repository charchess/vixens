package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// AgentFunctionalProfileSpec holds approved, non-secret, reusable instructions.
// This minimal v0.1 slice never grants runtime tools, models, workspace mounts or integrations.
type AgentFunctionalProfileSpec struct {
	// TenantRef pins this profile to one tenant; foreign-tenant use is forbidden.
	// +kubebuilder:validation:XValidation:rule="self.name == oldSelf.name",message="tenantRef.name is immutable"
	TenantRef ObjectReference `json:"tenantRef"`

	// Instructions are a reviewed, non-secret gateway overlay. Not a replacement for
	// each agent's private SOUL.md. Do not put provider credentials in this field.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=8192
	Instructions string `json:"instructions"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster,shortName=afprofile,categories=txo-fabric
// +kubebuilder:printcolumn:name="Tenant",type=string,JSONPath=`.spec.tenantRef.name`
type AgentFunctionalProfile struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              AgentFunctionalProfileSpec `json:"spec"`
}

// +kubebuilder:object:root=true
type AgentFunctionalProfileList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AgentFunctionalProfile `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AgentFunctionalProfile{}, &AgentFunctionalProfileList{})
}
