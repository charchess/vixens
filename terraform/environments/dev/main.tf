# ============================================================================
# VIXENS DEV ENVIRONMENT - 2-LEVEL ARCHITECTURE
# ============================================================================
# Direct module calls (env → modules) without base/ abstraction layer

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
    command = <<-EOT
      echo "⏳ Waiting for Kubernetes API to be ready..."

      # Phase 1: Wait for initial connectivity (40 attempts x 5s = 3min 20s)
      for i in {1..40}; do
        if kubectl --kubeconfig=${var.paths.kubeconfig} get nodes &>/dev/null; then
          echo "✅ Kubernetes API responded (attempt $i)"
          break
        fi
        echo "⏳ Attempt $i/40... (waiting 5s)"
        sleep 5

        if [ $i -eq 40 ]; then
          echo "❌ Timeout: API never responded"
          exit 1
        fi
      done

      # Phase 2: Wait for API to be stable (5 consecutive successful checks)
      echo "🔍 Verifying API stability (need 5 consecutive successful checks)..."
      CONSECUTIVE_SUCCESS=0
      for i in {1..20}; do
        if kubectl --kubeconfig=${var.paths.kubeconfig} get nodes &>/dev/null && \
           kubectl --kubeconfig=${var.paths.kubeconfig} get namespaces &>/dev/null && \
           kubectl --kubeconfig=${var.paths.kubeconfig} get --raw /healthz &>/dev/null; then
          CONSECUTIVE_SUCCESS=$((CONSECUTIVE_SUCCESS + 1))
          echo "✅ Stability check $CONSECUTIVE_SUCCESS/5 passed"

          if [ $CONSECUTIVE_SUCCESS -eq 5 ]; then
            echo "🎉 Kubernetes API is STABLE and ready!"
            exit 0
          fi
        else
          if [ $CONSECUTIVE_SUCCESS -gt 0 ]; then
            echo "⚠️  API became unstable, resetting counter (was at $CONSECUTIVE_SUCCESS/5)"
          fi
          CONSECUTIVE_SUCCESS=0
        fi
        sleep 3
      done

      echo "❌ API is responding but not stable enough"
      exit 1
    EOT
  }
}

# ----------------------------------------------------------------------------
# CILIUM CNI
# ----------------------------------------------------------------------------
module "cilium" {
  source = "../../modules/cilium"

  chart_version = module.shared.chart_versions.cilium

  # DRY: Capabilities from shared module
  cilium_agent_capabilities = module.shared.cilium_config.agent_capabilities
  cilium_clean_capabilities = module.shared.cilium_config.clean_capabilities
  control_plane_tolerations = module.shared.control_plane_tolerations

  timeout = module.shared.timeouts.helm_install

  talos_cluster_module = module.talos_cluster
  wait_for_k8s_api     = null_resource.wait_for_k8s_api

  kubeconfig_path     = var.paths.kubeconfig
  ip_pool_yaml_path   = var.paths.cilium_ip_pool_yaml
  l2_policy_yaml_path = var.paths.cilium_l2_policy_yaml
}

# ----------------------------------------------------------------------------
# ARGOCD
# ----------------------------------------------------------------------------
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

# ----------------------------------------------------------------------------
# OUTPUTS
# ----------------------------------------------------------------------------
output "cluster_endpoint" {
  description = "Kubernetes API endpoint"
  value       = var.cluster.endpoint
}

output "kubeconfig_path" {
  description = "Path to generated kubeconfig"
  value       = var.paths.kubeconfig
}

output "talosconfig_path" {
  description = "Path to generated talosconfig"
  value       = var.paths.talosconfig
}

output "argocd_url" {
  description = "ArgoCD URL"
  value       = "http://${var.argocd.loadbalancer_ip}"
}
