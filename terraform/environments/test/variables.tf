# Variables for dev environment

# ArgoCD Configuration
variable "argocd_service_type" {
  description = "ArgoCD server service type (ClusterIP, LoadBalancer)"
  type        = string
  default     = "ClusterIP"

  validation {
    condition     = contains(["ClusterIP", "LoadBalancer"], var.argocd_service_type)
    error_message = "Service type must be either ClusterIP or LoadBalancer."
  }
}

variable "argocd_loadbalancer_ip" {
  description = "ArgoCD server LoadBalancer IP (used when service_type is LoadBalancer)"
  type        = string
  default     = "192.168.208.71"
}

variable "argocd_hostname" {
  description = "ArgoCD server hostname for Ingress (Sprint 6+)"
  type        = string
  default     = "argocd.test.truxonline.com"
}

variable "argocd_insecure" {
  description = "Run ArgoCD in insecure mode (HTTP, no TLS) - dev/test: true, staging/prod: false"
  type        = bool
  default     = true
}

variable "argocd_anonymous_enabled" {
  description = "Enable anonymous access to ArgoCD (no login required) - dev: true, test/staging/prod: false"
  type        = bool
  default     = true
}

variable "argocd_disable_auth" {
  description = "Disable authentication for ArgoCD server (WARNING: INSECURE!)"
  type        = bool
  default     = false # Default to false for security
}

variable "environment" {
  description = "Environment name (dev, test, staging, prod)"
  type        = string
  default     = "dev"

  validation {
    condition     = contains(["dev", "test", "staging", "prod"], var.environment)
    error_message = "Environment must be one of: dev, test, staging, prod."
  }
}

variable "vlan_services_subnet" {
  description = "VLAN services subnet (208 for dev, 209 for test, etc.)"
  type        = string
  default     = "192.168.208.0/24"
}

# Git Configuration
variable "git_branch" {
  description = "Git branch for ArgoCD to track (dev, test, staging, main)"
  type        = string
  default     = "dev"

  validation {
    condition     = contains(["dev", "test", "staging", "main"], var.git_branch)
    error_message = "Git branch must be one of: dev, test, staging, main."
  }
}