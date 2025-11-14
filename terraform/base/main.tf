module "talos_cluster" {
  source = "../modules/talos"

  cluster_name        = var.cluster_name
  talos_version       = var.talos_version
  cluster_endpoint    = var.cluster_endpoint
  talos_image         = var.talos_image
  control_plane_nodes = var.control_plane_nodes
  worker_nodes        = var.worker_nodes
}

resource "local_file" "kubeconfig" {
  content         = module.talos_cluster.kubeconfig
  filename        = var.kubeconfig_path
  file_permission = "0600"
}

resource "local_file" "talosconfig" {
  content         = module.talos_cluster.talosconfig
  filename        = var.talosconfig_path
  file_permission = "0600"
}

module "cilium" {
  source = "../modules/cilium"

  release_name  = "cilium"
  chart_version = "1.18.3" # This could be a variable too
  namespace     = "kube-system"

  talos_cluster_module = module.talos_cluster
  wait_for_k8s_api     = null_resource.wait_for_k8s_api

  kubeconfig_path     = var.kubeconfig_path
  ip_pool_yaml_path   = var.cilium_ip_pool_yaml_path
  l2_policy_yaml_path = var.cilium_l2_policy_yaml_path
}