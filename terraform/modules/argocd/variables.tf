# ============================================================================
# ARGOCD MODULE VARIABLES
# ============================================================================

variable "chart_version" {
  description = "ArgoCD Helm chart version"
  type        = string
}

variable "namespace" {
  description = "Kubernetes namespace for ArgoCD"
  type        = string
  default     = "argocd"
}

variable "environment" {
  description = "Environment name (dev, test, staging, prod)"
  type        = string
}

variable "git_branch" {
  description = "Git branch for ArgoCD to track"
  type        = string
}

# --------------------------------------------------------------------------
# ARGOCD CONFIGURATION (typed object)
# --------------------------------------------------------------------------
variable "argocd_config" {
  description = "ArgoCD configuration object"
  type = object({
    service_type      = string
    loadbalancer_ip   = string
    hostname          = string
    insecure          = bool
    disable_auth      = bool
    anonymous_enabled = bool
  })
}

# --------------------------------------------------------------------------
# DRY CONFIGURATION
# --------------------------------------------------------------------------
variable "control_plane_tolerations" {
  description = "Control plane tolerations (from shared module)"
  type = list(object({
    key      = string
    operator = string
    effect   = string
  }))
}

variable "timeout" {
  description = "Helm installation timeout (seconds)"
  type        = number
  default     = 600
}

# --------------------------------------------------------------------------
# DEPENDENCIES
# --------------------------------------------------------------------------
variable "cilium_module" {
  description = "Cilium module reference for dependency"
  type        = any
  default     = null
}

variable "root_app_template_path" {
  description = "Path to root-app.yaml.tpl template"
  type        = string
}

# --------------------------------------------------------------------------
# INFISICAL BOOTSTRAP SECRET
# --------------------------------------------------------------------------
variable "infisical_secret_path" {
  description = "Path to Infisical universal auth secret YAML file (e.g., .secrets/dev/infisical-universal-auth.yaml)"
  type        = string
  default     = ""
}
