# modules/dns — Cloudflare DNS records for the platform.
# Disabled when var.cloudflare_zone_id == "" (top-level main.tf gates the count).

terraform {
  required_version = ">= 1.7.0"
}

variable "zone_id"       { type = string }
variable "domain"         { type = string }
variable "subdomains"    { type = map(string) }
variable "api_lb_dns"     { type = string }
variable "api_lb_zone_id" { type = string }

locals {
  common_tags = {
    ManagedBy = "terraform"
    Module     = "dns"
  }
}

# Root apex → ALB.
resource "cloudflare_record" "apex" {
  zone_id = var.zone_id
  name    = "@"
  value   = var.api_lb_dns
  type    = "CNAME"
  proxied = true
  ttl     = 1
  comment = "Civic platform apex → EKS ALB"
}

# One subdomain per entry in var.subdomains.
resource "cloudflare_record" "subdomains" {
  for_each = var.subdomains
  zone_id  = var.zone_id
  name     = each.key
  value    = var.api_lb_dns
  type     = "CNAME"
  proxied  = true
  ttl      = 1
  comment  = "Civic ${each.key} → EKS ALB"
}

# TXT record for SPF / DKIM / verification tokens.
resource "cloudflare_record" "spf" {
  zone_id = var.zone_id
  name    = "@"
  type    = "TXT"
  value   = "\"v=spf1 include:amazonses.com -all\""
  ttl     = 300
  proxied = false
}

# MX for inbound email (SES inbound).
resource "cloudflare_record" "mx" {
  zone_id  = var.zone_id
  name     = "@"
  type     = "MX"
  value    = "inbound-smtp.eu-west-1.amazonaws.com"
  priority = 10
  ttl      = 300
  proxied  = false
}

# -----------------------------------------------------------------------------

output "records" {
  description = "Map of created DNS records."
  value = {
    apex      = cloudflare_record.apex.hostname
    subdomains = { for k, r in cloudflare_record.subdomains : k => r.hostname }
  }
}
