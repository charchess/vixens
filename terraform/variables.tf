variable "env" {
  type    = string
  default = "dev"
}

variable "machines" {
  type = map(object({
    install_ip = string
    prod_ip    = string
  }))
  default = {
    onyx  = { install_ip = "192.168.208.164", prod_ip = "192.168.111.164" }
    obsy  = { install_ip = "192.168.208.162", prod_ip = "192.168.111.162" }
    opale = { install_ip = "192.168.208.163", prod_ip = "192.168.111.163" }
  }
}

variable "talosconfig_path" {
  type    = string
  default = "~/vixens/talosconfig-dev"
}

variable "controlplane_yaml" {
  type    = string
  default = "controlplane.yaml"
}