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

// WorkspaceScopeSpec defines the business-file and shared-skill modes exposed for
// one scope. Files and skills use separate PVCs/mount roots even when they share
// the same platform storage profile and access binding.
type WorkspaceScopeSpec struct {
	// Reference exposes authoritative business material read-only to the agent runtime.
	// +optional
	Reference bool `json:"reference,omitempty"`

	// Collaborative exposes a writable business exchange/collaboration area.
	// +optional
	Collaborative bool `json:"collaborative,omitempty"`

	// SkillsReference exposes a shared skill library read-only to Hermes. Agent-local
	// skills under /opt/data remain private and writable.
	// +optional
	SkillsReference bool `json:"skillsReference,omitempty"`

	// SkillsCollaborative exposes a shared skill library writable by authorized agents,
	// allowing deliberate cross-agent skill sharing and improvement.
	// +optional
	SkillsCollaborative bool `json:"skillsCollaborative,omitempty"`
}

// NamedWorkspaceScopeSpec defines a group or user scope available in one tenant.
// +kubebuilder:validation:XValidation:rule="self.reference || self.collaborative || self.skillsReference || self.skillsCollaborative",message="workspace scope must enable file or skill access"
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

	// +optional
	SkillsReference bool `json:"skillsReference,omitempty"`

	// +optional
	SkillsCollaborative bool `json:"skillsCollaborative,omitempty"`
}

// TenantWorkspaceSpec declares tenant-owned shared storage. Business files and
// shared skills remain separate trust domains and therefore receive different PVCs
// and mount roots even when backed by the same platform storage profile.
// TenantWorkspaceMigrationBinding stages replacement RWX PVCs without mutating
// an existing shared workspace claim's immutable StorageClassName. Cutover stays
// false while data is copied and independently verified.
type TenantWorkspaceMigrationBinding struct {
	// TargetStorageClassName is the explicit replacement RWX storage class.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=253
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="targetStorageClassName is immutable"
	TargetStorageClassName string `json:"targetStorageClassName"`

	// TargetClaimSuffix makes every replacement scope PVC deterministic and distinct.
	// +kubebuilder:validation:Pattern=`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`
	// +kubebuilder:validation:MaxLength=32
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="targetClaimSuffix is immutable"
	TargetClaimSuffix string `json:"targetClaimSuffix"`

	// CopyJobPrefix derives one immutable GitOps-owned evidence Job per source claim:
	// <copyJobPrefix>-<source-claim-name>. Every Job must complete and bind exact
	// source/target PVC UIDs before an AgentIdentity may mount migrated storage.
	// +optional
	// +kubebuilder:validation:Pattern=`^$|^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`
	CopyJobPrefix string `json:"copyJobPrefix,omitempty"`

	// Cutover selects prepared replacement claims for all tenant workspace mounts.
	// +optional
	Cutover bool `json:"cutover,omitempty"`
}

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

	// ClaimSuffix selects existing physical shared-workspace PVCs by name.
	// Nonempty means read-only adoption: never provision a missing claim.
	// +optional
	// +kubebuilder:validation:MaxLength=32
	// +kubebuilder:validation:Pattern=`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`
	ClaimSuffix string `json:"claimSuffix,omitempty"`

	// Migration is deprecated stored-CR compatibility, not active reconciliation.
	// +optional
	Migration *TenantWorkspaceMigrationBinding `json:"migration,omitempty"`

	// Organization declares tenant-wide file and skill-library scopes.
	// +optional
	Organization WorkspaceScopeSpec `json:"organization,omitempty"`

	// Groups declares team file and skill-library scopes that can be bound to agents.
	// +optional
	// +kubebuilder:validation:MaxItems=64
	// +listType=map
	// +listMapKey=name
	Groups []NamedWorkspaceScopeSpec `json:"groups,omitempty"`

	// Users declares human/user file and skill-library scopes that can be bound to agents.
	// +optional
	// +kubebuilder:validation:MaxItems=256
	// +listType=map
	// +listMapKey=name
	Users []NamedWorkspaceScopeSpec `json:"users,omitempty"`
}

// AgentAccessSpec binds an agent to tenant workspace populations. A future IAM
// integration may manage these bindings; Fabric does not become the employee directory.
type AgentAccessSpec struct {
	// ChatIAMGroups is an explicit agent.chat access intent referencing only
	// TenantBundle.humanAccess.web.iamGroups keys, NOT workspace group keys.
	// Authentik live membership, immutable UUID→Fabric ledger, and tenant
	// group-UUID pins are separately verified before any tuple is written.
	// Omitted/empty never implies an agent.chat permission.
	// +optional
	// +kubebuilder:validation:MaxItems=32
	// +listType=set
	ChatIAMGroups []string `json:"chatIAMGroups,omitempty"`

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
