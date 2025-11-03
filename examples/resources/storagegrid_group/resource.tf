# Create a StorageGrid group with S3 and management permissions
resource "storagegrid_group" "admin" {
  group_name = "admin-group"

  policies {
    management = {
      manage_all_containers     = true
      manage_endpoints          = true
      manage_own_s3_credentials = true
      root_access               = false
    }
    s3 = jsonencode({
      Statement = [
        {
          Sid    = "AllowFullS3Access"
          Effect = "Allow"
          Action = "s3:*"
          Resource = [
            "urn:sgws:s3:::*"
          ]
        }
      ]
    })
  }
}

# Create a group with limited S3 permissions for specific bucket
resource "storagegrid_group" "developers" {
  group_name = "developers"

  policies {
    management = {
      manage_own_s3_credentials = true
    }
    s3 = jsonencode({
      Statement = [
        {
          Sid    = "AllowDevBucketAccess"
          Effect = "Allow"
          Action = [
            "s3:GetObject",
            "s3:PutObject",
            "s3:DeleteObject",
            "s3:ListBucket"
          ]
          Resource = [
            "urn:sgws:s3:::dev-bucket",
            "urn:sgws:s3:::dev-bucket/*"
          ]
        }
      ]
    })
  }
}

# Create a read-only group using external policy file
resource "storagegrid_group" "readonly" {
  group_name = "readonly-group"

  policies {
    s3 = file("${path.module}/policies/readonly-policy.json")
  }
}
