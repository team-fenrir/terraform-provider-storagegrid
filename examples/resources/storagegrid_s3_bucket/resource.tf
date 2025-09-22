# Create a basic S3 bucket
resource "storagegrid_s3_bucket" "foo" {
  name   = "foo-bucket"
  region = "us-east-1"
}

# Create an S3 bucket with object lock enabled
resource "storagegrid_s3_bucket" "bar" {
  name                = "bar-bucket"
  region              = "us-east-1"
  object_lock_enabled = true
}

# Create an S3 bucket with object lock disabled
resource "storagegrid_s3_bucket" "replicated" {
  name                = "foo-replicated-bucket"
  region              = "us-east-1"
  object_lock_enabled = false
}
