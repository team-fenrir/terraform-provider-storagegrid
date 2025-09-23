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
	_ datasource.DataSource              = &S3BucketDataSource{}
	_ datasource.DataSourceWithConfigure = &S3BucketDataSource{}
)

func NewS3BucketDataSource() datasource.DataSource {
	return &S3BucketDataSource{}
}

// S3BucketDataSource defines the data source implementation.
type S3BucketDataSource struct {
	client *utils.Client
}

type S3BucketDataSourceModel struct {
	BucketName   types.String       `tfsdk:"bucket_name"`
	Name         types.String       `tfsdk:"name"`
	CreationTime types.String       `tfsdk:"creation_time"`
	Region       types.String       `tfsdk:"region"`
	Compliance   *ComplianceModel   `tfsdk:"compliance"`
	S3ObjectLock *S3ObjectLockModel `tfsdk:"s3_object_lock"`
	DeleteStatus *DeleteStatusModel `tfsdk:"delete_status"`
}

// ComplianceModel maps compliance configuration from the API response.
type ComplianceModel struct {
	AutoDelete             types.Bool  `tfsdk:"auto_delete"`
	LegalHold              types.Bool  `tfsdk:"legal_hold"`
	RetentionPeriodMinutes types.Int64 `tfsdk:"retention_period_minutes"`
}

// S3ObjectLockModel maps S3 object lock configuration from the API response.
type S3ObjectLockModel struct {
	Enabled                 types.Bool                    `tfsdk:"enabled"`
	DefaultRetentionSetting *DefaultRetentionSettingModel `tfsdk:"default_retention_setting"`
}

// DefaultRetentionSettingModel maps default retention settings.
type DefaultRetentionSettingModel struct {
	Mode  types.String `tfsdk:"mode"`
	Days  types.Int64  `tfsdk:"days"`
	Years types.Int64  `tfsdk:"years"`
}

// DeleteStatusModel maps delete object status from the API response.
type DeleteStatusModel struct {
	IsDeletingObjects  types.Bool   `tfsdk:"is_deleting_objects"`
	InitialObjectCount types.String `tfsdk:"initial_object_count"`
	InitialObjectBytes types.String `tfsdk:"initial_object_bytes"`
}

func (d *S3BucketDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_s3_bucket"
}

func (d *S3BucketDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches information about a StorageGrid S3 bucket.",
		Attributes: map[string]schema.Attribute{
			"bucket_name": schema.StringAttribute{
				Description: "The name of the S3 bucket to fetch.",
				Required:    true,
			},
			"name": schema.StringAttribute{
				Description: "The name of the bucket.",
				Computed:    true,
			},
			"creation_time": schema.StringAttribute{
				Description: "The time when the bucket was created.",
				Computed:    true,
			},
			"region": schema.StringAttribute{
				Description: "The region where the bucket is located.",
				Computed:    true,
				Optional:    true,
			},
			"compliance": schema.SingleNestedAttribute{
				Description: "Compliance settings for the bucket.",
				Computed:    true,
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"auto_delete": schema.BoolAttribute{
						Description: "Indicates if auto-delete is enabled.",
						Computed:    true,
					},
					"legal_hold": schema.BoolAttribute{
						Description: "Indicates if legal hold is enabled.",
						Computed:    true,
					},
					"retention_period_minutes": schema.Int64Attribute{
						Description: "Retention period in minutes.",
						Computed:    true,
					},
				},
			},
			"s3_object_lock": schema.SingleNestedAttribute{
				Description: "S3 object lock configuration for the bucket.",
				Computed:    true,
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"enabled": schema.BoolAttribute{
						Description: "Indicates if S3 object lock is enabled.",
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
			},
			"delete_status": schema.SingleNestedAttribute{
				Description: "Delete object status for the bucket.",
				Computed:    true,
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"is_deleting_objects": schema.BoolAttribute{
						Description: "Indicates if objects are being deleted.",
						Computed:    true,
					},
					"initial_object_count": schema.StringAttribute{
						Description: "Initial count of objects when deletion started.",
						Computed:    true,
					},
					"initial_object_bytes": schema.StringAttribute{
						Description: "Initial size in bytes when deletion started.",
						Computed:    true,
					},
				},
			},
		},
	}
}

func (d *S3BucketDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *S3BucketDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state S3BucketDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	bucketName := state.BucketName.ValueString()
	bucket, err := d.client.GetS3Bucket(bucketName)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to Read S3 Bucket %s", bucketName),
			err.Error(),
		)
		return
	}

	// Map API response data to the Terraform state model
	state.Name = types.StringValue(bucket.Name)
	state.CreationTime = types.StringValue(bucket.CreationTime)

	// Handle optional region field with default
	if bucket.Region != "" {
		state.Region = types.StringValue(bucket.Region)
	} else {
		state.Region = types.StringValue("us-east-1")
	}

	// Handle optional compliance configuration
	if bucket.Compliance != nil {
		state.Compliance = &ComplianceModel{
			AutoDelete:             types.BoolValue(bucket.Compliance.AutoDelete),
			LegalHold:              types.BoolValue(bucket.Compliance.LegalHold),
			RetentionPeriodMinutes: types.Int64Value(bucket.Compliance.RetentionPeriodMinutes),
		}
	}

	// Handle optional S3 object lock configuration
	if bucket.S3ObjectLock != nil {
		s3ObjectLock := &S3ObjectLockModel{
			Enabled: types.BoolValue(bucket.S3ObjectLock.Enabled),
		}

		if bucket.S3ObjectLock.DefaultRetentionSetting != nil {
			s3ObjectLock.DefaultRetentionSetting = &DefaultRetentionSettingModel{
				Mode: types.StringValue(bucket.S3ObjectLock.DefaultRetentionSetting.Mode),
			}

			if bucket.S3ObjectLock.DefaultRetentionSetting.Days > 0 {
				s3ObjectLock.DefaultRetentionSetting.Days = types.Int64Value(int64(bucket.S3ObjectLock.DefaultRetentionSetting.Days))
			} else {
				s3ObjectLock.DefaultRetentionSetting.Days = types.Int64Null()
			}

			if bucket.S3ObjectLock.DefaultRetentionSetting.Years > 0 {
				s3ObjectLock.DefaultRetentionSetting.Years = types.Int64Value(int64(bucket.S3ObjectLock.DefaultRetentionSetting.Years))
			} else {
				s3ObjectLock.DefaultRetentionSetting.Years = types.Int64Null()
			}
		}

		state.S3ObjectLock = s3ObjectLock
	}

	// Handle optional delete status configuration
	if bucket.DeleteStatus != nil {
		state.DeleteStatus = &DeleteStatusModel{
			IsDeletingObjects:  types.BoolValue(bucket.DeleteStatus.IsDeletingObjects),
			InitialObjectCount: types.StringValue(bucket.DeleteStatus.InitialObjectCount),
			InitialObjectBytes: types.StringValue(bucket.DeleteStatus.InitialObjectBytes),
		}
	}

	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
