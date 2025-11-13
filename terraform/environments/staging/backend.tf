terraform {
  backend "s3" {
    bucket = "terraform-state-staging"
    key    = "terraform.tfstate"
    region = "us-east-1"

    endpoint   = "http://synelia.internal.truxonline.com:9000"
    access_key = "terraform"
    secret_key = "terraform"

    skip_credentials_validation = true
    skip_metadata_api_check     = true
    skip_region_validation      = true
    skip_requesting_account_id  = true
    force_path_style            = true
  }
}
