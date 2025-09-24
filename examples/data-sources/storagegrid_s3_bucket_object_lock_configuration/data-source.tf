# Look up object lock configuration for the foo bucket
data "storagegrid_s3_bucket_object_lock_configuration" "foo" {
  bucket_name = "foo-bucket"
}

# Look up object lock configuration for the bar bucket
data "storagegrid_s3_bucket_object_lock_configuration" "bar" {
  bucket_name = "bar-bucket"
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
