aws_region    = "eu-west-1"
environment    = "dev"
project_name   = "civic"

# VPC
vpc_cidr         = "10.30.0.0/16"
vpc_az_count     = 3
enable_vpc_flow_logs = false

# RDS — single-AZ, smallest, no deletion protection (dev only!)
rds_instance_class        = "db.t4g.medium"
rds_storage_gb            = 50
rds_multi_az              = false
rds_deletion_protection   = false
rds_backup_retention_days = 3

# S3 — dev can use R2 to save AWS cost
storage_backend = "s3"

# Redis
redis_node_type           = "cache.t4g.small"
redis_cluster_size        = 1
redis_at_rest_encryption  = false
redis_transit_encryption  = false

# EKS
eks_kubernetes_version     = "1.30"
eks_endpoint_public_access = true
eks_node_instance_types    = ["t3.medium"]
eks_node_min_size          = 1
eks_node_max_size          = 4
eks_node_desired_size      = 1

# DNS — disabled in dev
cloudflare_zone_id = ""
cloudflare_api_token = ""
dns_domain          = "civic.local"
