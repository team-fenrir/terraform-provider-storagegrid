# Look up lifecycle configuration for the foo bucket
data "storagegrid_s3_bucket_lifecycle_configuration" "foo" {
  bucket_name = "foo-bucket"
}

# Look up lifecycle configuration for the bar bucket
data "storagegrid_s3_bucket_lifecycle_configuration" "bar" {
  bucket_name = "bar-bucket"
}

# Use lifecycle data to understand existing rules
locals {
  foo_has_expiration_rules = length([
    for rule in data.storagegrid_s3_bucket_lifecycle_configuration.foo.rule :
    rule if rule.expiration != null
  ]) > 0

  foo_max_expiration_days = max([
    for rule in data.storagegrid_s3_bucket_lifecycle_configuration.foo.rule :
    rule.expiration != null && rule.expiration.days != null ? rule.expiration.days : 0
  ]...)
}

# Create additional lifecycle rules based on existing configuration
resource "storagegrid_s3_bucket_lifecycle_configuration" "foo_additional" {
  count       = local.foo_has_expiration_rules ? 1 : 0
  bucket_name = "foo-additional-bucket"

  rule {
    id     = "foo-additional-cleanup"
    status = "Enabled"

    filter {
      prefix = "backup/"
    }

    # Set expiration based on existing rules
    expiration {
      days = local.foo_max_expiration_days * 2
    }
  }
}

# Output lifecycle information
output "foo_lifecycle_rules_count" {
  value = length(data.storagegrid_s3_bucket_lifecycle_configuration.foo.rule)
}

output "foo_lifecycle_rules" {
  value = [
    for rule in data.storagegrid_s3_bucket_lifecycle_configuration.foo.rule : {
      id              = rule.id
      status          = rule.status
      prefix          = rule.filter != null ? rule.filter.prefix : null
      expiration_days = rule.expiration != null ? rule.expiration.days : null
    }
  ]
}

output "bar_lifecycle_id" {
  value = data.storagegrid_s3_bucket_lifecycle_configuration.bar.id
}