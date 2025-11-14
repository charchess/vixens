# ============================================================================
# VIXENS PROD ENVIRONMENT - 2-LEVEL ARCHITECTURE
# ============================================================================
# Direct module calls (env → modules) without base/ abstraction layer
#
# Structure:
#   - main.tf      : Locals + shared module (DRY configurations)
#   - talos.tf     : Talos cluster + kubeconfig + wait script
#   - cilium.tf    : Cilium CNI
#   - argocd.tf    : ArgoCD GitOps
#   - outputs.tf   : Environment outputs
#   - variables.tf : Input variables
#   - versions.tf  : Provider versions
#   - providers.tf : Provider configurations
#   - backend.tf   : S3 backend configuration

locals {
  # Repository root for relative paths
  repo_root = "${path.module}/../../.."
}

# ----------------------------------------------------------------------------
# SHARED MODULE (DRY Configurations)
# ----------------------------------------------------------------------------
module "shared" {
  source = "../../modules/shared"

  environment     = var.environment
  loadbalancer_ip = var.argocd.loadbalancer_ip
}
