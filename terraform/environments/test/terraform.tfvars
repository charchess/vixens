git_branch = "test"
environment = "test"
vlan_services_subnet = "192.168.209.0/24"
argocd_service_type = "LoadBalancer"
argocd_loadbalancer_ip = "192.168.209.81"


# Talos Cluster Configuration
cluster_name = "vixens-test"
cluster_endpoint = "https://192.168.111.170:6443"
control_plane_nodes = {
  "topaze" = {
    hostname = "topaze"
    ip_address = "192.168.111.172"
    install_disk = "/dev/sda"
    nameservers = ["1.1.1.1", "8.8.8.8"]
    network_interfaces = [
      {
        iface = "eth1"
        vlan = 209
        addresses = ["192.168.209.172/24"]
        routes = [{ network = "0.0.0.0/0", gateway = "192.168.209.1" }]
      }
    ]
  },
  "turquoise" = {
    hostname = "turquoise"
    ip_address = "192.168.111.173"
    install_disk = "/dev/sda"
    nameservers = ["1.1.1.1", "8.8.8.8"]
    network_interfaces = [
      {
        iface = "eth1"
        vlan = 209
        addresses = ["192.168.209.173/24"]
        routes = [{ network = "0.0.0.0/0", gateway = "192.168.209.1" }]
      }
    ]
  },
  "tanzanite" = {
    hostname = "tanzanite"
    ip_address = "192.168.111.174"
    install_disk = "/dev/sda"
    nameservers = ["1.1.1.1", "8.8.8.8"]
    network_interfaces = [
      {
        iface = "eth1"
        vlan = 209
        addresses = ["192.168.209.174/24"]
        routes = [{ network = "0.0.0.0/0", gateway = "192.168.209.1" }]
      }
    ]
  }
}
worker_nodes = {}

# Cilium L2 Announcement
l2_pool_name = "test-pool"
l2_pool_ips = ["192.168.209.80-192.168.209.90"]
l2_policy_name = "test-l2-policy"
l2_policy_interfaces = ["eth1"]
l2_policy_node_selector_labels = {
  "kubernetes.io/hostname" = "topaze"
}
argocd_insecure = true
argocd_anonymous_enabled = true
argocd_disable_auth = true
argocd_hostname = "argocd.test.truxonline.com"
talos_version = "1.7"
kubernetes_version = "1.30"
force_destroy_time = "15m"
talos_config_path = "/root/.talos/config"
kubeconfig_path = "/root/vixens/terraform/environments/test/kubeconfig-test"
hyperv_host = "HV-TRUX-2"
vm_path = "D:\\VM"
vswitch_name = "TRUNK-EXT"
vlan_interco = 111
ram_mb = 4096
processors = 2
disk_size_gb = 40
iso_path = "C:\\ISO\\talos-amd64.iso"
cluster_vip = "192.168.111.170"
cluster_vip_ip = "192.168.111.170"
cluster_vip_iface = "eth0"
cluster_vip_node_selector = "node-role.kubernetes.io/control-plane"
cluster_vip_namespace = "kube-system"
cluster_vip_image = "ghcr.io/loxilb-io/kube-loxilb:0.8.4"
cluster_vip_service_account = "kube-loxilb"
cluster_vip_cluster_role = "kube-loxilb"
cluster_vip_cluster_role_binding = "kube-loxilb"
cluster_vip_configmap = "loxilb-config"
cluster_vip_configmap_data = {
  "LoxiLB-cm.yaml" = <<-EOT
    ---
    apiVersion: v1
    kind: ConfigMap
    metadata:
      name: loxilb-config
      namespace: kube-system
    data:
      LoxiLB-cm.yaml: |
        logLevel: "info"
        setBGP:
          bgpListenPort: 179
        setLBMode: "default"
        setRoles:
          - "full-loxilb"
        setUniqueIP: "public"
        setVTI:
          vtiName: "vti-1"
        setNetConf:
          - dev: "eth0"
            vlanID: 111
            port: 1111
            addr: "192.168.111.1/24"
          - dev: "eth1"
            vlanID: 209
            port: 2222
            addr: "192.168.209.1/24"
    EOT
}