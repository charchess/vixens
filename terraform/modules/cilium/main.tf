resource "helm_release" "cilium" {
  name       = var.release_name
  repository = "https://helm.cilium.io/"
  chart      = "cilium"
  version    = var.chart_version
  namespace  = var.namespace

  wait          = true
  wait_for_jobs = true
  timeout       = 600

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
    securityContext = {
      capabilities = {
        ciliumAgent = [
          "CHOWN",
          "KILL",
          "NET_ADMIN",
          "NET_RAW",
          "IPC_LOCK",
          "SYS_ADMIN",
          "SYS_RESOURCE",
          "DAC_OVERRIDE",
          "FOWNER",
          "SETGID",
          "SETUID"
        ]
        cleanCiliumState = [
          "NET_ADMIN",
          "SYS_ADMIN",
          "SYS_RESOURCE"
        ]
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
        enabled = true
        tolerations = [
          {
            key      = "node-role.kubernetes.io/control-plane"
            operator = "Exists"
            effect   = "NoSchedule"
          }
        ]
      }
      ui = {
        enabled = true
        tolerations = [
          {
            key      = "node-role.kubernetes.io/control-plane"
            operator = "Exists"
            effect   = "NoSchedule"
          }
        ]
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
      replicas = 1
      tolerations = [
        {
          key      = "node-role.kubernetes.io/control-plane"
          operator = "Exists"
          effect   = "NoSchedule"
        }
      ]
    }
  })]

  depends_on = [
    var.talos_cluster_module,
    var.wait_for_k8s_api
  ]
}

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

resource "kubectl_manifest" "cilium_ip_pool" {
  yaml_body = file(var.ip_pool_yaml_path)

  depends_on = [
    null_resource.wait_for_cilium_crds
  ]
}

resource "kubectl_manifest" "cilium_l2_policy" {
  yaml_body = file(var.l2_policy_yaml_path)

  depends_on = [
    null_resource.wait_for_cilium_crds
  ]
}
