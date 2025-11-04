# Create a basic S3 bucket
resource "storagegrid_s3_bucket" "data" {
  bucket_name = "my-data-bucket"
  region      = "us-east-1"
}

# Create an S3 bucket with object lock enabled
resource "storagegrid_s3_bucket" "compliance" {
  bucket_name         = "compliance-bucket"
  region              = "us-west-2"
  object_lock_enabled = true
}
