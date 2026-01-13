// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/team-fenrir/terraform-provider-storagegrid/internal/utils"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource                = &S3BucketLifecycleConfigurationResource{}
	_ resource.ResourceWithConfigure   = &S3BucketLifecycleConfigurationResource{}
	_ resource.ResourceWithImportState = &S3BucketLifecycleConfigurationResource{}
)

func NewS3BucketLifecycleConfigurationResource() resource.Resource {
	return &S3BucketLifecycleConfigurationResource{}
}

// S3BucketLifecycleConfigurationResource defines the resource implementation.
type S3BucketLifecycleConfigurationResource struct {
	client *utils.Client
}

// S3BucketLifecycleConfigurationResourceModel describes the resource data model.
type S3BucketLifecycleConfigurationResourceModel struct {
	BucketName types.String                 `tfsdk:"bucket_name"`
	Rules      []LifecycleRuleResourceModel `tfsdk:"rule"`
	ID         types.String                 `tfsdk:"id"`
}

// LifecycleRuleResourceModel represents a lifecycle rule.
type LifecycleRuleResourceModel struct {
	ID                          types.String                             `tfsdk:"id"`
	Status                      types.String                             `tfsdk:"status"`
	Filter                      *LifecycleFilterResourceModel            `tfsdk:"filter"`
	Expiration                  *LifecycleExpirationResourceModel        `tfsdk:"expiration"`
	NoncurrentVersionExpiration *LifecycleNoncurrentVersionResourceModel `tfsdk:"noncurrent_version_expiration"`
}

// LifecycleFilterResourceModel represents a lifecycle rule filter.
type LifecycleFilterResourceModel struct {
	Prefix types.String `tfsdk:"prefix"`
}

// LifecycleExpirationResourceModel represents expiration settings.
type LifecycleExpirationResourceModel struct {
	Days types.Int64  `tfsdk:"days"`
	Date types.String `tfsdk:"date"`
}

// LifecycleNoncurrentVersionResourceModel represents noncurrent version expiration settings.
type LifecycleNoncurrentVersionResourceModel struct {
	NoncurrentDays types.Int64 `tfsdk:"noncurrent_days"`
}

func (r *S3BucketLifecycleConfigurationResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_s3_bucket_lifecycle_configuration"
}

func (r *S3BucketLifecycleConfigurationResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages lifecycle configuration for a StorageGrid S3 bucket.",
		Attributes: map[string]schema.Attribute{
			"bucket_name": schema.StringAttribute{
				Description: "The name of the S3 bucket to configure lifecycle for.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"id": schema.StringAttribute{
				Description: "The unique identifier for the lifecycle configuration (same as bucket_name).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
		Blocks: map[string]schema.Block{
			"rule": schema.ListNestedBlock{
				Description: "Lifecycle rules for the bucket.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "Unique identifier for the rule.",
							Optional:    true,
							Computed:    true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
						"status": schema.StringAttribute{
							Description: "Status of the rule (Enabled or Disabled).",
							Required:    true,
						},
					},
					Blocks: map[string]schema.Block{
						"filter": schema.SingleNestedBlock{
							Description: "Filter for the lifecycle rule.",
							Attributes: map[string]schema.Attribute{
								"prefix": schema.StringAttribute{
									Description: "Object key prefix that identifies the objects to which the rule applies.",
									Optional:    true,
								},
							},
						},
						"expiration": schema.SingleNestedBlock{
							Description: "Expiration settings for current object versions.",
							Attributes: map[string]schema.Attribute{
								"days": schema.Int64Attribute{
									Description: "Number of days after object creation when the object expires.",
									Optional:    true,
								},
								"date": schema.StringAttribute{
									Description: "Date when objects expire (ISO 8601 format).",
									Optional:    true,
								},
							},
						},
						"noncurrent_version_expiration": schema.SingleNestedBlock{
							Description: "Expiration settings for noncurrent object versions.",
							Attributes: map[string]schema.Attribute{
								"noncurrent_days": schema.Int64Attribute{
									Description: "Number of days after an object becomes noncurrent when it expires.",
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

func (r *S3BucketLifecycleConfigurationResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *S3BucketLifecycleConfigurationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan S3BucketLifecycleConfigurationResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	bucketName := plan.BucketName.ValueString()

	// Convert Terraform model to API model
	lifecycleConfig := &utils.LifecycleConfiguration{
		Rules: make([]utils.Rule, len(plan.Rules)),
	}

	for i, rule := range plan.Rules {
		apiRule := utils.Rule{
			ID:     rule.ID.ValueString(),
			Status: rule.Status.ValueString(),
		}

		// Handle filter
		if rule.Filter != nil {
			apiRule.Filter = &utils.Filter{
				Prefix: rule.Filter.Prefix.ValueString(),
			}
		}

		// Handle expiration
		if rule.Expiration != nil {
			apiRule.Expiration = &utils.Expiration{}
			if !rule.Expiration.Days.IsNull() {
				apiRule.Expiration.Days = int(rule.Expiration.Days.ValueInt64())
			}
			if !rule.Expiration.Date.IsNull() {
				apiRule.Expiration.Date = rule.Expiration.Date.ValueString()
			}
		}

		// Handle noncurrent version expiration
		if rule.NoncurrentVersionExpiration != nil {
			apiRule.NoncurrentVersionExpiration = &utils.NoncurrentVersionExpiration{
				NoncurrentDays: int(rule.NoncurrentVersionExpiration.NoncurrentDays.ValueInt64()),
			}
		}

		lifecycleConfig.Rules[i] = apiRule
	}

	err := r.client.PutS3BucketLifecycleConfiguration(bucketName, lifecycleConfig)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to Create S3 Bucket Lifecycle Configuration for %s", bucketName),
			err.Error(),
		)
		return
	}

	// Set the ID (same as bucket name)
	plan.ID = types.StringValue(bucketName)

	// Save the plan to state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *S3BucketLifecycleConfigurationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state S3BucketLifecycleConfigurationResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	bucketName := state.BucketName.ValueString()
	lifecycleConfig, err := r.client.GetS3BucketLifecycleConfiguration(bucketName)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to Read S3 Bucket Lifecycle Configuration for %s", bucketName),
			err.Error(),
		)
		return
	}

	// Convert API model to Terraform model
	var rules []LifecycleRuleResourceModel
	for _, rule := range lifecycleConfig.Rules {
		ruleModel := LifecycleRuleResourceModel{
			ID:     types.StringValue(rule.ID),
			Status: types.StringValue(rule.Status),
		}

		// Handle filter
		if rule.Filter != nil {
			ruleModel.Filter = &LifecycleFilterResourceModel{
				Prefix: types.StringValue(rule.Filter.Prefix),
			}
		}

		// Handle expiration
		if rule.Expiration != nil {
			ruleModel.Expiration = &LifecycleExpirationResourceModel{}
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
			ruleModel.NoncurrentVersionExpiration = &LifecycleNoncurrentVersionResourceModel{
				NoncurrentDays: types.Int64Value(int64(rule.NoncurrentVersionExpiration.NoncurrentDays)),
			}
		}

		rules = append(rules, ruleModel)
	}

	state.Rules = rules
	state.ID = types.StringValue(bucketName)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *S3BucketLifecycleConfigurationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan S3BucketLifecycleConfigurationResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	bucketName := plan.BucketName.ValueString()

	// Convert Terraform model to API model
	lifecycleConfig := &utils.LifecycleConfiguration{
		Rules: make([]utils.Rule, len(plan.Rules)),
	}

	for i, rule := range plan.Rules {
		apiRule := utils.Rule{
			ID:     rule.ID.ValueString(),
			Status: rule.Status.ValueString(),
		}

		// Handle filter
		if rule.Filter != nil {
			apiRule.Filter = &utils.Filter{
				Prefix: rule.Filter.Prefix.ValueString(),
			}
		}

		// Handle expiration
		if rule.Expiration != nil {
			apiRule.Expiration = &utils.Expiration{}
			if !rule.Expiration.Days.IsNull() {
				apiRule.Expiration.Days = int(rule.Expiration.Days.ValueInt64())
			}
			if !rule.Expiration.Date.IsNull() {
				apiRule.Expiration.Date = rule.Expiration.Date.ValueString()
			}
		}

		// Handle noncurrent version expiration
		if rule.NoncurrentVersionExpiration != nil {
			apiRule.NoncurrentVersionExpiration = &utils.NoncurrentVersionExpiration{
				NoncurrentDays: int(rule.NoncurrentVersionExpiration.NoncurrentDays.ValueInt64()),
			}
		}

		lifecycleConfig.Rules[i] = apiRule
	}

	err := r.client.PutS3BucketLifecycleConfiguration(bucketName, lifecycleConfig)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to Update S3 Bucket Lifecycle Configuration for %s", bucketName),
			err.Error(),
		)
		return
	}

	// Save the updated plan to state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *S3BucketLifecycleConfigurationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state S3BucketLifecycleConfigurationResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	bucketName := state.BucketName.ValueString()

	err := r.client.DeleteS3BucketLifecycleConfiguration(bucketName)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to Delete S3 Bucket Lifecycle Configuration for %s", bucketName),
			err.Error(),
		)
		return
	}

	// State is automatically cleared on successful delete
}

func (r *S3BucketLifecycleConfigurationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import using the bucket name as the identifier
	bucketName := req.ID

	// Validate that the bucket exists and get lifecycle configuration
	lifecycleConfig, err := r.client.GetS3BucketLifecycleConfiguration(bucketName)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to Import S3 Bucket Lifecycle Configuration for %s", bucketName),
			fmt.Sprintf("Bucket does not exist or lifecycle configuration is not accessible: %s", err.Error()),
		)
		return
	}

	// Convert API model to Terraform model
	var rules []LifecycleRuleResourceModel
	for _, rule := range lifecycleConfig.Rules {
		ruleModel := LifecycleRuleResourceModel{
			ID:     types.StringValue(rule.ID),
			Status: types.StringValue(rule.Status),
		}

		// Handle filter
		if rule.Filter != nil {
			ruleModel.Filter = &LifecycleFilterResourceModel{
				Prefix: types.StringValue(rule.Filter.Prefix),
			}
		}

		// Handle expiration
		if rule.Expiration != nil {
			ruleModel.Expiration = &LifecycleExpirationResourceModel{}
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
			ruleModel.NoncurrentVersionExpiration = &LifecycleNoncurrentVersionResourceModel{
				NoncurrentDays: types.Int64Value(int64(rule.NoncurrentVersionExpiration.NoncurrentDays)),
			}
		}

		rules = append(rules, ruleModel)
	}

	// Set the imported lifecycle configuration in state
	state := S3BucketLifecycleConfigurationResourceModel{
		BucketName: types.StringValue(bucketName),
		Rules:      rules,
		ID:         types.StringValue(bucketName),
	}

	// Set the state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)

	// Set the ID attribute explicitly for import
	resource.ImportStatePassthroughID(ctx, path.Root("bucket_name"), req, resp)
}
