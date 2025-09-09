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
	_ datasource.DataSource              = &UserDataSource{}
	_ datasource.DataSourceWithConfigure = &UserDataSource{}
)

// NewUserDataSource is a factory function for the user data source.
func NewUserDataSource() datasource.DataSource {
	return &UserDataSource{}
}

// UserDataSource defines the data source implementation.
type UserDataSource struct {
	client *utils.Client
}

// UserDataSourceModel maps the user data to the Terraform schema.
type UserDataSourceModel struct {
	UserName   types.String   `tfsdk:"user_name"`
	Status     types.String   `tfsdk:"status"`
	APIVersion types.String   `tfsdk:"api_version"`
	Data       *UserDataModel `tfsdk:"data"`
}

// UserDataModel maps the nested 'data' object for a user.
// This struct is updated to reflect the new response body.
type UserDataModel struct {
	ID         types.String `tfsdk:"id"`
	AccountID  types.String `tfsdk:"account_id"`
	FullName   types.String `tfsdk:"full_name"`
	UniqueName types.String `tfsdk:"unique_name"`
	UserURN    types.String `tfsdk:"user_urn"`
	Federated  types.Bool   `tfsdk:"federated"`
	MemberOf   types.List   `tfsdk:"member_of"`
	Disable    types.Bool   `tfsdk:"disable"`
}

// Metadata returns the data source type name.
func (d *UserDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user"
}

// Schema defines the structure of the data source.
// The schema is updated to remove 'policies' and add the new fields.
func (d *UserDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches information about a StorageGrid User.",
		Attributes: map[string]schema.Attribute{
			"user_name": schema.StringAttribute{
				Description: "The unique name of the user to fetch (e.g., 'user/Test').",
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
				Description: "The main data object for the StorageGrid User.",
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						Description: "The unique identifier for the user.",
						Computed:    true,
					},
					"account_id": schema.StringAttribute{
						Description: "The account ID associated with the user.",
						Computed:    true,
					},
					"full_name": schema.StringAttribute{
						Description: "The full name of the user.",
						Computed:    true,
					},
					"unique_name": schema.StringAttribute{
						Description: "The unique name of the user.",
						Computed:    true,
					},
					"user_urn": schema.StringAttribute{
						Description: "The URN of the user.",
						Computed:    true,
					},
					"federated": schema.BoolAttribute{
						Description: "Indicates if the user is federated.",
						Computed:    true,
					},
					"member_of": schema.ListAttribute{
						Description: "A list of group IDs that the user is a member of.",
						Computed:    true,
						ElementType: types.StringType,
					},
					"disable": schema.BoolAttribute{
						Description: "Indicates if the user account is disabled.",
						Computed:    true,
					},
				},
			},
		},
	}
}

// Configure obtains the API client from the provider configuration.
func (d *UserDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

// Read fetches the user data from the API and sets the Terraform state.
func (d *UserDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state UserDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	userName := "user/" + state.UserName.ValueString()
	apiResponse, err := d.client.GetUser(userName)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to Read User %s", userName),
			err.Error(),
		)
		return
	}

	user := apiResponse.Data

	// Map the API response to the Terraform state model
	state.Status = types.StringValue(apiResponse.Status)
	state.APIVersion = types.StringValue(apiResponse.APIVersion)

	// Convert the 'memberOf' string slice to a types.List
	memberOfList, diags := types.ListValueFrom(ctx, types.StringType, user.MemberOf)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state.Data = &UserDataModel{
		ID:         types.StringValue(user.ID),
		AccountID:  types.StringValue(user.AccountID),
		FullName:   types.StringValue(user.FullName),
		UniqueName: types.StringValue(user.UniqueName),
		UserURN:    types.StringValue(user.UserURN),
		Federated:  types.BoolValue(user.Federated),
		MemberOf:   memberOfList,
		Disable:    types.BoolValue(user.Disable),
	}

	// Save the final state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}
