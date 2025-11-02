module "talos_cluster" {
  source = "../../modules/talos"

  cluster_name     = "vixens-test"
  talos_version    = "v1.11.3"
  cluster_endpoint = "https://192.168.111.170:6443" # VIP sur VLAN 111

  control_plane_nodes = {
    "carny" = {
      name         = "carny"
      ip_address   = "192.168.0.173"  # Maintenance IP for initial access
      mac_address  = "00:15:5d:00:cb:18"
      install_disk = "/dev/sda"
      network = {
        interface = "enx00155d00cb18"
        vlans = [
          {
            vlanId    = 111
            addresses = ["192.168.111.173/24"]
            gateway   = ""
          },
          {
            vlanId    = 209
            addresses = ["192.168.209.173/24"]
            gateway   = "192.168.209.1"
          }
        ]
      }
    },
    "celesty" = {
      name         = "celesty"
      ip_address   = "192.168.0.174"  # Maintenance IP for initial access
      mac_address  = "00:15:5d:00:cb:19"
      install_disk = "/dev/sda"
      network = {
        interface = "enx00155d00cb19"
        vlans = [
          {
            vlanId    = 111
            addresses = ["192.168.111.174/24"]
            gateway   = ""
          },
          {
            vlanId    = 209
            addresses = ["192.168.209.174/24"]
            gateway   = "192.168.209.1"
          }
        ]
      }
    },
    "citrine" = {
      name         = "citrine"
      ip_address   = "192.168.0.172"  # Maintenance IP for initial access
      mac_address  = "00:15:5d:00:cb:1a"
      install_disk = "/dev/sda"
      network = {
        interface = "enx00155d00cb1a"
        vlans = [
          {
            vlanId    = 111
            addresses = ["192.168.111.172/24"]
            gateway   = ""
          },
          {
            vlanId    = 209
            addresses = ["192.168.209.172/24"]
            gateway   = "192.168.209.1"
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
  filename        = "${path.module}/kubeconfig-test"
  file_permission = "0600"
}

# Output talosconfig to file
resource "local_file" "talosconfig" {
  content         = module.talos_cluster.talosconfig
  filename        = "${path.module}/talosconfig-test"
  file_permission = "0600"
}
