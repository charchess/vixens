git_branch             = "dev"
environment            = "dev"
vlan_services_subnet   = "192.168.210.0/24"
argocd_service_type    = "LoadBalancer"
argocd_loadbalancer_ip = "192.168.210.71"
argocd_disable_auth    = true
argocd_hostname        = "argocd.stg.truxonline.com"

# Talos Cluster Configuration
cluster_name     = "vixens-stg"
cluster_endpoint = "https://192.168.111.180:6443"
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
    ip_address   = "192.168.0.164"
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
    ip_address   = "192.168.0.165"
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
worker_nodes = {}

# Cilium L2 Announcement
l2_pool_name         = "dev-pool"
l2_pool_ips          = ["192.168.210.70-192.168.210.89"]
l2_policy_name       = "dev-l2-policy"
l2_policy_interfaces = ["enx"]
l2_policy_node_selector_labels = {
  "kubernetes.io/hostname" = "saphira"
}

argocd_insecure          = true
argocd_anonymous_enabled = true

cluster_vip = "192.168.111.180"

talos_version                = "v1.11.5"
talos_image                  = "factory.talos.dev/installer/613e1592b2da41ae5e265e8789429f22e121aab91cb4deb6bc3c0b6262961245:v1.11.5"
kubeconfig_path              = "./kubeconfig-dev"
talosconfig_path             = "./talosconfig-dev"
cilium_ip_pool_yaml_path     = "../../../apps/cilium-lb/overlays/dev/ippool.yaml"
cilium_l2_policy_yaml_path   = "../../../apps/cilium-lb/base/l2policy.yaml"
kubernetes_version           = "1.30.0"