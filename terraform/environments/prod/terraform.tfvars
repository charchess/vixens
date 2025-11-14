# ============================================================================
# VIXENS PROD ENVIRONMENT - CONFIGURATION VALUES
# ============================================================================

environment = "prod"
git_branch  = "main"

# ----------------------------------------------------------------------------
# CLUSTER CONFIGURATION
# ----------------------------------------------------------------------------
cluster = {
  name               = "vixens"
  endpoint           = "https://192.168.111.170:6443"
  talos_version      = "v1.11.5"
  talos_image        = "factory.talos.dev/installer/613e1592b2da41ae5e265e8789429f22e121aab91cb4deb6bc3c0b6262961245:v1.11.5"
  kubernetes_version = "1.30.0"
}

# ----------------------------------------------------------------------------
# CONTROL PLANE NODES
# ----------------------------------------------------------------------------
control_plane_nodes = {
  "perla" = {
    name         = "perla"
    ip_address   = "192.168.200.65"
    mac_address  = "68:1d:ef:4d:d6:a9"
    install_disk = "nvme0n1"
    network = {
      interface = "enp3s0"
      vlans = [
        {
          vlanId    = 111
          addresses = ["192.168.111.65/24"]
          gateway   = ""
        },
        {
          vlanId    = 200 # Assuming 200.x.x.x is VLAN 200
          addresses = ["192.168.200.65/24"]
          gateway   = "192.168.200.1" # Assuming a gateway
        }
      ]
    }
  },
  "peridot" = {
    name         = "peridot"
    ip_address   = "192.168.200.63"
    mac_address  = "68:1d:ef:56:d7:bb"
    install_disk = "nvme0n1"
    network = {
      interface = "enp2s0"
      vlans = [
        {
          vlanId    = 111
          addresses = ["192.168.111.60/24", "192.168.111.63/24"]
          gateway   = ""
        },
        {
          vlanId    = 200 # Assuming 200.x.x.x is VLAN 200
          addresses = ["192.168.200.63/24"]
          gateway   = "192.168.200.1" # Assuming a gateway
        }
      ]
    }
  },
  "purpuria" = {
    name         = "purpuria"
    ip_address   = "192.168.200.66"
    mac_address  = "00:e1:4f:68:0d:f8"
    install_disk = "sda"
    network = {
      interface = "enp2s0"
      vlans = [
        {
          vlanId    = 111
          addresses = ["192.168.111.66/24"]
          gateway   = ""
        },
        {
          vlanId    = 200 # Assuming 200.x.x.x is VLAN 200
          addresses = ["192.168.200.66/24"]
          gateway   = "192.168.200.1" # Assuming a gateway
        }

      ]
    }
  }
}

# ----------------------------------------------------------------------------
# WORKER NODES
# ----------------------------------------------------------------------------
worker_nodes = {}

# ----------------------------------------------------------------------------
# FILE PATHS
# ----------------------------------------------------------------------------
paths = {
  kubeconfig            = "./kubeconfig-prod"
  talosconfig           = "./talosconfig-prod"
  cilium_ip_pool_yaml   = "../../../apps/cilium-lb/overlays/prod/ippool.yaml"
  cilium_l2_policy_yaml = "../../../apps/cilium-lb/overlays/prod/l2policy.yaml"
}

# ----------------------------------------------------------------------------
# ARGOCD CONFIGURATION
# ----------------------------------------------------------------------------
argocd = {
  loadbalancer_ip   = "192.168.200.71"
  hostname          = "argocd.truxonline.com"
  admin_password    = "admin" # TODO: Change in production
  insecure          = true
  anonymous_enabled = true
  disable_auth      = true
}
