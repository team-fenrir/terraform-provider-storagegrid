// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/minio/minio-go/v7"
)

// bucketNameValidator validates S3 bucket naming rules that can't be expressed with regex
type bucketNameValidator struct{}

func (v bucketNameValidator) Description(ctx context.Context) string {
	return "Bucket name must follow S3 naming rules"
}

func (v bucketNameValidator) MarkdownDescription(ctx context.Context) string {
	return "Bucket name must follow S3 naming rules"
}

func (v bucketNameValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	value := req.ConfigValue.ValueString()

	// Cannot contain consecutive hyphens
	if strings.Contains(value, "--") {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid bucket name",
			"Bucket name cannot contain consecutive hyphens (--)",
		)
		return
	}

	// Cannot start with 'xn--' (internationalized domain names)
	if strings.HasPrefix(value, "xn--") {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid bucket name",
			"Bucket name cannot start with 'xn--' (reserved for internationalized domain names)",
		)
		return
	}

	// Cannot end with '-s3alias' (reserved)
	if strings.HasSuffix(value, "-s3alias") {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid bucket name",
			"Bucket name cannot end with '-s3alias' (reserved suffix)",
		)
		return
	}

	// Cannot be formatted as an IP address (basic check for 4 dot-separated numbers)
	if matched, _ := regexp.MatchString(`^\d+\.\d+\.\d+\.\d+$`, value); matched {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid bucket name",
			"Bucket name cannot be formatted as an IP address",
		)
		return
	}
}

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource                = &S3BucketResource{}
	_ resource.ResourceWithConfigure   = &S3BucketResource{}
	_ resource.ResourceWithImportState = &S3BucketResource{}
)

func NewS3BucketResource() resource.Resource {
	return &S3BucketResource{}
}

// S3BucketResource defines the resource implementation.
type S3BucketResource struct {
	s3Client *minio.Client
}

type S3BucketResourceModel struct {
	Bucket            types.String `tfsdk:"bucket"`
	ObjectLockEnabled types.Bool   `tfsdk:"object_lock_enabled"`
	Region            types.String `tfsdk:"region"`
	CreationDate      types.String `tfsdk:"creation_date"`
}

func (r *S3BucketResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_s3_bucket"
}

func (r *S3BucketResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a StorageGrid S3 bucket.",
		Attributes: map[string]schema.Attribute{
			"bucket": schema.StringAttribute{
				Description: "The name of the S3 bucket. Must follow S3 bucket naming rules.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthBetween(3, 63),
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[a-z0-9][a-z0-9\-]*[a-z0-9]$`),
						"Bucket name must start and end with a lowercase letter or number, and can only contain lowercase letters, numbers, and hyphens",
					),
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[^.]*[^.]$`),
						"Bucket name cannot end with a period",
					),
					bucketNameValidator{},
				},
			},
			"object_lock_enabled": schema.BoolAttribute{
				Description: "Enable S3 Object Lock for the bucket. Defaults to false.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"region": schema.StringAttribute{
				Description: "The region where the bucket is located.",
				Computed:    true,
			},
			"creation_date": schema.StringAttribute{
				Description: "The creation date of the bucket.",
				Computed:    true,
			},
		},
	}
}

func (r *S3BucketResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	clients, ok := req.ProviderData.(*StorageGridClients)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *StorageGridClients, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	// Validate that S3 client is available for S3 bucket operations
	if clients.S3Client == nil {
		resp.Diagnostics.AddError(
			"S3 API Client Not Configured",
			"The s3_bucket resource requires a StorageGrid S3 API endpoint to be configured. "+
				"Please configure the 's3' endpoint in the provider's endpoints block or set the STORAGEGRID_S3_ENDPOINT environment variable.",
		)
		return
	}

	r.s3Client = clients.S3Client
}

func (r *S3BucketResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan S3BucketResourceModel

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	bucketName := plan.Bucket.ValueString()
	objectLockEnabled := plan.ObjectLockEnabled.ValueBool()

	// Check if bucket already exists
	exists, err := r.s3Client.BucketExists(ctx, bucketName)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error checking bucket existence",
			fmt.Sprintf("Unable to check if bucket '%s' exists: %s", bucketName, err),
		)
		return
	}

	if exists {
		resp.Diagnostics.AddError(
			"Bucket already exists",
			fmt.Sprintf("S3 bucket '%s' already exists", bucketName),
		)
		return
	}

	// Create the bucket
	var createErr error
	if objectLockEnabled {
		// Create bucket with Object Lock enabled
		createErr = r.s3Client.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{
			ObjectLocking: true,
		})
	} else {
		// Create bucket without Object Lock
		createErr = r.s3Client.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{})
	}

	if createErr != nil {
		resp.Diagnostics.AddError(
			"Error creating bucket",
			fmt.Sprintf("Unable to create S3 bucket '%s': %s", bucketName, createErr),
		)
		return
	}

	// Read the bucket information after creation
	err = r.readBucket(ctx, &plan)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading created bucket",
			fmt.Sprintf("Unable to read bucket '%s' after creation: %s", bucketName, err),
		)
		return
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *S3BucketResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state S3BucketResourceModel

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.readBucket(ctx, &state)
	if err != nil {
		if strings.Contains(err.Error(), "does not exist") {
			// Bucket has been deleted outside of Terraform
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error reading bucket",
			fmt.Sprintf("Unable to read bucket '%s': %s", state.Bucket.ValueString(), err),
		)
		return
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

// readBucket reads bucket information and populates the model
func (r *S3BucketResource) readBucket(ctx context.Context, model *S3BucketResourceModel) error {
	bucketName := model.Bucket.ValueString()

	// Check if bucket exists
	exists, err := r.s3Client.BucketExists(ctx, bucketName)
	if err != nil {
		return fmt.Errorf("error checking bucket existence: %w", err)
	}

	if !exists {
		return fmt.Errorf("bucket '%s' does not exist", bucketName)
	}

	// Get bucket location (region)
	location, err := r.s3Client.GetBucketLocation(ctx, bucketName)
	if err != nil {
		return fmt.Errorf("error getting bucket location: %w", err)
	}

	// List buckets to get creation date
	buckets, err := r.s3Client.ListBuckets(ctx)
	if err != nil {
		return fmt.Errorf("error listing buckets: %w", err)
	}

	var creationDate string
	for _, bucket := range buckets {
		if bucket.Name == bucketName {
			creationDate = bucket.CreationDate.Format("2006-01-02T15:04:05Z")
			break
		}
	}

	// TODO: Check if Object Lock is enabled for the bucket
	// For now, we'll keep the value from the plan/state
	// This would require additional MinIO client calls to determine Object Lock status

	// Set computed attributes
	model.Region = types.StringValue(location)
	model.CreationDate = types.StringValue(creationDate)

	return nil
}

func (r *S3BucketResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// S3 buckets have very limited update operations
	// Most changes require replacement (handled by RequiresReplace plan modifier)
	resp.Diagnostics.AddError(
		"Bucket updates not supported",
		"S3 bucket updates are not currently supported. Most bucket configuration changes require replacement.",
	)
}

func (r *S3BucketResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state S3BucketResourceModel

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	bucketName := state.Bucket.ValueString()

	// Check if bucket exists before trying to delete
	exists, err := r.s3Client.BucketExists(ctx, bucketName)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error checking bucket existence",
			fmt.Sprintf("Unable to check if bucket '%s' exists: %s", bucketName, err),
		)
		return
	}

	if !exists {
		// Bucket doesn't exist, nothing to delete
		return
	}

	// Delete the bucket
	err = r.s3Client.RemoveBucket(ctx, bucketName)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting bucket",
			fmt.Sprintf("Unable to delete S3 bucket '%s': %s", bucketName, err),
		)
		return
	}
}

func (r *S3BucketResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	bucketName := req.ID

	// Validate bucket name format
	if len(bucketName) < 3 || len(bucketName) > 63 {
		resp.Diagnostics.AddError(
			"Invalid bucket name for import",
			fmt.Sprintf("Bucket name '%s' is invalid. S3 bucket names must be between 3 and 63 characters long.", bucketName),
		)
		return
	}

	// Create initial state
	state := S3BucketResourceModel{
		Bucket:            types.StringValue(bucketName),
		ObjectLockEnabled: types.BoolValue(false), // Default, will be updated in Read
	}

	// Read bucket information
	err := r.readBucket(ctx, &state)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error importing bucket",
			fmt.Sprintf("Unable to import bucket '%s': %s", bucketName, err),
		)
		return
	}

	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}
