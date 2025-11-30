# ============================================================================
# VIXENS PROD ENVIRONMENT - CONFIGURATION VALUES
# ============================================================================

environment = "prod"
git_branch  = "main"

# ----------------------------------------------------------------------------
# CLUSTER
# ----------------------------------------------------------------------------
cluster = {
  name               = "vixens"
  endpoint           = "https://192.168.111.190:6443"
  vip                = "192.168.111.190"
  talos_version      = "v1.11.5"
  talos_image        = "factory.talos.dev/installer/613e1592b2da41ae5e265e8789429f22e121aab91cb4deb6bc3c0b6262961245:v1.11.5"
  kubernetes_version = "1.30.0"
}

# ----------------------------------------------------------------------------
# CONTROL PLANE NODES
# ----------------------------------------------------------------------------
control_plane_nodes = {
  "powder" = {
    name         = "powder"
    ip_address   = "192.168.0.65"  # Maintenance IP for Terraform access
    mac_address  = "68:1d:ef:4d:d6:a9"
    install_disk = "nvme0n1"
    network = {
      interface = "enp3s0"
      vlans = [
        {
          vlanId    = 111
          addresses = ["192.168.111.193/24"]
          gateway   = ""
        },
        {
          vlanId    = 201
          addresses = ["192.168.201.193/24"]
          gateway   = "192.168.201.1"
        }
      ]
    }
  },
  "poison" = {
    name         = "poison"
    ip_address   = "192.168.0.63"  # Maintenance IP for Terraform access
    mac_address  = "68:1d:ef:56:d7:bb"
    install_disk = "nvme0n1"
    network = {
      interface = "enp2s0"
      vlans = [
        {
          vlanId    = 111
          addresses = ["192.168.111.190/24", "192.168.111.194/24"]
          gateway   = ""
        },
        {
          vlanId    = 201
          addresses = ["192.168.201.194/24"]
          gateway   = "192.168.201.1"
        }
      ]
    }
  },
  "phoebe" = {
    name         = "phoebe"
    ip_address   = "192.168.0.66"  # Maintenance IP for Terraform access
    mac_address  = "00:e1:4f:68:0d:f8"
    install_disk = "sda"
    network = {
      interface = "enp2s0"
      vlans = [
        {
          vlanId    = 111
          addresses = ["192.168.111.195/24"]
          gateway   = ""
        },
        {
          vlanId    = 201
          addresses = ["192.168.201.195/24"]
          gateway   = "192.168.201.1"
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
# ARGOCD CONFIGURATION
# ----------------------------------------------------------------------------
argocd = {
  service_type      = "LoadBalancer"
  loadbalancer_ip   = "192.168.201.71"
  hostname          = "argocd.truxonline.com"
  insecure          = true
  disable_auth      = true
  anonymous_enabled = true
}

# --------------------------------------------------------------------------
# CILIUM L2 ANNOUNCEMENTS
# --------------------------------------------------------------------------
cilium_l2 = {
  pool_name   = "prod-pool"
  pool_ips    = ["192.168.201.70-192.168.201.89"]
  policy_name = "prod-l2-policy"
  interfaces  = ["eth1"]
  node_selector = {
    "kubernetes.io/hostname" = "perla"
  }
}

# --------------------------------------------------------------------------
# NETWORK
# --------------------------------------------------------------------------
network = {
  vlan_services_subnet = "192.168.201.0/24"
}

# ----------------------------------------------------------------------------
# FILE PATHS
# ----------------------------------------------------------------------------
paths = {
  kubeconfig            = "./kubeconfig-prod"
  talosconfig           = "./talosconfig-prod"
  cilium_ip_pool_yaml   = "../../../apps/cilium-lb/overlays/prod/ippool.yaml"
  cilium_l2_policy_yaml = "../../../apps/cilium-lb/overlays/prod/l2policy.yaml"
  infisical_secret      = "../../../.secrets/prod/infisical-universal-auth.yaml"
}
