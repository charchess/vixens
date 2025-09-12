# 1. RESET + ATTENTE (local-exec)
resource "null_resource" "reset" {
  for_each = var.machines

  provisioner "local-exec" {
    command = <<-EOT
      talosctl reset \
        -n ${each.value.prod_ip} \
        -e ${each.value.prod_ip} \
        --talosconfig ${pathexpand(var.talosconfig_path)} \
        --system-labels-to-wipe STATE \
        --system-labels-to-wipe EPHEMERAL \
        --graceful=true --reboot --wait=false
    EOT
  }
}

resource "null_resource" "wait_install" {
  for_each = var.machines

  provisioner "local-exec" {
    command = "until ping -c1 -W1 ${each.value.install_ip} >/dev/null; do sleep 1; done"
  }

  depends_on = [null_resource.reset]
}

# 2. APPLY CONFIG
resource "talos_machine_configuration_apply" "apply" {
  for_each = var.machines

  node          = each.value.install_ip
  endpoint      = each.value.install_ip
  talos_config  = file(pathexpand(var.talosconfig_path))
  config_patches = [{
    op    = "replace"
    path  = "/machine"
    value = local.machine_patches[each.key]
  }]
  mode   = "interactive"
  reset  = false

  depends_on = [null_resource.wait_install]
}

# 3. BOOTSTRAP
resource "talos_cluster_bootstrap" "bootstrap" {
  node         = var.machines["opale"].install_ip
  endpoint     = var.machines["opale"].install_ip
  talos_config = file(pathexpand(var.talosconfig_path))

  depends_on = [talos_machine_configuration_apply.apply]
}

# 4. RECUPERER KUBECONFIG
data "talos_client_configuration" "this" {
  cluster_name = "vixens-${var.env}"
  endpoints    = [for m in var.machines : m.prod_ip]
  talos_config = file(pathexpand(var.talosconfig_path))

  depends_on = [talos_cluster_bootstrap.bootstrap]
}

resource "local_file" "kubeconfig" {
  content  = data.talos_client_configuration.this.kubeconfig_raw
  filename = pathexpand("~/vixens/kubeconfig-${var.env}")
}

resource "local_file" "talosconfig_out" {
  content  = data.talos_client_configuration.this.talos_config
  filename = pathexpand("~/vixens/talosconfig-${var.env}")
}