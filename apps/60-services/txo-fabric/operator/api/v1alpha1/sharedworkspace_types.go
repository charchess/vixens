package v1alpha1

import (
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	WorkspaceRetentionRetain = "Retain"
	WorkspaceRetentionDelete = "Delete"
)

// SharedWorkspaceProfile is platform-owned storage policy for tenant shared workspaces.
type SharedWorkspaceProfileSpec struct {
	// StorageClassName must provide ReadWriteMany semantics.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=253
	StorageClassName string `json:"storageClassName"`

	// Size is requested for each independently isolated shared scope PVC.
	// +kubebuilder:default="5Gi"
	Size resource.Quantity `json:"size,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster,shortName=swprofile,categories=txo-fabric
// +kubebuilder:printcolumn:name="StorageClass",type=string,JSONPath=`.spec.storageClassName`
// +kubebuilder:printcolumn:name="Size",type=string,JSONPath=`.spec.size`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
type SharedWorkspaceProfile struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              SharedWorkspaceProfileSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true
type SharedWorkspaceProfileList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SharedWorkspaceProfile `json:"items"`
}

// WorkspaceScopeSpec defines the two business-data trust modes exposed to agents.
// The zero value is valid for the organization scope so tenants can enable only
// group/user workspaces without fabricating an organization share.
type WorkspaceScopeSpec struct {
	// Reference exposes authoritative material read-only to the agent runtime.
	// +optional
	Reference bool `json:"reference,omitempty"`

	// Collaborative exposes a writable exchange/collaboration area.
	// +optional
	Collaborative bool `json:"collaborative,omitempty"`
}

// NamedWorkspaceScopeSpec defines a group or user scope available in one tenant.
// +kubebuilder:validation:XValidation:rule="self.reference || self.collaborative",message="workspace scope must enable reference or collaborative access"
type NamedWorkspaceScopeSpec struct {
	// Name is the stable scope key used in paths and agent bindings.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=32
	// +kubebuilder:validation:Pattern=`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`
	Name string `json:"name"`

	// +optional
	Reference bool `json:"reference,omitempty"`

	// +optional
	Collaborative bool `json:"collaborative,omitempty"`
}

// TenantWorkspaceSpec declares shared business-data scopes for one Fabric tenant.
type TenantWorkspaceSpec struct {
	// ProfileRef selects platform-owned RWX storage implementation policy.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=128
	ProfileRef string `json:"profileRef"`

	// RetentionPolicy controls tenant teardown. Retain is the safe default.
	// Changing destructive intent after storage exists would make teardown behavior
	// ambiguous, so it is immutable just like the private AgentIdentity workspace.
	// +kubebuilder:default=Retain
	// +kubebuilder:validation:Enum=Retain;Delete
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="retentionPolicy is immutable"
	RetentionPolicy string `json:"retentionPolicy,omitempty"`

	// Organization declares tenant-wide scopes.
	// +optional
	Organization WorkspaceScopeSpec `json:"organization,omitempty"`

	// Groups declares team scopes that can be bound to agents.
	// +optional
	// +kubebuilder:validation:MaxItems=64
	// +listType=map
	// +listMapKey=name
	Groups []NamedWorkspaceScopeSpec `json:"groups,omitempty"`

	// Users declares human/user scopes that can be bound to agents.
	// +optional
	// +kubebuilder:validation:MaxItems=256
	// +listType=map
	// +listMapKey=name
	Users []NamedWorkspaceScopeSpec `json:"users,omitempty"`
}

// AgentAccessSpec binds an agent to tenant workspace populations. A future IAM
// integration may manage these bindings; Fabric does not become the employee directory.
type AgentAccessSpec struct {
	// UserRef selects one declared tenant user workspace scope.
	// +optional
	// +kubebuilder:validation:MaxLength=32
	// +kubebuilder:validation:Pattern=`^$|^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`
	UserRef string `json:"userRef,omitempty"`

	// Groups selects declared tenant group workspace scopes.
	// +optional
	// +kubebuilder:validation:MaxItems=32
	// +listType=set
	Groups []string `json:"groups,omitempty"`
}

func init() {
	SchemeBuilder.Register(&SharedWorkspaceProfile{}, &SharedWorkspaceProfileList{})
}
