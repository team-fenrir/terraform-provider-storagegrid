// Copyright IBM Corp. 2025, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/team-fenrir/terraform-provider-storagegrid/internal/utils"
)

func TestBuildLifecycleConfiguration(t *testing.T) {
	tests := []struct {
		name  string
		rules []LifecycleRuleResourceModel
		want  *utils.LifecycleConfiguration
	}{
		{
			name:  "no rules",
			rules: nil,
			want:  &utils.LifecycleConfiguration{Rules: []utils.Rule{}},
		},
		{
			name: "minimal rule without optional blocks",
			rules: []LifecycleRuleResourceModel{
				{
					ID:     types.StringValue("rule-1"),
					Status: types.StringValue("Enabled"),
				},
			},
			want: &utils.LifecycleConfiguration{
				Rules: []utils.Rule{
					{ID: "rule-1", Status: "Enabled"},
				},
			},
		},
		{
			name: "rule with filter prefix",
			rules: []LifecycleRuleResourceModel{
				{
					ID:     types.StringValue("rule-1"),
					Status: types.StringValue("Enabled"),
					Filter: &LifecycleFilterResourceModel{
						Prefix: types.StringValue("logs/"),
					},
				},
			},
			want: &utils.LifecycleConfiguration{
				Rules: []utils.Rule{
					{
						ID:     "rule-1",
						Status: "Enabled",
						Filter: &utils.Filter{Prefix: "logs/"},
					},
				},
			},
		},
		{
			name: "expiration with days",
			rules: []LifecycleRuleResourceModel{
				{
					ID:     types.StringValue("rule-1"),
					Status: types.StringValue("Enabled"),
					Expiration: &LifecycleExpirationResourceModel{
						Days:                      types.Int64Value(30),
						Date:                      types.StringNull(),
						ExpiredObjectDeleteMarker: types.BoolNull(),
					},
				},
			},
			want: &utils.LifecycleConfiguration{
				Rules: []utils.Rule{
					{
						ID:         "rule-1",
						Status:     "Enabled",
						Expiration: &utils.Expiration{Days: 30},
					},
				},
			},
		},
		{
			name: "expiration with date",
			rules: []LifecycleRuleResourceModel{
				{
					ID:     types.StringValue("rule-1"),
					Status: types.StringValue("Enabled"),
					Expiration: &LifecycleExpirationResourceModel{
						Days:                      types.Int64Null(),
						Date:                      types.StringValue("2026-01-01T00:00:00Z"),
						ExpiredObjectDeleteMarker: types.BoolNull(),
					},
				},
			},
			want: &utils.LifecycleConfiguration{
				Rules: []utils.Rule{
					{
						ID:         "rule-1",
						Status:     "Enabled",
						Expiration: &utils.Expiration{Date: "2026-01-01T00:00:00Z"},
					},
				},
			},
		},
		{
			name: "expiration with expired_object_delete_marker true",
			rules: []LifecycleRuleResourceModel{
				{
					ID:     types.StringValue("rule-1"),
					Status: types.StringValue("Enabled"),
					Expiration: &LifecycleExpirationResourceModel{
						Days:                      types.Int64Null(),
						Date:                      types.StringNull(),
						ExpiredObjectDeleteMarker: types.BoolValue(true),
					},
				},
			},
			want: &utils.LifecycleConfiguration{
				Rules: []utils.Rule{
					{
						ID:         "rule-1",
						Status:     "Enabled",
						Expiration: &utils.Expiration{ExpiredObjectDeleteMarker: new(true)},
					},
				},
			},
		},
		{
			name: "expiration with expired_object_delete_marker false is still set",
			rules: []LifecycleRuleResourceModel{
				{
					ID:     types.StringValue("rule-1"),
					Status: types.StringValue("Enabled"),
					Expiration: &LifecycleExpirationResourceModel{
						Days:                      types.Int64Null(),
						Date:                      types.StringNull(),
						ExpiredObjectDeleteMarker: types.BoolValue(false),
					},
				},
			},
			want: &utils.LifecycleConfiguration{
				Rules: []utils.Rule{
					{
						ID:         "rule-1",
						Status:     "Enabled",
						Expiration: &utils.Expiration{ExpiredObjectDeleteMarker: new(false)},
					},
				},
			},
		},
		{
			name: "empty expiration block produces no expiration",
			rules: []LifecycleRuleResourceModel{
				{
					ID:     types.StringValue("rule-1"),
					Status: types.StringValue("Enabled"),
					Expiration: &LifecycleExpirationResourceModel{
						Days:                      types.Int64Null(),
						Date:                      types.StringNull(),
						ExpiredObjectDeleteMarker: types.BoolNull(),
					},
				},
			},
			want: &utils.LifecycleConfiguration{
				Rules: []utils.Rule{
					{ID: "rule-1", Status: "Enabled"},
				},
			},
		},
		{
			name: "noncurrent version expiration",
			rules: []LifecycleRuleResourceModel{
				{
					ID:     types.StringValue("rule-1"),
					Status: types.StringValue("Enabled"),
					NoncurrentVersionExpiration: &LifecycleNoncurrentVersionResourceModel{
						NoncurrentDays: types.Int64Value(7),
					},
				},
			},
			want: &utils.LifecycleConfiguration{
				Rules: []utils.Rule{
					{
						ID:                          "rule-1",
						Status:                      "Enabled",
						NoncurrentVersionExpiration: &utils.NoncurrentVersionExpiration{NoncurrentDays: 7},
					},
				},
			},
		},
		{
			name: "multiple rules preserve order and per-rule shape",
			rules: []LifecycleRuleResourceModel{
				{
					ID:     types.StringValue("rule-1"),
					Status: types.StringValue("Enabled"),
					Filter: &LifecycleFilterResourceModel{Prefix: types.StringValue("logs/")},
				},
				{
					ID:     types.StringValue("rule-2"),
					Status: types.StringValue("Disabled"),
					Expiration: &LifecycleExpirationResourceModel{
						Days:                      types.Int64Value(90),
						Date:                      types.StringNull(),
						ExpiredObjectDeleteMarker: types.BoolNull(),
					},
				},
			},
			want: &utils.LifecycleConfiguration{
				Rules: []utils.Rule{
					{
						ID:     "rule-1",
						Status: "Enabled",
						Filter: &utils.Filter{Prefix: "logs/"},
					},
					{
						ID:         "rule-2",
						Status:     "Disabled",
						Expiration: &utils.Expiration{Days: 90},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildLifecycleConfiguration(tt.rules)
			assertLifecycleConfigEqual(t, got, tt.want)
		})
	}
}

func TestMapLifecycleRules(t *testing.T) {
	tests := []struct {
		name   string
		config *utils.LifecycleConfiguration
		want   []LifecycleRuleResourceModel
	}{
		{
			name:   "no rules",
			config: &utils.LifecycleConfiguration{},
			want:   nil,
		},
		{
			name: "minimal rule",
			config: &utils.LifecycleConfiguration{
				Rules: []utils.Rule{{ID: "rule-1", Status: "Enabled"}},
			},
			want: []LifecycleRuleResourceModel{
				{ID: types.StringValue("rule-1"), Status: types.StringValue("Enabled")},
			},
		},
		{
			name: "rule with filter",
			config: &utils.LifecycleConfiguration{
				Rules: []utils.Rule{
					{ID: "rule-1", Status: "Enabled", Filter: &utils.Filter{Prefix: "logs/"}},
				},
			},
			want: []LifecycleRuleResourceModel{
				{
					ID:     types.StringValue("rule-1"),
					Status: types.StringValue("Enabled"),
					Filter: &LifecycleFilterResourceModel{Prefix: types.StringValue("logs/")},
				},
			},
		},
		{
			name: "expiration with days only nulls date and marker",
			config: &utils.LifecycleConfiguration{
				Rules: []utils.Rule{
					{ID: "rule-1", Status: "Enabled", Expiration: &utils.Expiration{Days: 30}},
				},
			},
			want: []LifecycleRuleResourceModel{
				{
					ID:     types.StringValue("rule-1"),
					Status: types.StringValue("Enabled"),
					Expiration: &LifecycleExpirationResourceModel{
						Days:                      types.Int64Value(30),
						Date:                      types.StringNull(),
						ExpiredObjectDeleteMarker: types.BoolNull(),
					},
				},
			},
		},
		{
			name: "expiration with expired_object_delete_marker",
			config: &utils.LifecycleConfiguration{
				Rules: []utils.Rule{
					{ID: "rule-1", Status: "Enabled", Expiration: &utils.Expiration{ExpiredObjectDeleteMarker: new(true)}},
				},
			},
			want: []LifecycleRuleResourceModel{
				{
					ID:     types.StringValue("rule-1"),
					Status: types.StringValue("Enabled"),
					Expiration: &LifecycleExpirationResourceModel{
						Days:                      types.Int64Null(),
						Date:                      types.StringNull(),
						ExpiredObjectDeleteMarker: types.BoolValue(true),
					},
				},
			},
		},
		{
			// mapLifecycleRules treats Expiration.Days <= 0 as "unset" and maps it
			// to a null Int64. This documents that asymmetry: an Expiration that is
			// present but carries no positive Days/Date/marker yields all-null fields.
			name: "expiration with zero days maps to null days",
			config: &utils.LifecycleConfiguration{
				Rules: []utils.Rule{
					{ID: "rule-1", Status: "Enabled", Expiration: &utils.Expiration{Days: 0}},
				},
			},
			want: []LifecycleRuleResourceModel{
				{
					ID:     types.StringValue("rule-1"),
					Status: types.StringValue("Enabled"),
					Expiration: &LifecycleExpirationResourceModel{
						Days:                      types.Int64Null(),
						Date:                      types.StringNull(),
						ExpiredObjectDeleteMarker: types.BoolNull(),
					},
				},
			},
		},
		{
			name: "noncurrent version expiration",
			config: &utils.LifecycleConfiguration{
				Rules: []utils.Rule{
					{ID: "rule-1", Status: "Enabled", NoncurrentVersionExpiration: &utils.NoncurrentVersionExpiration{NoncurrentDays: 7}},
				},
			},
			want: []LifecycleRuleResourceModel{
				{
					ID:     types.StringValue("rule-1"),
					Status: types.StringValue("Enabled"),
					NoncurrentVersionExpiration: &LifecycleNoncurrentVersionResourceModel{
						NoncurrentDays: types.Int64Value(7),
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mapLifecycleRules(tt.config)
			assertRuleModelsEqual(t, got, tt.want)
		})
	}
}

// TestBuildAndMapRoundTrip ensures a configuration survives a build -> map
// round trip without unexpected drift for the common field combinations.
func TestBuildAndMapRoundTrip(t *testing.T) {
	original := []LifecycleRuleResourceModel{
		{
			ID:     types.StringValue("rule-1"),
			Status: types.StringValue("Enabled"),
			Filter: &LifecycleFilterResourceModel{Prefix: types.StringValue("logs/")},
			Expiration: &LifecycleExpirationResourceModel{
				Days:                      types.Int64Value(30),
				Date:                      types.StringNull(),
				ExpiredObjectDeleteMarker: types.BoolNull(),
			},
			NoncurrentVersionExpiration: &LifecycleNoncurrentVersionResourceModel{
				NoncurrentDays: types.Int64Value(7),
			},
		},
	}

	roundTripped := mapLifecycleRules(buildLifecycleConfiguration(original))
	assertRuleModelsEqual(t, roundTripped, original)
}

func TestNonEmptyFilterValidator(t *testing.T) {
	ctx := context.Background()
	attrTypes := map[string]attr.Type{"prefix": types.StringType}

	tests := []struct {
		name      string
		value     types.Object
		wantError bool
	}{
		{
			name:      "null filter object is valid",
			value:     types.ObjectNull(attrTypes),
			wantError: false,
		},
		{
			name:      "unknown filter object is valid",
			value:     types.ObjectUnknown(attrTypes),
			wantError: false,
		},
		{
			name: "non-empty prefix is valid",
			value: types.ObjectValueMust(attrTypes, map[string]attr.Value{
				"prefix": types.StringValue("logs/"),
			}),
			wantError: false,
		},
		{
			name: "empty prefix is invalid",
			value: types.ObjectValueMust(attrTypes, map[string]attr.Value{
				"prefix": types.StringValue(""),
			}),
			wantError: true,
		},
		{
			name: "null prefix is invalid",
			value: types.ObjectValueMust(attrTypes, map[string]attr.Value{
				"prefix": types.StringNull(),
			}),
			wantError: true,
		},
		{
			name: "unknown prefix is valid",
			value: types.ObjectValueMust(attrTypes, map[string]attr.Value{
				"prefix": types.StringUnknown(),
			}),
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := validator.ObjectRequest{
				Path:        path.Root("filter"),
				ConfigValue: tt.value,
			}
			resp := &validator.ObjectResponse{}

			nonEmptyFilterValidator{}.ValidateObject(ctx, req, resp)

			if got := resp.Diagnostics.HasError(); got != tt.wantError {
				t.Fatalf("HasError() = %v, want %v (diagnostics: %v)", got, tt.wantError, resp.Diagnostics)
			}
		})
	}
}

// assertLifecycleConfigEqual compares two API lifecycle configurations field by field.
func assertLifecycleConfigEqual(t *testing.T, got, want *utils.LifecycleConfiguration) {
	t.Helper()

	if len(got.Rules) != len(want.Rules) {
		t.Fatalf("rule count = %d, want %d", len(got.Rules), len(want.Rules))
	}

	for i := range want.Rules {
		g, w := got.Rules[i], want.Rules[i]
		if g.ID != w.ID || g.Status != w.Status {
			t.Errorf("rule[%d] ID/Status = (%q, %q), want (%q, %q)", i, g.ID, g.Status, w.ID, w.Status)
		}

		assertFilterEqual(t, i, g.Filter, w.Filter)
		assertExpirationEqual(t, i, g.Expiration, w.Expiration)
		assertNoncurrentEqual(t, i, g.NoncurrentVersionExpiration, w.NoncurrentVersionExpiration)
	}
}

func assertFilterEqual(t *testing.T, i int, got, want *utils.Filter) {
	t.Helper()
	if (got == nil) != (want == nil) {
		t.Errorf("rule[%d] Filter presence = %v, want %v", i, got != nil, want != nil)
		return
	}
	if got != nil && got.Prefix != want.Prefix {
		t.Errorf("rule[%d] Filter.Prefix = %q, want %q", i, got.Prefix, want.Prefix)
	}
}

func assertExpirationEqual(t *testing.T, i int, got, want *utils.Expiration) {
	t.Helper()
	if (got == nil) != (want == nil) {
		t.Errorf("rule[%d] Expiration presence = %v, want %v", i, got != nil, want != nil)
		return
	}
	if got == nil {
		return
	}
	if got.Days != want.Days {
		t.Errorf("rule[%d] Expiration.Days = %d, want %d", i, got.Days, want.Days)
	}
	if got.Date != want.Date {
		t.Errorf("rule[%d] Expiration.Date = %q, want %q", i, got.Date, want.Date)
	}
	switch {
	case (got.ExpiredObjectDeleteMarker == nil) != (want.ExpiredObjectDeleteMarker == nil):
		t.Errorf("rule[%d] Expiration.ExpiredObjectDeleteMarker presence = %v, want %v",
			i, got.ExpiredObjectDeleteMarker != nil, want.ExpiredObjectDeleteMarker != nil)
	case got.ExpiredObjectDeleteMarker != nil && *got.ExpiredObjectDeleteMarker != *want.ExpiredObjectDeleteMarker:
		t.Errorf("rule[%d] Expiration.ExpiredObjectDeleteMarker = %v, want %v",
			i, *got.ExpiredObjectDeleteMarker, *want.ExpiredObjectDeleteMarker)
	}
}

func assertNoncurrentEqual(t *testing.T, i int, got, want *utils.NoncurrentVersionExpiration) {
	t.Helper()
	if (got == nil) != (want == nil) {
		t.Errorf("rule[%d] NoncurrentVersionExpiration presence = %v, want %v", i, got != nil, want != nil)
		return
	}
	if got != nil && got.NoncurrentDays != want.NoncurrentDays {
		t.Errorf("rule[%d] NoncurrentVersionExpiration.NoncurrentDays = %d, want %d", i, got.NoncurrentDays, want.NoncurrentDays)
	}
}

// assertRuleModelsEqual compares two slices of Terraform rule models field by field.
func assertRuleModelsEqual(t *testing.T, got, want []LifecycleRuleResourceModel) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("rule count = %d, want %d", len(got), len(want))
	}

	for i := range want {
		g, w := got[i], want[i]
		if !g.ID.Equal(w.ID) || !g.Status.Equal(w.Status) {
			t.Errorf("rule[%d] ID/Status = (%v, %v), want (%v, %v)", i, g.ID, g.Status, w.ID, w.Status)
		}

		if (g.Filter == nil) != (w.Filter == nil) {
			t.Errorf("rule[%d] Filter presence = %v, want %v", i, g.Filter != nil, w.Filter != nil)
		} else if g.Filter != nil && !g.Filter.Prefix.Equal(w.Filter.Prefix) {
			t.Errorf("rule[%d] Filter.Prefix = %v, want %v", i, g.Filter.Prefix, w.Filter.Prefix)
		}

		if (g.Expiration == nil) != (w.Expiration == nil) {
			t.Errorf("rule[%d] Expiration presence = %v, want %v", i, g.Expiration != nil, w.Expiration != nil)
		} else if g.Expiration != nil {
			if !g.Expiration.Days.Equal(w.Expiration.Days) {
				t.Errorf("rule[%d] Expiration.Days = %v, want %v", i, g.Expiration.Days, w.Expiration.Days)
			}
			if !g.Expiration.Date.Equal(w.Expiration.Date) {
				t.Errorf("rule[%d] Expiration.Date = %v, want %v", i, g.Expiration.Date, w.Expiration.Date)
			}
			if !g.Expiration.ExpiredObjectDeleteMarker.Equal(w.Expiration.ExpiredObjectDeleteMarker) {
				t.Errorf("rule[%d] Expiration.ExpiredObjectDeleteMarker = %v, want %v",
					i, g.Expiration.ExpiredObjectDeleteMarker, w.Expiration.ExpiredObjectDeleteMarker)
			}
		}

		if (g.NoncurrentVersionExpiration == nil) != (w.NoncurrentVersionExpiration == nil) {
			t.Errorf("rule[%d] NoncurrentVersionExpiration presence = %v, want %v",
				i, g.NoncurrentVersionExpiration != nil, w.NoncurrentVersionExpiration != nil)
		} else if g.NoncurrentVersionExpiration != nil &&
			!g.NoncurrentVersionExpiration.NoncurrentDays.Equal(w.NoncurrentVersionExpiration.NoncurrentDays) {
			t.Errorf("rule[%d] NoncurrentVersionExpiration.NoncurrentDays = %v, want %v",
				i, g.NoncurrentVersionExpiration.NoncurrentDays, w.NoncurrentVersionExpiration.NoncurrentDays)
		}
	}
}
