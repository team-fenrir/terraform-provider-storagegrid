// Copyright IBM Corp. 2025, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/team-fenrir/terraform-provider-storagegrid/internal/utils"
)

func TestParseS3BucketPolicyJSON(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name: "policy object",
			input: `{
				"Id": "123456",
				"Version": "2015-09-08",
				"Statement": [
					{
						"Effect": "Allow",
						"Action": "s3:GetObject",
						"Resource": "arn:aws:s3:::logs/*",
						"Principal": "*"
					}
				]
			}`,
		},
		{
			name: "empty statement array",
			input: `{
				"Version": "2015-09-08",
				"Statement": []
			}`,
		},
		{
			name: "api wrapper",
			input: `{
				"policy": {
					"Version": "2015-09-08",
					"Statement": []
				}
			}`,
			wantErr: true,
		},
		{
			name: "missing statement",
			input: `{
				"Version": "2015-09-08"
			}`,
			wantErr: true,
		},
		{
			name:    "invalid json",
			input:   `{`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseS3BucketPolicyJSON(tt.input)
			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestSetPolicyStateFromAPIEquivalentStorageGridPolicy(t *testing.T) {
	currentPolicy := types.StringValue(`{
		"Version": "2015-09-08",
		"Statement": [
			{
				"Effect": "Allow",
				"Principal": "*",
				"Action": "s3:GetObject",
				"Resource": "arn:aws:s3:::logs/*"
			}
		]
	}`)

	apiPolicy := &utils.S3Policy{
		Version: "2015-09-08",
		Statement: []utils.Statement{
			{
				Effect:    "Allow",
				Principal: "*",
				Action:    utils.StringOrSlice{"s3:GetObject"},
				Resource:  utils.StringOrSlice{"arn:aws:s3:::logs/*"},
			},
		},
	}

	got, err := setPolicyStateFromAPI(currentPolicy, apiPolicy)
	if err != nil {
		t.Fatalf("setPolicyStateFromAPI returned error: %v", err)
	}
	if got.ValueString() != currentPolicy.ValueString() {
		t.Fatalf("policy was not preserved from state")
	}
}

func TestIsNotFoundError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "status 404",
			err:  errString("error executing PUT request: status: 404, body: not found"),
			want: true,
		},
		{
			name: "bare not found",
			err:  errString("policy not found in response body"),
			want: false,
		},
		{
			name: "status 500 body mentions not found",
			err:  errString("status: 500, body: upstream not found"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isNotFoundError(tt.err); got != tt.want {
				t.Fatalf("isNotFoundError(%q) = %v, want %v", tt.err.Error(), got, tt.want)
			}
		})
	}
}

type errString string

func (e errString) Error() string {
	return string(e)
}
