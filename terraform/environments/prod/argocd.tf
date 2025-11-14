# ============================================================================
# ARGOCD GITOPS CONFIGURATION
# ============================================================================

module "argocd" {
  source = "../../modules/argocd"

  chart_version = module.shared.chart_versions.argocd
  environment   = var.environment
  git_branch    = var.git_branch

  argocd_config = var.argocd

  # DRY: Tolerations from shared module
  control_plane_tolerations = module.shared.control_plane_tolerations
  timeout                   = module.shared.timeouts.helm_install

  cilium_module          = module.cilium
  root_app_template_path = "${local.repo_root}/argocd/base/root-app.yaml.tpl"

  depends_on = [
    module.cilium
  ]
}
