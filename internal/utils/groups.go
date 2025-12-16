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

// GroupAPIResponse represents the full API response object.
type GroupAPIResponse struct {
	ResponseTime string    `json:"responseTime"`
	Status       string    `json:"status"`
	APIVersion   string    `json:"apiVersion"`
	Data         GroupData `json:"data"`
}

// Group represents the detailed information about a single group.
type GroupData struct {
	ID                 string   `json:"id"`
	AccountID          string   `json:"accountId"`
	DisplayName        string   `json:"displayName"`
	UniqueName         string   `json:"uniqueName"`
	GroupURN           string   `json:"groupURN"`
	Federated          bool     `json:"federated"`
	ManagementReadOnly bool     `json:"managementReadOnly"`
	Policies           Policies `json:"policies"`
}

// Policies contains the policy definitions for the group.
type Policies struct {
	Management ManagementPolicy `json:"management,omitempty"`
	S3         S3Policy         `json:"s3"`
}

// S3Policy holds the details for an S3 policy.
type S3Policy struct {
	Id        string      `json:"Id,omitempty"`
	Version   string      `json:"Version,omitempty"`
	Statement []Statement `json:"Statement"`
}

// Statement defines a single rule within a policy.
type Statement struct {
	Sid       string                              `json:"Sid,omitempty"`
	Effect    string                              `json:"Effect"`
	Action    StringOrSlice                       `json:"Action"`
	Resource  StringOrSlice                       `json:"Resource"`
	Condition map[string]map[string]StringOrSlice `json:"Condition,omitempty"`
}

type ManagementPolicy struct {
	ManageAllContainers       bool `json:"manageAllContainers"`
	ManageEndpoints           bool `json:"manageEndpoints"`
	ManageOwnContainerObjects bool `json:"manageOwnContainerObjects"`
	ManageOwnS3Credentials    bool `json:"manageOwnS3Credentials"`
	RootAccess                bool `json:"rootAccess"`
	ViewAllContainers         bool `json:"viewAllContainers"`
}

type GroupPayload struct {
	UniqueName         string   `json:"uniqueName"`
	DisplayName        string   `json:"displayName"`
	ManagementReadOnly bool     `json:"managementReadOnly"`
	Policies           Policies `json:"policies"`
}

func (c *Client) GetGroup(id string) (*GroupAPIResponse, error) {
	url := fmt.Sprintf("%s/api/v4/org/groups/%s", c.EndpointURL, id)
	log.Printf("%s", url)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	body, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}
	fmt.Print(string(body))

	group := GroupAPIResponse{}
	err = json.Unmarshal(body, &group)
	if err != nil {
		return nil, err
	}

	return &group, nil
}

func (c *Client) CreateGroup(payload GroupPayload) (*GroupAPIResponse, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("error marshaling create group payload: %w", err)
	}

	url := fmt.Sprintf("%s/api/v4/org/groups", c.EndpointURL)
	log.Printf("Executing POST request to URL: %s", url)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, err
	}

	body, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}

	var createdGroup GroupAPIResponse
	err = json.Unmarshal(body, &createdGroup)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling create group response: %w", err)
	}

	return &createdGroup, nil
}

func (c *Client) UpdateGroup(id string, payload GroupPayload) (*GroupAPIResponse, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("error marshaling update policies payload: %w", err)
	}

	url := fmt.Sprintf("%s/api/v4/org/groups/%s", c.EndpointURL, id)
	log.Printf("Executing PUT request to URL: %s", url)

	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, err
	}

	body, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}

	var updatedGroup GroupAPIResponse
	err = json.Unmarshal(body, &updatedGroup)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling update group response: %w", err)
	}

	return &updatedGroup, nil
}

func (c *Client) DeleteGroup(id string) error {
	url := fmt.Sprintf("%s/api/v4/org/groups/%s", c.EndpointURL, id)
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
