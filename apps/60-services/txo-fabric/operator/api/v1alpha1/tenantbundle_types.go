package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

type HumanAccessOIDCSpec struct {
	// Issuer is the HTTPS OpenID Connect issuer used by the runtime dashboard.
	// Runtime validation also enforces an HTTPS issuer before exposure.
	// +kubebuilder:validation:MinLength=8
	// +kubebuilder:validation:MaxLength=2048
	Issuer string `json:"issuer"`

	// ClientID is a public PKCE client identifier. Client secrets are not part
	// of the Fabric CRD contract.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=253
	ClientID string `json:"clientId"`

	// Scopes defaults to the identity claims required by Hermes.
	// +optional
	// +kubebuilder:default="openid profile email"
	// +kubebuilder:validation:MaxLength=512
	Scopes string `json:"scopes,omitempty"`
}

type HumanWebAccessSpec struct {
	// DomainSuffix is the platform-managed DNS suffix. Runtime hosts are derived
	// as <agentKey>-<tenantName>.<domainSuffix>.
	// +kubebuilder:validation:MinLength=3
	// +kubebuilder:validation:MaxLength=253
	DomainSuffix string `json:"domainSuffix"`

	// IngressClassName selects the platform ingress implementation.
	// +optional
	// +kubebuilder:default=traefik
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=63
	IngressClassName string `json:"ingressClassName,omitempty"`

	// TLSClusterIssuer selects the cert-manager ClusterIssuer for per-agent TLS.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=253
	TLSClusterIssuer string `json:"tlsClusterIssuer"`

	// PublicDNS opts generated agent Ingresses into the platform public-DNS
	// controller. Internal split DNS still follows the Ingress host normally.
	// +optional
	PublicDNS bool `json:"publicDNS,omitempty"`

	// DNSTarget optionally requests a CNAME-style target from external-dns.
	// +optional
	// +kubebuilder:validation:MaxLength=253
	DNSTarget string `json:"dnsTarget,omitempty"`

	// IAMGroups are tenant-local structural IAM group keys authorized to enter
	// Fabric human-facing surfaces. Fabric reconciles the corresponding Authentik
	// groups but never manages human membership in them.
	// +optional
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=32
	// +listType=set
	IAMGroups []string `json:"iamGroups,omitempty"`

	OIDC HumanAccessOIDCSpec `json:"oidc"`
}

type TenantHumanAccessSpec struct {
	// Web configures the first v0 human transport. Additional channels belong
	// beside Web rather than inside AgentIdentity.
	// +optional
	Web *HumanWebAccessSpec `json:"web,omitempty"`
}

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

	// Workspace declares tenant-owned shared business-data scopes. It is independent
	// from every AgentIdentity private /opt/data workspace.
	// +optional
	Workspace *TenantWorkspaceSpec `json:"workspace,omitempty"`

	// HumanAccess declares tenant-owned IAM/transport policy for authenticated
	// human interaction with explicitly opted-in AgentIdentity resources.
	// +optional
	HumanAccess *TenantHumanAccessSpec `json:"humanAccess,omitempty"`

	// Modules contains optional tenant capabilities. Core runtime, persistence and memory are
	// explicit fields because other resources depend on their resolved status.
	// +optional
	// +kubebuilder:validation:MaxItems=64
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
	// +kubebuilder:validation:MaxLength=63
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
// +kubebuilder:resource:scope=Cluster,shortName=tbundle,categories=txo-fabric
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
