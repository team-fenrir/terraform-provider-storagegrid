# Manage a bucket policy for an existing StorageGrid S3 bucket
resource "storagegrid_s3_bucket_policy" "logs" {
  bucket_name = "logs-bucket"

  policy = jsonencode({
    Id      = "logs-bucket-policy"
    Version = "2015-09-08"
    Statement = [
      {
        Sid       = "AllowObjectRead"
        Effect    = "Allow"
        Principal = "*"
        Action    = "s3:GetObject"
        Resource  = "arn:aws:s3:::logs-bucket/*"
      }
    ]
  })
}

