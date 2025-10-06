// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	awspolicy "github.com/hashicorp/awspolicyequivalence"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/team-fenrir/terraform-provider-storagegrid/internal/utils"
)

var (
	_ resource.Resource                = &GroupResource{}
	_ resource.ResourceWithConfigure   = &GroupResource{}
	_ resource.ResourceWithImportState = &GroupResource{}
)

var managementAttributeTypes = map[string]attr.Type{
	"manage_all_containers":        types.BoolType,
	"manage_endpoints":             types.BoolType,
	"manage_own_container_objects": types.BoolType,
	"manage_own_s3_credentials":    types.BoolType,
	"root_access":                  types.BoolType,
	"view_all_containers":          types.BoolType,
}

// normalizeDisplayName returns a plan modifier that sets display_name to match group_name.
func normalizeDisplayName() planmodifier.String {
	return &normalizeDisplayNameModifier{}
}

type normalizeDisplayNameModifier struct{}

func (m *normalizeDisplayNameModifier) Description(ctx context.Context) string {
	return "Normalizes display_name to match group_name"
}

func (m *normalizeDisplayNameModifier) MarkdownDescription(ctx context.Context) string {
	return "Normalizes display_name to match group_name"
}

func (m *normalizeDisplayNameModifier) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	// If we don't have a group_name in the plan, we can't normalize
	var plan GroupResourceModel
	if diags := req.Plan.Get(ctx, &plan); diags.HasError() {
		// Fall back to UseStateForUnknown behavior
		if req.StateValue.IsNull() {
			resp.PlanValue = types.StringUnknown()
		} else {
			resp.PlanValue = req.StateValue
		}
		return
	}

	// Set display_name to match group_name
	if !plan.GroupName.IsNull() && !plan.GroupName.IsUnknown() {
		resp.PlanValue = types.StringValue(plan.GroupName.ValueString())
	} else if req.StateValue.IsNull() {
		resp.PlanValue = types.StringUnknown()
	} else {
		resp.PlanValue = req.StateValue
	}
}

func NewGroupResource() resource.Resource {
	return &GroupResource{}
}

type GroupResource struct {
	client *utils.Client
}

type GroupResourceModel struct {
	GroupName          types.String          `tfsdk:"group_name"`
	Policies           PoliciesResourceModel `tfsdk:"policies"`
	ID                 types.String          `tfsdk:"id"`
	AccountID          types.String          `tfsdk:"account_id"`
	DisplayName        types.String          `tfsdk:"display_name"`
	UniqueName         types.String          `tfsdk:"unique_name"`
	GroupURN           types.String          `tfsdk:"group_urn"`
	Federated          types.Bool            `tfsdk:"federated"`
	ManagementReadOnly types.Bool            `tfsdk:"management_read_only"`
}

type PoliciesResourceModel struct {
	S3         types.String          `tfsdk:"s3"`
	Management ManagementPolicyModel `tfsdk:"management"`
}

func (r *GroupResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group"
}

func (r *GroupResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a StorageGrid Group.",
		Attributes: map[string]schema.Attribute{
			"group_name": schema.StringAttribute{
				Required:    true,
				Description: "The unique name for the group (e.g., 'my-new-group'). The 'group/' prefix is added automatically. This cannot be changed after creation.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"policies": schema.SingleNestedAttribute{
				Required:    true,
				Description: "Contains the policy definitions for the group.",
				Attributes: map[string]schema.Attribute{
					"s3": schema.StringAttribute{
						Required:    true,
						Description: "The S3 policy for the group, provided as a JSON string. Use the `file()` function to load from a file.",
						PlanModifiers: []planmodifier.String{
							suppressS3PolicyDiffs(),
						},
					},
					"management": schema.SingleNestedAttribute{
						Optional:    true,
						Computed:    true,
						Description: "Management policy permissions for the group. If omitted, all permissions default to false.",
						Default: objectdefault.StaticValue(
							types.ObjectValueMust(
								managementAttributeTypes,
								map[string]attr.Value{
									"manage_all_containers":        types.BoolValue(false),
									"manage_endpoints":             types.BoolValue(false),
									"manage_own_container_objects": types.BoolValue(false),
									"manage_own_s3_credentials":    types.BoolValue(false),
									"root_access":                  types.BoolValue(false),
									"view_all_containers":          types.BoolValue(false),
								},
							),
						),
						Attributes: map[string]schema.Attribute{
							"manage_all_containers": schema.BoolAttribute{
								Description: "Permission to manage all containers.",
								Optional:    true,
								Computed:    true,
								Default:     booldefault.StaticBool(false),
							},
							"manage_endpoints": schema.BoolAttribute{
								Description: "Permission to manage endpoints.",
								Optional:    true,
								Computed:    true,
								Default:     booldefault.StaticBool(false),
							},
							"manage_own_container_objects": schema.BoolAttribute{
								Description: "Permission to manage objects in own containers.",
								Optional:    true,
								Computed:    true,
								Default:     booldefault.StaticBool(false),
							},
							"manage_own_s3_credentials": schema.BoolAttribute{
								Description: "Permission to manage own S3 credentials.",
								Optional:    true,
								Computed:    true,
								Default:     booldefault.StaticBool(false),
							},
							"root_access": schema.BoolAttribute{
								Description: "Grants root access permissions to the group.",
								Optional:    true,
								Computed:    true,
								Default:     booldefault.StaticBool(false),
							},
							"view_all_containers": schema.BoolAttribute{
								Description: "Permission to view all containers.",
								Optional:    true,
								Computed:    true,
								Default:     booldefault.StaticBool(false),
							},
						},
					},
				},
			},
			"id": schema.StringAttribute{
				Description: "The unique identifier (ID) for the group, generated by StorageGrid.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"account_id": schema.StringAttribute{
				Description: "The account ID associated with the group.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"display_name": schema.StringAttribute{
				Description: "The display name of the group.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					normalizeDisplayName(),
				},
			},
			"unique_name": schema.StringAttribute{
				Description: "The canonical unique name of the group.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"group_urn": schema.StringAttribute{
				Description: "The URN of the group.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"federated": schema.BoolAttribute{
				Description: "Indicates if the group is federated.",
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"management_read_only": schema.BoolAttribute{
				Description: "Indicates if the group has read-only management access.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
		},
	}
}
func (r *GroupResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
	r.client = client
}

func (r *GroupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan GroupResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var s3Payload utils.S3Policy
	s3PolicyString := plan.Policies.S3.ValueString()
	err := json.Unmarshal([]byte(s3PolicyString), &s3Payload)
	if err != nil {
		resp.Diagnostics.AddError("Invalid S3 Policy JSON", "Could not unmarshal the provided S3 policy string: "+err.Error())
		return
	}

	managementPayload := utils.ManagementPolicy{
		ManageAllContainers:       plan.Policies.Management.ManageAllContainers.ValueBool(),
		ManageEndpoints:           plan.Policies.Management.ManageEndpoints.ValueBool(),
		ManageOwnContainerObjects: plan.Policies.Management.ManageOwnContainerObjects.ValueBool(),
		ManageOwnS3Credentials:    plan.Policies.Management.ManageOwnS3Credentials.ValueBool(),
		RootAccess:                plan.Policies.Management.RootAccess.ValueBool(),
		ViewAllContainers:         plan.Policies.Management.ViewAllContainers.ValueBool(),
	}
	groupName := plan.GroupName.ValueString()

	apiRequest := utils.GroupPayload{
		UniqueName:         "group/" + groupName,
		DisplayName:        groupName,
		ManagementReadOnly: plan.ManagementReadOnly.ValueBool(),
		Policies: utils.Policies{
			S3:         s3Payload,
			Management: managementPayload,
		},
	}

	createdGroup, err := r.client.CreateGroup(apiRequest)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Error creating StorageGrid Group: %s", groupName), "Could not create group, unexpected error: "+err.Error())
		return
	}

	groupData := createdGroup.Data
	plan.ID = types.StringValue(groupData.ID)
	plan.DisplayName = types.StringValue(groupData.DisplayName)
	plan.UniqueName = types.StringValue(groupData.UniqueName)
	plan.AccountID = types.StringValue(groupData.AccountID)
	plan.GroupURN = types.StringValue(groupData.GroupURN)
	plan.Federated = types.BoolValue(groupData.Federated)
	plan.ManagementReadOnly = types.BoolValue(groupData.ManagementReadOnly)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *GroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state GroupResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	groupNameFromState := state.ID.ValueString()
	id := state.ID.ValueString()
	apiGroup, err := r.client.GetGroup(id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading StorageGrid Group", fmt.Sprintf("Could not read StorageGrid group %s: %s", groupNameFromState, err.Error()))
		return
	}

	groupData := apiGroup.Data

	state.ID = types.StringValue(groupData.ID)
	state.GroupName = types.StringValue(strings.TrimPrefix(groupData.UniqueName, "group/"))
	state.DisplayName = types.StringValue(groupData.DisplayName)
	state.UniqueName = types.StringValue(groupData.UniqueName)
	state.AccountID = types.StringValue(groupData.AccountID)
	state.GroupURN = types.StringValue(groupData.GroupURN)
	state.Federated = types.BoolValue(groupData.Federated)
	state.ManagementReadOnly = types.BoolValue(groupData.ManagementReadOnly)

	state.Policies.Management = ManagementPolicyModel{
		ManageAllContainers:       types.BoolValue(groupData.Policies.Management.ManageAllContainers),
		ManageEndpoints:           types.BoolValue(groupData.Policies.Management.ManageEndpoints),
		ManageOwnContainerObjects: types.BoolValue(groupData.Policies.Management.ManageOwnContainerObjects),
		ManageOwnS3Credentials:    types.BoolValue(groupData.Policies.Management.ManageOwnS3Credentials),
		RootAccess:                types.BoolValue(groupData.Policies.Management.RootAccess),
		ViewAllContainers:         types.BoolValue(groupData.Policies.Management.ViewAllContainers),
	}

	s3PolicyFromAPIBytes, err := json.Marshal(groupData.Policies.S3)
	if err != nil {
		resp.Diagnostics.AddError("Error Processing S3 Policy", "Could not marshal S3 policy from API into string: "+err.Error())
		return
	}

	equal, err := awspolicy.PoliciesAreEquivalent(string(s3PolicyFromAPIBytes), state.Policies.S3.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("JSON Comparison Error", "Failed to compare S3 policies: "+err.Error())
		return
	}

	if !equal {
		state.Policies.S3 = types.StringValue(string(s3PolicyFromAPIBytes))
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
func (r *GroupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan GroupResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	var state GroupResourceModel
	diags.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	var s3Payload utils.S3Policy
	s3PolicyString := plan.Policies.S3.ValueString()
	if err := json.Unmarshal([]byte(s3PolicyString), &s3Payload); err != nil {
		resp.Diagnostics.AddError("Invalid S3 Policy JSON", "Could not unmarshal the provided S3 policy string: "+err.Error())
		return
	}

	managementPayload := utils.ManagementPolicy{
		ManageAllContainers:       plan.Policies.Management.ManageAllContainers.ValueBool(),
		ManageEndpoints:           plan.Policies.Management.ManageEndpoints.ValueBool(),
		ManageOwnContainerObjects: plan.Policies.Management.ManageOwnContainerObjects.ValueBool(),
		ManageOwnS3Credentials:    plan.Policies.Management.ManageOwnS3Credentials.ValueBool(),
		RootAccess:                plan.Policies.Management.RootAccess.ValueBool(),
		ViewAllContainers:         plan.Policies.Management.ViewAllContainers.ValueBool(),
	}

	groupName := state.GroupName.ValueString()
	apiRequest := utils.GroupPayload{
		UniqueName:         "group/" + groupName,
		DisplayName:        groupName,
		ManagementReadOnly: plan.ManagementReadOnly.ValueBool(),
		Policies: utils.Policies{
			S3:         s3Payload,
			Management: managementPayload,
		},
	}
	id := state.ID.ValueString()
	_, err := r.client.UpdateGroup(id, apiRequest)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating StorageGrid Group",
			fmt.Sprintf("Could not update group policies for ID %s: %s", groupName, err.Error()),
		)
		return
	}

	updatedGroup, err := r.client.GetGroup(id)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading StorageGrid Group",
			fmt.Sprintf("Could not read updated group data for %s after update: %s", groupName, err.Error()),
		)
		return
	}
	groupData := updatedGroup.Data

	plan.Policies.Management.ManageAllContainers = types.BoolValue(groupData.Policies.Management.ManageAllContainers)
	plan.Policies.Management.ManageEndpoints = types.BoolValue(groupData.Policies.Management.ManageEndpoints)
	plan.Policies.Management.ManageOwnContainerObjects = types.BoolValue(groupData.Policies.Management.ManageOwnContainerObjects)
	plan.Policies.Management.ManageOwnS3Credentials = types.BoolValue(groupData.Policies.Management.ManageOwnS3Credentials)
	plan.Policies.Management.RootAccess = types.BoolValue(groupData.Policies.Management.RootAccess)
	plan.Policies.Management.ViewAllContainers = types.BoolValue(groupData.Policies.Management.ViewAllContainers)

	plan.ID = types.StringValue(groupData.ID)
	plan.DisplayName = types.StringValue(groupData.DisplayName)
	plan.UniqueName = types.StringValue(groupData.UniqueName)
	plan.AccountID = types.StringValue(groupData.AccountID)
	plan.GroupURN = types.StringValue(groupData.GroupURN)
	plan.Federated = types.BoolValue(groupData.Federated)
	plan.ManagementReadOnly = types.BoolValue(groupData.ManagementReadOnly)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
func (r *GroupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state GroupResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.ID.ValueString()
	err := r.client.DeleteGroup(id)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting StorageGrid Group",
			fmt.Sprintf("Could not delete group with ID %s: %s", id, err.Error()),
		)
		return
	}
}

func (r *GroupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	groupName := req.ID

	// The API expects the unique name to be prefixed with "group/".
	apiUniqueName := "group/" + groupName

	apiGroup, err := r.client.GetGroup(apiUniqueName)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			resp.Diagnostics.AddError(
				"Group Not Found",
				fmt.Sprintf("Cannot import a group with unique name '%s' because it does not exist.", groupName),
			)
			return
		}
		resp.Diagnostics.AddError(
			"Error Importing StorageGrid Group",
			fmt.Sprintf("Could not import StorageGrid group with unique name %s: %s", groupName, err.Error()),
		)
		return
	}

	groupData := apiGroup.Data
	var state GroupResourceModel

	state.ID = types.StringValue(groupData.ID)

	groupName = strings.TrimPrefix(groupData.UniqueName, "group/")
	state.GroupName = types.StringValue(groupName)
	state.DisplayName = types.StringValue(groupName)
	state.UniqueName = types.StringValue(groupData.UniqueName)
	state.AccountID = types.StringValue(groupData.AccountID)
	state.GroupURN = types.StringValue(groupData.GroupURN)
	state.Federated = types.BoolValue(groupData.Federated)
	state.ManagementReadOnly = types.BoolValue(groupData.ManagementReadOnly)

	// Populate nested management policy object.
	state.Policies.Management = ManagementPolicyModel{
		ManageAllContainers:       types.BoolValue(groupData.Policies.Management.ManageAllContainers),
		ManageEndpoints:           types.BoolValue(groupData.Policies.Management.ManageEndpoints),
		ManageOwnContainerObjects: types.BoolValue(groupData.Policies.Management.ManageOwnContainerObjects),
		ManageOwnS3Credentials:    types.BoolValue(groupData.Policies.Management.ManageOwnS3Credentials),
		RootAccess:                types.BoolValue(groupData.Policies.Management.RootAccess),
		ViewAllContainers:         types.BoolValue(groupData.Policies.Management.ViewAllContainers),
	}

	// Marshal the S3 policy from the API into a string.
	s3PolicyFromAPIBytes, err := json.Marshal(groupData.Policies.S3)
	if err != nil {
		resp.Diagnostics.AddError("Error Processing S3 Policy on Import", "Could not marshal S3 policy from API into string: "+err.Error())
		return
	}
	state.Policies.S3 = types.StringValue(string(s3PolicyFromAPIBytes))

	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
