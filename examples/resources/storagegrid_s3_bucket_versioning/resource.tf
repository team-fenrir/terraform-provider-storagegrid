# Enable versioning on a bucket
resource "storagegrid_s3_bucket_versioning" "data" {
  bucket_name = storagegrid_s3_bucket.data.bucket_name
  status      = "Enabled"
}

# Suspend versioning on a bucket
resource "storagegrid_s3_bucket_versioning" "archive" {
  bucket_name = storagegrid_s3_bucket.archive.bucket_name
  status      = "Suspended"
}
