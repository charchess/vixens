# Call the reusable talos-cluster module
module "vixens_dev_cluster" {
  source = "../../modules/talos-cluster"

  cluster_name          = var.cluster_name
  cluster_endpoint      = var.cluster_endpoint
  control_plane_nodes = var.control_plane_nodes
}