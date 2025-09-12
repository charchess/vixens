locals {
  machine_patches = {
  for k in keys(var.machines) :
    k => file("path.module/vixens−dev−{k}.yaml")
  }
}