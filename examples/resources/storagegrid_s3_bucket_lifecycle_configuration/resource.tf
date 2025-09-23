# Configure lifecycle rules for the foo bucket
resource "storagegrid_s3_bucket_lifecycle_configuration" "foo" {
  bucket_name = storagegrid_s3_bucket.foo.bucket_name

  rule {
    id     = "foo-logs-expiration"
    status = "Enabled"

    filter {
      prefix = "logs/"
    }

    expiration {
      days = 90
    }
  }

  rule {
    id     = "foo-temp-cleanup"
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

# Configure lifecycle rules for the bar bucket with date-based expiration
resource "storagegrid_s3_bucket_lifecycle_configuration" "bar" {
  bucket_name = storagegrid_s3_bucket.bar.bucket_name

  rule {
    id     = "bar-archive-cleanup"
    status = "Enabled"

    filter {
      prefix = "archive/"
    }

    expiration {
      date = "2025-12-31T00:00:00Z"
    }
  }
}

# Simple lifecycle configuration for baz bucket (all objects)
resource "storagegrid_s3_bucket_lifecycle_configuration" "baz" {
  bucket_name = storagegrid_s3_bucket.baz.bucket_name

  rule {
    id     = "baz-global-expiration"
    status = "Enabled"

    # No filter = applies to all objects

    expiration {
      days = 365
    }
  }
}

# Reference the buckets created in the bucket example
resource "storagegrid_s3_bucket" "foo" {
  bucket_name = "foo-bucket"
  region      = "us-east-1"
}

resource "storagegrid_s3_bucket" "bar" {
  bucket_name = "bar-bucket"
  region      = "us-east-1"
}

resource "storagegrid_s3_bucket" "baz" {
  bucket_name = "baz-bucket"
  region      = "us-east-1"
}