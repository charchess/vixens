output "talosconfig" {
  description = "The generated talosconfig for cluster administration."
  value       = talos_client_configuration.vixens.talos_config
  sensitive   = true
}

output "kubeconfig" {
  description = "The generated kubeconfig for Kubernetes administration."
  value       = data.talos_kubeconfig.vixens.kubeconfig_raw
  sensitive   = true
}