// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/team-fenrir/terraform-provider-storagegrid/internal/utils"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource              = &S3BucketResource{}
	_ resource.ResourceWithConfigure = &S3BucketResource{}
)

func NewS3BucketResource() resource.Resource {
	return &S3BucketResource{}
}

// S3BucketResource defines the resource implementation.
type S3BucketResource struct {
	client *utils.Client
}

// S3BucketResourceModel describes the resource data model.
type S3BucketResourceModel struct {
	Name              types.String `tfsdk:"name"`
	Region            types.String `tfsdk:"region"`
	ObjectLockEnabled types.Bool   `tfsdk:"object_lock_enabled"`
	ID                types.String `tfsdk:"id"`
}

func (r *S3BucketResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_s3_bucket"
}

func (r *S3BucketResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a StorageGrid S3 bucket.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Description: "The name of the S3 bucket.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"region": schema.StringAttribute{
				Description: "The region where the bucket should be created.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("us-east-1"),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"object_lock_enabled": schema.BoolAttribute{
				Description: "Whether S3 Object Lock is enabled for this bucket. Defaults to false. When enabled, uses governance mode with 1 day retention as default.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
			"id": schema.StringAttribute{
				Description: "The unique identifier for the bucket (same as name).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *S3BucketResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*utils.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *utils.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

func (r *S3BucketResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan S3BucketResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create the bucket
	bucketName := plan.Name.ValueString()
	region := plan.Region.ValueString()
	objectLockEnabled := plan.ObjectLockEnabled.ValueBool()

	err := r.client.CreateS3Bucket(bucketName, region, objectLockEnabled)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to Create S3 Bucket %s", bucketName),
			err.Error(),
		)
		return
	}

	// Set the ID (same as name for S3 buckets)
	plan.ID = types.StringValue(bucketName)

	// Save the plan to state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *S3BucketResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state S3BucketResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	bucketName := state.Name.ValueString()
	bucket, err := r.client.GetS3Bucket(bucketName)
	if err != nil {
		// If bucket is not found, remove from state
		resp.Diagnostics.AddWarning(
			fmt.Sprintf("S3 Bucket %s not found", bucketName),
			"The bucket may have been deleted outside of Terraform. Removing from state.",
		)
		resp.State.RemoveResource(ctx)
		return
	}

	// Update state with current values
	state.Name = types.StringValue(bucket.Name)
	state.ID = types.StringValue(bucket.Name)

	if bucket.Region != "" {
		state.Region = types.StringValue(bucket.Region)
	} else {
		state.Region = types.StringValue("us-east-1")
	}

	// Set object lock enabled status from bucket data
	if bucket.S3ObjectLock != nil {
		state.ObjectLockEnabled = types.BoolValue(bucket.S3ObjectLock.Enabled)
	} else {
		state.ObjectLockEnabled = types.BoolValue(false)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *S3BucketResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Since name and region require replacement, this should not be called
	resp.Diagnostics.AddError(
		"Unexpected Update Call",
		"All attributes of this resource require replacement and should trigger a destroy/create instead of update.",
	)
}

func (r *S3BucketResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state S3BucketResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	bucketName := state.Name.ValueString()

	// TODO: Implement delete functionality when the API endpoint is available
	resp.Diagnostics.AddError(
		"Delete Not Implemented",
		fmt.Sprintf("Delete operation for S3 bucket %s is not yet implemented. Please delete the bucket manually.", bucketName),
	)
}