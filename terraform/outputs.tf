output "kubeconfig_path" {
  value = local_file.kubeconfig.filename
}

output "talosconfig_path" {
  value = local_file.talosconfig_out.filename
}
