// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package utils

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// S3BucketAPIResponse represents the API response structure for S3 bucket data.
type S3BucketAPIResponse struct {
	ResponseTime string         `json:"responseTime"`
	Status       string         `json:"status"`
	APIVersion   string         `json:"apiVersion"`
	Deprecated   bool           `json:"deprecated"`
	Data         []S3BucketData `json:"data"`
}

// S3BucketData represents the main data object for an S3 bucket.
type S3BucketData struct {
	Name         string                      `json:"name"`
	CreationTime string                      `json:"creationTime"`
	Region       string                      `json:"region,omitempty"`
	Compliance   *ComplianceConfig           `json:"compliance,omitempty"`
	S3ObjectLock *S3ObjectLockConfig         `json:"s3ObjectLock,omitempty"`
	DeleteStatus *DeleteObjectStatusConfig   `json:"deleteObjectStatus,omitempty"`
	Replication  *CrossGridReplicationConfig `json:"crossGridReplication,omitempty"`
}

// ComplianceConfig represents compliance settings for the bucket
type ComplianceConfig struct {
	AutoDelete             bool  `json:"autoDelete"`
	LegalHold              bool  `json:"legalHold"`
	RetentionPeriodMinutes int64 `json:"retentionPeriodMinutes"`
}

// S3ObjectLockConfig represents S3 object lock settings
type S3ObjectLockConfig struct {
	Enabled                 bool                     `json:"enabled"`
	DefaultRetentionSetting *DefaultRetentionSetting `json:"defaultRetentionSetting,omitempty"`
}

// DefaultRetentionSetting represents default retention settings for object lock
type DefaultRetentionSetting struct {
	Mode  string `json:"mode"`
	Days  int    `json:"days,omitempty"`
	Years int    `json:"years,omitempty"`
}

// DeleteObjectStatusConfig represents delete status for the bucket
type DeleteObjectStatusConfig struct {
	IsDeletingObjects  bool   `json:"isDeletingObjects"`
	InitialObjectCount string `json:"initialObjectCount"`
	InitialObjectBytes string `json:"initialObjectBytes"`
}

// CrossGridReplicationConfig represents cross-grid replication settings
type CrossGridReplicationConfig struct {
	Rules []interface{} `json:"rules"`
}

// getCachedBucketList retrieves the bucket list with caching support.
// Cache is valid for 5 minutes to balance between performance and freshness.
// NOTE: Using simple caching without mutex for now. In case of concurrent access issues,
// see the comment in Client struct for thread-safe implementation details.
func (c *Client) getCachedBucketList() ([]S3BucketData, error) {
	const cacheTimeout = 5 * time.Minute

	// Simple cache check - potential race condition but not catastrophic
	if time.Since(c.bucketCacheTime) < cacheTimeout && c.bucketCache != nil {
		return c.bucketCache, nil
	}

	// Cache is expired or empty, fetch fresh data
	url := fmt.Sprintf("%s/api/v4/org/containers", c.EndpointURL)

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

	// Update cache (potential race condition - multiple goroutines might update simultaneously)
	c.bucketCache = apiResponse.Data
	c.bucketCacheTime = time.Now()

	return c.bucketCache, nil
}

// GetS3Bucket retrieves information about a specific S3 bucket by name.
func (c *Client) GetS3Bucket(bucketName string) (*S3BucketData, error) {
	buckets, err := c.getCachedBucketList()
	if err != nil {
		return nil, err
	}

	// Find the specific bucket in the list
	for _, bucket := range buckets {
		if bucket.Name == bucketName {
			return &bucket, nil
		}
	}

	return nil, fmt.Errorf("bucket %s not found", bucketName)
}
