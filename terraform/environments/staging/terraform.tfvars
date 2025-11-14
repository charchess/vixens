# ============================================================================
# VIXENS STAGING ENVIRONMENT - CONFIGURATION VALUES
# ============================================================================

environment = "staging"
git_branch  = "staging"

# ----------------------------------------------------------------------------
# CLUSTER CONFIGURATION
# ----------------------------------------------------------------------------
cluster = {
  name               = "vixens-stg"
  endpoint           = "https://192.168.111.180:6443"
  talos_version      = "v1.11.5"
  talos_image        = "factory.talos.dev/installer/613e1592b2da41ae5e265e8789429f22e121aab91cb4deb6bc3c0b6262961245:v1.11.5"
  kubernetes_version = "1.30.0"
}

# ----------------------------------------------------------------------------
# CONTROL PLANE NODES
# ----------------------------------------------------------------------------
control_plane_nodes = {
  "serpentina" = {
    name         = "serpentina"
    ip_address   = "192.168.0.128"
    mac_address  = "00:15:5D:00:77:01"
    install_disk = "/dev/sda"
    network = {
      interface = "enx00155d007701"
      vlans = [
        {
          vlanId    = 111
          addresses = ["192.168.111.182/24"]
          gateway   = ""
        },
        {
          vlanId    = 210
          addresses = ["192.168.210.182/24"]
          gateway   = "192.168.210.1"
        }
      ]
    }
  },
  "spinelia" = {
    name         = "spinelia"
    ip_address   = "192.168.0.129"
    mac_address  = "00:15:5D:00:77:02"
    install_disk = "/dev/sda"
    network = {
      interface = "enx00155d007702"
      vlans = [
        {
          vlanId    = 111
          addresses = ["192.168.111.183/24"]
          gateway   = ""
        },
        {
          vlanId    = 210
          addresses = ["192.168.210.183/24"]
          gateway   = "192.168.210.1"
        }
      ]
    }
  },
  "saphira" = {
    name         = "saphira"
    ip_address   = "192.168.0.127"
    mac_address  = "00:15:5D:00:77:00"
    install_disk = "/dev/sda"
    network = {
      interface = "enx00155d007700"
      vlans = [
        {
          vlanId    = 111
          addresses = ["192.168.111.184/24"]
          gateway   = ""
        },
        {
          vlanId    = 210
          addresses = ["192.168.210.184/24"]
          gateway   = "192.168.210.1"
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
  kubeconfig            = "./kubeconfig-staging"
  talosconfig           = "./talosconfig-staging"
  cilium_ip_pool_yaml   = "../../../apps/cilium-lb/overlays/staging/ippool.yaml"
  cilium_l2_policy_yaml = "../../../apps/cilium-lb/overlays/staging/l2policy.yaml"
}

# ----------------------------------------------------------------------------
# ARGOCD CONFIGURATION
# ----------------------------------------------------------------------------
argocd = {
  loadbalancer_ip   = "192.168.210.71"
  hostname          = "argocd.stg.truxonline.com"
  admin_password    = "admin" # TODO: Change in production
  insecure          = true
  anonymous_enabled = true
  disable_auth      = true
}
