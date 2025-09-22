# Look up object lock configuration for the foo bucket
data "storagegrid_s3_bucket_object_lock_configuration" "foo" {
  bucket_name = "foo-bucket"
}

# Look up object lock configuration for the bar bucket
data "storagegrid_s3_bucket_object_lock_configuration" "bar" {
  bucket_name = "bar-bucket"
}

# Use object lock data to create complementary lifecycle rules
resource "storagegrid_s3_bucket_lifecycle_configuration" "foo" {
  bucket_name = "foo-bucket"

  rule {
    id     = "foo-object-cleanup"
    status = "Enabled"

    filter {
      prefix = "temp/"
    }

    # Set expiration longer than object lock retention period
    expiration {
      days = max(90,
        data.storagegrid_s3_bucket_object_lock_configuration.foo.default_retention_setting != null ?
        (data.storagegrid_s3_bucket_object_lock_configuration.foo.default_retention_setting.days > 0 ?
        data.storagegrid_s3_bucket_object_lock_configuration.foo.default_retention_setting.days + 30 : 90) : 90
      )
    }
  }
}

# Output object lock information
output "foo_default_retention_mode" {
  value = data.storagegrid_s3_bucket_object_lock_configuration.foo.default_retention_setting != null ? data.storagegrid_s3_bucket_object_lock_configuration.foo.default_retention_setting.mode : null
}

output "foo_default_retention_days" {
  value = data.storagegrid_s3_bucket_object_lock_configuration.foo.default_retention_setting != null ? data.storagegrid_s3_bucket_object_lock_configuration.foo.default_retention_setting.days : null
}

output "foo_default_retention_years" {
  value = data.storagegrid_s3_bucket_object_lock_configuration.foo.default_retention_setting != null ? data.storagegrid_s3_bucket_object_lock_configuration.foo.default_retention_setting.years : null
}

output "bar_object_lock_id" {
  value = data.storagegrid_s3_bucket_object_lock_configuration.bar.id
}