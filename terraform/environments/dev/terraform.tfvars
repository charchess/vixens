# ============================================================================
# VIXENS TERRAFORM - DEV ENVIRONMENT
# ============================================================================

environment = "dev"
git_branch  = "dev"

# --------------------------------------------------------------------------
# CLUSTER
# --------------------------------------------------------------------------
cluster = {
  name               = "vixens-dev"
  endpoint           = "https://192.168.111.160:6443"
  vip                = "192.168.111.160"
  talos_version      = "v1.11.5"
  talos_image        = "factory.talos.dev/installer/613e1592b2da41ae5e265e8789429f22e121aab91cb4deb6bc3c0b6262961245:v1.11.5"
  kubernetes_version = "1.30.0"
}

# --------------------------------------------------------------------------
# CONTROL PLANE NODES (3 HA)
# --------------------------------------------------------------------------
control_plane_nodes = {
  "obsy" = {
    name         = "obsy"
    ip_address   = "192.168.0.162"
    mac_address  = "00:15:5D:00:CB:10"
    install_disk = "/dev/sda"
    network = {
      interface = "enx00155d00cb10"
      vlans = [
        {
          vlanId    = 111
          addresses = ["192.168.111.162/24"]
          gateway   = ""
        },
        {
          vlanId    = 208
          addresses = ["192.168.208.162/24"]
          gateway   = "192.168.208.1"
        }
      ]
    }
  },
  "onyx" = {
    name         = "onyx"
    ip_address   = "192.168.0.164"
    mac_address  = "00:15:5D:00:CB:11"
    install_disk = "/dev/sda"
    network = {
      interface = "enx00155d00cb11"
      vlans = [
        {
          vlanId    = 111
          addresses = ["192.168.111.164/24"]
          gateway   = ""
        },
        {
          vlanId    = 208
          addresses = ["192.168.208.164/24"]
          gateway   = "192.168.208.1"
        }
      ]
    }
  },
  "opale" = {
    name         = "opale"
    ip_address   = "192.168.0.163"
    mac_address  = "00:15:5D:00:CB:0B"
    install_disk = "/dev/sda"
    network = {
      interface = "enx00155d00cb0b"
      vlans = [
        {
          vlanId    = 111
          addresses = ["192.168.111.163/24"]
          gateway   = ""
        },
        {
          vlanId    = 208
          addresses = ["192.168.208.163/24"]
          gateway   = "192.168.208.1"
        }
      ]
    }
  }
}

# --------------------------------------------------------------------------
# WORKER NODES (none for dev - all control plane)
# --------------------------------------------------------------------------
worker_nodes = {}

# --------------------------------------------------------------------------
# ARGOCD
# --------------------------------------------------------------------------
argocd = {
  loadbalancer_ip   = "192.168.208.71"
  hostname          = "argocd.dev.truxonline.com"
  admin_password    = "admin" # TODO: Change in production
  insecure          = true
  disable_auth      = true
  anonymous_enabled = true
}

# --------------------------------------------------------------------------
# CILIUM L2 ANNOUNCEMENTS
# --------------------------------------------------------------------------
cilium_l2 = {
  pool_name   = "dev-pool"
  pool_ips    = ["192.168.208.70-192.168.208.89"]
  policy_name = "dev-l2-policy"
  interfaces  = ["eth1"]
  node_selector = {
    "kubernetes.io/hostname" = "obsy"
  }
}

# --------------------------------------------------------------------------
# NETWORK
# --------------------------------------------------------------------------
network = {
  vlan_services_subnet = "192.168.208.0/24"
}

# --------------------------------------------------------------------------
# PATHS (using defaults from variables.tf)
# --------------------------------------------------------------------------
paths = {
  kubeconfig            = "./kubeconfig-dev"
  talosconfig           = "./talosconfig-dev"
  cilium_ip_pool_yaml   = "../../../apps/cilium-lb/overlays/dev/ippool.yaml"
  cilium_l2_policy_yaml = "../../../apps/cilium-lb/base/l2policy.yaml"
}
