# Look up versioning configuration for the foo bucket
data "storagegrid_s3_bucket_versioning" "foo" {
  bucket_name = "foo-bucket"
}

# Look up versioning configuration for the bar bucket
data "storagegrid_s3_bucket_versioning" "bar" {
  bucket_name = "bar-bucket"
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
