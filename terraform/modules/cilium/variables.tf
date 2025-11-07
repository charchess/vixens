variable "release_name" {
  description = "The name of the Helm release for Cilium."
  type        = string
  default     = "cilium"
}

variable "chart_version" {
  description = "The version of the Cilium Helm chart to deploy."
  type        = string
  default     = "1.18.3"
}

variable "namespace" {
  description = "The Kubernetes namespace to deploy Cilium into."
  type        = string
  default     = "kube-system"
}

variable "talos_cluster_module" {
  description = "A reference to the Talos cluster module to ensure proper dependency."
  type        = any
  default     = null
}

variable "wait_for_k8s_api" {
  description = "A reference to the wait_for_k8s_api resource to ensure API is ready."
  type        = any
  default     = null
}

variable "kubeconfig_path" {
  description = "The path to the kubeconfig file for running the wait script."
  type        = string
}

variable "ip_pool_yaml_path" {
  description = "The path to the CiliumLoadBalancerIPPool YAML file."
  type        = string
}

variable "l2_policy_yaml_path" {
  description = "The path to the CiliumL2AnnouncementPolicy YAML file."
  type        = string
}
