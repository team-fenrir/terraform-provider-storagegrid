// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/minio/minio-go/v7"
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
	s3Client *minio.Client
}

type S3BucketDataSourceModel struct {
	Name         types.String `tfsdk:"name"`
	CreationDate types.String `tfsdk:"creation_date"`
	Region       types.String `tfsdk:"region"`
}

func (d *S3BucketDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_s3_bucket"
}

func (d *S3BucketDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches information about a StorageGrid S3 bucket.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Description: "The name of the S3 bucket to look up.",
				Required:    true,
			},
			"creation_date": schema.StringAttribute{
				Description: "The creation date of the bucket.",
				Computed:    true,
			},
			"region": schema.StringAttribute{
				Description: "The region where the bucket is located.",
				Computed:    true,
			},
		},
	}
}

func (d *S3BucketDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	clients, ok := req.ProviderData.(*StorageGridClients)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *StorageGridClients, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	// Validate that S3 client is available for S3 bucket data source operations
	if clients.S3Client == nil {
		resp.Diagnostics.AddError(
			"S3 API Client Not Configured",
			"The s3_bucket data source requires a StorageGrid S3 API endpoint to be configured. "+
				"Please configure the 's3' endpoint in the provider's endpoints block or set the STORAGEGRID_S3_ENDPOINT environment variable.",
		)
		return
	}

	d.s3Client = clients.S3Client
}

func (d *S3BucketDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state S3BucketDataSourceModel

	diags := req.Config.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	bucketName := state.Name.ValueString()

	// Check if bucket exists and get its information
	exists, err := d.s3Client.BucketExists(ctx, bucketName)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error checking bucket existence",
			fmt.Sprintf("Unable to check if bucket '%s' exists: %s", bucketName, err),
		)
		return
	}

	if !exists {
		resp.Diagnostics.AddError(
			"Bucket not found",
			fmt.Sprintf("S3 bucket '%s' does not exist", bucketName),
		)
		return
	}

	// Get bucket location (region)
	location, err := d.s3Client.GetBucketLocation(ctx, bucketName)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error getting bucket location",
			fmt.Sprintf("Unable to get location for bucket '%s': %s", bucketName, err),
		)
		return
	}

	// List buckets to get creation date
	buckets, err := d.s3Client.ListBuckets(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error listing buckets",
			fmt.Sprintf("Unable to list buckets to get creation date: %s", err),
		)
		return
	}

	var creationDate string
	for _, bucket := range buckets {
		if bucket.Name == bucketName {
			creationDate = bucket.CreationDate.Format("2006-01-02T15:04:05Z")
			break
		}
	}

	// Set the computed attributes
	state.Region = types.StringValue(location)
	state.CreationDate = types.StringValue(creationDate)

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}