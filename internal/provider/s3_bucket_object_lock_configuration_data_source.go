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
	_ datasource.DataSource              = &S3BucketObjectLockConfigurationDataSource{}
	_ datasource.DataSourceWithConfigure = &S3BucketObjectLockConfigurationDataSource{}
)

func NewS3BucketObjectLockConfigurationDataSource() datasource.DataSource {
	return &S3BucketObjectLockConfigurationDataSource{}
}

// S3BucketObjectLockConfigurationDataSource defines the data source implementation.
type S3BucketObjectLockConfigurationDataSource struct {
	client *utils.Client
}

// S3BucketObjectLockConfigurationDataSourceModel describes the data source data model.
type S3BucketObjectLockConfigurationDataSourceModel struct {
	BucketName              types.String                            `tfsdk:"bucket_name"`
	Enabled                 types.Bool                              `tfsdk:"enabled"`
	DefaultRetentionSetting *DefaultRetentionSettingDataSourceModel `tfsdk:"default_retention_setting"`
}

// DefaultRetentionSettingDataSourceModel represents default retention settings.
type DefaultRetentionSettingDataSourceModel struct {
	Mode  types.String `tfsdk:"mode"`
	Days  types.Int64  `tfsdk:"days"`
	Years types.Int64  `tfsdk:"years"`
}

func (d *S3BucketObjectLockConfigurationDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_s3_bucket_object_lock_configuration"
}

func (d *S3BucketObjectLockConfigurationDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches object lock configuration for a StorageGrid S3 bucket.",
		Attributes: map[string]schema.Attribute{
			"bucket_name": schema.StringAttribute{
				Description: "The name of the S3 bucket to fetch object lock configuration for.",
				Required:    true,
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether object lock is enabled for the bucket.",
				Computed:    true,
			},
			"default_retention_setting": schema.SingleNestedAttribute{
				Description: "Default retention settings for object lock.",
				Computed:    true,
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"mode": schema.StringAttribute{
						Description: "The retention mode (compliance or governance).",
						Computed:    true,
					},
					"days": schema.Int64Attribute{
						Description: "Retention period in days.",
						Computed:    true,
						Optional:    true,
					},
					"years": schema.Int64Attribute{
						Description: "Retention period in years.",
						Computed:    true,
						Optional:    true,
					},
				},
			},
		},
	}
}

func (d *S3BucketObjectLockConfigurationDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *S3BucketObjectLockConfigurationDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state S3BucketObjectLockConfigurationDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	bucketName := state.BucketName.ValueString()
	objectLock, err := d.client.GetS3BucketObjectLock(bucketName)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to Read S3 Bucket Object Lock Configuration for %s", bucketName),
			err.Error(),
		)
		return
	}

	// Map API response data to the Terraform state model
	state.Enabled = types.BoolValue(objectLock.Enabled)

	// Handle optional default retention setting
	if objectLock.DefaultRetentionSetting != nil {
		state.DefaultRetentionSetting = &DefaultRetentionSettingDataSourceModel{
			Mode: types.StringValue(objectLock.DefaultRetentionSetting.Mode),
		}

		if objectLock.DefaultRetentionSetting.Days > 0 {
			state.DefaultRetentionSetting.Days = types.Int64Value(int64(objectLock.DefaultRetentionSetting.Days))
		} else {
			state.DefaultRetentionSetting.Days = types.Int64Null()
		}

		if objectLock.DefaultRetentionSetting.Years > 0 {
			state.DefaultRetentionSetting.Years = types.Int64Value(int64(objectLock.DefaultRetentionSetting.Years))
		} else {
			state.DefaultRetentionSetting.Years = types.Int64Null()
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
