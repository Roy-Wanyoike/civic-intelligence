aws_region    = "eu-west-1"
environment    = "prod"
project_name   = "civic"

vpc_cidr         = "10.32.0.0/16"
vpc_az_count     = 3
enable_vpc_flow_logs = true

# Prod: HA, large instance, encrypted backups
rds_instance_class        = "db.r6g.2xlarge"
rds_storage_gb            = 500
rds_multi_az              = true
rds_deletion_protection   = true
rds_backup_retention_days = 30
rds_pgvector_version      = "16.3"

# Prod can switch to R2 for cheaper egress-heavy buckets (e.g. raw documents)
storage_backend = "s3"
r2_account_id    = ""

redis_node_type           = "cache.r6g.2xlarge"
redis_cluster_size        = 5
redis_at_rest_encryption  = true
redis_transit_encryption  = true

eks_kubernetes_version     = "1.30"
eks_endpoint_public_access = false
eks_node_instance_types    = ["m6i.xlarge", "m6i.2xlarge"]
eks_node_min_size          = 5
eks_node_max_size          = 40
eks_node_desired_size      = 8

cloudflare_zone_id = "<set-in-CI>"
cloudflare_api_token = "<set-in-CI>"
dns_domain          = "civic.example"
