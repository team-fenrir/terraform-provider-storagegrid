// Copyright IBM Corp. 2025, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	awspolicy "github.com/hashicorp/awspolicyequivalence"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/team-fenrir/terraform-provider-storagegrid/internal/utils"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource                = &S3BucketPolicyResource{}
	_ resource.ResourceWithConfigure   = &S3BucketPolicyResource{}
	_ resource.ResourceWithImportState = &S3BucketPolicyResource{}
)

func NewS3BucketPolicyResource() resource.Resource {
	return &S3BucketPolicyResource{}
}

// S3BucketPolicyResource defines the resource implementation.
type S3BucketPolicyResource struct {
	client *utils.Client
}

// S3BucketPolicyResourceModel describes the resource data model.
type S3BucketPolicyResourceModel struct {
	BucketName types.String `tfsdk:"bucket_name"`
	Policy     types.String `tfsdk:"policy"`
	ID         types.String `tfsdk:"id"`
}

func (r *S3BucketPolicyResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_s3_bucket_policy"
}

func (r *S3BucketPolicyResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages the S3 bucket policy for a StorageGrid S3 bucket. StorageGrid does not provide a DELETE method for bucket policies, so destroying this resource clears the policy with a PUT request containing a null policy.",
		Attributes: map[string]schema.Attribute{
			"bucket_name": schema.StringAttribute{
				Description: "The name of the S3 bucket to configure the policy for.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"policy": schema.StringAttribute{
				Description: "The bucket policy as a JSON string. Provide the policy object itself; the provider wraps it in the StorageGrid API request body.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					suppressS3PolicyDiffs(),
				},
			},
			"id": schema.StringAttribute{
				Description: "The unique identifier for the bucket policy (same as bucket_name).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *S3BucketPolicyResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func parseS3BucketPolicyJSON(policyJSON string) (utils.S3Policy, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal([]byte(policyJSON), &raw); err != nil {
		return utils.S3Policy{}, err
	}
	if _, ok := raw["policy"]; ok {
		return utils.S3Policy{}, fmt.Errorf("the policy JSON must contain the policy object only, without the top-level StorageGrid API wrapper field \"policy\"")
	}

	var policy utils.S3Policy
	if err := json.Unmarshal([]byte(policyJSON), &policy); err != nil {
		return utils.S3Policy{}, err
	}
	if policy.Statement == nil {
		return utils.S3Policy{}, fmt.Errorf("the policy JSON must include a Statement array")
	}

	return policy, nil
}

func marshalS3BucketPolicy(policy *utils.S3Policy) (string, error) {
	policyBytes, err := json.Marshal(policy)
	if err != nil {
		return "", err
	}
	return string(policyBytes), nil
}

func setPolicyStateFromAPI(currentPolicy types.String, apiPolicy *utils.S3Policy) (types.String, error) {
	apiPolicyString, err := marshalS3BucketPolicy(apiPolicy)
	if err != nil {
		return types.StringNull(), err
	}

	if currentPolicy.IsNull() || currentPolicy.IsUnknown() {
		return types.StringValue(apiPolicyString), nil
	}

	equal, err := awspolicy.PoliciesAreEquivalent(apiPolicyString, currentPolicy.ValueString())
	if err != nil {
		return types.StringNull(), err
	}
	if equal {
		return currentPolicy, nil
	}

	return types.StringValue(apiPolicyString), nil
}

func emptyS3BucketPolicy() utils.S3Policy {
	return utils.S3Policy{
		Version:   "2015-09-08",
		Statement: []utils.Statement{},
	}
}

func isNotFoundError(err error) bool {
	return strings.Contains(err.Error(), "status: 404")
}

func (r *S3BucketPolicyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan S3BucketPolicyResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	bucketName := plan.BucketName.ValueString()
	policy, err := parseS3BucketPolicyJSON(plan.Policy.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid S3 Bucket Policy JSON", "Could not parse the provided S3 bucket policy string: "+err.Error())
		return
	}

	err = r.client.PutS3BucketPolicy(bucketName, policy)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to Create S3 Bucket Policy for %s", bucketName),
			err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(bucketName)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *S3BucketPolicyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state S3BucketPolicyResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	bucketName := state.BucketName.ValueString()
	policy, err := r.client.GetS3BucketPolicy(bucketName)
	if err != nil {
		if isNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to Read S3 Bucket Policy for %s", bucketName),
			err.Error(),
		)
		return
	}

	state.Policy, err = setPolicyStateFromAPI(state.Policy, policy)
	if err != nil {
		resp.Diagnostics.AddError("S3 Bucket Policy Comparison Error", "Failed to compare S3 bucket policies: "+err.Error())
		return
	}
	state.ID = types.StringValue(bucketName)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *S3BucketPolicyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan S3BucketPolicyResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	bucketName := plan.BucketName.ValueString()
	policy, err := parseS3BucketPolicyJSON(plan.Policy.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid S3 Bucket Policy JSON", "Could not parse the provided S3 bucket policy string: "+err.Error())
		return
	}

	err = r.client.PutS3BucketPolicy(bucketName, policy)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to Update S3 Bucket Policy for %s", bucketName),
			err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(bucketName)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *S3BucketPolicyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state S3BucketPolicyResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	bucketName := state.BucketName.ValueString()
	err := r.client.ClearS3BucketPolicy(bucketName)
	if err != nil {
		if isNotFoundError(err) {
			return
		}
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to Delete S3 Bucket Policy for %s", bucketName),
			"StorageGrid does not provide a DELETE method for bucket policies. The provider attempted to clear the policy with PUT and received: "+err.Error(),
		)
		return
	}
}

func (r *S3BucketPolicyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	bucketName := req.ID

	policy, err := r.client.GetS3BucketPolicy(bucketName)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to Import S3 Bucket Policy for %s", bucketName),
			fmt.Sprintf("Bucket does not exist or bucket policy is not accessible: %s", err.Error()),
		)
		return
	}

	policyString, err := marshalS3BucketPolicy(policy)
	if err != nil {
		resp.Diagnostics.AddError("Error Processing S3 Bucket Policy on Import", "Could not marshal S3 bucket policy from API into string: "+err.Error())
		return
	}

	state := S3BucketPolicyResourceModel{
		BucketName: types.StringValue(bucketName),
		Policy:     types.StringValue(policyString),
		ID:         types.StringValue(bucketName),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
