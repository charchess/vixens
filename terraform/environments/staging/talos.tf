# ============================================================================
# TALOS CLUSTER CONFIGURATION
# ============================================================================

# ----------------------------------------------------------------------------
# TALOS CLUSTER
# ----------------------------------------------------------------------------
module "talos_cluster" {
  source = "../../modules/talos"

  cluster_name        = var.cluster.name
  talos_version       = var.cluster.talos_version
  cluster_endpoint    = var.cluster.endpoint
  talos_image         = var.cluster.talos_image
  kubernetes_version  = var.cluster.kubernetes_version
  control_plane_nodes = var.control_plane_nodes
  worker_nodes        = var.worker_nodes
}

# ----------------------------------------------------------------------------
# KUBECONFIG & TALOSCONFIG FILES
# ----------------------------------------------------------------------------
resource "local_file" "kubeconfig" {
  content         = module.talos_cluster.kubeconfig
  filename        = var.paths.kubeconfig
  file_permission = "0600"
}

resource "local_file" "talosconfig" {
  content         = module.talos_cluster.talosconfig
  filename        = var.paths.talosconfig
  file_permission = "0600"
}

# ----------------------------------------------------------------------------
# WAIT FOR KUBERNETES API
# ----------------------------------------------------------------------------
resource "null_resource" "wait_for_k8s_api" {
  depends_on = [
    module.talos_cluster,
    local_file.kubeconfig
  ]

  provisioner "local-exec" {
    command = "${local.repo_root}/scripts/wait-for-k8s-api.sh ${var.paths.kubeconfig}"
  }
}
