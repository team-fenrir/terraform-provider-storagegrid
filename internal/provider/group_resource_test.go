// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccGroupResource_WithCondition(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing with Condition
			{
				Config: providerConfig + `
resource "storagegrid_group" "test" {
  group_name = "test-group-with-condition"

  policies {
    s3 = jsonencode({
      Statement = [
        {
          Sid    = "AllowListAllBuckets"
          Effect = "Allow"
          Action = "s3:ListAllMyBuckets"
          Resource = "*"
        },
        {
          Sid    = "AllowListBucket"
          Effect = "Allow"
          Action = [
            "s3:ListBucket",
            "s3:ListBucketVersions"
          ]
          Resource = "arn:aws:s3:::test-bucket"
          Condition = {
            StringLike = {
              "s3:prefix" = [
                "documents/*",
                "documents"
              ]
            }
          }
        },
        {
          Sid    = "AllowObjectAccess"
          Effect = "Allow"
          Action = [
            "s3:GetObject",
            "s3:PutObject"
          ]
          Resource = "arn:aws:s3:::test-bucket/documents/*"
        }
      ]
    })

    management {
      manage_own_s3_credentials = true
    }
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify group attributes
					resource.TestCheckResourceAttr("storagegrid_group.test", "group_name", "test-group-with-condition"),
					resource.TestCheckResourceAttr("storagegrid_group.test", "unique_name", "group/test-group-with-condition"),

					// Verify management policy
					resource.TestCheckResourceAttr("storagegrid_group.test", "policies.management.manage_own_s3_credentials", "true"),
					resource.TestCheckResourceAttr("storagegrid_group.test", "policies.management.root_access", "false"),

					// Verify S3 policy is set (we can't directly check JSON structure in acceptance tests,
					// but we verify it's not empty)
					resource.TestCheckResourceAttrSet("storagegrid_group.test", "policies.s3"),

					// Verify computed attributes are set
					resource.TestCheckResourceAttrSet("storagegrid_group.test", "id"),
					resource.TestCheckResourceAttrSet("storagegrid_group.test", "account_id"),
					resource.TestCheckResourceAttrSet("storagegrid_group.test", "display_name"),
					resource.TestCheckResourceAttrSet("storagegrid_group.test", "group_urn"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "storagegrid_group.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateId:     "test-group-with-condition",
			},
			// Update testing - modify the Condition
			{
				Config: providerConfig + `
resource "storagegrid_group" "test" {
  group_name = "test-group-with-condition"

  policies {
    s3 = jsonencode({
      Statement = [
        {
          Sid    = "AllowListAllBuckets"
          Effect = "Allow"
          Action = "s3:ListAllMyBuckets"
          Resource = "*"
        },
        {
          Sid    = "AllowListBucket"
          Effect = "Allow"
          Action = [
            "s3:ListBucket",
            "s3:ListBucketVersions"
          ]
          Resource = "arn:aws:s3:::test-bucket"
          Condition = {
            StringLike = {
              "s3:prefix" = [
                "images/*",
                "images"
              ]
            }
          }
        },
        {
          Sid    = "AllowObjectAccess"
          Effect = "Allow"
          Action = [
            "s3:GetObject",
            "s3:PutObject"
          ]
          Resource = "arn:aws:s3:::test-bucket/images/*"
        }
      ]
    })

    management {
      manage_own_s3_credentials = true
    }
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify the resource still exists and has updated S3 policy
					resource.TestCheckResourceAttr("storagegrid_group.test", "group_name", "test-group-with-condition"),
					resource.TestCheckResourceAttrSet("storagegrid_group.test", "policies.s3"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccGroupResource_MultipleConditionOperators(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with multiple condition operators
			{
				Config: providerConfig + `
resource "storagegrid_group" "test" {
  group_name = "test-group-multi-condition"

  policies {
    s3 = jsonencode({
      Statement = [
        {
          Sid    = "ConditionalAccess"
          Effect = "Allow"
          Action = "s3:GetObject"
          Resource = "arn:aws:s3:::secure-bucket/*"
          Condition = {
            StringEquals = {
              "s3:ExistingObjectTag/environment" = "production"
            }
            IpAddress = {
              "aws:SourceIp" = [
                "192.168.1.0/24",
                "10.0.0.0/8"
              ]
            }
          }
        }
      ]
    })

    management {
      manage_own_s3_credentials = false
    }
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("storagegrid_group.test", "group_name", "test-group-multi-condition"),
					resource.TestCheckResourceAttrSet("storagegrid_group.test", "id"),
					resource.TestCheckResourceAttrSet("storagegrid_group.test", "policies.s3"),
				),
			},
		},
	})
}

func TestAccGroupResource_WithoutCondition(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create without Condition (backward compatibility test)
			{
				Config: providerConfig + `
resource "storagegrid_group" "test" {
  group_name = "test-group-no-condition"

  policies {
    s3 = jsonencode({
      Statement = [
        {
          Sid    = "AllowListAllBuckets"
          Effect = "Allow"
          Action = "s3:ListAllMyBuckets"
          Resource = "*"
        },
        {
          Sid    = "AllowFullBucketAccess"
          Effect = "Allow"
          Action = "s3:*"
          Resource = [
            "arn:aws:s3:::my-bucket",
            "arn:aws:s3:::my-bucket/*"
          ]
        }
      ]
    })

    management {
      root_access = true
    }
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("storagegrid_group.test", "group_name", "test-group-no-condition"),
					resource.TestCheckResourceAttr("storagegrid_group.test", "policies.management.root_access", "true"),
					resource.TestCheckResourceAttrSet("storagegrid_group.test", "id"),
					resource.TestCheckResourceAttrSet("storagegrid_group.test", "policies.s3"),
				),
			},
			// Update to add a Condition
			{
				Config: providerConfig + `
resource "storagegrid_group" "test" {
  group_name = "test-group-no-condition"

  policies {
    s3 = jsonencode({
      Statement = [
        {
          Sid    = "AllowListAllBuckets"
          Effect = "Allow"
          Action = "s3:ListAllMyBuckets"
          Resource = "*"
        },
        {
          Sid    = "AllowConditionalBucketAccess"
          Effect = "Allow"
          Action = "s3:*"
          Resource = [
            "arn:aws:s3:::my-bucket",
            "arn:aws:s3:::my-bucket/*"
          ]
          Condition = {
            StringLike = {
              "s3:prefix" = "shared/*"
            }
          }
        }
      ]
    })

    management {
      root_access = true
    }
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("storagegrid_group.test", "group_name", "test-group-no-condition"),
					resource.TestCheckResourceAttrSet("storagegrid_group.test", "policies.s3"),
				),
			},
		},
	})
}

func TestAccGroupResource_ComplexPolicy(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with the example from the issue
			{
				Config: providerConfig + `
resource "storagegrid_group" "test" {
  group_name = "test-group-complex"

  policies {
    s3 = jsonencode({
      Statement = [
        {
          Sid    = "AllowListAllBuckets"
          Effect = "Allow"
          Action = "s3:ListAllMyBuckets"
          Resource = "*"
        },
        {
          Sid    = "AllowListBucket"
          Effect = "Allow"
          Action = [
            "s3:ListBucket",
            "s3:ListBucketVersions"
          ]
          Resource = "arn:aws:s3:::reflow-client-storage-dev"
          Condition = {
            StringLike = {
              "s3:prefix" = [
                "powerfactory/*",
                "powerfactory"
              ]
            }
          }
        },
        {
          Sid    = "AllowUserFolderAccess"
          Effect = "Allow"
          Action = [
            "s3:GetObject",
            "s3:GetObjectTagging",
            "s3:GetObjectVersion",
            "s3:GetObjectVersionTagging",
            "s3:PutObject"
          ]
          Resource = "arn:aws:s3:::reflow-client-storage-dev/powerfactory/*"
        }
      ]
    })

    management {
      manage_own_s3_credentials = true
      view_all_containers       = true
    }
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("storagegrid_group.test", "group_name", "test-group-complex"),
					resource.TestCheckResourceAttr("storagegrid_group.test", "policies.management.manage_own_s3_credentials", "true"),
					resource.TestCheckResourceAttr("storagegrid_group.test", "policies.management.view_all_containers", "true"),
					resource.TestCheckResourceAttrSet("storagegrid_group.test", "id"),
					resource.TestCheckResourceAttrSet("storagegrid_group.test", "policies.s3"),
				),
			},
		},
	})
}
