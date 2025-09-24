data "storagegrid_s3_bucket" "foo" {
  bucket_name = "foo-bucket"
}

# Output bucket information
output "foo_bucket_region" {
  value = data.storagegrid_s3_bucket.foo.region
}

output "foo_bucket_creation_time" {
  value = data.storagegrid_s3_bucket.foo.creation_time
}

output "bar_bucket_object_lock_enabled" {
  value = data.storagegrid_s3_bucket.foo.object_lock_enabled
}
