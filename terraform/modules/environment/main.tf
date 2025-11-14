# ============================================================================
# ENVIRONMENT MODULE - MAIN CONFIGURATION
# ============================================================================
# This module encapsulates the common infrastructure logic for all environments.
# It orchestrates the deployment of Talos cluster, Cilium CNI, and ArgoCD GitOps.

locals {
  # Repository root for relative paths (assuming module is at modules/environment/)
  repo_root = "${path.module}/../../.."
}

# ----------------------------------------------------------------------------
# SHARED MODULE (DRY Configurations)
# ----------------------------------------------------------------------------
module "shared" {
  source = "../shared"

  environment     = var.environment
  loadbalancer_ip = var.argocd.loadbalancer_ip
}
