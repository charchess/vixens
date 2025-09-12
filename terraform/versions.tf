terraform {
  required_version = ">= 1.5"

  required_providers {
    talos = {
      source  = "siderolabs/talos"
      version = "~> 0.6"
    }
    local = { source = "hashicorp/local" }
    null  = { source = "hashicorp/null" }
  }
}