// Copyright IBM Corp. 2025, 2026
// SPDX-License-Identifier: MPL-2.0

package utils

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestStringOrSliceUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    StringOrSlice
		wantErr bool
	}{
		{
			name:  "single string",
			input: `"s3:GetObject"`,
			want:  StringOrSlice{"s3:GetObject"},
		},
		{
			name:  "string slice",
			input: `["s3:GetObject","s3:PutObject"]`,
			want:  StringOrSlice{"s3:GetObject", "s3:PutObject"},
		},
		{
			name:  "empty string slice",
			input: `[]`,
			want:  StringOrSlice{},
		},
		{
			name:    "non string value",
			input:   `123`,
			wantErr: true,
		},
		{
			name:    "mixed slice",
			input:   `["s3:GetObject",123]`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got StringOrSlice
			err := json.Unmarshal([]byte(tt.input), &got)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("got %#v, want %#v", got, tt.want)
			}
		})
	}
}
