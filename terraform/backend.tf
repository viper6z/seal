#state on s3
terraform {
  backend "s3" {
    bucket       = "homelab-tfstate-bucket-1337"
    key          = "homelab/main/terraform.tfstate"
    region       = "eu-north-1"
    profile      = "homelab-terraform"
    use_lockfile = true
  }
}

