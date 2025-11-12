variable "git_branch" {
  description = "The Git branch to use for ArgoCD applications."
  type        = string
}

variable "environment" {
  description = "The name of the deployment environment (e.g., dev, test, prod)."
  type        = string
}

variable "vlan_services_subnet" {
  description = "The VLAN subnet for services."
  type        = string
}

variable "argocd_service_type" {
  description = "The type of Kubernetes service to use for the ArgoCD server (e.g., ClusterIP, LoadBalancer)."
  type        = string
  default     = "LoadBalancer"
}

variable "argocd_loadbalancer_ip" {
  description = "The IP address to assign to the ArgoCD server LoadBalancer."
  type        = string
}

variable "argocd_disable_auth" {
  description = "If true, disables authentication on the ArgoCD server. WARNING: INSECURE."
  type        = bool
  default     = false
}

variable "argocd_hostname" {
  description = "The hostname for the ArgoCD Ingress."
  type        = string
}

variable "cluster_name" {
  description = "The name of the Talos cluster and Kubernetes cluster."
  type        = string
}

variable "cluster_endpoint" {
  description = "The Kubernetes API endpoint (VIP) for the cluster."
  type        = string
}

variable "control_plane_nodes" {
  description = "Map of control plane nodes with their complete configuration."
  type = map(object({
    name         = string
    ip_address   = string
    mac_address  = string
    install_disk = string
    network = object({
      interface = string
      vlans = list(object({
        vlanId    = number
        addresses = list(string)
        gateway   = string
      }))
    })
  }))
}

variable "worker_nodes" {
  description = "Map of worker nodes with their complete configuration."
  type = map(object({
    name         = string
    ip_address   = string
    mac_address  = string
    install_disk = string
    network = object({
      interface = string
      vlans = list(object({
        vlanId    = number
        addresses = list(string)
        gateway   = string
      }))
    })
  }))
  default = {}
}

variable "l2_pool_name" {
  description = "The name of the Cilium L2 LoadBalancer IP Pool."
  type        = string
}

variable "l2_pool_ips" {
  description = "List of IP ranges for the Cilium L2 LoadBalancer IP Pool."
  type        = list(string)
}

variable "l2_policy_name" {
  description = "The name of the Cilium L2 Announcement Policy."
  type        = string
}

variable "l2_policy_interfaces" {
  description = "List of interfaces for the Cilium L2 Announcement Policy."
  type        = list(string)
}

variable "l2_policy_node_selector_labels" {
  description = "Node selector labels for the Cilium L2 Announcement Policy."
  type        = map(string)
  default     = {}
}

variable "cluster_vip" {
  description = "The Virtual IP address for the Kubernetes API server."
  type        = string
}

variable "argocd_insecure" {
  description = "If true, run the ArgoCD server in insecure (HTTP) mode."
  type        = bool
  default     = true
}

variable "argocd_anonymous_enabled" {
  description = "If true, enables anonymous user access to ArgoCD."
  type        = bool
  default     = true
}

variable "talos_version" {
  description = "The version of Talos to use for the cluster nodes."
  type        = string
  default     = "v1.11.5" # Based on current usage
}

variable "kubernetes_version" {
  description = "The version of Kubernetes to deploy on the cluster."
  type        = string
  default     = "1.30.0" # Based on current usage
}

variable "talos_image" {
  description = "The custom Talos image URL from factory (e.g., factory.talos.dev/installer/<schematic_id>:v1.11.3) for extensions like iSCSI. Leave empty to use default image."
  type        = string
  default     = ""
}

variable "kubeconfig_path" {
  description = "The path where the generated kubeconfig file will be saved."
  type        = string
}

variable "talosconfig_path" {
  description = "The path where the generated talosconfig file will be saved."
  type        = string
}

variable "cilium_ip_pool_yaml_path" {
  description = "The path to the Cilium LoadBalancer IP Pool YAML file."
  type        = string
}

variable "cilium_l2_policy_yaml_path" {
  description = "The path to the Cilium L2 Announcement Policy YAML file."
  type        = string
}