# Global variables for the Civic Intelligence Platform Terraform stack.
# Per-environment overrides live in environments/<env>/terraform.tfvars.

# -----------------------------------------------------------------------------
# Top-level
# -----------------------------------------------------------------------------
variable "aws_region" {
  description = "AWS region to deploy into."
  type        = string
  default     = "eu-west-1"
}

variable "environment" {
  description = "Deployment environment (dev|staging|prod)."
  type        = string
  validation {
    condition     = contains(["dev", "staging", "prod"], var.environment)
    error_message = "environment must be one of: dev, staging, prod."
  }
}

variable "project_name" {
  description = "Project prefix for resource naming."
  type        = string
  default     = "civic"
}

# -----------------------------------------------------------------------------
# VPC
# -----------------------------------------------------------------------------
variable "vpc_cidr" {
  description = "VPC CIDR block."
  type        = string
  default     = "10.30.0.0/16"
}

variable "vpc_az_count" {
  description = "Number of AZs to spread workloads across (min 3 for prod)."
  type        = number
  default     = 3
}

variable "enable_vpc_flow_logs" {
  description = "Enable VPC flow logs to CloudWatch."
  type        = bool
  default     = true
}

# -----------------------------------------------------------------------------
# RDS (Postgres + pgvector)
# -----------------------------------------------------------------------------
variable "rds_instance_class" {
  description = "RDS instance class."
  type        = string
  default     = "db.r6g.large"
}

variable "rds_storage_gb" {
  description = "Allocated storage in GiB."
  type        = number
  default     = 200
}

variable "rds_multi_az" {
  description = "Multi-AZ Postgres cluster."
  type        = bool
  default     = true
}

variable "rds_deletion_protection" {
  description = "Prevent accidental RDS deletion."
  type        = bool
  default     = true
}

variable "rds_backup_retention_days" {
  description = "Automated backup retention."
  type        = number
  default     = 30
}

variable "rds_pgvector_version" {
  description = "Postgres major version compatible with pgvector."
  type        = string
  default     = "16.3"
}

# -----------------------------------------------------------------------------
# S3-compatible object storage
# -----------------------------------------------------------------------------
variable "storage_backend" {
  description = "Object storage backend: 's3' (AWS) or 'r2' (Cloudflare R2)."
  type        = string
  default     = "s3"
  validation {
    condition     = contains(["s3", "r2"], var.storage_backend)
    error_message = "storage_backend must be 's3' or 'r2'."
  }
}

variable "r2_account_id" {
  description = "Cloudflare account ID (used only if storage_backend=r2)."
  type        = string
  default     = ""
}

# -----------------------------------------------------------------------------
# ElastiCache Redis
# -----------------------------------------------------------------------------
variable "redis_node_type" {
  description = "ElastiCache node type."
  type        = string
  default     = "cache.r6g.large"
}

variable "redis_cluster_size" {
  description = "Number of cache nodes (1 = standalone, 2+ = cluster mode)."
  type        = number
  default     = 3
}

variable "redis_at_rest_encryption" {
  description = "Encrypt Redis at rest."
  type        = bool
  default     = true
}

variable "redis_transit_encryption" {
  description = "Encrypt Redis in transit (TLS)."
  type        = bool
  default     = true
}

# -----------------------------------------------------------------------------
# NATS (currently self-hosted on EKS via Helm; this var tunes the ASG size)
# -----------------------------------------------------------------------------
variable "nats_jetstream_storage_gb" {
  description = "Persistent volume size for NATS JetStream file store."
  type        = number
  default     = 50
}

# -----------------------------------------------------------------------------
# EKS
# -----------------------------------------------------------------------------
variable "eks_kubernetes_version" {
  description = "EKS Kubernetes version."
  type        = string
  default     = "1.30"
}

variable "eks_endpoint_public_access" {
  description = "Public API endpoint for EKS. False in prod."
  type        = bool
  default     = false
}

variable "eks_node_instance_types" {
  description = "EC2 instance types for the worker node group."
  type        = list(string)
  default     = ["m6i.large", "m6i.xlarge"]
}

variable "eks_node_min_size" {
  description = "Minimum worker nodes."
  type        = number
  default     = 3
}

variable "eks_node_max_size" {
  description = "Maximum worker nodes (autoscaler upper bound)."
  type        = number
  default     = 30
}

variable "eks_node_desired_size" {
  description = "Desired worker node count."
  type        = number
  default     = 3
}

# -----------------------------------------------------------------------------
# DNS
# -----------------------------------------------------------------------------
variable "cloudflare_zone_id" {
  description = "Cloudflare zone ID for DNS records. Empty = DNS module disabled."
  type        = string
  default     = ""
}

variable "cloudflare_api_token" {
  description = "Cloudflare API token (sensitive). Empty = DNS module disabled."
  type        = string
  default    = ""
  sensitive  = true
}

variable "dns_domain" {
  description = "Root domain for the platform."
  type        = string
  default     = "civic.example"
}

variable "dns_subdomains" {
  description = "Subdomains to create A records for."
  type        = map(string)
  default = {
    "api"   = "api"
    "app"   = "web"
    "ops"   = "grafana"
    "keycloak" = "keycloak"
  }
}
