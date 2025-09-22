# Look up versioning configuration for the foo bucket
data "storagegrid_s3_bucket_versioning" "foo" {
  bucket_name = "foo-bucket"
}

# Look up versioning configuration for the bar bucket
data "storagegrid_s3_bucket_versioning" "bar" {
  bucket_name = "bar-bucket"
}

# Use versioning data to make decisions
locals {
  foo_versioning_status = data.storagegrid_s3_bucket_versioning.foo.versioning_enabled ? "enabled" : (
    data.storagegrid_s3_bucket_versioning.foo.versioning_suspended ? "suspended" : "disabled"
  )
}

# Conditionally create lifecycle rules based on versioning status
resource "storagegrid_s3_bucket_lifecycle_configuration" "foo" {
  count       = data.storagegrid_s3_bucket_versioning.foo.versioning_enabled ? 1 : 0
  bucket_name = "foo-bucket"

  rule {
    id     = "foo-version-cleanup"
    status = "Enabled"

    noncurrent_version_expiration {
      noncurrent_days = 30
    }
  }
}

# Output versioning information
output "foo_versioning_enabled" {
  value = data.storagegrid_s3_bucket_versioning.foo.versioning_enabled
}

output "foo_versioning_suspended" {
  value = data.storagegrid_s3_bucket_versioning.foo.versioning_suspended
}

output "foo_versioning_status" {
  value = local.foo_versioning_status
}

output "bar_versioning_id" {
  value = data.storagegrid_s3_bucket_versioning.bar.id
}