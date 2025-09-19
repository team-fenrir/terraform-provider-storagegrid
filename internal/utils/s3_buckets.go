// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
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

// S3BucketCreateRequest represents the request body for creating an S3 bucket
type S3BucketCreateRequest struct {
	Name         string                    `json:"name"`
	Region       string                    `json:"region"`
	S3ObjectLock *S3BucketCreateObjectLock `json:"s3ObjectLock,omitempty"`
}

// S3BucketCreateObjectLock represents object lock settings for bucket creation
type S3BucketCreateObjectLock struct {
	Enabled                 bool                            `json:"enabled"`
	DefaultRetentionSetting *S3BucketCreateRetentionSetting `json:"defaultRetentionSetting,omitempty"`
}

// S3BucketCreateRetentionSetting represents default retention settings for bucket creation
type S3BucketCreateRetentionSetting struct {
	Mode string `json:"mode"`
	Days int    `json:"days"`
}

// S3BucketCreateResponse represents the API response structure for bucket creation
type S3BucketCreateResponse struct {
	ResponseTime string                  `json:"responseTime"`
	Status       string                  `json:"status"`
	APIVersion   string                  `json:"apiVersion"`
	Deprecated   bool                    `json:"deprecated"`
	Data         S3BucketCreateData      `json:"data"`
	Metadata     *S3BucketCreateMetadata `json:"metadata,omitempty"`
}

// S3BucketCreateData represents the data returned after bucket creation
type S3BucketCreateData struct {
	Name   string `json:"name"`
	Region string `json:"region"`
}

// S3BucketCreateMetadata represents metadata including alerts
type S3BucketCreateMetadata struct {
	Alerts []S3BucketAlert `json:"alerts,omitempty"`
}

// S3BucketAlert represents alert information
type S3BucketAlert struct {
	Deprecated bool   `json:"deprecated"`
	Severity   string `json:"severity"`
	Text       string `json:"text"`
	Key        string `json:"key"`
}

// CreateS3Bucket creates a new S3 bucket with the specified name, region, and object lock settings
func (c *Client) CreateS3Bucket(bucketName, region string, objectLockEnabled bool) error {
	url := fmt.Sprintf("%s/api/v4/org/containers", c.EndpointURL)
	log.Printf("Executing POST request to URL: %s", url)

	createRequest := S3BucketCreateRequest{
		Name:   bucketName,
		Region: region,
	}

	// Add object lock configuration if enabled
	if objectLockEnabled {
		createRequest.S3ObjectLock = &S3BucketCreateObjectLock{
			Enabled: true,
			DefaultRetentionSetting: &S3BucketCreateRetentionSetting{
				Mode: "governance", // Lighter than compliance mode
				Days: 1,            // Default to 1 day to avoid problems
			},
		}
	} else {
		createRequest.S3ObjectLock = &S3BucketCreateObjectLock{
			Enabled: false,
		}
	}

	requestBody, err := json.Marshal(createRequest)
	if err != nil {
		return fmt.Errorf("error marshalling bucket create request: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	body, err := c.doRequest(req)
	if err != nil {
		return fmt.Errorf("error executing request: %w", err)
	}

	var apiResponse S3BucketCreateResponse
	if err := json.Unmarshal(body, &apiResponse); err != nil {
		return fmt.Errorf("error unmarshalling bucket create response: %w", err)
	}

	if apiResponse.Status != "success" {
		return fmt.Errorf("bucket creation failed with status: %s", apiResponse.Status)
	}

	// Clear cache since we created a new bucket
	c.bucketCache = nil
	c.bucketCacheTime = time.Time{}

	return nil
}

// DeleteS3Bucket deletes an S3 bucket by name
func (c *Client) DeleteS3Bucket(bucketName string) error {
	url := fmt.Sprintf("%s/api/v4/org/containers/%s", c.EndpointURL, bucketName)
	log.Printf("Executing DELETE request to URL: %s", url)

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("error creating DELETE request: %w", err)
	}

	_, err = c.doRequest(req)
	if err != nil {
		// Check if this is a timeout error
		if isTimeoutError(err) {
			log.Printf("Delete request timed out, checking if bucket was actually deleted...")

			// Wait a moment for the operation to complete
			time.Sleep(2 * time.Second)

			// Check if bucket still exists
			_, checkErr := c.GetS3Bucket(bucketName)
			if checkErr != nil && strings.Contains(checkErr.Error(), "not found") {
				// Bucket was successfully deleted despite timeout
				log.Printf("Bucket %s was successfully deleted despite timeout", bucketName)
				c.bucketCache = nil
				c.bucketCacheTime = time.Time{}
				return nil
			}
		}
		return fmt.Errorf("error executing DELETE request: %w", err)
	}

	// Clear cache since we successfully deleted a bucket
	c.bucketCache = nil
	c.bucketCacheTime = time.Time{}

	return nil
}

// isTimeoutError checks if an error is a timeout error
func isTimeoutError(err error) bool {
	if err == nil {
		return false
	}

	// Check for context deadline exceeded
	if err == context.DeadlineExceeded {
		return true
	}

	// Check for net timeout errors
	if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
		return true
	}

	// Check for timeout in error message
	errStr := err.Error()
	return strings.Contains(errStr, "timeout") ||
		   strings.Contains(errStr, "deadline exceeded") ||
		   strings.Contains(errStr, "Client.Timeout exceeded")
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
