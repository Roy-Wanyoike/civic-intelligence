terraform {
  required_version = ">= 1.7.0, < 2.0.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.50"
    }
    random = {
      source  = "hashicorp/random"
      version = "~> 3.6"
    }
    cloudflare = {
      source  = "cloudflare/cloudflare"
      version = "~> 4.40"
    }
  }

  # Backend: S3 + DynamoDB for state locking. Per-env:
  #   terraform init -backend-config=backend-dev.hcl
  backend "s3" {
    # Placeholder values overridden by env-specific backend config files.
    bucket       = "civic-terraform-state"
    key          = "civic-intelligence/terraform.tfstate"
    region       = "eu-west-1"
    encrypt      = true
    use_lockfile = true
  }
}

provider "aws" {
  region  = var.aws_region
  default_tags {
    tags = {
      Project     = "civic-intelligence"
      Environment = var.environment
      ManagedBy   = "terraform"
    }
  }
}

# Cloudflare provider is only active when var.cloudflare_zone_id != "".
provider "cloudflare" {
  api_token = var.cloudflare_api_token
}
