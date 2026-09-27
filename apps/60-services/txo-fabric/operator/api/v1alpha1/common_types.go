package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

type ObjectReference struct {
	// Name is the metadata.name of the referenced cluster-scoped Fabric object.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=253
	Name string `json:"name"`
}

type TenantIsolationSpec struct {
	// Profile selects the isolation policy for the tenant cell.
	// +kubebuilder:validation:Enum=Shared;Dedicated
	// +kubebuilder:default=Shared
	Profile string `json:"profile,omitempty"`
}

type PostgreSQLSpec struct {
	// Mode selects logical resources on the platform shared cluster or a tenant-dedicated cluster.
	// +kubebuilder:validation:Enum=Shared;Dedicated
	// +kubebuilder:default=Shared
	Mode string `json:"mode,omitempty"`

	// ProfileRef selects the platform-owned PostgreSQLProfile used to satisfy this capability.
	ProfileRef ObjectReference `json:"profileRef"`
}

type TenantPersistenceSpec struct {
	// PostgreSQL requests PostgreSQL capability for this tenant. Nil means no PostgreSQL dependency.
	// +optional
	PostgreSQL *PostgreSQLSpec `json:"postgresql,omitempty"`
}

type HindsightMemorySpec struct {
	// ProfileRef selects the platform-owned HindsightProfile used to satisfy this capability.
	ProfileRef ObjectReference `json:"profileRef"`
}

type TenantMemorySpec struct {
	// Hindsight enables tenant-scoped durable agent memory through Hindsight.
	// +optional
	Hindsight *HindsightMemorySpec `json:"hindsight,omitempty"`
}

type TenantModuleSpec struct {
	// Name is the stable Fabric module key (for example paperclip or valkey).
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=63
	// +kubebuilder:validation:Pattern=`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`
	Name string `json:"name"`

	// Enabled defaults to true when the module entry is present.
	// +kubebuilder:default=true
	Enabled bool `json:"enabled,omitempty"`

	// ProfileRef selects a platform-owned implementation profile for the module.
	// +optional
	// +kubebuilder:validation:MaxLength=128
	ProfileRef string `json:"profileRef,omitempty"`
}

type ComponentStatus struct {
	// Phase is a concise machine-readable lifecycle state.
	// +optional
	Phase string `json:"phase,omitempty"`

	// Endpoint is the resolved in-cluster endpoint, when the component exposes one.
	// +optional
	Endpoint string `json:"endpoint,omitempty"`

	// Message contains a human-readable explanation for non-ready states.
	// +optional
	Message string `json:"message,omitempty"`
}

type TenantPersistenceStatus struct {
	// +optional
	PostgreSQL *ComponentStatus `json:"postgresql,omitempty"`
}

type TenantMemoryStatus struct {
	// +optional
	Hindsight *ComponentStatus `json:"hindsight,omitempty"`
}

type TenantModuleStatus struct {
	Name            string `json:"name"`
	ComponentStatus `json:",inline"`
}

type RuntimeStatus struct {
	// DeploymentName is the generated Hermes Deployment.
	// +optional
	DeploymentName string `json:"deploymentName,omitempty"`

	// PVCName is the generated runtime workspace PVC.
	// +optional
	PVCName string `json:"pvcName,omitempty"`
}

type MemoryBindingStatus struct {
	// BankID is the resolved logical memory-bank identity.
	// +optional
	BankID string `json:"bankId,omitempty"`

	// Phase describes memory provisioning for this identity.
	// +optional
	Phase string `json:"phase,omitempty"`
}

func conditionSliceCopy(in []metav1.Condition) []metav1.Condition {
	if in == nil {
		return nil
	}
	out := make([]metav1.Condition, len(in))
	copy(out, in)
	return out
}
