# ============================================================================
# ENVIRONMENT OUTPUTS
# ============================================================================

output "cluster_endpoint" {
  description = "Kubernetes API endpoint"
  value       = module.environment.cluster_endpoint
}

output "kubeconfig_path" {
  description = "Path to generated kubeconfig"
  value       = module.environment.kubeconfig_path
}

output "talosconfig_path" {
  description = "Path to generated talosconfig"
  value       = module.environment.talosconfig_path
}

output "argocd_url" {
  description = "ArgoCD URL"
  value       = module.environment.argocd_url
}
