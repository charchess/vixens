package controller

// Manager-level permissions live here so controller-gen remains the single source
// of truth for config/rbac/role.yaml. These permissions are not tied to one
// reconciler: controller-runtime uses Leases for leader election and emits Events
// for controller activity.
//
// +kubebuilder:rbac:groups=coordination.k8s.io,resources=leases,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch
