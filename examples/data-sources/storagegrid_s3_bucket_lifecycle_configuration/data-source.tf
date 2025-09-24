# Look up lifecycle configuration for the foo bucket
data "storagegrid_s3_bucket_lifecycle_configuration" "foo" {
  bucket_name = "foo-bucket"
}

# Look up lifecycle configuration for the bar bucket
data "storagegrid_s3_bucket_lifecycle_configuration" "bar" {
  bucket_name = "bar-bucket"
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
