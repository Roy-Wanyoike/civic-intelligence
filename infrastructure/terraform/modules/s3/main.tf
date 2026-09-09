# modules/s3 — S3-compatible object storage.
# Supports AWS S3 (default) or Cloudflare R2 (alternative, via S3-compatible API).

terraform {
  required_version = ">= 1.7.0"
}

variable "environment"     { type = string }
variable "project_name"    { type = string }
variable "storage_backend" { type = string }
variable "r2_account_id"    { type = string, default = "" }
variable "buckets" {
  description = "Map of bucket logical name → settings (versioning, lifecycle_days)."
  type = map(object({
    versioning    = bool
    lifecycle_days = number
  }))
}

locals {
  common_tags = {
    Project     = var.project_name
    Environment = var.environment
    ManagedBy   = "terraform"
    Module      = "s3"
  }
  is_r2 = var.storage_backend == "r2"
}

# -----------------------------------------------------------------------------
# AWS S3 path (default)
# -----------------------------------------------------------------------------
resource "aws_kms_key" "this" {
  count                       = local.is_r2 ? 0 : 1
  description                 = "KMS key for ${var.project_name}-${var.environment} S3 buckets"
  deletion_window_in_days     = 30
  enable_key_rotation         = true
  tags                        = local.common_tags
}

resource "aws_kms_alias" "this" {
  count         = local.is_r2 ? 0 : 1
  name          = "alias/${var.project_name}-${var.environment}-s3"
  target_key_id = aws_kms_key.this[0].key_id
}

resource "aws_s3_bucket" "this" {
  for_each = local.is_r2 ? {} : var.buckets
  bucket  = "${var.project_name}-${var.environment}-${each.key}"
  tags    = merge(local.common_tags, { Name = each.key })

  lifecycle {
    prevent_destroy = var.environment == "prod"
  }
}

resource "aws_s3_bucket_versioning" "this" {
  for_each = local.is_r2 ? {} : var.buckets
  bucket   = aws_s3_bucket.this[each.key].id
  versioning_configuration {
    status = each.value.versioning ? "Enabled" : "Suspended"
  }
}

resource "aws_s3_bucket_server_side_encryption_configuration" "this" {
  for_each = local.is_r2 ? {} : var.buckets
  bucket   = aws_s3_bucket.this[each.key].id
  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm     = "aws:kms"
      kms_master_key_id = aws_kms_key.this[0].arn
    }
    bucket_key_enabled = true
  }
}

resource "aws_s3_bucket_public_access_block" "this" {
  for_each                = local.is_r2 ? {} : var.buckets
  bucket                  = aws_s3_bucket.this[each.key].id
  block_public_acls        = true
  block_public_policy      = true
  ignore_public_acls       = true
  restrict_public_buckets = true
}

resource "aws_s3_bucket_lifecycle_configuration" "this" {
  for_each = local.is_r2 ? {} : var.buckets
  bucket   = aws_s3_bucket.this[each.key].id

  rule {
    id     = "transition-to-intelligent-tiering"
    status = "Enabled"
    transition {
      days          = max(each.value.lifecycle_days - 30, 0)
      storage_class = "INTELLIGENT_TIERING"
    }
    expiration {
      days = each.value.lifecycle_days
    }
    noncurrent_version_expiration {
      noncurrent_days = 30
    }
  }
}

# -----------------------------------------------------------------------------
# Cloudflare R2 path (alternative). R2 is S3-compatible; buckets are managed
# via the Cloudflare provider. Application code reads S3_ENDPOINT env var.
# -----------------------------------------------------------------------------
resource "cloudflare_r2_bucket" "this" {
  for_each     = local.is_r2 ? var.buckets : {}
  account_id   = var.r2_account_id
  name         = "${var.project_name}-${var.environment}-${each.key}"
  location     = "APAC"
}

# -----------------------------------------------------------------------------

output "bucket_arns" {
  description = "ARNs (AWS) or IDs (R2) of created buckets."
  value = local.is_r2 ? {
    for k, b in cloudflare_r2_bucket.this : k => b.id
  } : {
    for k, b in aws_s3_bucket.this : k => b.arn
  }
}

output "bucket_names" {
  description = "Logical → bucket name mapping."
  value = local.is_r2 ? {
    for k, b in cloudflare_r2_bucket.this : k => b.name
  } : {
    for k, b in aws_s3_bucket.this : k => b.id
  }
}
