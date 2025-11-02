module "talos_cluster" {
  source = "../../modules/talos"

  cluster_name     = "vixens-dev"
  talos_version    = "v1.11.3"
  cluster_endpoint = "https://192.168.111.160:6443" # VIP sur VLAN 111

  # Optional: Custom image with iSCSI extension for Synology CSI
  # talos_image = "factory.talos.dev/installer/<schematic_id>:v1.11.3"

  # HA: 3 control planes (etcd quorum)
  control_plane_nodes = {
    "obsy" = {
      name         = "obsy"
      ip_address   = "192.168.0.162"  # Maintenance IP for initial access
      mac_address  = "00:15:5D:00:CB:10"
      install_disk = "/dev/sda"
      network = {
        interface = "enx00155d00cb10"
        vlans = [
          {
            vlanId    = 111
            addresses = ["192.168.111.162/24"]
            gateway   = ""
          },
          {
            vlanId    = 208
            addresses = ["192.168.208.162/24"]
            gateway   = "192.168.208.1"
          }
        ]
      }
    }
    "onyx" = {
      name         = "onyx"
      ip_address   = "192.168.0.164"  # Maintenance IP for initial access
      mac_address  = "00:15:5D:00:CB:11"
      install_disk = "/dev/sda"
      network = {
        interface = "enx00155d00cb11"
        vlans = [
          {
            vlanId    = 111
            addresses = ["192.168.111.164/24"]
            gateway   = ""
          },
          {
            vlanId    = 208
            addresses = ["192.168.208.164/24"]
            gateway   = "192.168.208.1"
          }
        ]
      }
    }
    "opale" = {
      name         = "opale"
      ip_address   = "192.168.0.163"  # VLAN 111 IP for initial access
      mac_address  = "00:15:5D:00:CB:0B"
      install_disk = "/dev/sda"
      network = {
        interface = "enx00155d00cb0b"
        vlans = [
          {
            vlanId    = 111
            addresses = ["192.168.111.163/24"]
            gateway   = ""
          },
          {
            vlanId    = 208
            addresses = ["192.168.208.163/24"]
            gateway   = "192.168.208.1"
          }
        ]
      }
    }
  }

  worker_nodes = {}
}

# Output kubeconfig to file for easy access
resource "local_file" "kubeconfig" {
  content         = module.talos_cluster.kubeconfig
  filename        = "${path.module}/kubeconfig-dev"
  file_permission = "0600"
}

# Output talosconfig to file
resource "local_file" "talosconfig" {
  content         = module.talos_cluster.talosconfig
  filename        = "${path.module}/talosconfig-dev"
  file_permission = "0600"
}
