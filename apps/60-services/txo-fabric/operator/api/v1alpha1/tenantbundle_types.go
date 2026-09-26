package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// TenantBundleSpec is the desired state of one Fabric Cell.
type TenantBundleSpec struct {
	// TenantID is the immutable business identifier. Kubernetes metadata.name is the canonical
	// machine slug and is used to derive tenant-<name> namespaces.
	// +kubebuilder:validation:Pattern=`^TEN[0-9]{5,}$`
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="tenantId is immutable"
	TenantID string `json:"tenantId"`

	// DisplayName is the human-readable tenant name.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=253
	DisplayName string `json:"displayName"`

	// +optional
	Isolation TenantIsolationSpec `json:"isolation,omitempty"`

	// +optional
	Persistence TenantPersistenceSpec `json:"persistence,omitempty"`

	// Memory declares the durable memory capability consumed by AgentIdentity resources.
	// +optional
	Memory TenantMemorySpec `json:"memory,omitempty"`

	// Modules contains optional tenant capabilities. Core runtime, persistence and memory are
	// explicit fields because other resources depend on their resolved status.
	// +optional
	// +listType=map
	// +listMapKey=name
	Modules []TenantModuleSpec `json:"modules,omitempty"`
}

// TenantBundleStatus is the observed state of a Fabric Cell.
type TenantBundleStatus struct {
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// +kubebuilder:validation:Enum=Pending;Provisioning;Ready;Degraded;Failed;Deleting
	// +optional
	Phase string `json:"phase,omitempty"`

	// Namespace is the resolved tenant namespace.
	// +optional
	Namespace string `json:"namespace,omitempty"`

	// +optional
	Persistence TenantPersistenceStatus `json:"persistence,omitempty"`

	// +optional
	Memory TenantMemoryStatus `json:"memory,omitempty"`

	// +optional
	// +listType=map
	// +listMapKey=name
	Modules []TenantModuleStatus `json:"modules,omitempty"`

	// Conditions use the standard Kubernetes condition shape.
	// +optional
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,shortName=tbundle
// +kubebuilder:printcolumn:name="Tenant",type=string,JSONPath=`.spec.tenantId`
// +kubebuilder:printcolumn:name="Namespace",type=string,JSONPath=`.status.namespace`
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
type TenantBundle struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TenantBundleSpec   `json:"spec,omitempty"`
	Status TenantBundleStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type TenantBundleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TenantBundle `json:"items"`
}

func init() {
	SchemeBuilder.Register(&TenantBundle{}, &TenantBundleList{})
}
