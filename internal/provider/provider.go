// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/team-fenrir/terraform-provider-storagegrid/internal/utils"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
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
	Endpoints       *EndpointsModel `tfsdk:"endpoints"`
	AccountID       types.String    `tfsdk:"accountid"`
	Username        types.String    `tfsdk:"username"`
	Password        types.String    `tfsdk:"password"`
	AccessKey       types.String    `tfsdk:"access_key"`
	SecretAccessKey types.String    `tfsdk:"secret_access_key"`
}

// EndpointsModel describes the endpoints configuration block.
type EndpointsModel struct {
	Mgmt types.String `tfsdk:"mgmt"`
	S3   types.String `tfsdk:"s3"`
}

// StorageGridClients holds the various API clients for StorageGrid services.
type StorageGridClients struct {
	MgmtClient         *utils.Client // nil if no management endpoint configured
	S3Client           *minio.Client // nil if no S3 endpoint configured
	tempAccessKeyID    string        // temporary access key created for S3 operations
	tempSecretKey      string        // temporary secret key created for S3 operations
	tempAccessKeyAlias string        // alias of temporary access key for cleanup
	username           string        // username for access key creation
}

// createTemporaryAccessKey creates a temporary S3 access key for the configured user
func (c *StorageGridClients) createTemporaryAccessKey(ctx context.Context) error {
	if c.MgmtClient == nil {
		return fmt.Errorf("management client not available for access key creation")
	}

	// Get the user ID using the management API
	userUniqueName := "user/" + c.username
	userResponse, err := c.MgmtClient.GetUser(userUniqueName)
	if err != nil {
		return fmt.Errorf("failed to get user info for %s: %w", c.username, err)
	}

	userID := userResponse.Data.ID

	// Create a temporary access key
	payload := utils.S3AccessKeyCreatePayload{
		// No expiry for now - we'll manage cleanup manually
		Expires: nil,
	}

	accessKeyResponse, err := c.MgmtClient.CreateS3AccessKey(userID, payload)
	if err != nil {
		return fmt.Errorf("failed to create temporary S3 access key: %w", err)
	}

	// Store the temporary credentials
	c.tempAccessKeyID = accessKeyResponse.Data.AccessKey
	c.tempSecretKey = accessKeyResponse.Data.SecretAccessKey
	c.tempAccessKeyAlias = accessKeyResponse.Data.ID // Use the key ID for deletion

	tflog.Debug(ctx, "Created temporary S3 access key", map[string]interface{}{
		"user_id":    userID,
		"access_key": c.tempAccessKeyID,
	})

	return nil
}

// cleanupTemporaryAccessKey removes the temporary access key
func (c *StorageGridClients) cleanupTemporaryAccessKey(ctx context.Context) error {
	if c.MgmtClient == nil || c.tempAccessKeyAlias == "" {
		return nil // nothing to clean up
	}

	// Get the user ID for deletion
	userUniqueName := "user/" + c.username
	userResponse, err := c.MgmtClient.GetUser(userUniqueName)
	if err != nil {
		tflog.Warn(ctx, "Failed to get user info for cleanup", map[string]interface{}{
			"username": c.username,
			"error":    err.Error(),
		})
		return nil // Don't fail on cleanup
	}

	userID := userResponse.Data.ID

	// Delete the temporary access key
	err = c.MgmtClient.DeleteS3AccessKey(userID, c.tempAccessKeyAlias)
	if err != nil {
		tflog.Warn(ctx, "Failed to cleanup temporary access key", map[string]interface{}{
			"key_id": c.tempAccessKeyAlias,
			"error":  err.Error(),
		})
	} else {
		tflog.Debug(ctx, "Successfully cleaned up temporary S3 access key", map[string]interface{}{
			"key_id": c.tempAccessKeyAlias,
		})
	}

	// Clear the temporary key info
	c.tempAccessKeyID = ""
	c.tempSecretKey = ""
	c.tempAccessKeyAlias = ""

	return nil
}

// initCleanup sets up automatic cleanup for temporary access keys
func (c *StorageGridClients) initCleanup() {
	if c.tempAccessKeyAlias != "" {
		// Use a finalizer to ensure cleanup happens when the clients object is garbage collected
		runtime.SetFinalizer(c, (*StorageGridClients).finalize)
	}
}

// finalize is called by the garbage collector to clean up temporary access keys
func (c *StorageGridClients) finalize() {
	if c.tempAccessKeyAlias != "" {
		// Create a background context for cleanup
		ctx := context.Background()
		c.cleanupTemporaryAccessKey(ctx)
	}
}

func (p *StorageGridProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "storagegrid"
	resp.Version = p.version
}

// Schema defines the provider-level schema for configuration data.
func (p *StorageGridProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The StorageGrid provider enables Terraform management of StorageGrid management and S3 resources including users, groups, access keys, buckets, and bucket configurations. This provider requires StorageGrid with v4 API support and is not compatible with older StorageGrid versions that only support v3 or earlier APIs.",
		Attributes: map[string]schema.Attribute{
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
			"access_key": schema.StringAttribute{
				Description: "S3 access key for direct S3 authentication when management endpoint is not configured. May also be provided via STORAGEGRID_ACCESS_KEY environment variable.",
				Optional:    true,
			},
			"secret_access_key": schema.StringAttribute{
				Description: "S3 secret access key for direct S3 authentication when management endpoint is not configured. May also be provided via STORAGEGRID_SECRET_ACCESS_KEY environment variable.",
				Optional:    true,
				Sensitive:   true,
			},
		},
		Blocks: map[string]schema.Block{
			"endpoints": schema.SingleNestedBlock{
				Description: "Configuration block for StorageGrid service endpoints. Individual endpoints may also be provided via STORAGEGRID_MGMT_ENDPOINT and STORAGEGRID_S3_ENDPOINT environment variables.",
				Attributes: map[string]schema.Attribute{
					"mgmt": schema.StringAttribute{
						Description: "URI for StorageGrid Management API.",
						Optional:    true,
					},
					"s3": schema.StringAttribute{
						Description: "URI for StorageGrid S3 API.",
						Optional:    true,
					},
				},
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

	if config.Endpoints != nil {
		if config.Endpoints.Mgmt.IsUnknown() {
			resp.Diagnostics.AddAttributeError(
				path.Root("endpoints").AtName("mgmt"),
				"Unknown StorageGrid Management API Endpoint",
				"The provider cannot create the StorageGrid API client as there is an unknown configuration value for the StorageGrid Management API endpoint. "+
					"Either target apply the source of the value first, set the value statically in the configuration, or use the STORAGEGRID_MGMT_ENDPOINT environment variable.",
			)
		}

		if config.Endpoints.S3.IsUnknown() {
			resp.Diagnostics.AddAttributeError(
				path.Root("endpoints").AtName("s3"),
				"Unknown StorageGrid S3 API Endpoint",
				"The provider cannot create the StorageGrid API client as there is an unknown configuration value for the StorageGrid S3 API endpoint. "+
					"Either target apply the source of the value first, set the value statically in the configuration, or use the STORAGEGRID_S3_ENDPOINT environment variable.",
			)
		}
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

	if config.AccessKey.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("access_key"),
			"Unknown StorageGrid S3 Access Key",
			"The provider cannot create the StorageGrid S3 client as there is an unknown configuration value for the S3 access key. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the STORAGEGRID_ACCESS_KEY environment variable.",
		)
	}

	if config.SecretAccessKey.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("secret_access_key"),
			"Unknown StorageGrid S3 Secret Access Key",
			"The provider cannot create the StorageGrid S3 client as there is an unknown configuration value for the S3 secret access key. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the STORAGEGRID_SECRET_ACCESS_KEY environment variable.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	// Default values to environment variables, but override
	// with Terraform configuration value if set.
	// CHOICE: this means that ENV vars overwrite what's in the TF object!
	mgmtEndpoint := os.Getenv("STORAGEGRID_MGMT_ENDPOINT")
	s3Endpoint := os.Getenv("STORAGEGRID_S3_ENDPOINT")
	accountID := os.Getenv("STORAGEGRID_ACCOUNTID")
	username := os.Getenv("STORAGEGRID_USERNAME")
	password := os.Getenv("STORAGEGRID_PASSWORD")
	accessKey := os.Getenv("STORAGEGRID_ACCESS_KEY")
	secretAccessKey := os.Getenv("STORAGEGRID_SECRET_ACCESS_KEY")

	// Extract endpoints from configuration if provided
	if config.Endpoints != nil {
		if !config.Endpoints.Mgmt.IsNull() {
			mgmtEndpoint = config.Endpoints.Mgmt.ValueString()
		}
		if !config.Endpoints.S3.IsNull() {
			s3Endpoint = config.Endpoints.S3.ValueString()
		}
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

	if !config.AccessKey.IsNull() {
		accessKey = config.AccessKey.ValueString()
	}

	if !config.SecretAccessKey.IsNull() {
		secretAccessKey = config.SecretAccessKey.ValueString()
	}

	// If any of the expected configurations are missing, return
	// errors with provider-specific guidance.

	if mgmtEndpoint == "" && s3Endpoint == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("endpoints"),
			"Missing StorageGrid API Endpoints",
			"The provider cannot create the StorageGrid API client as there are no endpoints configured. "+
				"At least one of 'mgmt' or 's3' endpoints must be provided in the endpoints configuration map or via "+
				"STORAGEGRID_MGMT_ENDPOINT and/or STORAGEGRID_S3_ENDPOINT environment variables.",
		)
	}

	// Only validate management-related credentials if management endpoint is configured
	if mgmtEndpoint != "" {
		if accountID == "" {
			resp.Diagnostics.AddAttributeError(
				path.Root("accountid"),
				"Missing StorageGrid API AccountID",
				"The provider cannot create the StorageGrid Management API client as there is a missing or empty value for the StorageGrid API accountID. "+
					"Set the accountID value in the configuration or use the STORAGEGRID_ACCOUNTID environment variable. "+
					"If either is already set, ensure the value is not empty.",
			)
		}

		if username == "" {
			resp.Diagnostics.AddAttributeError(
				path.Root("username"),
				"Missing StorageGrid API Username",
				"The provider cannot create the StorageGrid Management API client as there is a missing or empty value for the StorageGrid API username. "+
					"Set the username value in the configuration or use the STORAGEGRID_USERNAME environment variable. "+
					"If either is already set, ensure the value is not empty.",
			)
		}

		if password == "" {
			resp.Diagnostics.AddAttributeError(
				path.Root("password"),
				"Missing StorageGrid API Password",
				"The provider cannot create the StorageGrid Management API client as there is a missing or empty value for the StorageGrid API password. "+
					"Set the password value in the configuration or use the STORAGEGRID_PASSWORD environment variable. "+
					"If either is already set, ensure the value is not empty.",
			)
		}
	}

	// For S3-only configurations, validate that S3 credentials are provided
	if mgmtEndpoint == "" && s3Endpoint != "" {
		if accessKey == "" || secretAccessKey == "" {
			resp.Diagnostics.AddAttributeError(
				path.Root("access_key"),
				"Missing S3 Credentials for S3-Only Configuration",
				"When only the S3 endpoint is configured (without management endpoint), explicit S3 credentials are required. "+
					"Please provide both 'access_key' and 'secret_access_key' in the provider configuration, or "+
					"set both STORAGEGRID_ACCESS_KEY and STORAGEGRID_SECRET_ACCESS_KEY environment variables.",
			)
		}
	}

	if resp.Diagnostics.HasError() {
		return
	}

	ctx = tflog.SetField(ctx, "storagegrid_mgmt_endpoint", mgmtEndpoint)
	ctx = tflog.SetField(ctx, "storagegrid_s3_endpoint", s3Endpoint)
	ctx = tflog.SetField(ctx, "storagegrid_account_id", accountID)
	ctx = tflog.SetField(ctx, "storagegrid_username", username)

	tflog.Debug(ctx, "Creating StorageGrid clients")

	// Create the clients container
	clients := &StorageGridClients{
		username: username, // Store username for access key operations
	}

	// Create management client if endpoint is provided
	if mgmtEndpoint != "" {
		mgmtClient, err := utils.NewClient(&mgmtEndpoint, &accountID, &username, &password)
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to Create StorageGrid Management API Client",
				"An unexpected error occurred when creating the StorageGrid Management API client. "+
					"If the error is not clear, please contact the provider developers.\n\n"+
					"StorageGrid Management Client Error: "+err.Error(),
			)
			return
		}
		clients.MgmtClient = mgmtClient
		tflog.Debug(ctx, "Successfully created StorageGrid Management client")
	} else {
		tflog.Debug(ctx, "No management endpoint provided, skipping management client creation")
	}

	// Create S3 client if endpoint is provided
	if s3Endpoint != "" {
		// Parse S3 endpoint to extract hostname and determine security
		endpoint := s3Endpoint
		secure := true

		if strings.HasPrefix(endpoint, "https://") {
			endpoint = strings.TrimPrefix(endpoint, "https://")
			secure = true
		} else if strings.HasPrefix(endpoint, "http://") {
			endpoint = strings.TrimPrefix(endpoint, "http://")
			secure = false
		}

		// Determine credentials to use for S3 client
		var s3AccessKey, s3SecretKey string

		if clients.MgmtClient != nil {
			// Both management and S3 endpoints configured - create temporary access keys
			tflog.Debug(ctx, "Both management and S3 endpoints configured, creating temporary access keys")

			err := clients.createTemporaryAccessKey(ctx)
			if err != nil {
				resp.Diagnostics.AddError(
					"Unable to Create Temporary S3 Access Keys",
					"Failed to create temporary S3 access keys for S3 client authentication. "+
						"If the error is not clear, please contact the provider developers.\n\n"+
						"Temporary Access Key Error: "+err.Error(),
				)
				return
			}

			s3AccessKey = clients.tempAccessKeyID
			s3SecretKey = clients.tempSecretKey

			// Set up automatic cleanup for the temporary access keys
			clients.initCleanup()

			tflog.Debug(ctx, "Using temporary access keys for S3 client")
		} else {
			// Only S3 endpoint configured - use explicit S3 credentials
			// (validation already done earlier to ensure these are not empty)
			s3AccessKey = accessKey
			s3SecretKey = secretAccessKey
			tflog.Debug(ctx, "Using explicit access_key/secret_access_key for S3 client")
		}

		// Create S3 client with appropriate credentials
		s3Client, err := minio.New(endpoint, &minio.Options{
			Creds:  credentials.NewStaticV4(s3AccessKey, s3SecretKey, ""),
			Secure: secure,
		})
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to Create StorageGrid S3 API Client",
				"An unexpected error occurred when creating the StorageGrid S3 API client. "+
					"If the error is not clear, please contact the provider developers.\n\n"+
					"StorageGrid S3 Client Error: "+err.Error(),
			)
			return
		}
		clients.S3Client = s3Client
		tflog.Debug(ctx, "Successfully created StorageGrid S3 client")
	} else {
		tflog.Debug(ctx, "No S3 endpoint provided, skipping S3 client creation")
	}

	// Make the StorageGrid clients available during DataSource and Resource
	// type Configure methods.
	resp.DataSourceData = clients
	resp.ResourceData = clients

	tflog.Info(ctx, "Configured StorageGrid client", map[string]any{"success": true})
}

func (p *StorageGridProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewGroupResource,
		NewUserResource,
		NewAccessKeysResource,
		NewS3BucketResource,
	}
}
func (p *StorageGridProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewGroupDataSource,
		NewUserDataSource,
		NewS3BucketDataSource,
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &StorageGridProvider{
			version: version,
		}
	}
}
