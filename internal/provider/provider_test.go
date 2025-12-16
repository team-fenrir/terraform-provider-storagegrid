// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// testAccProtoV6ProviderFactories are used to instantiate a provider during
// acceptance testing. The factory function will be invoked for every Terraform
// CLI command executed to create a provider server to which the CLI can
// reattach.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"storagegrid": providerserver.NewProtocol6WithError(New("test")()),
}

// testAccPreCheck validates that the required environment variables are set
// for acceptance tests to run.
func testAccPreCheck(t *testing.T) {
	t.Helper()

	// Check if acceptance tests are enabled (must be explicitly "1")
	if os.Getenv("TF_ACC") != "1" {
		t.Skip("Acceptance tests skipped unless env 'TF_ACC=1' is set")
	}

	// Check required environment variables
	requiredEnvVars := []string{
		"STORAGEGRID_ENDPOINT",
		"STORAGEGRID_ACCOUNTID",
		"STORAGEGRID_USERNAME",
		"STORAGEGRID_PASSWORD",
	}

	missingVars := []string{}
	for _, envVar := range requiredEnvVars {
		if os.Getenv(envVar) == "" {
			missingVars = append(missingVars, envVar)
		}
	}

	if len(missingVars) > 0 {
		t.Skipf("Acceptance tests skipped: missing required environment variables: %v", missingVars)
	}
}

// providerConfig returns the provider configuration using environment variables.
// This allows tests to connect to a real StorageGRID instance.
const providerConfig = `
provider "storagegrid" {
  # Configuration is read from environment variables:
  # - STORAGEGRID_ENDPOINT
  # - STORAGEGRID_S3_ENDPOINT (optional)
  # - STORAGEGRID_ACCOUNTID
  # - STORAGEGRID_USERNAME
  # - STORAGEGRID_PASSWORD
}
`
