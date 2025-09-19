// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"os"

	"github.com/team-fenrir/terraform-provider-storagegrid/internal/utils"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure StorageGridProvider satisfies various provider interfaces.
var _ provider.Provider = &StorageGridProvider{}

// StorageGridProvider defines the provider implementation.
type StorageGridProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// StorageGridProviderModel describes the provider data model.
type StorageGridProviderModel struct {
	Endpoint  types.String `tfsdk:"endpoint"`
	AccountID types.String `tfsdk:"accountid"`
	Username  types.String `tfsdk:"username"`
	Password  types.String `tfsdk:"password"`
}

func (p *StorageGridProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "storagegrid"
	resp.Version = p.version
}

// Schema defines the provider-level schema for configuration data.
func (p *StorageGridProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The StorageGrid provider enables Terraform management of StorageGrid IAM resources including users, groups, and access keys. This provider requires StorageGrid with v4 API support and is not compatible with older StorageGrid versions that only support v3 or earlier APIs.",
		Attributes: map[string]schema.Attribute{
			"endpoint": schema.StringAttribute{
				Description: "URI for StorageGrid API. May also be provided via STORAGEGRID_ENDPOINT environment variable.",
				Optional:    true,
			},
			"accountid": schema.StringAttribute{
				Description: "Account ID for target StorageGrid tenant. May also be provided via STORAGEGRID_ACCOUNTID environment variable.",
				Optional:    true,
			},
			"username": schema.StringAttribute{
				Description: "Username for StorageGrid tenant. May also be provided via STORAGEGRID_USERNAME environment variable.",
				Optional:    true,
			},
			"password": schema.StringAttribute{
				Description: "Password for StorageGrid tenant. May also be provided via STORAGEGRID_PASSWORD environment variable.",
				Optional:    true,
				Sensitive:   true,
			},
		},
	}
}

func (p *StorageGridProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	tflog.Info(ctx, "Configuring StorageGrid client")
	// Retrieve provider data from configuration
	var config StorageGridProviderModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// If practitioner provided a configuration value for any of the
	// attributes, it must be a known value.

	if config.Endpoint.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("endpoint"),
			"Unknown StorageGrid API Endpoint",
			"The provider cannot create the StorageGrid API client as there is an unknown configuration value for the StorageGrid API endpoint. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the STORAGEGRID_ENDPOINT environment variable.",
		)
	}

	if config.AccountID.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("accountid"),
			"Unknown StorageGrid API Account ID",
			"The provider cannot create the StorageGrid API client as there is an unknown configuration value for the StorageGrid API account ID. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the STORAGEGRID_ACCOUNTID environment variable.",
		)
	}

	if config.Username.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("username"),
			"Unknown StorageGrid API Username",
			"The provider cannot create the StorageGrid API client as there is an unknown configuration value for the StorageGrid API username. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the STORAGEGRID_USERNAME environment variable.",
		)
	}

	if config.Password.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("password"),
			"Unknown StorageGrid API Password",
			"The provider cannot create the StorageGrid API client as there is an unknown configuration value for the StorageGrid API password. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the STORAGEGRID_PASSWORD environment variable.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	// Default values to environment variables, but override
	// with Terraform configuration value if set.
	// CHOICE: this means that ENV vars overwrite what's in the TF object!
	endpoint := os.Getenv("STORAGEGRID_ENDPOINT")
	accountID := os.Getenv("STORAGEGRID_ACCOUNTID")
	username := os.Getenv("STORAGEGRID_USERNAME")
	password := os.Getenv("STORAGEGRID_PASSWORD")

	if !config.Endpoint.IsNull() {
		endpoint = config.Endpoint.ValueString()
	}

	if !config.AccountID.IsNull() {
		accountID = config.AccountID.ValueString()
	}

	if !config.Username.IsNull() {
		username = config.Username.ValueString()
	}

	if !config.Password.IsNull() {
		password = config.Password.ValueString()
	}

	// If any of the expected configurations are missing, return
	// errors with provider-specific guidance.

	if endpoint == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("endpoint"),
			"Missing StorageGrid API Endpoint",
			"The provider cannot create the StorageGrid API client as there is a missing or empty value for the StorageGrid API endpoint. "+
				"Set the endpoint value in the configuration or use the STORAGEGRID_ENDPOINT environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}

	if accountID == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("accountid"),
			"Missing StorageGrid API AccountID",
			"The provider cannot create the StorageGrid API client as there is a missing or empty value for the StorageGrid API accountID. "+
				"Set the accountID value in the configuration or use the STORAGEGRID_ACCOUNTID environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}

	if username == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("username"),
			"Missing StorageGrid API Username",
			"The provider cannot create the StorageGrid API client as there is a missing or empty value for the StorageGrid API username. "+
				"Set the username value in the configuration or use the STORAGEGRID_USERNAME environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}

	if password == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("password"),
			"Missing StorageGrid API Password",
			"The provider cannot create the StorageGrid API client as there is a missing or empty value for the StorageGrid API password. "+
				"Set the password value in the configuration or use the STORAGEGRID_PASSWORD environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	ctx = tflog.SetField(ctx, "storagegrid_endpoint", endpoint)
	ctx = tflog.SetField(ctx, "storagegrid_account_id", accountID)
	ctx = tflog.SetField(ctx, "storagegrid_username", username)

	tflog.Debug(ctx, "Creating StorageGrid client")
	client, err := utils.NewClient(&endpoint, &accountID, &username, &password)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create StorageGrid API Client",
			"An unexpected error occurred when creating the StorageGrid API client. "+
				"If the error is not clear, please contact the provider developers.\n\n"+
				"StorageGrid Client Error: "+err.Error(),
		)
		return
	}

	// Make the StorageGrid client available during DataSource and Resource
	// type Configure methods.
	resp.DataSourceData = client
	resp.ResourceData = client

	tflog.Info(ctx, "Configured StorageGrid client", map[string]any{"success": true})
}

func (p *StorageGridProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewGroupResource,
		NewUserResource,
		NewAccessKeysResource,
		NewS3BucketResource,
		NewS3BucketVersioningResource,
	}
}
func (p *StorageGridProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewGroupDataSource,
		NewUserDataSource,
		NewS3BucketDataSource,
		NewS3BucketVersioningDataSource,
		NewS3BucketObjectLockConfigurationDataSource,
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &StorageGridProvider{
			version: version,
		}
	}
}
