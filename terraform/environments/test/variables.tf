variable "environment" {
  type = string
}

variable "vlan_services_subnet" {
  type = string
}

variable "argocd_insecure" {
  type    = bool
  default = false
}

variable "argocd_disable_auth" {
  type    = bool
  default = false
}

variable "argocd_service_type" {
  type = string
}

variable "argocd_loadbalancer_ip" {
  type = string
}

variable "argocd_anonymous_enabled" {
  type    = bool
  default = false
}

variable "argocd_hostname" {
  type = string
  default = ""
}

variable "git_branch" {
  type = string
}
