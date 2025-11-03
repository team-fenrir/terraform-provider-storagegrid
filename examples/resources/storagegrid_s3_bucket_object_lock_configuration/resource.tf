# Configure object lock with compliance mode and days-based retention
resource "storagegrid_s3_bucket_object_lock_configuration" "compliance" {
  bucket_name = storagegrid_s3_bucket.compliance.bucket_name

  default_retention_setting {
    mode  = "compliance"
    days  = 30
    years = 0
  }
}

# Configure object lock with governance mode and years-based retention
resource "storagegrid_s3_bucket_object_lock_configuration" "audit" {
  bucket_name = storagegrid_s3_bucket.audit.bucket_name

  default_retention_setting {
    mode  = "governance"
    days  = 0
    years = 7
  }
}
