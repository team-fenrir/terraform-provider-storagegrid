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
	_ datasource.DataSource              = &S3BucketVersioningDataSource{}
	_ datasource.DataSourceWithConfigure = &S3BucketVersioningDataSource{}
)

func NewS3BucketVersioningDataSource() datasource.DataSource {
	return &S3BucketVersioningDataSource{}
}

// S3BucketVersioningDataSource defines the data source implementation.
type S3BucketVersioningDataSource struct {
	client *utils.Client
}

// S3BucketVersioningDataSourceModel describes the data source data model.
type S3BucketVersioningDataSourceModel struct {
	BucketName types.String `tfsdk:"bucket_name"`
	Status     types.String `tfsdk:"status"`
}

func (d *S3BucketVersioningDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_s3_bucket_versioning"
}

func (d *S3BucketVersioningDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches versioning configuration for a StorageGrid S3 bucket.",
		Attributes: map[string]schema.Attribute{
			"bucket_name": schema.StringAttribute{
				Description: "The name of the S3 bucket to fetch versioning information for.",
				Required:    true,
			},
			"status": schema.StringAttribute{
				Description: "The versioning status for the bucket. Possible values are 'Disabled', 'Enabled' or 'Suspended'.",
				Computed:    true,
			},
		},
	}
}

func (d *S3BucketVersioningDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *S3BucketVersioningDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state S3BucketVersioningDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	bucketName := state.BucketName.ValueString()
	versioning, err := d.client.GetS3BucketVersioning(bucketName)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to Read S3 Bucket Versioning for %s", bucketName),
			err.Error(),
		)
		return
	}

	// Convert API boolean fields to status string
	status := apiBoolsToStatus(versioning.VersioningEnabled, versioning.VersioningSuspended)

	// Map API response data to the Terraform state model
	state.Status = types.StringValue(status)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
