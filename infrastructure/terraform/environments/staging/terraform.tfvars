aws_region    = "eu-west-1"
environment    = "staging"
project_name   = "civic"

vpc_cidr         = "10.31.0.0/16"
vpc_az_count     = 3
enable_vpc_flow_logs = true

rds_instance_class        = "db.r6g.large"
rds_storage_gb            = 100
rds_multi_az              = true
rds_deletion_protection   = true
rds_backup_retention_days = 7

storage_backend = "s3"

redis_node_type           = "cache.r6g.large"
redis_cluster_size        = 3
redis_at_rest_encryption  = true
redis_transit_encryption  = true

eks_kubernetes_version     = "1.30"
eks_endpoint_public_access = false
eks_node_instance_types    = ["m6i.large"]
eks_node_min_size          = 2
eks_node_max_size          = 10
eks_node_desired_size      = 3

cloudflare_zone_id = "<set-in-CI>"
cloudflare_api_token = "<set-in-CI>"
dns_domain          = "staging.civic.example"
