variable "chart_version" {
  description = "The version of the ArgoCD Helm chart to deploy."
  type        = string
  default     = "7.7.7"
}

variable "namespace" {
  description = "The Kubernetes namespace to deploy ArgoCD into."
  type        = string
  default     = "argocd"
}

variable "argocd_loadbalancer_ip" {
  description = "The IP address to assign to the ArgoCD server LoadBalancer."
  type        = string
}

variable "argocd_service_type" {
  description = "The type of Kubernetes service to use for the ArgoCD server."
  type        = string
  default     = "ClusterIP"
}

variable "environment" {
  description = "The name of the deployment environment."
  type        = string
}

variable "argocd_hostname" {
  description = "The hostname for the ArgoCD Ingress."
  type        = string
}

variable "argocd_insecure" {
  description = "If true, run the ArgoCD server in insecure (HTTP) mode."
  type        = bool
  default     = false
}

variable "argocd_anonymous_enabled" {
  description = "If true, enables anonymous user access to ArgoCD."
  type        = bool
  default     = false
}

variable "argocd_disable_auth" {
  description = "If true, disables authentication on the ArgoCD server. WARNING: INSECURE."
  type        = bool
  default     = false
}

variable "cilium_module" {
  description = "A reference to the Cilium module to ensure proper dependency."
  type        = any
  default     = null
}

variable "root_app_template_path" {
  description = "The path to the root-app.yaml.tpl file."
  type        = string
}

variable "git_branch" {
  description = "The Git branch for ArgoCD to track."
  type        = string
}
