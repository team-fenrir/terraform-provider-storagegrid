# Enable versioning on the foo bucket
resource "storagegrid_s3_bucket_versioning" "foo" {
  bucket_name          = storagegrid_s3_bucket.foo.bucket_name
  versioning_enabled   = true
  versioning_suspended = false
}

# Suspend versioning on the bar bucket
resource "storagegrid_s3_bucket_versioning" "bar" {
  bucket_name          = storagegrid_s3_bucket.bar.bucket_name
  versioning_enabled   = false
  versioning_suspended = true
}

# Reference the buckets created in the bucket example
resource "storagegrid_s3_bucket" "foo" {
  bucket_name = "foo-bucket"
  region      = "us-east-1"
}

resource "storagegrid_s3_bucket" "bar" {
  bucket_name = "bar-bucket"
  region      = "us-east-1"
  # Note: Cannot modify versioning on object lock enabled buckets
  # object_lock_enabled = false
}