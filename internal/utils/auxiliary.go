// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package utils

import (
	"encoding/json"
	"fmt"
)

type StringOrSlice []string

func (s *StringOrSlice) UnmarshalJSON(b []byte) error {
	var single string
	if err := json.Unmarshal(b, &single); err == nil {
		*s = []string{single}
		return nil
	}

	var slice []string
	if err := json.Unmarshal(b, &slice); err == nil {
		*s = slice
		return nil
	}

	return fmt.Errorf("failed to unmarshal string or slice of strings")
}
