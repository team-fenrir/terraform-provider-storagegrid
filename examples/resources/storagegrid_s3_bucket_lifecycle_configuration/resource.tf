# Configure lifecycle with multiple rules and filters
resource "storagegrid_s3_bucket_lifecycle_configuration" "data" {
  bucket_name = storagegrid_s3_bucket.data.bucket_name

  rule {
    id     = "logs-expiration"
    status = "Enabled"

    filter {
      prefix = "logs/"
    }

    expiration {
      days = 90
    }

    noncurrent_version_expiration {
      noncurrent_days = 30
    }
  }

  rule {
    id     = "temp-cleanup"
    status = "Enabled"

    filter {
      prefix = "temp/"
    }

    expiration {
      days = 7
    }

    noncurrent_version_expiration {
      noncurrent_days = 1
    }
  }
}

# Configure lifecycle with date-based expiration
resource "storagegrid_s3_bucket_lifecycle_configuration" "archive" {
  bucket_name = storagegrid_s3_bucket.archive.bucket_name

  rule {
    id     = "archive-cleanup"
    status = "Enabled"

    expiration {
      date = "2026-12-31T00:00:00.000Z"
    }

    noncurrent_version_expiration {
      noncurrent_days = 90
    }
  }
}

# Configure lifecycle without noncurrent version expiration (for non-versioned buckets)
resource "storagegrid_s3_bucket_lifecycle_configuration" "simple" {
  bucket_name = storagegrid_s3_bucket.simple.bucket_name

  rule {
    id     = "cleanup-old-objects"
    status = "Enabled"

    filter {
      prefix = "data/"
    }

    expiration {
      days = 30
    }
  }
}
