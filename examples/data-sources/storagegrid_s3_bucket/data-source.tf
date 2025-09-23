# Look up an existing S3 bucket by name
data "storagegrid_s3_bucket" "foo" {
  bucket_name = "foo-bucket"
}

# Look up an S3 bucket and use its configuration
data "storagegrid_s3_bucket" "bar" {
  bucket_name = "bar-bucket"
}

# Use bucket data to configure lifecycle rules
resource "storagegrid_s3_bucket_lifecycle_configuration" "foo" {
  bucket_name = data.storagegrid_s3_bucket.foo.bucket_name

  rule {
    id     = "foo-cleanup"
    status = "Enabled"

    expiration {
      days = 30
    }
  }
}

# Conditionally create object lock configuration if bucket has object lock enabled
resource "storagegrid_s3_bucket_object_lock_configuration" "bar" {
  count       = data.storagegrid_s3_bucket.bar.object_lock_enabled ? 1 : 0
  bucket_name = data.storagegrid_s3_bucket.bar.bucket_name

  default_retention_setting {
    mode = "compliance"
    days = 7
  }
}

# Output bucket information
output "foo_bucket_region" {
  value = data.storagegrid_s3_bucket.foo.region
}

output "foo_bucket_creation_time" {
  value = data.storagegrid_s3_bucket.foo.creation_time
}

output "bar_bucket_object_lock_enabled" {
  value = data.storagegrid_s3_bucket.bar.object_lock_enabled
}