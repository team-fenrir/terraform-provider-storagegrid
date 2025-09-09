// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	awspolicy "github.com/hashicorp/awspolicyequivalence"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
)

func suppressS3PolicyDiffs() planmodifier.String {
	return &s3PolicyDiffSuppressor{}
}

type s3PolicyDiffSuppressor struct{}

func (s *s3PolicyDiffSuppressor) Description(ctx context.Context) string {
	return "Suppresses differences in S3 Policy JSON strings that are semantically equal."
}

func (s *s3PolicyDiffSuppressor) MarkdownDescription(ctx context.Context) string {
	return s.Description(ctx)
}

func (s *s3PolicyDiffSuppressor) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if req.PlanValue.IsNull() || req.PlanValue.IsUnknown() {
		return
	}
	if req.StateValue.IsNull() {
		return
	}

	planJSON := req.PlanValue.ValueString()
	stateJSON := req.StateValue.ValueString()

	equal, err := awspolicy.PoliciesAreEquivalent(planJSON, stateJSON)
	if err != nil {
		resp.Diagnostics.AddError("S3 Policy Comparison Error", "Failed to compare JSON strings: "+err.Error())
		return
	}

	if equal {
		resp.PlanValue = req.StateValue
	}
}
