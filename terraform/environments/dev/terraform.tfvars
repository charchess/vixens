git_branch             = "dev"
environment            = "dev"
vlan_services_subnet   = "192.168.208.0/24"
argocd_service_type    = "LoadBalancer"
argocd_loadbalancer_ip = "192.168.208.71"
argocd_disable_auth    = true
argocd_hostname        = "argocd.dev.truxonline.com"

# Talos Cluster Configuration
cluster_name     = "vixens-dev"
cluster_endpoint = "https://192.168.111.160:6443"
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
worker_nodes = {}

# Cilium L2 Announcement
l2_pool_name         = "dev-pool"
l2_pool_ips          = ["192.168.208.70-192.168.208.89"]
l2_policy_name       = "dev-l2-policy"
l2_policy_interfaces = ["eth1"]
l2_policy_node_selector_labels = {
  "kubernetes.io/hostname" = "obsy"
}

argocd_insecure          = true
argocd_anonymous_enabled = true

cluster_vip = "192.168.111.160"

talos_version                = "v1.11.5"
talos_image                  = "factory.talos.dev/installer/613e1592b2da41ae5e265e8789429f22e121aab91cb4deb6bc3c0b6262961245:v1.11.5"
kubeconfig_path              = "./kubeconfig-dev"
talosconfig_path             = "./talosconfig-dev"
cilium_ip_pool_yaml_path     = "../../../apps/cilium-lb/overlays/dev/ippool.yaml"
cilium_l2_policy_yaml_path   = "../../../apps/cilium-lb/base/l2policy.yaml"
kubernetes_version           = "1.30.0"
