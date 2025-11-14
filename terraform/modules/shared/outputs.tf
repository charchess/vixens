# ============================================================================
# SHARED MODULE OUTPUTS
# ============================================================================
# Export all locals for use in other modules

output "chart_versions" {
  description = "Chart versions for all Helm deployments"
  value       = local.chart_versions
}

output "control_plane_tolerations" {
  description = "Standard tolerations for control plane workloads"
  value       = local.control_plane_tolerations
}

output "cilium_config" {
  description = "Cilium configuration including capabilities and security contexts"
  value       = local.cilium_config
}

output "env_config" {
  description = "Environment-specific configuration (VLANs, domains)"
  value       = local.env_config[var.environment]
}

output "common_labels" {
  description = "Common labels to apply to all resources"
  value       = local.common_labels
}

output "network" {
  description = "Network configuration defaults"
  value       = local.network
}

output "security" {
  description = "Security defaults for containers"
  value       = local.security
}

output "timeouts" {
  description = "Timeout values for various operations"
  value       = local.timeouts
}

output "environment" {
  description = "Current environment name"
  value       = var.environment
}
