# ============================================================================
# VIXENS TEST ENVIRONMENT - CONFIGURATION VALUES
# ============================================================================
# Nodes: citrine, carny, celesty (3 CP HA)

environment = "test"
git_branch  = "test"

# ----------------------------------------------------------------------------
# CLUSTER
# ----------------------------------------------------------------------------
cluster = {
  name               = "vixens-test"
  endpoint           = "https://192.168.111.170:6443"
  vip                = "192.168.111.170"
  talos_version      = "v1.11.5"
  talos_image        = "factory.talos.dev/installer/613e1592b2da41ae5e265e8789429f22e121aab91cb4deb6bc3c0b6262961245:v1.11.5"
  kubernetes_version = "1.30.0"
}

# ----------------------------------------------------------------------------
# CONTROL PLANE NODES (3 HA)
# ----------------------------------------------------------------------------
control_plane_nodes = {
  "citrine" = {
    name         = "citrine"
    ip_address   = "192.168.0.172"
    mac_address  = "00:15:5d:00:cb:1a"
    install_disk = "/dev/sda"
    network = {
      interface = "enx00155d00cb1a"
      vlans = [
        {
          vlanId    = 111
          addresses = ["192.168.111.172/24"]
          gateway   = ""
        },
        {
          vlanId    = 209
          addresses = ["192.168.209.172/24"]
          gateway   = "192.168.209.1"
        }
      ]
    }
  },
  "carny" = {
    name         = "carny"
    ip_address   = "192.168.0.173"
    mac_address  = "00:15:5d:00:cb:18"
    install_disk = "/dev/sda"
    network = {
      interface = "enx00155d00cb18"
      vlans = [
        {
          vlanId    = 111
          addresses = ["192.168.111.173/24"]
          gateway   = ""
        },
        {
          vlanId    = 209
          addresses = ["192.168.209.173/24"]
          gateway   = "192.168.209.1"
        }
      ]
    }
  },
  "celesty" = {
    name         = "celesty"
    ip_address   = "192.168.0.174"
    mac_address  = "00:15:5d:00:cb:19"
    install_disk = "/dev/sda"
    network = {
      interface = "enx00155d00cb19"
      vlans = [
        {
          vlanId    = 111
          addresses = ["192.168.111.174/24"]
          gateway   = ""
        },
        {
          vlanId    = 209
          addresses = ["192.168.209.174/24"]
          gateway   = "192.168.209.1"
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
# ARGOCD
# ----------------------------------------------------------------------------
argocd = {
  service_type      = "LoadBalancer"
  loadbalancer_ip   = "192.168.209.71"
  hostname          = "argocd.test.truxonline.com"
  insecure          = true
  disable_auth      = true
  anonymous_enabled = true
}

# ----------------------------------------------------------------------------
# CILIUM L2 ANNOUNCEMENTS
# ----------------------------------------------------------------------------
cilium_l2 = {
  pool_name   = "test-pool"
  pool_ips    = ["192.168.209.70-192.168.209.89"]
  policy_name = "test-l2-policy"
  interfaces  = ["eth1"]
  node_selector = {
    "kubernetes.io/hostname" = "citrine"
  }
}

# --------------------------------------------------------------------------
# NETWORK
# --------------------------------------------------------------------------
network = {
  vlan_services_subnet = "192.168.209.0/24"
}

# ----------------------------------------------------------------------------
# FILE PATHS
# ----------------------------------------------------------------------------
paths = {
  kubeconfig            = "./kubeconfig-test"
  talosconfig           = "./talosconfig-test"
  cilium_ip_pool_yaml   = "../../../apps/cilium-lb/overlays/test/ippool.yaml"
  cilium_l2_policy_yaml = "../../../apps/cilium-lb/base/l2policy.yaml"
}
