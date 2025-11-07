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
    hostname     = "obsy"
    ip_address   = "192.168.111.162"
    install_disk = "/dev/sda"
    nameservers  = ["1.1.1.1", "8.8.8.8"]
    network_interfaces = [
      {
        iface     = "eth1"
        vlan      = 208
        addresses = ["192.168.208.162/24"]
        routes    = [{ network = "0.0.0.0/0", gateway = "192.168.208.1" }]
      }
    ]
  },
  "opale" = {
    hostname     = "opale"
    ip_address   = "192.168.111.163"
    install_disk = "/dev/sda"
    nameservers  = ["1.1.1.1", "8.8.8.8"]
    network_interfaces = [
      {
        iface     = "eth1"
        vlan      = 208
        addresses = ["192.168.208.163/24"]
        routes    = [{ network = "0.0.0.0/0", gateway = "192.168.208.1" }]
      }
    ]
  },
  "onyx" = {
    hostname     = "onyx"
    ip_address   = "192.168.111.164"
    install_disk = "/dev/sda"
    nameservers  = ["1.1.1.1", "8.8.8.8"]
    network_interfaces = [
      {
        iface     = "eth1"
        vlan      = 208
        addresses = ["192.168.208.164/24"]
        routes    = [{ network = "0.0.0.0/0", gateway = "192.168.208.1" }]
      }
    ]
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

# Common Variables (assuming they are the same as test)
talos_version      = "1.7"
kubernetes_version = "1.30"
force_destroy_time = "15m"
talos_config_path  = "/root/.talos/config"
kubeconfig_path    = "/root/vixens/terraform/environments/dev/kubeconfig-dev"
hyperv_host        = "HV-TRUX-2"
vm_path            = "D:\\VM"
vswitch_name       = "TRUNK-EXT"
vlan_interco       = 111
ram_mb             = 4096
processors         = 2
disk_size_gb       = 40
iso_path           = "C:\\ISO\\talos-amd64.iso"
cluster_vip        = "192.168.111.160"

argocd_insecure          = true
argocd_anonymous_enabled = true
