// Copyright IBM Corp. 2025, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import "testing"

func TestStatusToAPIBools(t *testing.T) {
	tests := []struct {
		name          string
		status        string
		wantEnabled   bool
		wantSuspended bool
	}{
		{
			name:          "enabled",
			status:        "Enabled",
			wantEnabled:   true,
			wantSuspended: false,
		},
		{
			name:          "suspended",
			status:        "Suspended",
			wantEnabled:   false,
			wantSuspended: true,
		},
		{
			name:          "unknown defaults to enabled",
			status:        "Disabled",
			wantEnabled:   true,
			wantSuspended: false,
		},
		{
			name:          "empty defaults to enabled",
			status:        "",
			wantEnabled:   true,
			wantSuspended: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotEnabled, gotSuspended := statusToAPIBools(tt.status)
			if gotEnabled != tt.wantEnabled || gotSuspended != tt.wantSuspended {
				t.Fatalf("statusToAPIBools(%q) = (%v, %v), want (%v, %v)", tt.status, gotEnabled, gotSuspended, tt.wantEnabled, tt.wantSuspended)
			}
		})
	}
}

func TestAPIBoolsToStatus(t *testing.T) {
	tests := []struct {
		name      string
		enabled   bool
		suspended bool
		want      string
	}{
		{
			name:      "enabled",
			enabled:   true,
			suspended: false,
			want:      "Enabled",
		},
		{
			name:      "suspended",
			enabled:   false,
			suspended: true,
			want:      "Suspended",
		},
		{
			name:      "disabled",
			enabled:   false,
			suspended: false,
			want:      "Disabled",
		},
		{
			name:      "enabled wins if both flags are true",
			enabled:   true,
			suspended: true,
			want:      "Enabled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := apiBoolsToStatus(tt.enabled, tt.suspended); got != tt.want {
				t.Fatalf("apiBoolsToStatus(%v, %v) = %q, want %q", tt.enabled, tt.suspended, got, tt.want)
			}
		})
	}
}
