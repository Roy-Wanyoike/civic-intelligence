# modules/vpc — civic platform VPC with public/private/db subnets across N AZs.

terraform {
  required_version = ">= 1.7.0"
}

variable "environment" { type = string }
variable "project_name" { type = string }
variable "cidr" { type = string }
variable "az_count" { type = number }
variable "enable_flow_logs" { type = bool }

locals {
  azs = slice(data.aws_availability_zones.available.names, 0, min(var.az_count, 3))
  # /16 → split into /20 public, /19 private, /20 db, /20 spare
  public_cidrs   = [for k in range(length(local.azs)) : cidrsubnet(var.cidr, 4, k)]
  private_cidrs  = [for k in range(length(local.azs)) : cidrsubnet(var.cidr, 4, k + 4)]
  db_cidrs       = [for k in range(length(local.azs)) : cidrsubnet(var.cidr, 4, k + 8)]
  common_tags = {
    Project     = var.project_name
    Environment = var.environment
    ManagedBy   = "terraform"
    Module      = "vpc"
  }
}

data "aws_availability_zones" "available" {
  state = "available"
}

resource "aws_vpc" "this" {
  cidr_block           = var.cidr
  enable_dns_support   = true
  enable_dns_hostnames = true
  tags = merge(local.common_tags, {
    Name = "${var.project_name}-${var.environment}-vpc"
  })
}

resource "aws_internet_gateway" "this" {
  vpc_id = aws_vpc.this.id
  tags   = merge(local.common_tags, { Name = "${var.project_name}-${var.environment}-igw" })
}

# Elastic IPs for the NAT gateways (one per AZ with a public subnet).
resource "aws_eip" "nat" {
  count  = length(local.azs)
  domain = "vpc"
  tags   = merge(local.common_tags, { Name = "${var.project_name}-${var.environment}-nat-${local.azs[count.index]}" })
}

resource "aws_nat_gateway" "this" {
  count         = length(local.azs)
  allocation_id = aws_eip.nat[count.index].id
  subnet_id     = aws_subnet.public[count.index].id
  tags          = merge(local.common_tags, { Name = "${var.project_name}-${var.environment}-nat-${local.azs[count.index]}" })
  depends_on    = [aws_internet_gateway.this]
}

resource "aws_subnet" "public" {
  count                   = length(local.azs)
  vpc_id                  = aws_vpc.this.id
  cidr_block              = local.public_cidrs[count.index]
  availability_zone       = local.azs[count.index]
  map_public_ip_on_launch = true
  tags = merge(local.common_tags, {
    Name = "${var.project_name}-${var.environment}-public-${local.azs[count.index]}"
    Tier = "public"
    "kubernetes.io/role/elb" = "1"
  })
}

resource "aws_subnet" "private" {
  count             = length(local.azs)
  vpc_id            = aws_vpc.this.id
  cidr_block         = local.private_cidrs[count.index]
  availability_zone = local.azs[count.index]
  tags = merge(local.common_tags, {
    Name = "${var.project_name}-${var.environment}-private-${local.azs[count.index]}"
    Tier = "private"
    "kubernetes.io/role/internal-elb" = "1"
  })
}

resource "aws_subnet" "db" {
  count             = length(local.azs)
  vpc_id            = aws_vpc.this.id
  cidr_block         = local.db_cidrs[count.index]
  availability_zone = local.azs[count.index]
  tags = merge(local.common_tags, {
    Name = "${var.project_name}-${var.environment}-db-${local.azs[count.index]}"
    Tier = "database"
  })
}

resource "aws_route_table" "public" {
  vpc_id = aws_vpc.this.id
  route {
    cidr_block = "0.0.0.0/0"
    gateway_id = aws_internet_gateway.this.id
  }
  tags = merge(local.common_tags, { Name = "${var.project_name}-${var.environment}-rt-public" })
}

resource "aws_route_table_association" "public" {
  count          = length(local.azs)
  subnet_id      = aws_subnet.public[count.index].id
  route_table_id = aws_route_table.public.id
}

resource "aws_route_table" "private" {
  count  = length(local.azs)
  vpc_id = aws_vpc.this.id
  route {
    cidr_block     = "0.0.0.0/0"
    nat_gateway_id = aws_nat_gateway.this[count.index].id
  }
  tags = merge(local.common_tags, { Name = "${var.project_name}-${var.environment}-rt-private-${local.azs[count.index]}" })
}

resource "aws_route_table_association" "private" {
  count          = length(local.azs)
  subnet_id      = aws_subnet.private[count.index].id
  route_table_id = aws_route_table.private[count.index].id
}

resource "aws_route_table_association" "db" {
  count          = length(local.azs)
  subnet_id      = aws_subnet.db[count.index].id
  route_table_id = aws_route_table.private[count.index].id
}

# Flow logs (s3 destination).
resource "aws_flow_log" "this" {
  count             = var.enable_flow_logs ? 1 : 0
  log_destination   = "cloud-watch-logs"
  log_destination_type = "cloud-watch-logs"
  traffic_type      = "ALL"
  vpc_id             = aws_vpc.this.id
  tags               = merge(local.common_tags, { Name = "${var.project_name}-${var.environment}-flowlog" })
}

# Security group for the EKS worker nodes (used by EKS module too).
resource "aws_security_group" "workers" {
  name        = "${var.project_name}-${var.environment}-workers-sg"
  description = "EKS worker nodes"
  vpc_id      = aws_vpc.this.id
  tags        = local.common_tags
}

resource "aws_security_group_rule" "workers_egress" {
  type              = "egress"
  from_port         = 0
  to_port           = 0
  protocol          = "-1"
  cidr_blocks        = ["0.0.0.0/0"]
  security_group_id = aws_security_group.workers.id
}

resource "aws_security_group_rule" "workers_internal" {
  type              = "ingress"
  from_port         = 0
  to_port           = 0
  protocol          = "-1"
  self              = true
  security_group_id = aws_security_group.workers.id
}

# -----------------------------------------------------------------------------

output "vpc_id"              { value = aws_vpc.this.id }
output "vpc_cidr_block"      { value = aws_vpc.this.cidr_block }
output "public_subnet_ids"   { value = aws_subnet.public[*].id }
output "private_subnet_ids"  { value = aws_subnet.private[*].id }
output "db_subnet_ids"       { value = aws_subnet.db[*].id }
output "availability_zones"  { value = local.azs }
output "worker_security_group_id" { value = aws_security_group.workers.id }
