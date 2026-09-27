package v1alpha1

import (
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type PostgreSQLClusterReference struct {
	// Name is the CloudNativePG Cluster resource name.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=253
	Name string `json:"name"`

	// Namespace is the namespace containing the CloudNativePG Cluster.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=63
	Namespace string `json:"namespace"`
}

type SharedPostgreSQLProfileSpec struct {
	// ClusterRef selects the existing platform-owned CloudNativePG cluster.
	ClusterRef PostgreSQLClusterReference `json:"clusterRef"`

	// DatabaseNamePrefix is prepended to operator-generated tenant database names.
	// +kubebuilder:default=txo_
	// +kubebuilder:validation:MaxLength=32
	// +kubebuilder:validation:Pattern=`^[a-z][a-z0-9_]*$`
	DatabaseNamePrefix string `json:"databaseNamePrefix,omitempty"`

	// RoleNamePrefix is prepended to operator-generated tenant login roles.
	// +kubebuilder:default=txo_
	// +kubebuilder:validation:MaxLength=32
	// +kubebuilder:validation:Pattern=`^[a-z][a-z0-9_]*$`
	RoleNamePrefix string `json:"roleNamePrefix,omitempty"`
}

type DedicatedPostgreSQLStorageSpec struct {
	// Size is the PVC request for each dedicated CloudNativePG instance.
	// +kubebuilder:default="8Gi"
	Size resource.Quantity `json:"size,omitempty"`

	// StorageClassName is explicit because Vixens intentionally has no default StorageClass.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=253
	StorageClassName string `json:"storageClassName"`
}

type DedicatedPostgreSQLProfileSpec struct {
	// Instances is the number of CloudNativePG instances for a dedicated tenant cluster.
	// +kubebuilder:default=2
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=5
	Instances int32 `json:"instances,omitempty"`

	Storage DedicatedPostgreSQLStorageSpec `json:"storage"`
}

type PostgreSQLProfileSpec struct {
	// Provider identifies the PostgreSQL lifecycle implementation.
	// +kubebuilder:validation:Enum=CloudNativePG
	// +kubebuilder:default=CloudNativePG
	Provider string `json:"provider,omitempty"`

	// Topology defines whether tenants consume an existing shared cluster or receive a dedicated cluster.
	// +kubebuilder:validation:Enum=SharedCluster;DedicatedCluster
	Topology string `json:"topology"`

	// RequiredExtensions are PostgreSQL extensions that must be present in every tenant database created through this profile.
	// The controller maps them to CloudNativePG Database.spec.extensions; binaries must already be available in the target cluster image/runtime.
	// +optional
	// +listType=set
	RequiredExtensions []string `json:"requiredExtensions,omitempty"`

	// Shared configures an existing shared CloudNativePG cluster.
	// +optional
	Shared *SharedPostgreSQLProfileSpec `json:"shared,omitempty"`

	// Dedicated configures operator-created per-tenant CloudNativePG clusters.
	// +optional
	Dedicated *DedicatedPostgreSQLProfileSpec `json:"dedicated,omitempty"`

	// DatabaseReclaimPolicy controls whether the tenant database is retained when the TenantBundle stops requesting PostgreSQL.
	// +kubebuilder:validation:Enum=Retain;Delete
	// +kubebuilder:default=Retain
	DatabaseReclaimPolicy string `json:"databaseReclaimPolicy,omitempty"`

	// RoleReclaimPolicy controls whether the tenant login role is retained with its database.
	// +kubebuilder:validation:Enum=Retain;Delete
	// +kubebuilder:default=Retain
	RoleReclaimPolicy string `json:"roleReclaimPolicy,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster,shortName=pgprofile,categories=txo-fabric
// +kubebuilder:printcolumn:name="Provider",type=string,JSONPath=`.spec.provider`
// +kubebuilder:printcolumn:name="Topology",type=string,JSONPath=`.spec.topology`
// +kubebuilder:printcolumn:name="DB Reclaim",type=string,JSONPath=`.spec.databaseReclaimPolicy`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
type PostgreSQLProfile struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              PostgreSQLProfileSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true
type PostgreSQLProfileList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PostgreSQLProfile `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PostgreSQLProfile{}, &PostgreSQLProfileList{})
}
