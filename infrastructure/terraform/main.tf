terraform {
  required_version = ">= 1.5"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

provider "aws" {
  region = var.region
}

variable "region" {
  type    = string
  default = "eu-west-1"
}

variable "project_name" {
  type    = string
  default = "civic-intelligence"
}

variable "environment" {
  type    = string
  default = "dev"
}

# RDS PostgreSQL with pgvector
module "rds" {
  source = "./modules/rds"
  
  project_name = var.project_name
  environment  = var.environment
  db_name      = "civic_intelligence"
  db_username  = "civic"
  db_password  = var.db_password
}

# S3 bucket for document archive
module "storage" {
  source = "./modules/s3"
  
  project_name = var.project_name
  environment  = var.environment
}

variable "db_password" {
  type      = string
  sensitive = true
}

output "rds_endpoint" {
  value = module.rds.endpoint
}

output "s3_bucket" {
  value = module.storage.bucket_name
}
