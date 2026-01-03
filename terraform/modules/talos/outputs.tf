output "kubeconfig" {
  description = "Kubeconfig for accessing the Kubernetes cluster"
  value       = talos_cluster_kubeconfig.this.kubeconfig_raw
  sensitive   = true
}

output "talosconfig" {
  description = "Talosconfig for managing Talos nodes"
  value       = data.talos_client_configuration.this.talos_config
  sensitive   = true
}

output "control_plane_configs" {
  description = "Talos machine configurations for control plane nodes"
  value = {
    for k, v in data.talos_machine_configuration.control_plane : k => v.machine_configuration
  }
  sensitive = true
}

output "cluster_endpoint" {
  description = "Kubernetes API endpoint"
  value       = var.cluster_endpoint
}

# Outputs for Helm provider configuration
output "kubernetes_host" {
  description = "Kubernetes API server host for Helm provider"
  value       = talos_cluster_kubeconfig.this.kubernetes_client_configuration.host
  sensitive   = false
}

output "kubernetes_client_certificate" {
  description = "Base64 encoded client certificate for Helm provider"
  value       = talos_cluster_kubeconfig.this.kubernetes_client_configuration.client_certificate
  sensitive   = true
}

output "kubernetes_client_key" {
  description = "Base64 encoded client key for Helm provider"
  value       = talos_cluster_kubeconfig.this.kubernetes_client_configuration.client_key
  sensitive   = true
}

output "kubernetes_ca_certificate" {
  description = "Base64 encoded CA certificate for Helm provider"
  value       = talos_cluster_kubeconfig.this.kubernetes_client_configuration.ca_certificate
  sensitive   = true
}

# Debug: Output node patches to verify nameservers are included
output "debug_node_patches" {
  description = "Generated node patches for debugging"
  value       = { for k, v in var.control_plane_nodes : k => yamldecode(local.node_patches[k]) }
  sensitive   = false
}
