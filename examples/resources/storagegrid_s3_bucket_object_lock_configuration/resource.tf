# Configure object lock with compliance mode and days-based retention
resource "storagegrid_s3_bucket_object_lock_configuration" "foo" {
  bucket_name = storagegrid_s3_bucket.foo.bucket_name

  default_retention_setting {
    mode  = "compliance"
    days  = 30
    years = 0
  }
}

# Configure object lock with governance mode and years-based retention
resource "storagegrid_s3_bucket_object_lock_configuration" "bar" {
  bucket_name = storagegrid_s3_bucket.bar.bucket_name

  default_retention_setting {
    mode  = "governance"
    days  = 0
    years = 1
  }
}

# Object lock configuration without default retention settings
resource "storagegrid_s3_bucket_object_lock_configuration" "baz" {
  bucket_name = storagegrid_s3_bucket.baz.bucket_name
  # No default_retention_setting block = no default retention
}

# Reference buckets with object lock enabled
resource "storagegrid_s3_bucket" "foo" {
  bucket_name         = "foo-bucket"
  region              = "us-east-1"
  object_lock_enabled = true
}

resource "storagegrid_s3_bucket" "bar" {
  bucket_name         = "bar-bucket"
  region              = "us-east-1"
  object_lock_enabled = true
}

resource "storagegrid_s3_bucket" "baz" {
  bucket_name         = "baz-bucket"
  region              = "us-east-1"
  object_lock_enabled = true
}