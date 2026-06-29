// Copyright IBM Corp. 2025, 2026
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
	_ datasource.DataSource              = &S3BucketPolicyDataSource{}
	_ datasource.DataSourceWithConfigure = &S3BucketPolicyDataSource{}
)

func NewS3BucketPolicyDataSource() datasource.DataSource {
	return &S3BucketPolicyDataSource{}
}

// S3BucketPolicyDataSource defines the data source implementation.
type S3BucketPolicyDataSource struct {
	client *utils.Client
}

// S3BucketPolicyDataSourceModel describes the data source data model.
type S3BucketPolicyDataSourceModel struct {
	BucketName types.String `tfsdk:"bucket_name"`
	Policy     types.String `tfsdk:"policy"`
	ID         types.String `tfsdk:"id"`
}

func (d *S3BucketPolicyDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_s3_bucket_policy"
}

func (d *S3BucketPolicyDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches the S3 bucket policy for a StorageGrid S3 bucket.",
		Attributes: map[string]schema.Attribute{
			"bucket_name": schema.StringAttribute{
				Description: "The name of the S3 bucket to fetch the policy for.",
				Required:    true,
			},
			"policy": schema.StringAttribute{
				Description: "The bucket policy as a JSON string.",
				Computed:    true,
			},
			"id": schema.StringAttribute{
				Description: "The unique identifier for the bucket policy (same as bucket_name).",
				Computed:    true,
			},
		},
	}
}

func (d *S3BucketPolicyDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *S3BucketPolicyDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state S3BucketPolicyDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	bucketName := state.BucketName.ValueString()
	policy, err := d.client.GetS3BucketPolicy(bucketName)
	if err != nil {
		if isNotFoundError(err) {
			if _, bucketErr := d.client.GetS3Bucket(bucketName); bucketErr != nil {
				resp.Diagnostics.AddError(
					fmt.Sprintf("Unable to Read S3 Bucket Policy for %s", bucketName),
					fmt.Sprintf("Bucket policy was not found and the bucket could not be verified: %s", bucketErr.Error()),
				)
				return
			}

			emptyPolicy := emptyS3BucketPolicy()
			policy = &emptyPolicy
		} else {
			resp.Diagnostics.AddError(
				fmt.Sprintf("Unable to Read S3 Bucket Policy for %s", bucketName),
				err.Error(),
			)
			return
		}
	}

	policyString, err := marshalS3BucketPolicy(policy)
	if err != nil {
		resp.Diagnostics.AddError("Error Processing S3 Bucket Policy", "Could not marshal S3 bucket policy from API into string: "+err.Error())
		return
	}

	state.Policy = types.StringValue(policyString)
	state.ID = types.StringValue(bucketName)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
