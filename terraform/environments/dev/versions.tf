terraform {
  required_version = ">= 1.5.0"

  # S3 backend for remote state storage (Minio)
  backend "s3" {
    bucket = "terraform-state-dev"
    key    = "terraform.tfstate"
    region = "us-east-1" # Fake region for Minio compatibility

    # Minio endpoint configuration
    endpoint   = "http://synelia.internal.truxonline.com:9000"
    access_key = "terraform"
    secret_key = "terraform"

    # Minio-specific settings
    skip_credentials_validation = true
    skip_metadata_api_check     = true
    skip_region_validation      = true
    skip_requesting_account_id  = true # Skip AWS account ID check for Minio
    force_path_style            = true # Required for Minio
  }

  required_providers {
    talos = {
      source  = "siderolabs/talos"
      version = "~> 0.9"
    }
    local = {
      source  = "hashicorp/local"
      version = "~> 2.4"
    }
    helm = {
      source  = "hashicorp/helm"
      version = "~> 2.12"
    }
    kubectl = {
      source  = "gavinbunney/kubectl"
      version = "~> 1.19.0"
    }
  }
}
