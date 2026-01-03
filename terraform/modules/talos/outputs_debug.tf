# Temporary output for debugging nameservers patch
output "debug_node_patch_daphne" {
  value     = try(yamldecode(local.node_patches["daphne"]), "not found")
  sensitive = false
}
