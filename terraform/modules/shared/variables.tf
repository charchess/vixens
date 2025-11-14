variable "environment" {
  description = "Environment name (dev, test, staging, prod)"
  type        = string

  validation {
    condition     = contains(["dev", "test", "staging", "prod"], var.environment)
    error_message = "Environment must be one of: dev, test, staging, prod"
  }
}

variable "loadbalancer_ip" {
  description = "LoadBalancer IP for ArgoCD service (optional)"
  type        = string
  default     = ""

  validation {
    condition     = var.loadbalancer_ip == "" || can(regex("^\\d{1,3}\\.\\d{1,3}\\.\\d{1,3}\\.\\d{1,3}$", var.loadbalancer_ip))
    error_message = "LoadBalancer IP must be a valid IPv4 address or empty string"
  }
}
