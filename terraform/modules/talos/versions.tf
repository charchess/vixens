terraform {
  required_providers {
    talos = {
      source  = "siderolabs/talos"
      version = "~> 0.9.0" # Use the latest version, allow patch updates
    }
  }
}