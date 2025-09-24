// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/team-fenrir/terraform-provider-storagegrid/internal/utils"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ datasource.DataSource              = &S3BucketLifecycleConfigurationDataSource{}
	_ datasource.DataSourceWithConfigure = &S3BucketLifecycleConfigurationDataSource{}
)

func NewS3BucketLifecycleConfigurationDataSource() datasource.DataSource {
	return &S3BucketLifecycleConfigurationDataSource{}
}

// S3BucketLifecycleConfigurationDataSource defines the data source implementation.
type S3BucketLifecycleConfigurationDataSource struct {
	client *utils.Client
}

// S3BucketLifecycleConfigurationDataSourceModel describes the data source data model.
type S3BucketLifecycleConfigurationDataSourceModel struct {
	BucketName types.String                   `tfsdk:"bucket_name"`
	Rules      []LifecycleRuleDataSourceModel `tfsdk:"rule"`
}

// LifecycleRuleDataSourceModel represents a lifecycle rule.
type LifecycleRuleDataSourceModel struct {
	ID                          types.String                               `tfsdk:"id"`
	Status                      types.String                               `tfsdk:"status"`
	Filter                      *LifecycleFilterDataSourceModel            `tfsdk:"filter"`
	Expiration                  *LifecycleExpirationDataSourceModel        `tfsdk:"expiration"`
	NoncurrentVersionExpiration *LifecycleNoncurrentVersionDataSourceModel `tfsdk:"noncurrent_version_expiration"`
}

// LifecycleFilterDataSourceModel represents a lifecycle rule filter.
type LifecycleFilterDataSourceModel struct {
	Prefix types.String `tfsdk:"prefix"`
}

// LifecycleExpirationDataSourceModel represents expiration settings.
type LifecycleExpirationDataSourceModel struct {
	Days types.Int64  `tfsdk:"days"`
	Date types.String `tfsdk:"date"`
}

// LifecycleNoncurrentVersionDataSourceModel represents noncurrent version expiration settings.
type LifecycleNoncurrentVersionDataSourceModel struct {
	NoncurrentDays types.Int64 `tfsdk:"noncurrent_days"`
}

func (d *S3BucketLifecycleConfigurationDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_s3_bucket_lifecycle_configuration"
}

func (d *S3BucketLifecycleConfigurationDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches lifecycle configuration for a StorageGrid S3 bucket.",
		Attributes: map[string]schema.Attribute{
			"bucket_name": schema.StringAttribute{
				Description: "The name of the S3 bucket to fetch lifecycle configuration for.",
				Required:    true,
			},
		},
		Blocks: map[string]schema.Block{
			"rule": schema.ListNestedBlock{
				Description: "Lifecycle rules for the bucket.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "Unique identifier for the rule.",
							Computed:    true,
						},
						"status": schema.StringAttribute{
							Description: "Status of the rule (Enabled or Disabled).",
							Computed:    true,
						},
					},
					Blocks: map[string]schema.Block{
						"filter": schema.SingleNestedBlock{
							Description: "Filter for the lifecycle rule.",
							Attributes: map[string]schema.Attribute{
								"prefix": schema.StringAttribute{
									Description: "Object key prefix that identifies the objects to which the rule applies.",
									Computed:    true,
									Optional:    true,
								},
							},
						},
						"expiration": schema.SingleNestedBlock{
							Description: "Expiration settings for current object versions.",
							Attributes: map[string]schema.Attribute{
								"days": schema.Int64Attribute{
									Description: "Number of days after object creation when the object expires.",
									Computed:    true,
									Optional:    true,
								},
								"date": schema.StringAttribute{
									Description: "Date when objects expire (ISO 8601 format).",
									Computed:    true,
									Optional:    true,
								},
							},
						},
						"noncurrent_version_expiration": schema.SingleNestedBlock{
							Description: "Expiration settings for noncurrent object versions.",
							Attributes: map[string]schema.Attribute{
								"noncurrent_days": schema.Int64Attribute{
									Description: "Number of days after an object becomes noncurrent when it expires.",
									Computed:    true,
									Optional:    true,
								},
							},
						},
					},
				},
			},
		},
	}
}

func (d *S3BucketLifecycleConfigurationDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*utils.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *utils.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = client
}

func (d *S3BucketLifecycleConfigurationDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state S3BucketLifecycleConfigurationDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	bucketName := state.BucketName.ValueString()
	lifecycleConfig, err := d.client.GetS3BucketLifecycleConfiguration(bucketName)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to Read S3 Bucket Lifecycle Configuration for %s", bucketName),
			err.Error(),
		)
		return
	}

	// Map API response data to the Terraform state model
	var rules []LifecycleRuleDataSourceModel
	for _, rule := range lifecycleConfig.Rules {
		ruleModel := LifecycleRuleDataSourceModel{
			ID:     types.StringValue(rule.ID),
			Status: types.StringValue(rule.Status),
		}

		// Handle filter
		if rule.Filter != nil {
			ruleModel.Filter = &LifecycleFilterDataSourceModel{
				Prefix: types.StringValue(rule.Filter.Prefix),
			}
		}

		// Handle expiration
		if rule.Expiration != nil {
			ruleModel.Expiration = &LifecycleExpirationDataSourceModel{}
			if rule.Expiration.Days > 0 {
				ruleModel.Expiration.Days = types.Int64Value(int64(rule.Expiration.Days))
			} else {
				ruleModel.Expiration.Days = types.Int64Null()
			}
			if rule.Expiration.Date != "" {
				ruleModel.Expiration.Date = types.StringValue(rule.Expiration.Date)
			} else {
				ruleModel.Expiration.Date = types.StringNull()
			}
		}

		// Handle noncurrent version expiration
		if rule.NoncurrentVersionExpiration != nil {
			ruleModel.NoncurrentVersionExpiration = &LifecycleNoncurrentVersionDataSourceModel{
				NoncurrentDays: types.Int64Value(int64(rule.NoncurrentVersionExpiration.NoncurrentDays)),
			}
		}

		rules = append(rules, ruleModel)
	}

	state.Rules = rules

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
