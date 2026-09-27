package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

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
	// ClusterRef selects the existing CloudNativePG cluster used by this profile.
	// The Fabric operator does not own or mutate the Cluster resource itself.
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

type PostgreSQLProfileSpec struct {
	// Provider identifies the PostgreSQL lifecycle implementation.
	// +kubebuilder:validation:Enum=CloudNativePG
	// +kubebuilder:default=CloudNativePG
	Provider string `json:"provider,omitempty"`

	// Topology is SharedCluster in the initial Fabric contract. Dedicated tenant
	// clusters are intentionally deferred until the shared path is validated end-to-end.
	// +kubebuilder:validation:Enum=SharedCluster
	Topology string `json:"topology"`

	// RequiredExtensions are PostgreSQL extensions that must be present in every tenant database created through this profile.
	// The controller maps them to CloudNativePG Database.spec.extensions; binaries must already be available in the target cluster image/runtime.
	// +optional
	// +listType=set
	RequiredExtensions []string `json:"requiredExtensions,omitempty"`

	// Shared configures an existing shared CloudNativePG cluster.
	Shared SharedPostgreSQLProfileSpec `json:"shared"`

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
