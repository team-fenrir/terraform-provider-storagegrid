// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package utils

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// S3BucketAPIResponse represents the API response structure for S3 bucket data.
type S3BucketAPIResponse struct {
	ResponseTime string        `json:"responseTime"`
	Status       string        `json:"status"`
	APIVersion   string        `json:"apiVersion"`
	Deprecated   bool          `json:"deprecated"`
	Data         S3BucketData  `json:"data"`
}

// S3BucketData represents the main data object for an S3 bucket.
type S3BucketData struct {
	ID                     string `json:"id"`
	Name                   string `json:"name"`
	CreationTime           string `json:"creationTime"`
	Region                 string `json:"region"`
	ObjectLockEnabled      bool   `json:"objectLockEnabled"`
	ComplianceEnabled      bool   `json:"complianceEnabled"`
	ConsistencyLevel       string `json:"consistencyLevel"`
	LastAccessTimeEnabled  bool   `json:"lastAccessTimeEnabled"`
}

// GetS3Bucket retrieves information about a specific S3 bucket by name.
func (c *Client) GetS3Bucket(bucketName string) (*S3BucketAPIResponse, error) {
	url := fmt.Sprintf("%s/api/v4/org/containers/%s", c.EndpointURL, bucketName)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("accept", "application/json")

	body, err := c.doRequest(req)
	if err != nil {
		return nil, fmt.Errorf("error executing request: %w", err)
	}

	var apiResponse S3BucketAPIResponse
	if err := json.Unmarshal(body, &apiResponse); err != nil {
		return nil, fmt.Errorf("error unmarshalling S3 bucket response: %w", err)
	}

	return &apiResponse, nil
}