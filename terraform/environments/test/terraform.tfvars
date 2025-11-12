git_branch             = "test"
environment            = "test"
vlan_services_subnet   = "192.168.209.0/24"
argocd_service_type    = "LoadBalancer"
argocd_loadbalancer_ip = "192.168.209.71"
argocd_disable_auth    = true
argocd_hostname        = "argocd.test.truxonline.com"

# Talos Cluster Configuration
cluster_name     = "vixens-test"
cluster_endpoint = "https://192.168.111.170:6443"
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
worker_nodes = {}

# Cilium L2 Announcement
l2_pool_name         = "test-pool"
l2_pool_ips          = ["192.168.209.70-192.168.209.89"]
l2_policy_name       = "test-l2-policy"
l2_policy_interfaces = ["enx*"]
l2_policy_node_selector_labels = {
  "kubernetes.io/hostname" = "citrine"
}

argocd_insecure          = true
argocd_anonymous_enabled = true

cluster_vip = "192.168.111.170"

talos_version                = "v1.11.5"
talos_image                  = "factory.talos.dev/installer/613e1592b2da41ae5e265e8789429f22e121aab91cb4deb6bc3c0b6262961245:v1.11.5"
kubeconfig_path              = "./kubeconfig-test"
talosconfig_path             = "./talosconfig-test"
cilium_ip_pool_yaml_path     = "../../../apps/cilium-lb/overlays/test/ippool.yaml"
cilium_l2_policy_yaml_path   = "../../../apps/cilium-lb/base/l2policy.yaml"
kubernetes_version           = "1.30.0"
