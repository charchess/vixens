variable "cluster_name" {
  description = "The name of the Talos cluster."
  type        = string
}

variable "cluster_endpoint" {
  description = "The Virtual IP (VIP) or FQDN for the Kubernetes API."
  type        = string
}

variable "control_plane_nodes" {
  description = "A map of control plane node objects."
  type = map(object({
    hostname        = string
    install_disk    = string
    ip_address_int  = string # Internal network (VLAN 111)
    ip_address_ext  = string # External/Service network (VLAN 208, etc.)
    gateway_int     = string
    gateway_ext     = string
    dns_servers     = list(string)
    vlan_id_ext     = number
  }))
}