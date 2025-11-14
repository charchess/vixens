# ============================================================================
# VIXENS DEV ENVIRONMENT - OUTPUTS
# ============================================================================

output "cluster_endpoint" {
  description = "Kubernetes API endpoint"
  value       = var.cluster.endpoint
}

output "kubeconfig_path" {
  description = "Path to generated kubeconfig"
  value       = var.paths.kubeconfig
}

output "talosconfig_path" {
  description = "Path to generated talosconfig"
  value       = var.paths.talosconfig
}

output "argocd_url" {
  description = "ArgoCD URL"
  value       = "http://${var.argocd.loadbalancer_ip}"
}
