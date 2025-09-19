// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/team-fenrir/terraform-provider-storagegrid/internal/utils"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource                = &S3BucketVersioningResource{}
	_ resource.ResourceWithConfigure   = &S3BucketVersioningResource{}
	_ resource.ResourceWithImportState = &S3BucketVersioningResource{}
)

func NewS3BucketVersioningResource() resource.Resource {
	return &S3BucketVersioningResource{}
}

// S3BucketVersioningResource defines the resource implementation.
type S3BucketVersioningResource struct {
	client *utils.Client
}

// S3BucketVersioningResourceModel describes the resource data model.
type S3BucketVersioningResourceModel struct {
	BucketName          types.String `tfsdk:"bucket_name"`
	VersioningEnabled   types.Bool   `tfsdk:"versioning_enabled"`
	VersioningSuspended types.Bool   `tfsdk:"versioning_suspended"`
	ID                  types.String `tfsdk:"id"`
}

func (r *S3BucketVersioningResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_s3_bucket_versioning"
}

func (r *S3BucketVersioningResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages versioning configuration for a StorageGrid S3 bucket.",
		Attributes: map[string]schema.Attribute{
			"bucket_name": schema.StringAttribute{
				Description: "The name of the S3 bucket to configure versioning for.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"versioning_enabled": schema.BoolAttribute{
				Description: "Whether versioning is enabled for the bucket.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"versioning_suspended": schema.BoolAttribute{
				Description: "Whether versioning is suspended for the bucket.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"id": schema.StringAttribute{
				Description: "The unique identifier for the versioning configuration (same as bucket_name).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *S3BucketVersioningResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *S3BucketVersioningResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan S3BucketVersioningResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	bucketName := plan.BucketName.ValueString()
	versioningEnabled := plan.VersioningEnabled.ValueBool()
	versioningSuspended := plan.VersioningSuspended.ValueBool()

	err := r.client.UpdateS3BucketVersioning(bucketName, versioningEnabled, versioningSuspended)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to Create S3 Bucket Versioning Configuration for %s", bucketName),
			err.Error(),
		)
		return
	}

	// Set the ID (same as bucket name)
	plan.ID = types.StringValue(bucketName)

	// Save the plan to state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *S3BucketVersioningResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state S3BucketVersioningResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	bucketName := state.BucketName.ValueString()
	versioning, err := r.client.GetS3BucketVersioning(bucketName)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to Read S3 Bucket Versioning Configuration for %s", bucketName),
			err.Error(),
		)
		return
	}

	// Update state with current values
	state.VersioningEnabled = types.BoolValue(versioning.VersioningEnabled)
	state.VersioningSuspended = types.BoolValue(versioning.VersioningSuspended)
	state.ID = types.StringValue(bucketName)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *S3BucketVersioningResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan S3BucketVersioningResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	bucketName := plan.BucketName.ValueString()
	versioningEnabled := plan.VersioningEnabled.ValueBool()
	versioningSuspended := plan.VersioningSuspended.ValueBool()

	err := r.client.UpdateS3BucketVersioning(bucketName, versioningEnabled, versioningSuspended)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to Update S3 Bucket Versioning Configuration for %s", bucketName),
			err.Error(),
		)
		return
	}

	// Save the updated plan to state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *S3BucketVersioningResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state S3BucketVersioningResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	bucketName := state.BucketName.ValueString()

	// When deleting the versioning resource, set versioning to suspended state
	// (versioningEnabled=false, versioningSuspended=true) since StorageGrid
	// requires at least one of them to be true
	err := r.client.UpdateS3BucketVersioning(bucketName, false, true)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to Delete S3 Bucket Versioning Configuration for %s", bucketName),
			err.Error(),
		)
		return
	}

	// State is automatically cleared on successful delete
}

func (r *S3BucketVersioningResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import using the bucket name as the identifier
	bucketName := req.ID

	// Validate that the bucket exists and get versioning configuration
	versioning, err := r.client.GetS3BucketVersioning(bucketName)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to Import S3 Bucket Versioning Configuration for %s", bucketName),
			fmt.Sprintf("Bucket does not exist or versioning configuration is not accessible: %s", err.Error()),
		)
		return
	}

	// Set the imported versioning configuration in state
	state := S3BucketVersioningResourceModel{
		BucketName:          types.StringValue(bucketName),
		VersioningEnabled:   types.BoolValue(versioning.VersioningEnabled),
		VersioningSuspended: types.BoolValue(versioning.VersioningSuspended),
		ID:                  types.StringValue(bucketName),
	}

	// Set the state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)

	// Set the ID attribute explicitly for import
	resource.ImportStatePassthroughID(ctx, path.Root("bucket_name"), req, resp)
}
