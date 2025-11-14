# ============================================================================
# VIXENS DEV ENVIRONMENT
# ============================================================================
# Uses the shared environment module for DRY infrastructure deployment

module "environment" {
  source = "../../modules/environment"

  environment         = var.environment
  git_branch          = var.git_branch
  cluster             = var.cluster
  control_plane_nodes = var.control_plane_nodes
  worker_nodes        = var.worker_nodes
  paths               = var.paths
  argocd              = var.argocd
}
