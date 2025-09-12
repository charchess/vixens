locals {
  # On fusionne controlplane.yaml + patch machine
  machine_configs = {
    for k, v in var.machines :
    k => base64encode(templatefile("${path.module}/vixens-dev-${k}.yaml", {}))
  }
}