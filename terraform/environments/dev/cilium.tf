# Cilium CNI Deployment
# Cilium v1.18.3 with kube-proxy replacement, Hubble observability

resource "helm_release" "cilium" {
  name       = "cilium"
  repository = "https://helm.cilium.io/"
  chart      = "cilium"
  version    = "1.18.3"
  namespace  = "kube-system"

  # Wait for Cilium to be ready before marking as complete
  wait          = true
  wait_for_jobs = true
  timeout       = 600 # 10 minutes

  values = [yamlencode({
    # Kube-proxy replacement mode
    kubeProxyReplacement = true

    # Kubernetes API server configuration - Use KubePrism for Talos compatibility
    k8sServiceHost = "localhost" # KubePrism local endpoint
    k8sServicePort = 7445        # KubePrism port (instead of 6443)

    # IPAM configuration
    ipam = {
      mode = "kubernetes"
    }

    # Routing mode (replaces deprecated tunnel parameter)
    routingMode = "tunnel"
    tunnelProtocol = "vxlan"

    # Talos-specific security context configuration
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

    # Talos-specific cgroup configuration
    cgroup = {
      autoMount = {
        enabled = false # Talos manages cgroups differently
      }
      hostRoot = "/sys/fs/cgroup"
    }

    # Required for Talos DNS forwarding compatibility
    bpf = {
      hostLegacyRouting = true # Talos kube-dns forwarding needs this
    }

    # Hubble observability
    hubble = {
      enabled = true

      relay = {
        enabled = true
        # Tolerate control-plane taint for full control-plane cluster
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
        # Tolerate control-plane taint for full control-plane cluster
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

    # Operator configuration
    operator = {
      replicas = 1 # Single replica for dev (3 control planes)
      # Tolerate control-plane taint for full control-plane cluster
      tolerations = [
        {
          key      = "node-role.kubernetes.io/control-plane"
          operator = "Exists"
          effect   = "NoSchedule"
        }
      ]
    }
  })]

  # Ensure Talos cluster is ready before deploying Cilium
  depends_on = [
    module.talos_cluster
  ]
}
