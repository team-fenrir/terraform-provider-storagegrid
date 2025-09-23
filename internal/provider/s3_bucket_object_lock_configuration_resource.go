// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/team-fenrir/terraform-provider-storagegrid/internal/utils"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource                = &S3BucketObjectLockConfigurationResource{}
	_ resource.ResourceWithConfigure   = &S3BucketObjectLockConfigurationResource{}
	_ resource.ResourceWithImportState = &S3BucketObjectLockConfigurationResource{}
)

func NewS3BucketObjectLockConfigurationResource() resource.Resource {
	return &S3BucketObjectLockConfigurationResource{}
}

// S3BucketObjectLockConfigurationResource defines the resource implementation.
type S3BucketObjectLockConfigurationResource struct {
	client *utils.Client
}

// S3BucketObjectLockConfigurationResourceModel describes the resource data model.
type S3BucketObjectLockConfigurationResourceModel struct {
	BucketName              types.String                          `tfsdk:"bucket_name"`
	DefaultRetentionSetting *DefaultRetentionSettingResourceModel `tfsdk:"default_retention_setting"`
	ID                      types.String                          `tfsdk:"id"`
}

// DefaultRetentionSettingResourceModel represents default retention settings for the resource.
type DefaultRetentionSettingResourceModel struct {
	Mode  types.String `tfsdk:"mode"`
	Days  types.Int64  `tfsdk:"days"`
	Years types.Int64  `tfsdk:"years"`
}

func (r *S3BucketObjectLockConfigurationResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_s3_bucket_object_lock_configuration"
}

func (r *S3BucketObjectLockConfigurationResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages default retention settings for a StorageGrid S3 bucket with object lock enabled. " +
			"NOTE: This resource can only be used on buckets that already have object lock enabled at creation time. " +
			"Object lock must be enabled using the storagegrid_s3_bucket resource with object_lock_enabled=true.",
		Attributes: map[string]schema.Attribute{
			"bucket_name": schema.StringAttribute{
				Description: "The name of the S3 bucket to configure object lock for.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"id": schema.StringAttribute{
				Description: "The unique identifier for the object lock configuration (same as bucket_name).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
		Blocks: map[string]schema.Block{
			"default_retention_setting": schema.SingleNestedBlock{
				Description: "Default retention settings for object lock.",
				Attributes: map[string]schema.Attribute{
					"mode": schema.StringAttribute{
						Description: "The retention mode (compliance or governance).",
						Optional:    true,
						Computed:    true,
						Default:     stringdefault.StaticString("compliance"),
					},
					"days": schema.Int64Attribute{
						Description: "Retention period in days.",
						Optional:    true,
						Computed:    true,
						Default:     int64default.StaticInt64(1),
					},
					"years": schema.Int64Attribute{
						Description: "Retention period in years.",
						Optional:    true,
						Computed:    true,
						Default:     int64default.StaticInt64(1),
					},
				},
			},
		},
	}
}

func (r *S3BucketObjectLockConfigurationResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *S3BucketObjectLockConfigurationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan S3BucketObjectLockConfigurationResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	bucketName := plan.BucketName.ValueString()

	// Get current object lock status to validate this resource can be applied
	currentObjectLock, err := r.client.GetS3BucketObjectLock(bucketName)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to Check Current Object Lock Status for %s", bucketName),
			err.Error(),
		)
		return
	}

	// Validate that object lock is actually enabled on the bucket
	if !currentObjectLock.Enabled {
		resp.Diagnostics.AddError(
			"Object Lock Not Enabled on Bucket",
			fmt.Sprintf("Bucket %s does not have object lock enabled. Object lock must be enabled at bucket creation time using the storagegrid_s3_bucket resource with object_lock_enabled=true. This resource can only be used on buckets that already have object lock enabled.", bucketName),
		)
		return
	}

	var defaultRetentionSetting *utils.DefaultRetentionSetting
	if plan.DefaultRetentionSetting != nil {
		defaultRetentionSetting = &utils.DefaultRetentionSetting{
			Mode: plan.DefaultRetentionSetting.Mode.ValueString(),
		}

		// Only set days OR years, not both - StorageGrid requires exactly one
		days := int(plan.DefaultRetentionSetting.Days.ValueInt64())
		years := int(plan.DefaultRetentionSetting.Years.ValueInt64())

		if years > 0 {
			defaultRetentionSetting.Years = years
		} else {
			defaultRetentionSetting.Days = days
		}
	}

	err = r.client.UpdateS3BucketObjectLock(bucketName, true, defaultRetentionSetting)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to Create S3 Bucket Object Lock Configuration for %s", bucketName),
			err.Error(),
		)
		return
	}

	// Set the ID (same as bucket name)
	plan.ID = types.StringValue(bucketName)

	// Save the plan to state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *S3BucketObjectLockConfigurationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state S3BucketObjectLockConfigurationResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	bucketName := state.BucketName.ValueString()
	objectLock, err := r.client.GetS3BucketObjectLock(bucketName)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to Read S3 Bucket Object Lock Configuration for %s", bucketName),
			err.Error(),
		)
		return
	}

	// Update state with current values
	state.ID = types.StringValue(bucketName)

	// Handle default retention setting
	if objectLock.DefaultRetentionSetting != nil {
		state.DefaultRetentionSetting = &DefaultRetentionSettingResourceModel{
			Mode:  types.StringValue(objectLock.DefaultRetentionSetting.Mode),
			Days:  types.Int64Value(int64(objectLock.DefaultRetentionSetting.Days)),
			Years: types.Int64Value(int64(objectLock.DefaultRetentionSetting.Years)),
		}
	} else {
		state.DefaultRetentionSetting = nil
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *S3BucketObjectLockConfigurationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan S3BucketObjectLockConfigurationResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	bucketName := plan.BucketName.ValueString()

	var defaultRetentionSetting *utils.DefaultRetentionSetting
	if plan.DefaultRetentionSetting != nil {
		defaultRetentionSetting = &utils.DefaultRetentionSetting{
			Mode: plan.DefaultRetentionSetting.Mode.ValueString(),
		}

		// Only set days OR years, not both - StorageGrid requires exactly one
		days := int(plan.DefaultRetentionSetting.Days.ValueInt64())
		years := int(plan.DefaultRetentionSetting.Years.ValueInt64())

		if years > 0 {
			defaultRetentionSetting.Years = years
		} else {
			defaultRetentionSetting.Days = days
		}
	}

	err := r.client.UpdateS3BucketObjectLock(bucketName, true, defaultRetentionSetting)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to Update S3 Bucket Object Lock Configuration for %s", bucketName),
			err.Error(),
		)
		return
	}

	// Save the updated plan to state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *S3BucketObjectLockConfigurationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state S3BucketObjectLockConfigurationResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	bucketName := state.BucketName.ValueString()

	// When deleting object lock configuration, try to disable object lock
	// but if that fails (which it often does), just clear default retention settings
	err := r.client.UpdateS3BucketObjectLock(bucketName, false, nil)
	if err != nil {
		// Check if this is specifically an "Invalid ObjectLockEnabled value" error
		if strings.Contains(err.Error(), "Invalid ObjectLockEnabled value") {
			// Try to just clear the default retention settings instead
			err2 := r.client.UpdateS3BucketObjectLock(bucketName, true, nil)
			if err2 != nil {
				resp.Diagnostics.AddWarning(
					"Cannot Disable Object Lock",
					fmt.Sprintf("Object lock cannot be disabled on bucket %s once enabled. The object lock configuration resource has been removed from Terraform state, but object lock will remain enabled on the bucket with no default retention settings.", bucketName),
				)
			} else {
				resp.Diagnostics.AddWarning(
					"Object Lock Remains Enabled",
					fmt.Sprintf("Object lock cannot be disabled on bucket %s once enabled. Default retention settings have been cleared, but object lock remains active.", bucketName),
				)
			}
			// Continue with the delete - state will be cleared automatically
		} else {
			resp.Diagnostics.AddError(
				fmt.Sprintf("Unable to Delete S3 Bucket Object Lock Configuration for %s", bucketName),
				err.Error(),
			)
			return
		}
	}

	// State is automatically cleared on successful delete
}

func (r *S3BucketObjectLockConfigurationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import using the bucket name as the identifier
	bucketName := req.ID

	// Validate that the bucket exists and get object lock configuration
	objectLock, err := r.client.GetS3BucketObjectLock(bucketName)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to Import S3 Bucket Object Lock Configuration for %s", bucketName),
			fmt.Sprintf("Bucket does not exist or object lock configuration is not accessible: %s", err.Error()),
		)
		return
	}

	// Validate that object lock is enabled on the bucket
	if !objectLock.Enabled {
		resp.Diagnostics.AddError(
			"Object Lock Not Enabled on Bucket",
			fmt.Sprintf("Cannot import object lock configuration for bucket %s because object lock is not enabled. This resource can only be used on buckets that have object lock enabled.", bucketName),
		)
		return
	}

	// Set the imported object lock configuration in state
	state := S3BucketObjectLockConfigurationResourceModel{
		BucketName: types.StringValue(bucketName),
		ID:         types.StringValue(bucketName),
	}

	// Handle default retention setting
	if objectLock.DefaultRetentionSetting != nil {
		state.DefaultRetentionSetting = &DefaultRetentionSettingResourceModel{
			Mode:  types.StringValue(objectLock.DefaultRetentionSetting.Mode),
			Days:  types.Int64Value(int64(objectLock.DefaultRetentionSetting.Days)),
			Years: types.Int64Value(int64(objectLock.DefaultRetentionSetting.Years)),
		}
	}

	// Set the state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)

	// Set the ID attribute explicitly for import
	resource.ImportStatePassthroughID(ctx, path.Root("bucket_name"), req, resp)
}
