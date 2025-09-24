# Create a StorageGrid group with S3 and management permissions
resource "storagegrid_group" "foo" {
  group_name = "foo-group"

  policies {
    management = {
      manage_all_containers     = true
      manage_endpoints          = false
      manage_own_s3_credentials = true
      root_access               = false
    }
    s3 = {
      statement = [
        {
          sid    = "AllowFooBucketAccess"
          effect = "Allow"
          action = [
            "s3:GetObject",
            "s3:PutObject",
            "s3:DeleteObject",
            "s3:ListBucket"
          ]
          resource = [
            "urn:sgws:s3:::foo-bucket",
            "urn:sgws:s3:::foo-bucket/*"
          ]
        }
      ]
    }
  }
}

# Create a read-only group for bar resources
resource "storagegrid_group" "bar" {
  group_name = "bar-readonly"

  policies {
    management = {
      manage_all_containers     = false
      manage_endpoints          = false
      manage_own_s3_credentials = true
      root_access               = false
    }
    s3 = {
      statement = [
        {
          sid    = "AllowBarBucketRead"
          effect = "Allow"
          action = [
            "s3:GetObject",
            "s3:ListBucket"
          ]
          resource = [
            "urn:sgws:s3:::bar-bucket",
            "urn:sgws:s3:::bar-bucket/*"
          ]
        }
      ]
    }
  }
}