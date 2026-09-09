# Civic Intelligence Platform — top-level Terraform stack.
# Composes the per-bounded-context modules into a complete environment.

terraform {
  required_version = ">= 1.7.0"
}

provider "aws" {
  region = var.aws_region
}

# -----------------------------------------------------------------------------

module "vpc" {
  source = "./modules/vpc"

  environment          = var.environment
  project_name         = var.project_name
  cidr                  = var.vpc_cidr
  az_count              = var.vpc_az_count
  enable_flow_logs      = var.enable_vpc_flow_logs
}

module "rds" {
  source = "./modules/rds"

  environment             = var.environment
  project_name            = var.project_name
  vpc_id                  = module.vpc.vpc_id
  subnet_ids              = module.vpc.private_subnet_ids
  allowed_cidrs           = [module.vpc.vpc_cidr]
  instance_class          = var.rds_instance_class
  storage_gb              = var.rds_storage_gb
  multi_az                = var.rds_multi_az
  deletion_protection     = var.rds_deletion_protection
  backup_retention_days   = var.rds_backup_retention_days
  postgres_engine_version = var.rds_pgvector_version

  depends_on = [module.vpc]
}

module "s3" {
  source = "./modules/s3"

  environment      = var.environment
  project_name     = var.project_name
  storage_backend  = var.storage_backend
  r2_account_id    = var.r2_account_id
  buckets = {
    raw-documents = { versioning = true, lifecycle_days = 365 }
    exports       = { versioning = true, lifecycle_days = 90 }
    evidence      = { versioning = true, lifecycle_days = 3650 }
    terraform-state = { versioning = true, lifecycle_days = 365 }
  }
}

module "redis" {
  source = "./modules/redis"

  environment         = var.environment
  project_name        = var.project_name
  vpc_id              = module.vpc.vpc_id
  subnet_ids          = module.vpc.private_subnet_ids
  node_type           = var.redis_node_type
  cluster_size        = var.redis_cluster_size
  at_rest_encryption  = var.redis_at_rest_encryption
  transit_encryption  = var.redis_transit_encryption
}

module "nats" {
  source = "./modules/nats"

  environment         = var.environment
  project_name        = var.project_name
  jetstream_storage_gb = var.nats_jetstream_storage_gb
  eks_cluster_name    = module.eks.cluster_name
}

module "eks" {
  source = "./modules/eks"

  environment              = var.environment
  project_name             = var.project_name
  vpc_id                   = module.vpc.vpc_id
  subnet_ids               = module.vpc.private_subnet_ids
  kubernetes_version       = var.eks_kubernetes_version
  endpoint_public_access   = var.eks_endpoint_public_access
  node_instance_types      = var.eks_node_instance_types
  node_min_size            = var.eks_node_min_size
  node_max_size            = var.eks_node_max_size
  node_desired_size        = var.eks_node_desired_size
}

module "dns" {
  count  = var.cloudflare_zone_id == "" ? 0 : 1
  source = "./modules/dns"

  zone_id        = var.cloudflare_zone_id
  domain         = var.dns_domain
  subdomains     = var.dns_subdomains
  api_lb_dns     = module.eks.alb_dns_name
  api_lb_zone_id = module.eks.alb_zone_id
}

# -----------------------------------------------------------------------------
# Outputs
# -----------------------------------------------------------------------------

output "rds_endpoint" {
  description = "RDS Postgres endpoint."
  value       = module.rds.endpoint
  sensitive   = true
}

output "redis_endpoint" {
  description = "ElastiCache Redis primary endpoint."
  value       = module.redis.primary_endpoint
  sensitive   = true
}

output "s3_bucket_arns" {
  description = "S3 bucket ARNs (or R2 bucket IDs)."
  value       = module.s3.bucket_arns
}

output "eks_cluster_name" {
  description = "EKS cluster name."
  value       = module.eks.cluster_name
}

output "eks_oidc_provider_arn" {
  description = "EKS OIDC provider ARN (for IRSA)."
  value       = module.eks.oidc_provider_arn
}

output "alb_dns_name" {
  description = "Public ALB DNS for ingress."
  value       = module.eks.alb_dns_name
}
