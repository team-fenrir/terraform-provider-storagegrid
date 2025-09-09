// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

// S3AccessKeyBaseResponse contains common fields for all S3 key API responses.
type S3AccessKeyBaseResponse struct {
	ResponseTime string `json:"responseTime"`
	Status       string `json:"status"`
	APIVersion   string `json:"apiVersion"`
	Deprecated   bool   `json:"deprecated"`
}

// S3AccessKeyData represents the non-sensitive data for a single S3 access key.
// This is used when listing existing keys.
type S3AccessKeyData struct {
	ID          string `json:"id"`
	AccountID   string `json:"accountId"`
	DisplayName string `json:"displayName"`
	UserURN     string `json:"userURN"`
	UserUUID    string `json:"userUUID"`
	Expires     string `json:"expires,omitempty"`
}

// S3AccessKeyCreateData represents the data returned on key creation,
// including the sensitive secret.
type S3AccessKeyCreateData struct {
	S3AccessKeyData        // Embeds all fields from S3AccessKeyData
	AccessKey       string `json:"accessKey"`
	SecretAccessKey string `json:"secretAccessKey"`
}

// S3AccessKeyCreateAPIResponse is the full API response for a POST (create) request.
type S3AccessKeyCreateAPIResponse struct {
	S3AccessKeyBaseResponse
	Data S3AccessKeyCreateData `json:"data"`
}

// S3AccessKeyListAPIResponse is the full API response for a GET (list) request.
type S3AccessKeyListAPIResponse struct {
	S3AccessKeyBaseResponse
	Data []S3AccessKeyData `json:"data"`
}

// S3AccessKeyCreatePayload defines the request body for creating a new access key.
type S3AccessKeyCreatePayload struct {
	Expires *string `json:"expires,omitempty"`
}

// GetS3AccessKeys fetches all S3 access keys for a given user.
func (c *Client) GetS3AccessKeys(userID string) (*S3AccessKeyListAPIResponse, error) {
	url := fmt.Sprintf("%s/api/v4/org/users/%s/s3-access-keys?includeCloneStatus=false", c.EndpointURL, userID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	body, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}

	var keysResponse S3AccessKeyListAPIResponse
	if err := json.Unmarshal(body, &keysResponse); err != nil {
		return nil, fmt.Errorf("error unmarshaling list s3 access keys response: %w", err)
	}

	return &keysResponse, nil
}

// CreateS3AccessKey creates a new S3 access key for a user.
func (c *Client) CreateS3AccessKey(userID string, payload S3AccessKeyCreatePayload) (*S3AccessKeyCreateAPIResponse, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("error marshaling create s3 access key payload: %w", err)
	}

	url := fmt.Sprintf("%s/api/v4/org/users/%s/s3-access-keys", c.EndpointURL, userID)
	log.Printf("Executing POST request to URL: %s", url)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, err
	}

	body, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}

	var createdKeyResponse S3AccessKeyCreateAPIResponse
	if err := json.Unmarshal(body, &createdKeyResponse); err != nil {
		return nil, fmt.Errorf("error unmarshaling create s3 access key response: %w", err)
	}

	return &createdKeyResponse, nil
}

// DeleteS3AccessKey deletes a specific S3 access key.
func (c *Client) DeleteS3AccessKey(userID, keyID string) error {
	url := fmt.Sprintf("%s/api/v4/org/users/%s/s3-access-keys/%s", c.EndpointURL, userID, keyID)
	log.Printf("Executing DELETE request to URL: %s", url)

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("error creating DELETE request: %w", err)
	}

	_, err = c.doRequest(req)
	if err != nil {
		// The API returns a 204 No Content on success, so a non-error response is sufficient.
		return fmt.Errorf("error executing DELETE request: %w", err)
	}

	return nil
}
