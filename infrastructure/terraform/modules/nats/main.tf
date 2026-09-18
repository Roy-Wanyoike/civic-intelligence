# modules/nats — NATS JetStream runs on EKS via Helm (see ../main.tf for EKS).
# This module provisions supporting AWS resources only:
#   * IAM role for the IRSA service account used by the NATS StatefulSet
#   * Persistent volume claims size hint (consumed by the Helm chart)
#   * CloudWatch log group

terraform {
  required_version = ">= 1.7.0"
}

variable "environment"           { type = string }
variable "project_name"          { type = string }
variable "jetstream_storage_gb" { type = number }
variable "eks_cluster_name"      { type = string }

locals {
  common_tags = {
    Project     = var.project_name
    Environment = var.environment
    ManagedBy   = "terraform"
    Module      = "nats"
  }
  full_name = "${var.project_name}-${var.environment}-nats"
}

# IRSA: EKS OIDC → IAM role for the NATS service account.
data "aws_iam_openid_connect_provider" "eks" {
  url = data.aws_eks_cluster.this.identity[0].oidc[0].issuer
}

data "aws_eks_cluster" "this" {
  name = var.eks_cluster_name
}

resource "aws_iam_role" "nats_irsa" {
  name = "${local.full_name}-irsa"
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect = "Allow"
      Principal = {
        Federated = data.aws_iam_openid_connect_provider.eks.arn
      }
      Action = "sts:AssumeRoleWithWebIdentity"
      Condition = {
        StringEquals = {
          "${replace(data.aws_iam_openid_connect_provider.eks.url, "https://", "")}:sub" = "system:serviceaccount:civic:${local.full_name}"
        }
      }
    }]
  })
  tags = local.common_tags
}

# Permissive S3 access for backup/restore of JetStream state.
# (Scope down to platform buckets in prod.)
resource "aws_iam_role_policy" "nats_irsa" {
  name = "${local.full_name}-irsa-policy"
  role = aws_iam_role.nats_irsa.id
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "s3:GetObject", "s3:PutObject", "s3:DeleteObject",
          "s3:ListBucket", "s3:GetBucketLocation"
        ]
        Resource = [
          "arn:aws:s3:::${var.project_name}-${var.environment}-*",
          "arn:aws:s3:::${var.project_name}-${var.environment}-*/*"
        ]
      }
    ]
  })
}

resource "aws_cloudwatch_log_group" "nats" {
  name              = "/civic/${var.environment}/nats"
  retention_in_days = 30
  tags              = local.common_tags
}

# -----------------------------------------------------------------------------

output "irsa_role_arn"          { value = aws_iam_role.nats_irsa.arn }
output "log_group_name"          { value = aws_cloudwatch_log_group.nats.name }
output "jetstream_storage_gb"   { value = var.jetstream_storage_gb }
output "service_account_name"   { value = local.full_name }
output "namespace"               { value = "civic" }
