# modules/redis — ElastiCache Redis (cluster mode) with TLS + at-rest encryption.

terraform {
  required_version = ">= 1.7.0"
}

variable "environment"        { type = string }
variable "project_name"       { type = string }
variable "vpc_id"              { type = string }
variable "subnet_ids"         { type = list(string) }
variable "node_type"           { type = string }
variable "cluster_size"       { type = number }
variable "at_rest_encryption"  { type = bool }
variable "transit_encryption" { type = bool }

locals {
  common_tags = {
    Project     = var.project_name
    Environment = var.environment
    ManagedBy   = "terraform"
    Module      = "redis"
  }
  full_name = "${var.project_name}-${var.environment}-redis"
}

resource "random_password" "auth_token" {
  length           = 48
  special          = true
  override_special = "!$#%&*()-_=+"
}

resource "aws_elasticache_subnet_group" "this" {
  name        = "${local.full_name}-sng"
  subnet_ids  = var.subnet_ids
  description = "ElastiCache subnet group for ${local.full_name}"
}

resource "aws_security_group" "this" {
  name        = "${local.full_name}-sg"
  description = "Redis for ${local.full_name}"
  vpc_id      = var.vpc_id
  tags        = local.common_tags
}

# Ingress: 6379 from the VPC. Tighten further in prod by passing worker SG.
resource "aws_security_group_rule" "ingress" {
  type              = "ingress"
  from_port         = var.transit_encryption ? 6379 : 6379
  to_port           = var.transit_encryption ? 6379 : 6379
  protocol          = "tcp"
  cidr_blocks        = [data.aws_vpc.this.cidr_block]
  security_group_id = aws_security_group.this.id
}

resource "aws_security_group_rule" "egress" {
  type              = "egress"
  from_port         = 0
  to_port           = 0
  protocol          = "-1"
  cidr_blocks        = ["0.0.0.0/0"]
  security_group_id = aws_security_group.this.id
}

data "aws_vpc" "this" {
  id = var.vpc_id
}

resource "aws_elasticache_replication_group" "this" {
  replication_group_id          = local.full_name
  description                    = "Redis cluster for ${local.full_name}"
  node_type                      = var.node_type
  num_cache_clusters             = var.cluster_size
  engine                         = "redis"
  engine_version                 = "7.1"
  parameter_group_name          = "default.redis7.cluster.on"
  subnet_group_name              = aws_elasticache_subnet_group.this.name
  security_group_ids            = [aws_security_group.this.id]
  at_rest_encryption_enabled    = var.at_rest_encryption
  transit_encryption_enabled    = var.transit_encryption
  auth_token                     = var.transit_encryption ? random_password.auth_token.result : null
  multi_az_enabled              = var.cluster_size >= 2
  automatic_failover_enabled    = var.cluster_size >= 2
  snapshot_retention_limit       = 7
  snapshot_window                = "03:00-05:00"
  maintenance_window             = "sun:05:00-sun:07:00"
  snapshot_arns                  = []
  final_snapshot_identifier      = "${local.full_name}-final"
  tags                           = local.common_tags
}

resource "aws_ssm_parameter" "auth_token" {
  name        = "/civic/${var.environment}/redis/auth_token"
  description = "AUTH token for ${local.full_name} (only set when TLS enabled)"
  type        = "SecureString"
  value       = var.transit_encryption ? random_password.auth_token.result : "disabled"
  tags        = local.common_tags
}

# -----------------------------------------------------------------------------

output "primary_endpoint"   { value = aws_elasticache_replication_group.this.primary_endpoint_address }
output "reader_endpoint"     { value = aws_elasticache_replication_group.this.reader_endpoint_address }
output "security_group_id"   { value = aws_security_group.this.id }
output "auth_token_ssm_arn"  { value = aws_ssm_parameter.auth_token.arn }
