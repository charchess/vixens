cluster_name     = "vixens-dev"
cluster_endpoint = "192.168.111.10"

control_plane_nodes = {
  "obsy" = {
    hostname        = "obsy"
    install_disk    = "/dev/sda"
    ip_address_int  = "192.168.111.11"
    gateway_int     = "192.168.111.1"
    ip_address_ext  = "192.168.208.11"
    gateway_ext     = "192.168.208.1"
    vlan_id_ext     = 208
    dns_servers     = ["192.168.111.1", "8.8.8.8"]
  },
  "opale" = {
    hostname        = "opale"
    install_disk    = "/dev/sda"
    ip_address_int  = "192.168.111.12"
    gateway_int     = "192.168.111.1"
    ip_address_ext  = "192.168.208.12"
    gateway_ext     = "192.168.208.1"
    vlan_id_ext     = 208
    dns_servers     = ["192.168.111.1", "8.8.8.8"]
  },
  "onyx" = {
    hostname        = "onyx"
    install_disk    = "/dev/sda"
    ip_address_int  = "192.168.111.13"
    gateway_int     = "192.168.111.1"
    ip_address_ext  = "192.168.208.13"
    gateway_ext     = "192.168.208.1"
    vlan_id_ext     = 208
    dns_servers     = ["192.168.111.1", "8.8.8.8"]
  }
}