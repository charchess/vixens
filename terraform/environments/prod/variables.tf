# ============================================================================
# VIXENS TERRAFORM - DEV ENVIRONMENT VARIABLES
# ============================================================================
# Strongly typed variables with validation

# --------------------------------------------------------------------------
# ENVIRONMENT
# --------------------------------------------------------------------------
variable "environment" {
  description = "Environment name"
  type        = string

  validation {
    condition     = contains(["dev", "test", "staging", "prod"], var.environment)
    error_message = "Environment must be one of: dev, test, staging, prod"
  }
}

variable "git_branch" {
  description = "Git branch for ArgoCD targetRevision"
  type        = string
}

# --------------------------------------------------------------------------
# CLUSTER CONFIGURATION
# --------------------------------------------------------------------------
variable "cluster" {
  description = "Talos cluster configuration"
  type = object({
    name               = string
    endpoint           = string
    vip                = string
    talos_version      = string
    talos_image        = string
    kubernetes_version = string
  })

  validation {
    condition     = can(regex("^https://.*:6443$", var.cluster.endpoint))
    error_message = "Cluster endpoint must be a valid Kubernetes API URL (https://...6443)"
  }

  validation {
    condition     = can(regex("^v\\d+\\.\\d+\\.\\d+$", var.cluster.talos_version))
    error_message = "Talos version must be in format vX.Y.Z"
  }

  validation {
    condition     = can(regex("^\\d{1,3}\\.\\d{1,3}\\.\\d{1,3}\\.\\d{1,3}$", var.cluster.vip))
    error_message = "Cluster VIP must be a valid IPv4 address"
  }
}

# --------------------------------------------------------------------------
# NODES CONFIGURATION
# --------------------------------------------------------------------------
variable "control_plane_nodes" {
  description = "Control plane nodes configuration"
  type = map(object({
    name         = string
    ip_address   = string
    mac_address  = string
    install_disk = string
    nameservers  = optional(list(string), [])
    network = object({
      interface = string
      vlans = list(object({
        vlanId    = number
        addresses = list(string)
        gateway   = string
      }))
    })
  }))

  validation {
    condition     = length(var.control_plane_nodes) % 2 == 1
    error_message = "Control plane nodes must be an odd number (1, 3, 5) for etcd quorum"
  }
}

variable "worker_nodes" {
  description = "Worker nodes configuration (optional)"
  type = map(object({
    name         = string
    ip_address   = string
    mac_address  = string
    install_disk = string
    nameservers  = optional(list(string), [])
    network = object({
      interface = string
      vlans = list(object({
        vlanId    = number
        addresses = list(string)
        gateway   = string
      }))
    })
  }))
  default = {}
}

# --------------------------------------------------------------------------
# ARGOCD CONFIGURATION
# --------------------------------------------------------------------------
variable "argocd" {
  description = "ArgoCD configuration"
  type = object({
    service_type      = string
    loadbalancer_ip   = string
    hostname          = string
    insecure          = bool
    disable_auth      = bool
    anonymous_enabled = bool
  })

  validation {
    condition     = contains(["LoadBalancer", "ClusterIP", "NodePort"], var.argocd.service_type)
    error_message = "ArgoCD service type must be LoadBalancer, ClusterIP, or NodePort"
  }

  validation {
    condition     = can(regex("^\\d{1,3}\\.\\d{1,3}\\.\\d{1,3}\\.\\d{1,3}$", var.argocd.loadbalancer_ip))
    error_message = "ArgoCD LoadBalancer IP must be a valid IPv4 address"
  }
}

# --------------------------------------------------------------------------
# CILIUM L2 ANNOUNCEMENTS
# --------------------------------------------------------------------------
variable "cilium_l2" {
  description = "Cilium L2 Announcements configuration"
  type = object({
    pool_name     = string
    pool_ips      = list(string)
    policy_name   = string
    interfaces    = list(string)
    node_selector = map(string)
  })

  validation {
    condition     = length(var.cilium_l2.pool_ips) > 0
    error_message = "At least one IP pool range must be specified"
  }
}

# --------------------------------------------------------------------------
# NETWORK
# --------------------------------------------------------------------------
variable "network" {
  description = "Network configuration"
  type = object({
    vlan_services_subnet = string
  })

  validation {
    condition     = can(cidrhost(var.network.vlan_services_subnet, 0))
    error_message = "VLAN services subnet must be a valid CIDR notation"
  }
}

# --------------------------------------------------------------------------
# FILE PATHS
# --------------------------------------------------------------------------
variable "paths" {
  description = "File paths for generated configurations"
  type = object({
    kubeconfig            = string
    talosconfig           = string
    cilium_ip_pool_yaml   = string
    cilium_l2_policy_yaml = string
    infisical_secret      = string
  })

  default = {
    kubeconfig            = "./kubeconfig-prod"
    talosconfig           = "./talosconfig-prod"
    cilium_ip_pool_yaml   = "../../../apps/cilium-lb/overlays/prod/ippool.yaml"
    cilium_l2_policy_yaml = "../../../apps/cilium-lb/base/l2policy.yaml"
    infisical_secret      = "../../../.secrets/prod/infisical-universal-auth.yaml"
  }
}
