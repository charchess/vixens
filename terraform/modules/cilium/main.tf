# ============================================================================
# CILIUM MODULE - OPTIMIZED WITH DRY PRINCIPLE
# ============================================================================
# Eliminates hardcoding by using shared configurations

resource "helm_release" "cilium" {
  name       = var.release_name
  repository = "https://helm.cilium.io/"
  chart      = "cilium"
  version    = var.chart_version
  namespace  = var.namespace

  wait          = true
  wait_for_jobs = true
  timeout       = var.timeout

  values = [yamlencode({
    kubeProxyReplacement = true
    k8sServiceHost       = "localhost"
    k8sServicePort       = 7445

    l2announcements = {
      enabled = true
    }

    k8sClientRateLimit = {
      qps   = 10
      burst = 20
    }

    externalIPs = {
      enabled = true
    }

    ipam = {
      mode = "kubernetes"
    }

    routingMode    = "tunnel"
    tunnelProtocol = "vxlan"

    # DRY: Capabilities from shared module
    securityContext = {
      capabilities = {
        ciliumAgent      = var.cilium_agent_capabilities.add
        cleanCiliumState = var.cilium_clean_capabilities.add
      }
    }

    cgroup = {
      autoMount = {
        enabled = false
      }
      hostRoot = "/sys/fs/cgroup"
    }

    bpf = {
      hostLegacyRouting = true
    }

    hubble = {
      enabled = true
      relay = {
        enabled     = true
        tolerations = var.control_plane_tolerations
      }
      ui = {
        enabled     = false
        tolerations = var.control_plane_tolerations
      }
      metrics = {
        enabled = [
          "dns",
          "drop",
          "tcp",
          "flow",
          "port-distribution",
          "icmp",
          "http"
        ]
      }
    }

    operator = {
      replicas    = 1
      tolerations = var.control_plane_tolerations
    }

    # DNS Proxy Configuration
    # Disable transparent mode to fix CoreDNS timeout issues
    # See: vixens-l31o (P0 bug - affects dev + prod)
    dnsProxy = {
      enableTransparentMode = false
    }
  })]

  depends_on = [
    var.talos_cluster_module,
    var.wait_for_k8s_api
  ]
}

# --------------------------------------------------------------------------
# WAIT FOR CILIUM CRDs
# --------------------------------------------------------------------------
resource "null_resource" "wait_for_cilium_crds" {
  depends_on = [
    helm_release.cilium
  ]

  provisioner "local-exec" {
    command = "bash ${path.module}/wait_for_cilium_crds.sh"

    environment = {
      KUBECONFIG_PATH = var.kubeconfig_path
    }
  }
}

# --------------------------------------------------------------------------
# CILIUM L2 ANNOUNCEMENTS RESOURCES
# --------------------------------------------------------------------------
resource "kubectl_manifest" "cilium_ip_pool" {
  yaml_body = file(var.ip_pool_yaml_path)

  depends_on = [
    null_resource.wait_for_cilium_crds
  ]
}

resource "kubectl_manifest" "cilium_l2_policy" {
  yaml_body = file(var.l2_policy_yaml_path)

  depends_on = [
    kubectl_manifest.cilium_ip_pool
  ]
}
