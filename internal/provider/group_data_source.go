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
	_ datasource.DataSource              = &GroupDataSource{}
	_ datasource.DataSourceWithConfigure = &GroupDataSource{}
)

func NewGroupDataSource() datasource.DataSource {
	return &GroupDataSource{}
}

// GroupDataSource defines the data source implementation.
type GroupDataSource struct {
	client *utils.Client
}

type GroupDataSourceModel struct {
	GroupName  types.String    `tfsdk:"group_name"`
	Status     types.String    `tfsdk:"status"`
	APIVersion types.String    `tfsdk:"api_version"`
	Data       *GroupDataModel `tfsdk:"data"`
}

// GroupDataModel maps the nested 'data' object.
type GroupDataModel struct {
	ID                 types.String  `tfsdk:"id"`
	AccountID          types.String  `tfsdk:"account_id"`
	DisplayName        types.String  `tfsdk:"display_name"`
	UniqueName         types.String  `tfsdk:"unique_name"`
	GroupURN           types.String  `tfsdk:"group_urn"`
	Federated          types.Bool    `tfsdk:"federated"`
	ManagementReadOnly types.Bool    `tfsdk:"management_read_only"`
	Policies           PoliciesModel `tfsdk:"policies"`
}

// PoliciesModel maps the nested 'policies' object.
type PoliciesModel struct {
	S3         S3PolicyModel         `tfsdk:"s3"`
	Management ManagementPolicyModel `tfsdk:"management"`
}

// S3PolicyModel maps the nested 's3' policy object.
type S3PolicyModel struct {
	Version   types.String     `tfsdk:"version"`
	Statement []StatementModel `tfsdk:"statement"`
}

// StatementModel maps the objects within the 'Statement' list.
type StatementModel struct {
	Effect   types.String   `tfsdk:"effect"`
	Action   []types.String `tfsdk:"action"`
	Resource []types.String `tfsdk:"resource"`
}

type ManagementPolicyModel struct {
	ManageAllContainers       types.Bool `tfsdk:"manage_all_containers"`
	ManageEndpoints           types.Bool `tfsdk:"manage_endpoints"`
	ManageOwnContainerObjects types.Bool `tfsdk:"manage_own_container_objects"`
	ManageOwnS3Credentials    types.Bool `tfsdk:"manage_own_s3_credentials"`
	RootAccess                types.Bool `tfsdk:"root_access"`
	ViewAllContainers         types.Bool `tfsdk:"view_all_containers"`
}

func (d *GroupDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group"
}

func (d *GroupDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches information about a StorageGrid Group.",
		Attributes: map[string]schema.Attribute{
			"group_name": schema.StringAttribute{
				Description: "The unique name of the group to fetch (e.g., 'group/example').",
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
				Description: "The main data object for the StorageGrid Group.",
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						Description: "The unique identifier for the group.",
						Computed:    true,
					},
					"account_id": schema.StringAttribute{
						Description: "The account ID associated with the group.",
						Computed:    true,
					},
					"display_name": schema.StringAttribute{
						Description: "The display name of the group.",
						Computed:    true,
					},
					"unique_name": schema.StringAttribute{
						Description: "The unique name of the group.",
						Computed:    true,
					},
					"group_urn": schema.StringAttribute{
						Description: "The URN of the group.",
						Computed:    true,
					},
					"federated": schema.BoolAttribute{
						Description: "Indicates if the group is federated.",
						Computed:    true,
					},
					"management_read_only": schema.BoolAttribute{
						Description: "Indicates if the group has read-only management access.",
						Computed:    true,
					},
					"policies": schema.SingleNestedAttribute{
						Description: "Contains the policy definitions for the group.",
						Computed:    true,
						Attributes: map[string]schema.Attribute{
							"s3": schema.SingleNestedAttribute{
								Description: "S3 policy details.",
								Computed:    true,
								Attributes: map[string]schema.Attribute{
									"version": schema.StringAttribute{
										Description: "The version of the policy.",
										Computed:    true,
									},
									"statement": schema.ListNestedAttribute{
										Description: "A list of policy statements.",
										Computed:    true,
										NestedObject: schema.NestedAttributeObject{
											Attributes: map[string]schema.Attribute{
												"effect": schema.StringAttribute{
													Description: "The effect of the statement (e.g., 'Allow' or 'Deny').",
													Computed:    true,
												},
												"action": schema.ListAttribute{
													Description: "A list of actions allowed or denied by the statement.",
													Computed:    true,
													ElementType: types.StringType,
												},
												"resource": schema.ListAttribute{
													Description: "A list of resources to which the statement applies.",
													Computed:    true,
													ElementType: types.StringType,
												},
											},
										},
									},
								},
							},
							"management": schema.SingleNestedAttribute{
								Description: "Management policy details.",
								Computed:    true,
								Attributes: map[string]schema.Attribute{
									"manage_all_containers": schema.BoolAttribute{
										Description: "Permission to manage all containers.",
										Computed:    true,
									},
									"manage_endpoints": schema.BoolAttribute{
										Description: "Permission to manage endpoints.",
										Computed:    true,
									},
									"manage_own_container_objects": schema.BoolAttribute{
										Description: "Permission to manage objects in own containers.",
										Computed:    true,
									},
									"manage_own_s3_credentials": schema.BoolAttribute{
										Description: "Permission to manage own S3 credentials.",
										Computed:    true,
									},
									"root_access": schema.BoolAttribute{
										Description: "Root access permissions.",
										Computed:    true,
									},
									"view_all_containers": schema.BoolAttribute{
										Description: "Permission to view all containers.",
										Computed:    true,
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func (d *GroupDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *GroupDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state GroupDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	groupName := "group/" + state.GroupName.ValueString()
	apiResponse, err := d.client.GetGroup(groupName)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to Read Group %s", groupName),
			err.Error(),
		)
		return
	}

	group := apiResponse.Data

	// Map API response data to the Terraform state model
	state.Status = types.StringValue(apiResponse.Status)
	state.APIVersion = types.StringValue(apiResponse.APIVersion)

	state.Data = &GroupDataModel{
		ID:                 types.StringValue(group.ID),
		AccountID:          types.StringValue(group.AccountID),
		DisplayName:        types.StringValue(group.DisplayName),
		UniqueName:         types.StringValue(group.UniqueName),
		GroupURN:           types.StringValue(group.GroupURN),
		Federated:          types.BoolValue(group.Federated),
		ManagementReadOnly: types.BoolValue(group.ManagementReadOnly),
		Policies: PoliciesModel{
			S3: S3PolicyModel{
				Version: types.StringValue(group.Policies.S3.Version),
			},
			Management: ManagementPolicyModel{
				ManageAllContainers:       types.BoolValue(group.Policies.Management.ManageAllContainers),
				ManageEndpoints:           types.BoolValue(group.Policies.Management.ManageEndpoints),
				ManageOwnContainerObjects: types.BoolValue(group.Policies.Management.ManageOwnContainerObjects),
				ManageOwnS3Credentials:    types.BoolValue(group.Policies.Management.ManageOwnS3Credentials),
				RootAccess:                types.BoolValue(group.Policies.Management.RootAccess),
				ViewAllContainers:         types.BoolValue(group.Policies.Management.ViewAllContainers),
			},
		},
	}

	// Map S3 policy statements
	var statements []StatementModel
	for _, stmt := range group.Policies.S3.Statement {
		statementState := StatementModel{
			Effect:   types.StringValue(stmt.Effect),
			Action:   make([]types.String, len(stmt.Action)),
			Resource: make([]types.String, len(stmt.Resource)),
		}
		for i, action := range stmt.Action {
			statementState.Action[i] = types.StringValue(action)
		}
		for i, resource := range stmt.Resource {
			statementState.Resource[i] = types.StringValue(resource)
		}
		statements = append(statements, statementState)
	}
	state.Data.Policies.S3.Statement = statements

	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
