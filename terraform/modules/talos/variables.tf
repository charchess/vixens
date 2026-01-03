variable "control_plane_nodes" {
  description = "Map of control plane nodes with their complete configuration"
  type = map(object({
    name         = string
    ip_address   = string
    mac_address  = string
    install_disk = string
    nameservers  = optional(list(string), [])
    network = object({
      interface = string
      vlans = list(object({
        vlanId    = number
        addresses = list(string)
        gateway   = string
      }))
    })
  }))

  validation {
    condition     = length(var.control_plane_nodes) % 2 == 1
    error_message = "Le nombre de control planes doit être impair (quorum etcd)."
  }
}

variable "worker_nodes" {
  description = "Map of worker nodes with their complete configuration"
  type = map(object({
    name         = string
    ip_address   = string
    mac_address  = string
    install_disk = string
    nameservers  = optional(list(string), [])
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

variable "cluster_name" {
  description = "The name of the Talos cluster."
  type        = string
}

variable "talos_version" {
  description = "Talos Linux version to deploy"
  type        = string
  default     = "v1.11.3"
}

variable "kubernetes_version" {
  description = "Kubernetes version to deploy"
  type        = string
  default     = "v1.30.0"
}

variable "talos_image" {
  description = "Custom Talos image URL from factory (e.g., factory.talos.dev/installer/<schematic_id>:v1.11.3) for extensions like iSCSI. Leave empty to use default image."
  type        = string
  default     = ""
}

variable "cluster_endpoint" {
  description = "Kubernetes API endpoint (VIP)"
  type        = string
}

variable "pod_subnet" {
  description = "Pod CIDR subnet"
  type        = string
  default     = "10.244.0.0/16"
}

variable "service_subnet" {
  description = "Service CIDR subnet"
  type        = string
  default     = "10.96.0.0/12"
}
