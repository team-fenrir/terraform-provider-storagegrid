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

// UserAPIResponse represents the full API response for a single user.
type UserAPIResponse struct {
	ResponseTime string   `json:"responseTime"`
	Status       string   `json:"status"`
	APIVersion   string   `json:"apiVersion"`
	Data         UserData `json:"data"`
}

// UserData represents the detailed information about a single user.
type UserData struct {
	ID         string   `json:"id"`
	AccountID  string   `json:"accountId"`
	FullName   string   `json:"fullName"`
	UniqueName string   `json:"uniqueName"`
	UserURN    string   `json:"userURN"`
	Federated  bool     `json:"federated"`
	MemberOf   []string `json:"memberOf"`
	Disable    bool     `json:"disable"`
}

// UserPayload defines the request body for creating or updating a user.
type UserPayload struct {
	UniqueName string   `json:"uniqueName"`
	FullName   string   `json:"fullName"`
	MemberOf   []string `json:"memberOf"`
	Disable    bool     `json:"disable"`
}

// ChangePasswordPayload defines the request body for changing a user's password.
type ChangePasswordPayload struct {
	Password string `json:"password"`
}

func (c *Client) GetUser(id string) (*UserAPIResponse, error) {
	url := fmt.Sprintf("%s/api/v4/org/users/%s", c.EndpointURL, id)
	log.Printf("Executing GET request to URL: %s", url)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating GET request: %w", err)
	}

	body, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}

	var userResponse UserAPIResponse
	err = json.Unmarshal(body, &userResponse)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling user response: %w", err)
	}

	return &userResponse, nil
}

func (c *Client) CreateUser(payload UserPayload) (*UserAPIResponse, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("error marshaling create user payload: %w", err)
	}

	url := fmt.Sprintf("%s/api/v4/org/users", c.EndpointURL)
	log.Printf("Executing POST request to URL: %s with payload %s", url, string(payloadBytes))

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, fmt.Errorf("error creating create user request: %w", err)
	}

	body, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}

	var createdUser UserAPIResponse
	err = json.Unmarshal(body, &createdUser)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling create user response: %w", err)
	}

	return &createdUser, nil
}

func (c *Client) UpdateUser(id string, payload UserPayload) (*UserAPIResponse, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("error marshaling update user payload: %w", err)
	}

	url := fmt.Sprintf("%s/api/v4/org/users/%s", c.EndpointURL, id)
	log.Printf("Executing PUT request to URL: %s with payload %s", url, string(payloadBytes))

	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, fmt.Errorf("error creating update user request: %w", err)
	}

	body, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}

	var updatedUser UserAPIResponse
	err = json.Unmarshal(body, &updatedUser)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling update user response: %w", err)
	}

	return &updatedUser, nil
}

func (c *Client) DeleteUser(id string) error {
	url := fmt.Sprintf("%s/api/v4/org/users/%s", c.EndpointURL, id)
	log.Printf("Executing DELETE request to URL: %s", url)

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("error creating DELETE request: %w", err)
	}

	_, err = c.doRequest(req)
	if err != nil {
		return fmt.Errorf("error executing DELETE request: %w", err)
	}

	return nil
}

// ChangeUserPassword updates the password for a local tenant user.
// The shortName parameter should be the user's unique name (e.g., "user/username").
func (c *Client) ChangeUserPassword(shortName string, password string) error {
	payload := ChangePasswordPayload{
		Password: password,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("error marshaling change password payload: %w", err)
	}

	url := fmt.Sprintf("%s/api/v4/org/users/%s/change-password", c.EndpointURL, shortName)
	log.Printf("Executing POST request to URL: %s", url)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return fmt.Errorf("error creating change password request: %w", err)
	}

	_, err = c.doRequest(req)
	if err != nil {
		return fmt.Errorf("error executing change password request: %w", err)
	}

	return nil
}
