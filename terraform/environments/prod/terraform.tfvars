# ============================================================================
# VIXENS PROD ENVIRONMENT - CONFIGURATION VALUES
# ============================================================================

environment = "prod"
git_branch  = "prod-stable"  # Tag-based deployment (trunk-based workflow)

# ----------------------------------------------------------------------------
# CLUSTER
# ----------------------------------------------------------------------------
cluster = {
  name               = "vixens"
  endpoint           = "https://192.168.111.190:6443"
  vip                = "192.168.111.190"
  talos_version      = "v1.12.0"
  talos_image        = "factory.talos.dev/metal-installer/249d9135de54962744e917cfe654117000cba369f9152fbab9d055a00aa3664f:v1.12.0"
  kubernetes_version = "1.30.0"
}

# ----------------------------------------------------------------------------
# CONTROL PLANE NODES
# ----------------------------------------------------------------------------
control_plane_nodes = {
  "powder" = {
    name         = "powder"
    ip_address   = "192.168.0.65" # Maintenance IP for Terraform access
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
    ip_address   = "192.168.0.63" # Maintenance IP for Terraform access
    mac_address  = "68:1d:ef:56:d7:bb"
    install_disk = "nvme0n1"
    network = {
      interface = "enp2s0"
      vlans = [
        {
          vlanId    = 111
          addresses = ["192.168.111.194/24"] # IP node uniquement (VIP gérée par Talos)
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
    ip_address   = "192.168.0.66" # Maintenance IP for Terraform access
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
# NOTE: Configuration mise à jour pour refléter le YAML déployé manuellement
# Le module Terraform utilise file() pour charger les YAMLs existants
cilium_l2 = {
  pool_name     = "prod-pool"
  pool_ips      = ["192.168.201.70-192.168.201.89"]
  policy_name   = "prod-l2-policy"
  interfaces    = ["^en.*\\.201$"] # Regex VLAN 201 pour nodes physiques
  node_selector = {}               # Tous les nodes (correspond au YAML déployé)
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
  cilium_ip_pool_yaml   = "../../../apps/00-infra/cilium-lb/overlays/prod/ippool.yaml"
  cilium_l2_policy_yaml = "../../../apps/00-infra/cilium-lb/overlays/prod/l2policy.yaml"
  infisical_secret      = "../../../.secrets/prod/infisical-universal-auth.yaml"
}
