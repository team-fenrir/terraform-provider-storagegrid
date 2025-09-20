// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/minio/minio-go/v7/pkg/lifecycle"
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
	Mode  string `json:"-"`
	Days  int    `json:"-"`
	Years int    `json:"-"`
}

// UnmarshalJSON handles conversion of string or number days/years to integers
func (d *DefaultRetentionSetting) UnmarshalJSON(data []byte) error {
	// First try to unmarshal into a flexible structure that can handle both strings and numbers
	aux := &struct {
		Mode  string      `json:"mode"`
		Days  interface{} `json:"days,omitempty"`
		Years interface{} `json:"years,omitempty"`
	}{}

	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	// Set the mode
	d.Mode = aux.Mode

	// Handle days field - can be string or number
	if aux.Days != nil {
		switch v := aux.Days.(type) {
		case string:
			if v != "" {
				if days, err := strconv.Atoi(v); err == nil {
					d.Days = days
				}
			}
		case float64:
			d.Days = int(v)
		case int:
			d.Days = v
		}
	}

	// Handle years field - can be string or number
	if aux.Years != nil {
		switch v := aux.Years.(type) {
		case string:
			if v != "" {
				if years, err := strconv.Atoi(v); err == nil {
					d.Years = years
				}
			}
		case float64:
			d.Years = int(v)
		case int:
			d.Years = v
		}
	}

	return nil
}

// MarshalJSON handles conversion of integer days/years for API requests
func (d *DefaultRetentionSetting) MarshalJSON() ([]byte, error) {
	// Create a struct that includes the fields we want to marshal
	aux := &struct {
		Mode  string `json:"mode"`
		Days  *int   `json:"days,omitempty"`
		Years *int   `json:"years,omitempty"`
	}{
		Mode: d.Mode,
	}

	// Only include the field that has a value > 0, prioritizing years
	if d.Years > 0 {
		aux.Years = &d.Years
	} else {
		// Always send days if years is not set, even if it's 0
		aux.Days = &d.Days
	}

	return json.Marshal(aux)
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

// S3BucketVersioningAPIResponse represents the API response structure for bucket versioning
type S3BucketVersioningAPIResponse struct {
	ResponseTime string                 `json:"responseTime"`
	Status       string                 `json:"status"`
	APIVersion   string                 `json:"apiVersion"`
	Deprecated   bool                   `json:"deprecated"`
	Data         S3BucketVersioningData `json:"data"`
}

// S3BucketVersioningData represents the versioning configuration for an S3 bucket
type S3BucketVersioningData struct {
	VersioningEnabled   bool `json:"versioningEnabled"`
	VersioningSuspended bool `json:"versioningSuspended"`
}

// GetS3BucketVersioning retrieves versioning configuration for a specific S3 bucket
func (c *Client) GetS3BucketVersioning(bucketName string) (*S3BucketVersioningData, error) {
	url := fmt.Sprintf("%s/api/v4/org/containers/%s/versioning", c.EndpointURL, bucketName)
	log.Printf("Executing GET request to URL: %s", url)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	body, err := c.doRequest(req)
	if err != nil {
		return nil, fmt.Errorf("error executing request: %w", err)
	}

	var apiResponse S3BucketVersioningAPIResponse
	if err := json.Unmarshal(body, &apiResponse); err != nil {
		return nil, fmt.Errorf("error unmarshalling S3 bucket versioning response: %w", err)
	}

	return &apiResponse.Data, nil
}

// S3BucketVersioningUpdateRequest represents the request body for updating bucket versioning
type S3BucketVersioningUpdateRequest struct {
	VersioningEnabled   bool `json:"versioningEnabled"`
	VersioningSuspended bool `json:"versioningSuspended"`
}

// UpdateS3BucketVersioning updates versioning configuration for a specific S3 bucket
func (c *Client) UpdateS3BucketVersioning(bucketName string, versioningEnabled, versioningSuspended bool) error {
	url := fmt.Sprintf("%s/api/v4/org/containers/%s/versioning", c.EndpointURL, bucketName)
	log.Printf("Executing PUT request to URL: %s", url)

	updateRequest := S3BucketVersioningUpdateRequest{
		VersioningEnabled:   versioningEnabled,
		VersioningSuspended: versioningSuspended,
	}

	requestBody, err := json.Marshal(updateRequest)
	if err != nil {
		return fmt.Errorf("error marshalling bucket versioning update request: %w", err)
	}

	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return fmt.Errorf("error creating PUT request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	body, err := c.doRequest(req)
	if err != nil {
		return fmt.Errorf("error executing PUT request: %w", err)
	}

	var apiResponse S3BucketVersioningAPIResponse
	if err := json.Unmarshal(body, &apiResponse); err != nil {
		return fmt.Errorf("error unmarshalling bucket versioning update response: %w", err)
	}

	if apiResponse.Status != "success" {
		return fmt.Errorf("bucket versioning update failed with status: %s", apiResponse.Status)
	}

	return nil
}

// S3BucketObjectLockAPIResponse represents the API response structure for bucket object lock
type S3BucketObjectLockAPIResponse struct {
	ResponseTime string                 `json:"responseTime"`
	Status       string                 `json:"status"`
	APIVersion   string                 `json:"apiVersion"`
	Deprecated   bool                   `json:"deprecated"`
	Data         S3BucketObjectLockData `json:"data"`
}

// S3BucketObjectLockData represents the object lock configuration for an S3 bucket
type S3BucketObjectLockData struct {
	Enabled                 bool                     `json:"enabled"`
	DefaultRetentionSetting *DefaultRetentionSetting `json:"defaultRetentionSetting,omitempty"`
}

// GetS3BucketObjectLock retrieves object lock configuration for a specific S3 bucket
func (c *Client) GetS3BucketObjectLock(bucketName string) (*S3BucketObjectLockData, error) {
	url := fmt.Sprintf("%s/api/v4/org/containers/%s/object-lock", c.EndpointURL, bucketName)
	log.Printf("Executing GET request to URL: %s", url)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	body, err := c.doRequest(req)
	if err != nil {
		return nil, fmt.Errorf("error executing request: %w", err)
	}

	var apiResponse S3BucketObjectLockAPIResponse
	if err := json.Unmarshal(body, &apiResponse); err != nil {
		return nil, fmt.Errorf("error unmarshalling S3 bucket object lock response: %w", err)
	}

	return &apiResponse.Data, nil
}

// S3BucketObjectLockUpdateRequest represents the request body for updating bucket object lock
type S3BucketObjectLockUpdateRequest struct {
	Enabled                 bool                     `json:"enabled"`
	DefaultRetentionSetting *DefaultRetentionSetting `json:"defaultRetentionSetting,omitempty"`
}

// UpdateS3BucketObjectLock updates object lock configuration for a specific S3 bucket
func (c *Client) UpdateS3BucketObjectLock(bucketName string, enabled bool, defaultRetentionSetting *DefaultRetentionSetting) error {
	url := fmt.Sprintf("%s/api/v4/org/containers/%s/object-lock", c.EndpointURL, bucketName)
	log.Printf("Executing PUT request to URL: %s", url)

	updateRequest := S3BucketObjectLockUpdateRequest{
		Enabled:                 enabled,
		DefaultRetentionSetting: defaultRetentionSetting,
	}

	requestBody, err := json.Marshal(updateRequest)
	if err != nil {
		return fmt.Errorf("error marshalling bucket object lock update request: %w", err)
	}

	log.Printf("Request body: %s", string(requestBody))

	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return fmt.Errorf("error creating PUT request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	body, err := c.doRequest(req)
	if err != nil {
		return fmt.Errorf("error executing PUT request: %w", err)
	}

	var apiResponse S3BucketObjectLockAPIResponse
	if err := json.Unmarshal(body, &apiResponse); err != nil {
		return fmt.Errorf("error unmarshalling bucket object lock update response: %w", err)
	}

	if apiResponse.Status != "success" {
		return fmt.Errorf("bucket object lock update failed with status: %s", apiResponse.Status)
	}

	return nil
}

// S3 Lifecycle Configuration structures for XML marshalling/unmarshalling

// LifecycleConfiguration represents the root lifecycle configuration
type LifecycleConfiguration struct {
	XMLName xml.Name `xml:"LifecycleConfiguration"`
	Rules   []Rule   `xml:"Rule"`
}

// Rule represents a lifecycle rule
type Rule struct {
	ID                          string                       `xml:"ID,omitempty"`
	Status                      string                       `xml:"Status"`
	Filter                      *Filter                      `xml:"Filter,omitempty"`
	Expiration                  *Expiration                  `xml:"Expiration,omitempty"`
	NoncurrentVersionExpiration *NoncurrentVersionExpiration `xml:"NoncurrentVersionExpiration,omitempty"`
}

// Filter represents the filter for a lifecycle rule
type Filter struct {
	Prefix string `xml:"Prefix,omitempty"`
}

// Expiration represents expiration settings for current versions
type Expiration struct {
	Days int    `xml:"Days,omitempty"`
	Date string `xml:"Date,omitempty"`
}

// NoncurrentVersionExpiration represents expiration settings for noncurrent versions
type NoncurrentVersionExpiration struct {
	NoncurrentDays int `xml:"NoncurrentDays,omitempty"`
}

// S3AccessKeyResponse represents the API response for access key creation
type S3AccessKeyResponse struct {
	ResponseTime string      `json:"responseTime"`
	Status       string      `json:"status"`
	APIVersion   string      `json:"apiVersion"`
	Data         s3AccessKey `json:"data"`
}

// GetS3EndpointURL converts the management endpoint to S3 endpoint (port 10443)
func (c *Client) GetS3EndpointURL() string {
	// TODO: Make this configurable later - hardcoded for testing
	return strings.Replace(c.EndpointURL, ":9443", ":10443", 1)
}

// createTemporaryAccessKey creates a temporary access key for S3 operations
func (c *Client) createTemporaryAccessKey() (*s3AccessKey, error) {
	url := fmt.Sprintf("%s/api/v4/org/users/current-user/s3-access-keys", c.EndpointURL)
	log.Printf("Creating temporary access key via URL: %s", url)

	// Create request body for temporary access key with future expiration
	expirationTime := time.Now().Add(24 * time.Hour) // Expire in 24 hours
	requestBody := []byte(fmt.Sprintf(`{"expires": "%s"}`, expirationTime.Format("2006-01-02T15:04:05.000Z")))

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, fmt.Errorf("error creating access key request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	body, err := c.doRequest(req)
	if err != nil {
		return nil, fmt.Errorf("error executing access key request: %w", err)
	}

	var response S3AccessKeyResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("error unmarshalling access key response: %w", err)
	}

	if response.Status != "success" {
		return nil, fmt.Errorf("access key creation failed with status: %s", response.Status)
	}

	return &response.Data, nil
}

// deleteAccessKey deletes a temporary access key
func (c *Client) deleteAccessKey(accessKeyID string) error {
	url := fmt.Sprintf("%s/api/v4/org/users/current-user/s3-access-keys/%s", c.EndpointURL, accessKeyID)
	log.Printf("Deleting access key via URL: %s", url)

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("error creating delete access key request: %w", err)
	}

	_, err = c.doRequest(req)
	if err != nil {
		return fmt.Errorf("error executing delete access key request: %w", err)
	}

	return nil
}

// GetS3Client returns a cached MinIO client, creating it if necessary
func (c *Client) GetS3Client() (*minio.Client, error) {
	// Return cached client if available
	if c.s3Client != nil {
		return c.s3Client, nil
	}

	// Create temporary access key
	accessKey, err := c.createTemporaryAccessKey()
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary access key: %w", err)
	}

	// Parse S3 endpoint
	s3EndpointURL := c.GetS3EndpointURL()
	parsedURL, err := url.Parse(s3EndpointURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse S3 endpoint URL: %w", err)
	}

	// Create MinIO client
	minioClient, err := minio.New(parsedURL.Host, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey.AccessKey, accessKey.SecretKey, ""),
		Secure: parsedURL.Scheme == "https",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create MinIO client: %w", err)
	}

	// Cache the client and access key
	c.s3Client = minioClient
	c.s3AccessKey = accessKey

	log.Printf("Created and cached S3 client with temporary access key")
	return c.s3Client, nil
}

// clearS3ClientCache clears the S3 client cache and deletes the access key
func (c *Client) clearS3ClientCache() {
	if c.s3AccessKey != nil {
		if err := c.deleteAccessKey(c.s3AccessKey.ID); err != nil {
			log.Printf("Warning: failed to delete temporary access key: %v", err)
		}
	}
	c.s3Client = nil
	c.s3AccessKey = nil
}

// executeS3Operation executes an S3 operation with retry on authentication failure
func (c *Client) executeS3Operation(operation func(*minio.Client) error) error {
	client, err := c.GetS3Client()
	if err != nil {
		return fmt.Errorf("failed to get S3 client: %w", err)
	}

	// Try the operation
	err = operation(client)
	if err != nil {
		// Check if it's an authentication/authorization error that might indicate expired session
		errStr := err.Error()
		if strings.Contains(errStr, "AccessDenied") ||
			strings.Contains(errStr, "InvalidAccessKey") ||
			strings.Contains(errStr, "TokenRefreshRequired") ||
			strings.Contains(errStr, "ExpiredToken") {

			log.Printf("S3 operation failed with auth error, clearing cache and retrying: %v", err)

			// Clear cache and retry once
			c.clearS3ClientCache()
			client, retryErr := c.GetS3Client()
			if retryErr != nil {
				return fmt.Errorf("failed to refresh S3 client after auth error: %w", retryErr)
			}

			// Retry the operation
			if retryErr := operation(client); retryErr != nil {
				return fmt.Errorf("S3 operation failed after retry: %w", retryErr)
			}
			return nil
		}
	}

	return err
}

// CleanupS3Client cleans up the cached S3 client and deletes the temporary access key
func (c *Client) CleanupS3Client() {
	c.clearS3ClientCache()
	log.Printf("Cleaned up S3 client and deleted temporary access key")
}

// GetS3AccessKey returns the current S3 access key (for debugging)
func (c *Client) GetS3AccessKey() *s3AccessKey {
	return c.s3AccessKey
}

// GetS3BucketLifecycleConfiguration retrieves lifecycle configuration for a specific S3 bucket
func (c *Client) GetS3BucketLifecycleConfiguration(bucketName string) (*LifecycleConfiguration, error) {
	var result *LifecycleConfiguration
	var operationErr error

	err := c.executeS3Operation(func(client *minio.Client) error {
		log.Printf("Getting lifecycle configuration for bucket: %s", bucketName)

		// Get lifecycle configuration using MinIO client
		lifecycle, err := client.GetBucketLifecycle(context.Background(), bucketName)
		if err != nil {
			return fmt.Errorf("error getting bucket lifecycle configuration: %w", err)
		}

		// Convert MinIO lifecycle to our struct
		lifecycleConfig := &LifecycleConfiguration{
			Rules: make([]Rule, len(lifecycle.Rules)),
		}

		for i, rule := range lifecycle.Rules {
			lifecycleRule := Rule{
				ID:     rule.ID,
				Status: rule.Status,
			}

			// Handle filter
			if rule.RuleFilter.Prefix != "" {
				lifecycleRule.Filter = &Filter{
					Prefix: rule.RuleFilter.Prefix,
				}
			}

			// Handle expiration
			if rule.Expiration.Days > 0 || !rule.Expiration.Date.Time.IsZero() {
				lifecycleRule.Expiration = &Expiration{}
				if rule.Expiration.Days > 0 {
					lifecycleRule.Expiration.Days = int(rule.Expiration.Days)
				}
				if !rule.Expiration.Date.Time.IsZero() {
					lifecycleRule.Expiration.Date = rule.Expiration.Date.Time.Format("2006-01-02T15:04:05.000Z")
				}
			}

			// Handle noncurrent version expiration
			if rule.NoncurrentVersionExpiration.NoncurrentDays > 0 {
				lifecycleRule.NoncurrentVersionExpiration = &NoncurrentVersionExpiration{
					NoncurrentDays: int(rule.NoncurrentVersionExpiration.NoncurrentDays),
				}
			}

			lifecycleConfig.Rules[i] = lifecycleRule
		}

		result = lifecycleConfig
		return nil
	})

	if err != nil {
		return nil, err
	}
	if operationErr != nil {
		return nil, operationErr
	}

	return result, nil
}

// PutS3BucketLifecycleConfiguration sets lifecycle configuration for a specific S3 bucket
func (c *Client) PutS3BucketLifecycleConfiguration(bucketName string, lifecycleConfig *LifecycleConfiguration) error {
	return c.executeS3Operation(func(client *minio.Client) error {
		log.Printf("Setting lifecycle configuration for bucket: %s", bucketName)

		// Convert our struct to MinIO lifecycle format using the proper lifecycle package
		config := lifecycle.NewConfiguration()
		config.Rules = make([]lifecycle.Rule, len(lifecycleConfig.Rules))

		for i, rule := range lifecycleConfig.Rules {
			minioRule := lifecycle.Rule{
				ID:     rule.ID,
				Status: rule.Status,
			}

			// Handle filter
			if rule.Filter != nil {
				minioRule.RuleFilter = lifecycle.Filter{
					Prefix: rule.Filter.Prefix,
				}
			}

			// Handle expiration
			if rule.Expiration != nil {
				if rule.Expiration.Days > 0 {
					minioRule.Expiration.Days = lifecycle.ExpirationDays(rule.Expiration.Days)
				}
				if rule.Expiration.Date != "" {
					if date, err := time.Parse("2006-01-02T15:04:05.000Z", rule.Expiration.Date); err == nil {
						minioRule.Expiration.Date = lifecycle.ExpirationDate{Time: date}
					}
				}
			}

			// Handle noncurrent version expiration
			if rule.NoncurrentVersionExpiration != nil {
				minioRule.NoncurrentVersionExpiration.NoncurrentDays = lifecycle.ExpirationDays(rule.NoncurrentVersionExpiration.NoncurrentDays)
			}

			config.Rules[i] = minioRule
		}

		// Set lifecycle configuration using MinIO client
		err := client.SetBucketLifecycle(context.Background(), bucketName, config)
		if err != nil {
			return fmt.Errorf("error setting bucket lifecycle configuration: %w", err)
		}

		return nil
	})
}

// DeleteS3BucketLifecycleConfiguration deletes lifecycle configuration for a specific S3 bucket
func (c *Client) DeleteS3BucketLifecycleConfiguration(bucketName string) error {
	return c.executeS3Operation(func(client *minio.Client) error {
		log.Printf("Deleting lifecycle configuration for bucket: %s", bucketName)

		// Remove lifecycle configuration using MinIO client
		err := client.SetBucketLifecycle(context.Background(), bucketName, lifecycle.NewConfiguration())
		if err != nil {
			return fmt.Errorf("error removing bucket lifecycle configuration: %w", err)
		}

		return nil
	})
}
