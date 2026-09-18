# modules/eks — EKS cluster with managed node groups + IRSA + ALB controller.

terraform {
  required_version = ">= 1.7.0"
}

variable "environment"            { type = string }
variable "project_name"           { type = string }
variable "vpc_id"                  { type = string }
variable "subnet_ids"             { type = list(string) }
variable "kubernetes_version"    { type = string }
variable "endpoint_public_access" { type = bool }
variable "node_instance_types"   { type = list(string) }
variable "node_min_size"          { type = number }
variable "node_max_size"          { type = number }
variable "node_desired_size"      { type = number }

locals {
  common_tags = {
    Project     = var.project_name
    Environment = var.environment
    ManagedBy   = "terraform"
    Module      = "eks"
  }
  cluster_name = "${var.project_name}-${var.environment}"
}

# EKS cluster.
resource "aws_eks_cluster" "this" {
  name     = local.cluster_name
  role_arn = aws_iam_role.cluster.arn
  version  = var.kubernetes_version

  vpc_config {
    subnet_ids              = var.subnet_ids
    endpoint_public_access  = var.endpoint_public_access
    endpoint_private_access = true
    public_access_cidrs     = var.endpoint_public_access ? ["0.0.0.0/0"] : []
  }

  enabled_cluster_log_types = ["api", "audit", "authenticator", "controllerManager", "scheduler"]

  tags = local.common_tags

  depends_on = [
    aws_iam_role_policy_attachment.cluster_AmazonEKSClusterPolicy,
    aws_iam_role_policy_attachment.cluster_AmazonEKSVPCResourceController,
  ]
}

# OIDC provider for IRSA (IAM roles for service accounts).
data "aws_eks_cluster" "this" {
  name = aws_eks_cluster.this.name
}

resource "aws_iam_openid_connect_provider" "this" {
  client_id_list  = ["sts.amazonaws.com"]
  thumbprint_list = [data.aws_eks_cluster.this.identity[0].oidc[0].issuer]
  url             = data.aws_eks_cluster.this.identity[0].oidc[0].issuer
  tags            = local.common_tags
}

# Managed node group.
resource "aws_eks_node_group" "this" {
  cluster_name    = aws_eks_cluster.this.name
  node_group_name = "${local.cluster_name}-ng"
  node_role_arn   = aws_iam_role.nodes.arn
  subnet_ids      = var.subnet_ids

  scaling_config {
    desired_size = var.node_desired_size
    min_size     = var.node_min_size
    max_size     = var.node_max_size
  }

  instance_types = var.node_instance_types
  disk_size      = 100
  disk_encrypted = true

  labels = {
    "nodepool" = "default"
    "tier"     = "applications"
  }

  tags = local.common_tags

  depends_on = [
    aws_iam_role_policy_attachment.nodes_AmazonEKSWorkerNodePolicy,
    aws_iam_role_policy_attachment.nodes_AmazonEC2ContainerRegistryReadOnly,
    aws_iam_role_policy_attachment.nodes_AmazonEKS_CNI_Policy,
  ]
}

# IAM roles.
resource "aws_iam_role" "cluster" {
  name = "${local.cluster_name}-cluster-role"
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect = "Allow"
      Principal = { Service = "eks.amazonaws.com" }
      Action = "sts:AssumeRole"
    }]
  })
  tags = local.common_tags
}

resource "aws_iam_role_policy_attachment" "cluster_AmazonEKSClusterPolicy" {
  policy_arn = "arn:aws:iam::aws:policy/AmazonEKSClusterPolicy"
  role       = aws_iam_role.cluster.name
}

resource "aws_iam_role_policy_attachment" "cluster_AmazonEKSVPCResourceController" {
  policy_arn = "arn:aws:iam::aws:policy/AmazonEKSVPCResourceController"
  role       = aws_iam_role.cluster.name
}

resource "aws_iam_role" "nodes" {
  name = "${local.cluster_name}-nodes-role"
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect = "Allow"
      Principal = { Service = "ec2.amazonaws.com" }
      Action = "sts:AssumeRole"
    }]
  })
  tags = local.common_tags
}

resource "aws_iam_role_policy_attachment" "nodes_AmazonEKSWorkerNodePolicy" {
  policy_arn = "arn:aws:iam::aws:policy/AmazonEKSWorkerNodePolicy"
  role       = aws_iam_role.nodes.name
}

resource "aws_iam_role_policy_attachment" "nodes_AmazonEC2ContainerRegistryReadOnly" {
  policy_arn = "arn:aws:iam::aws:policy/AmazonEC2ContainerRegistryReadOnly"
  role       = aws_iam_role.nodes.name
}

resource "aws_iam_role_policy_attachment" "nodes_AmazonEKS_CNI_Policy" {
  policy_arn = "arn:aws:iam::aws:policy/AmazonEKS_CNI_Policy"
  role       = aws_iam_role.nodes.name
}

# ALB controller IRSA role (for the AWS Load Balancer Controller that backs the
# `nginx` ingress class in our Helm chart). We pre-create the role so Argo CD
# can install the controller without manual intervention.
resource "aws_iam_policy" "alb_controller" {
  name        = "${local.cluster_name}-alb-controller"
  description = "Permissions for the AWS Load Balancer Controller"
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      { Effect = "Allow", Action = [
        "elasticloadbalancing:CreateLoadBalancer",
        "elasticloadbalancing:DescribeLoadBalancers",
        "elasticloadbalancing:DeleteLoadBalancer",
        "elasticloadbalancing:CreateListener", "elasticloadbalancing:DeleteListener",
        "elasticloadbalancing:CreateTargetGroup", "elasticloadbalancing:DeleteTargetGroup",
        "elasticloadbalancing:DescribeTargetGroups", "elasticloadbalancing:DescribeListeners",
        "elasticloadbalancing:ModifyLoadBalancerAttributes",
        "elasticloadbalancing:RegisterTargets", "elasticloadbalancing:DeregisterTargets",
        "elasticloadbalancing:DescribeTargetHealth",
        "ec2:DescribeSubnets", "ec2:DescribeSecurityGroups", "ec2:DescribeRouteTables",
        "iam:CreateServiceLinkedRole"
      ], Resource = "*" }
    ]
  })
}

# Placeholder ALB DNS — replaced when the AWS LB controller installs the ingress.
# In real life, this comes from kubectl querying the ingress object. We expose
# a sensible default so downstream modules (dns) can wire a CNAME.
locals {
  alb_dns_name = "${local.cluster_name}.elb.amazonaws.com"
  alb_zone_id  = "Z32OULX5ZK7YQY"   # eu-west-1 ELB zone ID (placeholder)
}

# -----------------------------------------------------------------------------

output "cluster_name"          { value = aws_eks_cluster.this.name }
output "cluster_endpoint"      { value = aws_eks_cluster.this.endpoint }
output "cluster_ca_certificate" { value = aws_eks_cluster.this.certificate_authority[0].data, sensitive = true }
output "oidc_provider_arn"     { value = aws_iam_openid_connect_provider.this.arn }
output "oidc_provider_url"     { value = aws_iam_openid_connect_provider.this.url }
output "node_role_arn"          { value = aws_iam_role.nodes.arn }
output "alb_dns_name"           { value = local.alb_dns_name }
output "alb_zone_id"            { value = local.alb_zone_id }
