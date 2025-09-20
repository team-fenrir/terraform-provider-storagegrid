// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/minio/minio-go/v7"
)

// Client holds the client configuration.
type Client struct {
	EndpointURL string
	HTTPClient  *http.Client
	Token       string

	// Cache for bucket list
	// NOTE: Currently using simple caching without mutex for simplicity.
	// If concurrent access issues arise (multiple goroutines corrupting cache or causing panics),
	// add thread safety with sync.RWMutex:
	//   bucketCacheMux  sync.RWMutex
	// Then wrap cache reads with bucketCacheMux.RLock()/RUnlock() and
	// cache writes with bucketCacheMux.Lock()/Unlock() using double-checked locking pattern
	// to prevent race conditions where multiple goroutines fetch/update cache simultaneously.
	bucketCache     []S3BucketData
	bucketCacheTime time.Time

	// S3 client cache for lifecycle operations
	s3Client    *minio.Client
	s3AccessKey *s3AccessKey
}

// s3AccessKey represents temporary access keys for S3 operations
type s3AccessKey struct {
	AccessKey string `json:"accessKey"`
	SecretKey string `json:"secretKey"`
	ID        string `json:"id"`
}

// SignInBody represents the request body for the authentication request.
type SignInBody struct {
	AccountID string `json:"accountId"`
	Username  string `json:"username"`
	Password  string `json:"password"`
	Cookie    bool   `json:"cookie"`
	CsrfToken bool   `json:"csrfToken"`
}

// AuthResponse maps to the JSON response from the authorization endpoint.
type AuthResponse struct {
	ResponseTime string `json:"responseTime"`
	Status       string `json:"status"`
	APIVersion   string `json:"apiVersion"`
	Deprecated   bool   `json:"deprecated"`
	Token        string `json:"data"`
}

// NewClient creates and configures a new API client.
func NewClient(endpoint, accountID, username, password *string) (*Client, error) {
	c := Client{
		EndpointURL: *endpoint,
		HTTPClient:  &http.Client{Timeout: 60 * time.Second}, // Increased timeout for bucket operations
	}

	// If endpoint is not provided, return the client without authenticating.
	if username == nil || password == nil || accountID == nil || endpoint == nil {
		return &c, nil
	}

	// Authenticate and store the token
	authPayload := SignInBody{
		AccountID: *accountID,
		Username:  *username,
		Password:  *password,
		Cookie:    true,
		CsrfToken: false,
	}

	ar, err := c.SignIn(authPayload)
	if err != nil {
		return nil, fmt.Errorf("failed to sign in: %w", err)
	}

	c.Token = ar.Token

	return &c, nil
}

// SignIn handles the authentication process and retrieves a token.
func (c *Client) SignIn(authPayload SignInBody) (*AuthResponse, error) {
	// Marshal the authentication payload into JSON
	payloadBytes, err := json.Marshal(authPayload)
	if err != nil {
		return nil, fmt.Errorf("error marshalling auth payload: %w", err)
	}

	// Create the HTTP request
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/api/v4/authorize", c.EndpointURL), bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	// Set the necessary headers
	req.Header.Set("accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	// Execute the request
	res, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error executing request: %w", err)
	}
	defer res.Body.Close()

	// Read the response body
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status: %d, body: %s", res.StatusCode, body)
	}

	// Unmarshal the response into our AuthResponse struct
	var authResponse AuthResponse
	if err := json.Unmarshal(body, &authResponse); err != nil {
		return nil, fmt.Errorf("error unmarshalling auth response: %w", err)
	}

	return &authResponse, nil
}

// doRequest executes an authenticated API request.
func (c *Client) doRequest(req *http.Request) ([]byte, error) {
	// Set the authorization header with the token obtained during sign-in
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.Token))

	res, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("status: %d, body: %s", res.StatusCode, body)
	}

	return body, nil
}
