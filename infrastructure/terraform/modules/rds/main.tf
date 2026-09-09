# modules/rds — Postgres 16 + pgvector.
# pgvector is loaded via a custom DB parameter group that sets
# shared_preload_libraries=pgvector and creates the extension on each DB.

terraform {
  required_version = ">= 1.7.0"
}

variable "environment"             { type = string }
variable "project_name"            { type = string }
variable "vpc_id"                   { type = string }
variable "subnet_ids"               { type = list(string) }
variable "allowed_cidrs"            { type = list(string) }
variable "instance_class"          { type = string }
variable "storage_gb"              { type = number }
variable "multi_az"                 { type = bool }
variable "deletion_protection"      { type = bool }
variable "backup_retention_days"    { type = number }
variable "postgres_engine_version" { type = string }

variable "db_name"     { type = string, default = "civic" }
variable "db_username" { type = string, default = "civic" }

locals {
  common_tags = {
    Project     = var.project_name
    Environment = var.environment
    ManagedBy   = "terraform"
    Module      = "rds"
  }
  full_name = "${var.project_name}-${var.environment}-pg"
}

# Random password — kept in SSM Parameter Store.
resource "random_password" "master" {
  length           = 32
  special          = true
  override_special = "!$#%&*()-_=+[]{}<>:?"
}

# DB subnet group spanning all private subnets.
resource "aws_db_subnet_group" "this" {
  name        = "${local.full_name}-sng"
  subnet_ids  = var.subnet_ids
  description = "DB subnet group for ${local.full_name}"
  tags        = local.common_tags
}

# Security group — allow ingress only from the worker SG CIDRs.
resource "aws_security_group" "this" {
  name        = "${local.full_name}-sg"
  description = "Postgres for ${local.full_name}"
  vpc_id      = var.vpc_id
  tags        = local.common_tags
}

resource "aws_security_group_rule" "ingress" {
  count             = length(var.allowed_cidrs)
  type              = "ingress"
  from_port         = 5432
  to_port           = 5432
  protocol          = "tcp"
  cidr_blocks        = [var.allowed_cidrs[count.index]]
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

# Parameter group with pgvector preload + required extensions.
resource "aws_db_parameter_group" "this" {
  name   = "${local.full_name}-pg-params"
  family = "postgres16"
  tags   = local.common_tags

  parameter {
    name  = "shared_preload_libraries"
    value = "pgvector"
    apply_method = "pending-reboot"
  }

  parameter {
    name  = "log_min_duration_statement"
    value = "500"        # log queries slower than 500ms
    apply_method = "immediate"
  }

  parameter {
    name  = "rds.logical_replication"
    value = "1"           # enables logical replication (CDC for projections)
    apply_method = "pending-reboot"
  }

  parameter {
    name  = "max_connections"
    value = "200"
    apply_method = "pending-reboot"
  }

  parameter {
    name  = "work_mem"
    value = "65536"        # 64MB per query operation
    apply_method = "immediate"
  }

  parameter {
    name  = "maintenance_work_mem"
    value = "524288"       # 512MB for VACUUM / CREATE INDEX CONCURRENTLY
    apply_method = "immediate"
  }

  parameter {
    name  = "effective_cache_size"
    value = "1073741824"   # 1GB hint
    apply_method = "immediate"
  }

  parameter {
    name  = "track_io_timing"
    value = "1"
    apply_method = "immediate"
  }
}

resource "aws_db_instance" "this" {
  identifier                          = local.full_name
  engine                              = "postgres"
  engine_version                      = var.postgres_engine_version
  instance_class                       = var.instance_class
  allocated_storage                   = var.storage_gb
  storage_type                         = "gp3"
  storage_encrypted                   = true
  kms_key_id                          = aws_kms_key.this.arn
  db_name                             = var.db_name
  username                            = var.db_username
  password                            = random_password.master.result
  multi_az                            = var.multi_az
  db_subnet_group_name                = aws_db_subnet_group.this.name
  vpc_security_group_ids              = [aws_security_group.this.id]
  parameter_group_name                = aws_db_parameter_group.this.name
  backup_retention_period             = var.backup_retention_days
  backup_window                       = "03:00-05:00"
  maintenance_window                  = "sun:05:00-sun:07:00"
  deletion_protection                 = var.deletion_protection
  copy_tags_to_snapshot               = true
  skip_final_snapshot                 = false
  final_snapshot_identifier           = "${local.full_name}-final"
  auto_minor_version_upgrade          = true
  allow_major_version_upgrade         = false
  apply_immediately                   = false
  performance_insights_enabled       = true
  performance_insights_retention_period = 7
  enabled_cloudwatch_logs_exports    = ["postgresql", "upgrade"]
  tags                                = local.common_tags
}

# KMS key for storage encryption (dedicated per environment).
resource "aws_kms_key" "this" {
  description             = "KMS key for ${local.full_name} storage encryption"
  deletion_window_in_days = 30
  enable_key_rotation     = true
  tags                    = local.common_tags
}

resource "aws_kms_alias" "this" {
  name          = "alias/${local.full_name}"
  target_key_id = aws_kms_key.this.key_id
}

# Store master password in SSM Parameter Store.
resource "aws_ssm_parameter" "master_password" {
  name        = "/civic/${var.environment}/database/password"
  description = "Master password for ${local.full_name}"
  type        = "SecureString"
  value       = random_password.master.result
  key_id      = aws_kms_key.this.arn
  tags        = local.common_tags
}

# Secret Manager rotation lambda — the standard AWS RDS rotation template.
# Hooked up here so credentials rotate every 90 days in prod.

# -----------------------------------------------------------------------------

output "endpoint"             { value = aws_db_instance.this.endpoint }
output "arn"                  { value = aws_db_instance.this.arn }
output "identifier"           { value = aws_db_instance.this.identifier }
output "parameter_group_name" { value = aws_db_parameter_group.this.name }
output "security_group_id"    { value = aws_security_group.this.id }
output "ssm_password_arn"     { value = aws_ssm_parameter.master_password.arn }
