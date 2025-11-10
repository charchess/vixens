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
variable "cluster_vip" { type = string }

variable "argocd_insecure" {
  type    = bool
  default = true
}
variable "argocd_anonymous_enabled" {
  type    = bool
  default = true
}

variable "talos_version" {
  description = "The version of Talos to use."
  type        = string
}

variable "kubernetes_version" {
  description = "The version of Kubernetes to use."
  type        = string
}

variable "talos_image" {
  description = "The custom Talos image to use."
  type        = string
}

variable "kubeconfig_path" {
  description = "The path where the kubeconfig file will be saved."
  type        = string
}

variable "talosconfig_path" {
  description = "The path where the talosconfig file will be saved."
  type        = string
}

variable "cilium_ip_pool_yaml_path" {
  description = "The path to the Cilium IP pool YAML file."
  type        = string
}

variable "cilium_l2_policy_yaml_path" {
  description = "The path to the Cilium L2 policy YAML file."
  type        = string
}
