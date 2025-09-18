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
	BucketName types.String           `tfsdk:"bucket_name"`
	Status     types.String           `tfsdk:"status"`
	APIVersion types.String           `tfsdk:"api_version"`
	Data       *S3BucketDataModel     `tfsdk:"data"`
}

// S3BucketDataModel maps the nested 'data' object from the API response.
type S3BucketDataModel struct {
	ID                     types.String `tfsdk:"id"`
	Name                   types.String `tfsdk:"name"`
	CreationTime           types.String `tfsdk:"creation_time"`
	Region                 types.String `tfsdk:"region"`
	ObjectLockEnabled      types.Bool   `tfsdk:"object_lock_enabled"`
	ComplianceEnabled      types.Bool   `tfsdk:"compliance_enabled"`
	ConsistencyLevel       types.String `tfsdk:"consistency_level"`
	LastAccessTimeEnabled  types.Bool   `tfsdk:"last_access_time_enabled"`
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
			"status": schema.StringAttribute{
				Description: "The status of the API response.",
				Computed:    true,
			},
			"api_version": schema.StringAttribute{
				Description: "The version of the API.",
				Computed:    true,
			},
			"data": schema.SingleNestedAttribute{
				Description: "The main data object for the StorageGrid S3 bucket.",
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						Description: "The unique identifier for the bucket.",
						Computed:    true,
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
					},
					"object_lock_enabled": schema.BoolAttribute{
						Description: "Indicates if object lock is enabled for the bucket.",
						Computed:    true,
					},
					"compliance_enabled": schema.BoolAttribute{
						Description: "Indicates if compliance is enabled for the bucket.",
						Computed:    true,
					},
					"consistency_level": schema.StringAttribute{
						Description: "The consistency level configured for the bucket.",
						Computed:    true,
					},
					"last_access_time_enabled": schema.BoolAttribute{
						Description: "Indicates if last access time tracking is enabled.",
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
	apiResponse, err := d.client.GetS3Bucket(bucketName)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to Read S3 Bucket %s", bucketName),
			err.Error(),
		)
		return
	}

	bucket := apiResponse.Data

	// Map API response data to the Terraform state model
	state.Status = types.StringValue(apiResponse.Status)
	state.APIVersion = types.StringValue(apiResponse.APIVersion)

	state.Data = &S3BucketDataModel{
		ID:                     types.StringValue(bucket.ID),
		Name:                   types.StringValue(bucket.Name),
		CreationTime:           types.StringValue(bucket.CreationTime),
		Region:                 types.StringValue(bucket.Region),
		ObjectLockEnabled:      types.BoolValue(bucket.ObjectLockEnabled),
		ComplianceEnabled:      types.BoolValue(bucket.ComplianceEnabled),
		ConsistencyLevel:       types.StringValue(bucket.ConsistencyLevel),
		LastAccessTimeEnabled:  types.BoolValue(bucket.LastAccessTimeEnabled),
	}

	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}