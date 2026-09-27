package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

type NamespacedObjectReference struct {
	// Name is the metadata.name of the referenced namespaced object.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=253
	Name string `json:"name"`

	// Namespace is the namespace of the referenced object.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=63
	Namespace string `json:"namespace"`
}

type SecretStoreReference struct {
	// Name is the External Secrets store name.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=253
	Name string `json:"name"`

	// Kind selects a namespaced or cluster-scoped External Secrets store.
	// +kubebuilder:validation:Enum=SecretStore;ClusterSecretStore
	// +kubebuilder:default=ClusterSecretStore
	Kind string `json:"kind,omitempty"`
}

type PostgreSQLCredentialProfileSpec struct {
	// SecretStoreRef identifies the platform secret store used for tenant database credentials.
	SecretStoreRef SecretStoreReference `json:"secretStoreRef"`

	// RemoteKeyPrefix is the platform-owned prefix below which tenant credential records live.
	// The reconciler appends the canonical tenant name when materializing credentials.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=512
	RemoteKeyPrefix string `json:"remoteKeyPrefix"`
}

type PostgreSQLProfileSpec struct {
	// Provider identifies the database lifecycle implementation.
	// +kubebuilder:validation:Enum=CloudNativePG
	// +kubebuilder:default=CloudNativePG
	Provider string `json:"provider,omitempty"`

	// Mode must match the TenantBundle persistence request that selects this profile.
	// +kubebuilder:validation:Enum=Shared;Dedicated
	// +kubebuilder:default=Shared
	Mode string `json:"mode,omitempty"`

	// ClusterRef identifies the existing CloudNativePG Cluster for Shared mode.
	// Dedicated profiles may omit it because the Fabric operator will own the tenant cluster lifecycle.
	// +optional
	ClusterRef *NamespacedObjectReference `json:"clusterRef,omitempty"`

	// DatabaseNamePrefix is prepended to the canonical tenant name when deriving the logical database/user name.
	// +kubebuilder:default=txo_
	// +kubebuilder:validation:MaxLength=32
	// +kubebuilder:validation:Pattern=`^[a-z][a-z0-9_]*$`
	DatabaseNamePrefix string `json:"databaseNamePrefix,omitempty"`

	// RequiredExtensions are PostgreSQL extensions that must exist in every tenant database created by this profile.
	// +optional
	// +listType=set
	RequiredExtensions []string `json:"requiredExtensions,omitempty"`

	// Credentials defines where database credential material is sourced.
	Credentials PostgreSQLCredentialProfileSpec `json:"credentials"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster,shortName=pgprofile,categories=txo-fabric
// +kubebuilder:printcolumn:name="Provider",type=string,JSONPath=`.spec.provider`
// +kubebuilder:printcolumn:name="Mode",type=string,JSONPath=`.spec.mode`
// +kubebuilder:printcolumn:name="Cluster",type=string,JSONPath=`.spec.clusterRef.name`
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
