git_branch             = "prod"
environment            = "prod"
vlan_services_subnet   = "192.168.208.0/24"
argocd_service_type    = "LoadBalancer"
argocd_loadbalancer_ip = "192.168.208.71" # Assuming same LB IP for now, might need adjustment
argocd_disable_auth    = true
argocd_hostname        = "argocd.truxonline.com"

# Talos Cluster Configuration
cluster_name     = "vixens-prod"
cluster_endpoint = "https://192.168.111.170:6443" # New VIP for prod
control_plane_nodes = {
  "emy" = {
    name         = "emy"
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
  "jade" = {
    name         = "jade"
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
  "ruby" = {
    name         = "ruby"
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
worker_nodes = {}

# Cilium L2 Announcement
l2_pool_name         = "prod-pool"
l2_pool_ips          = ["192.168.208.100-192.168.208.119"] # Placeholder, needs verification
l2_policy_name       = "prod-l2-policy"
l2_policy_interfaces = ["enp3s0", "enp2s0"] # Based on get links output
l2_policy_node_selector_labels = {
  "kubernetes.io/hostname" = "emy" # Assuming all control plane nodes are selected
}

argocd_insecure          = true
argocd_anonymous_enabled = true

cluster_vip = "192.168.111.170" # New VIP for prod

talos_version                = "v1.11.5" # Keep same for now, might need update
talos_image                  = "factory.talos.dev/installer/613e1592b2da41ae5e265e8789429f22e121aab91cb4deb6bc3c0b6262961245:v1.11.5" # Keep same for now
kubeconfig_path              = "./kubeconfig-prod"
talosconfig_path             = "./talosconfig-prod"
cilium_ip_pool_yaml_path     = "../../../apps/cilium-lb/overlays/prod/ippool.yaml"
cilium_l2_policy_yaml_path   = "../../../apps/cilium-lb/overlays/prod/l2policy.yaml"
kubernetes_version           = "1.30.0" # Keep same for now
