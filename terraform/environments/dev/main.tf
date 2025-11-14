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
      set -e
      echo "⏳ Waiting for Kubernetes API to be ready..."
      echo "⏸️  Initial delay: waiting 60 seconds for Talos bootstrap..."
      sleep 60

      # Phase 1: Wait for API server to respond (60 attempts x 10s = 10min)
      echo "📡 Phase 1: Waiting for API server to respond..."
      i=1
      while [ $i -le 60 ]; do
        if kubectl --kubeconfig=${var.paths.kubeconfig} get --raw /healthz &>/dev/null; then
          echo "✅ API server responded on attempt $i"
          break
        fi
        echo "⏳ Attempt $i/60 - API not ready yet (waiting 10s)..."
        sleep 10
        i=$((i + 1))
      done

      if [ $i -gt 60 ]; then
        echo "❌ Timeout: API never responded after 10 minutes"
        exit 1
      fi

      # Phase 2: Wait for control plane pods to be ready (checks kube-apiserver, kube-controller, kube-scheduler, etcd)
      echo "🔍 Phase 2: Waiting for control plane components to be ready..."
      READY=0
      ATTEMPT=1
      MAX_ATTEMPTS=60

      while [ $ATTEMPT -le $MAX_ATTEMPTS ]; do
        # Check if we can list nodes (API operational)
        if ! kubectl --kubeconfig=${var.paths.kubeconfig} get nodes &>/dev/null; then
          echo "⏳ Attempt $ATTEMPT/$MAX_ATTEMPTS - API not fully operational yet (10s)..."
          sleep 10
          ATTEMPT=$((ATTEMPT + 1))
          continue
        fi

        # Check control plane pods in kube-system
        APISERVER_COUNT=$(kubectl --kubeconfig=${var.paths.kubeconfig} get pods -n kube-system -l component=kube-apiserver --no-headers 2>/dev/null | grep -c Running || echo "0")
        CONTROLLER_COUNT=$(kubectl --kubeconfig=${var.paths.kubeconfig} get pods -n kube-system -l component=kube-controller-manager --no-headers 2>/dev/null | grep -c Running || echo "0")
        SCHEDULER_COUNT=$(kubectl --kubeconfig=${var.paths.kubeconfig} get pods -n kube-system -l component=kube-scheduler --no-headers 2>/dev/null | grep -c Running || echo "0")
        ETCD_COUNT=$(kubectl --kubeconfig=${var.paths.kubeconfig} get pods -n kube-system -l component=etcd --no-headers 2>/dev/null | grep -c Running || echo "0")

        echo "📊 Control plane status: kube-apiserver=$APISERVER_COUNT kube-controller=$CONTROLLER_COUNT kube-scheduler=$SCHEDULER_COUNT etcd=$ETCD_COUNT"

        # We need at least 1 of each (for single control plane) or 2+ for HA
        if [ "$APISERVER_COUNT" -ge 1 ] && [ "$CONTROLLER_COUNT" -ge 1 ] && [ "$SCHEDULER_COUNT" -ge 1 ] && [ "$ETCD_COUNT" -ge 1 ]; then
          READY=$((READY + 1))
          echo "✅ Control plane ready ($READY/3 consecutive checks)"

          if [ $READY -ge 3 ]; then
            echo "🎉 Kubernetes control plane is STABLE and ready!"
            echo "📋 Final status:"
            kubectl --kubeconfig=${var.paths.kubeconfig} get nodes
            kubectl --kubeconfig=${var.paths.kubeconfig} get pods -n kube-system | grep -E "(kube-|etcd)"
            exit 0
          fi
          sleep 5
        else
          if [ $READY -gt 0 ]; then
            echo "⚠️  Control plane became unstable (was at $READY/3)"
          fi
          READY=0
          echo "⏳ Waiting for control plane... (10s)"
          sleep 10
        fi

        ATTEMPT=$((ATTEMPT + 1))
      done

      echo "❌ Control plane not ready after $MAX_ATTEMPTS attempts (10 minutes)"
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
