// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package utils

import (
	"encoding/json"
	"testing"
)

func TestStatementWithCondition_Unmarshal(t *testing.T) {
	testCases := []struct {
		name           string
		policyJSON     string
		expectedError  bool
		validateResult func(t *testing.T, policy S3Policy)
	}{
		{
			name: "Policy with StringLike condition",
			policyJSON: `{
				"Statement": [
					{
						"Sid": "AllowListBucket",
						"Effect": "Allow",
						"Action": ["s3:ListBucket", "s3:ListBucketVersions"],
						"Resource": "arn:aws:s3:::reflow-client-storage-dev",
						"Condition": {
							"StringLike": {
								"s3:prefix": ["powerfactory/*", "powerfactory"]
							}
						}
					}
				]
			}`,
			expectedError: false,
			validateResult: func(t *testing.T, policy S3Policy) {
				if len(policy.Statement) != 1 {
					t.Fatalf("Expected 1 statement, got %d", len(policy.Statement))
				}
				stmt := policy.Statement[0]
				if stmt.Sid != "AllowListBucket" {
					t.Errorf("Expected Sid 'AllowListBucket', got '%s'", stmt.Sid)
				}
				if stmt.Condition == nil {
					t.Fatal("Expected Condition to be present, got nil")
				}
				if _, ok := stmt.Condition["StringLike"]; !ok {
					t.Error("Expected StringLike operator in Condition")
				}
				prefixValues := stmt.Condition["StringLike"]["s3:prefix"]
				if len(prefixValues) != 2 {
					t.Errorf("Expected 2 prefix values, got %d", len(prefixValues))
				}
				expectedPrefixes := []string{"powerfactory/*", "powerfactory"}
				for i, expected := range expectedPrefixes {
					if prefixValues[i] != expected {
						t.Errorf("Expected prefix[%d] to be '%s', got '%s'", i, expected, prefixValues[i])
					}
				}
			},
		},
		{
			name: "Policy with multiple condition operators",
			policyJSON: `{
				"Statement": [
					{
						"Effect": "Allow",
						"Action": "s3:GetObject",
						"Resource": "*",
						"Condition": {
							"StringEquals": {
								"s3:ExistingObjectTag/environment": "production"
							},
							"IpAddress": {
								"aws:SourceIp": ["192.168.1.0/24", "10.0.0.0/8"]
							}
						}
					}
				]
			}`,
			expectedError: false,
			validateResult: func(t *testing.T, policy S3Policy) {
				if len(policy.Statement) != 1 {
					t.Fatalf("Expected 1 statement, got %d", len(policy.Statement))
				}
				stmt := policy.Statement[0]
				if stmt.Condition == nil {
					t.Fatal("Expected Condition to be present, got nil")
				}
				if len(stmt.Condition) != 2 {
					t.Errorf("Expected 2 condition operators, got %d", len(stmt.Condition))
				}
				// Check StringEquals
				if _, ok := stmt.Condition["StringEquals"]; !ok {
					t.Error("Expected StringEquals operator in Condition")
				}
				envValue := stmt.Condition["StringEquals"]["s3:ExistingObjectTag/environment"]
				if len(envValue) != 1 || envValue[0] != "production" {
					t.Errorf("Expected environment tag value 'production', got %v", envValue)
				}
				// Check IpAddress
				if _, ok := stmt.Condition["IpAddress"]; !ok {
					t.Error("Expected IpAddress operator in Condition")
				}
				ipValues := stmt.Condition["IpAddress"]["aws:SourceIp"]
				if len(ipValues) != 2 {
					t.Errorf("Expected 2 IP values, got %d", len(ipValues))
				}
			},
		},
		{
			name: "Policy without Condition (backward compatibility)",
			policyJSON: `{
				"Statement": [
					{
						"Sid": "AllowListAllBuckets",
						"Effect": "Allow",
						"Action": "s3:ListAllMyBuckets",
						"Resource": "*"
					}
				]
			}`,
			expectedError: false,
			validateResult: func(t *testing.T, policy S3Policy) {
				if len(policy.Statement) != 1 {
					t.Fatalf("Expected 1 statement, got %d", len(policy.Statement))
				}
				stmt := policy.Statement[0]
				if stmt.Condition != nil {
					t.Errorf("Expected Condition to be nil for statement without condition, got %v", stmt.Condition)
				}
				if stmt.Sid != "AllowListAllBuckets" {
					t.Errorf("Expected Sid 'AllowListAllBuckets', got '%s'", stmt.Sid)
				}
			},
		},
		{
			name: "Complete policy with mixed statements",
			policyJSON: `{
				"Statement": [
					{
						"Sid": "AllowListAllBuckets",
						"Effect": "Allow",
						"Action": "s3:ListAllMyBuckets",
						"Resource": "*"
					},
					{
						"Sid": "AllowListBucket",
						"Effect": "Allow",
						"Action": ["s3:ListBucket", "s3:ListBucketVersions"],
						"Resource": "arn:aws:s3:::reflow-client-storage-dev",
						"Condition": {
							"StringLike": {
								"s3:prefix": ["powerfactory/*", "powerfactory"]
							}
						}
					},
					{
						"Sid": "AllowUserFolderAccess",
						"Effect": "Allow",
						"Action": [
							"s3:GetObject",
							"s3:GetObjectTagging",
							"s3:GetObjectVersion",
							"s3:GetObjectVersionTagging",
							"s3:PutObject"
						],
						"Resource": "arn:aws:s3:::reflow-client-storage-dev/powerfactory/*"
					}
				]
			}`,
			expectedError: false,
			validateResult: func(t *testing.T, policy S3Policy) {
				if len(policy.Statement) != 3 {
					t.Fatalf("Expected 3 statements, got %d", len(policy.Statement))
				}
				// First statement - no condition
				if policy.Statement[0].Condition != nil {
					t.Error("Expected first statement to have no Condition")
				}
				// Second statement - has condition
				if policy.Statement[1].Condition == nil {
					t.Error("Expected second statement to have Condition")
				}
				// Third statement - no condition
				if policy.Statement[2].Condition != nil {
					t.Error("Expected third statement to have no Condition")
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var policy S3Policy
			err := json.Unmarshal([]byte(tc.policyJSON), &policy)

			if tc.expectedError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tc.expectedError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if err == nil && tc.validateResult != nil {
				tc.validateResult(t, policy)
			}
		})
	}
}

func TestStatementWithCondition_Marshal(t *testing.T) {
	testCases := []struct {
		name     string
		policy   S3Policy
		validate func(t *testing.T, jsonOutput string)
	}{
		{
			name: "Statement with Condition marshals correctly",
			policy: S3Policy{
				Version: "2012-10-17",
				Statement: []Statement{
					{
						Sid:      "TestStatement",
						Effect:   "Allow",
						Action:   StringOrSlice{"s3:ListBucket"},
						Resource: StringOrSlice{"arn:aws:s3:::my-bucket"},
						Condition: map[string]map[string]StringOrSlice{
							"StringLike": {
								"s3:prefix": {"documents/*", "images/*"},
							},
						},
					},
				},
			},
			validate: func(t *testing.T, jsonOutput string) {
				var unmarshaled S3Policy
				if err := json.Unmarshal([]byte(jsonOutput), &unmarshaled); err != nil {
					t.Fatalf("Failed to unmarshal output: %v", err)
				}
				stmt := unmarshaled.Statement[0]
				if stmt.Condition == nil {
					t.Fatal("Condition should be present in marshaled output")
				}
				if _, ok := stmt.Condition["StringLike"]; !ok {
					t.Error("StringLike operator should be present")
				}
			},
		},
		{
			name: "Statement without Condition omits field",
			policy: S3Policy{
				Version: "2012-10-17",
				Statement: []Statement{
					{
						Sid:      "TestStatement",
						Effect:   "Allow",
						Action:   StringOrSlice{"s3:ListBucket"},
						Resource: StringOrSlice{"arn:aws:s3:::my-bucket"},
					},
				},
			},
			validate: func(t *testing.T, jsonOutput string) {
				// Parse as map to check if Condition key is absent
				var raw map[string]interface{}
				if err := json.Unmarshal([]byte(jsonOutput), &raw); err != nil {
					t.Fatalf("Failed to unmarshal output: %v", err)
				}
				statements := raw["Statement"].([]interface{})
				stmt := statements[0].(map[string]interface{})
				if _, exists := stmt["Condition"]; exists {
					t.Error("Condition field should be omitted when nil due to omitempty tag")
				}
			},
		},
		{
			name: "Multiple condition operators marshal correctly",
			policy: S3Policy{
				Statement: []Statement{
					{
						Effect:   "Allow",
						Action:   StringOrSlice{"s3:GetObject"},
						Resource: StringOrSlice{"*"},
						Condition: map[string]map[string]StringOrSlice{
							"StringEquals": {
								"s3:ExistingObjectTag/environment": {"production"},
							},
							"IpAddress": {
								"aws:SourceIp": {"192.168.1.0/24"},
							},
						},
					},
				},
			},
			validate: func(t *testing.T, jsonOutput string) {
				var unmarshaled S3Policy
				if err := json.Unmarshal([]byte(jsonOutput), &unmarshaled); err != nil {
					t.Fatalf("Failed to unmarshal output: %v", err)
				}
				stmt := unmarshaled.Statement[0]
				if len(stmt.Condition) != 2 {
					t.Errorf("Expected 2 condition operators, got %d", len(stmt.Condition))
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			jsonBytes, err := json.Marshal(tc.policy)
			if err != nil {
				t.Fatalf("Failed to marshal policy: %v", err)
			}
			if tc.validate != nil {
				tc.validate(t, string(jsonBytes))
			}
		})
	}
}

func TestStatementCondition_RoundTrip(t *testing.T) {
	// Test that we can unmarshal and then marshal back to equivalent JSON
	originalJSON := `{
		"Statement": [
			{
				"Sid": "AllowListBucket",
				"Effect": "Allow",
				"Action": ["s3:ListBucket", "s3:ListBucketVersions"],
				"Resource": "arn:aws:s3:::reflow-client-storage-dev",
				"Condition": {
					"StringLike": {
						"s3:prefix": ["powerfactory/*", "powerfactory"]
					}
				}
			}
		]
	}`

	var policy S3Policy
	err := json.Unmarshal([]byte(originalJSON), &policy)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	remarshaled, err := json.Marshal(policy)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var policy2 S3Policy
	err = json.Unmarshal(remarshaled, &policy2)
	if err != nil {
		t.Fatalf("Failed to unmarshal remarshaled: %v", err)
	}

	// Verify the data is preserved
	if len(policy2.Statement) != 1 {
		t.Fatalf("Expected 1 statement after round-trip, got %d", len(policy2.Statement))
	}
	stmt := policy2.Statement[0]
	if stmt.Condition == nil {
		t.Fatal("Condition should be preserved after round-trip")
	}
	if len(stmt.Condition["StringLike"]["s3:prefix"]) != 2 {
		t.Error("Prefix values should be preserved after round-trip")
	}
}
