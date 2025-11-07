variable "git_branch" { type = string }
variable "environment" { type = string }
variable "vlan_services_subnet" { type = string }
variable "argocd_service_type" { type = string }
variable "argocd_loadbalancer_ip" { type = string }
variable "argocd_disable_auth" { type = bool }
variable "argocd_hostname" { type = string }
variable "cluster_name" { type = string }
variable "cluster_endpoint" { type = string }
variable "control_plane_nodes" { type = any }
variable "worker_nodes" { type = any }
variable "l2_pool_name" { type = string }
variable "l2_pool_ips" { type = list(string) }
variable "l2_policy_name" { type = string }
variable "l2_policy_interfaces" { type = list(string) }
variable "l2_policy_node_selector_labels" { type = map(string) }
# variable "talos_version" { type = string }
# variable "kubernetes_version" { type = string }
# variable "force_destroy_time" { type = string }
# variable "talos_config_path" { type = string }
# variable "kubeconfig_path" { type = string }
# variable "hyperv_host" { type = string }
# variable "vm_path" { type = string }
# variable "vswitch_name" { type = string }
# variable "vlan_interco" { type = number }
# variable "ram_mb" { type = number }
# variable "processors" { type = number }
# variable "disk_size_gb" { type = number }
# variable "iso_path" { type = string }
variable "cluster_vip" { type = string }

variable "argocd_insecure" {
  type    = bool
  default = true
}
variable "argocd_anonymous_enabled" {
  type    = bool
  default = true
}