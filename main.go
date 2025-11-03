// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/team-fenrir/terraform-provider-storagegrid/internal/provider"
	"github.com/team-fenrir/terraform-provider-storagegrid/internal/utils"
)

var (
	// these will be set by the goreleaser configuration
	// to appropriate values for the compiled binary.
	version string = "dev"

	// goreleaser can pass other information to the main package, such as the specific commit
	// https://goreleaser.com/cookbooks/using-main.version/
)

func main() {
	var debug bool

	flag.BoolVar(&debug, "debug", false, "set to true to run the provider with support for debuggers like delve")
	flag.Parse()

	// Setup cleanup handler for graceful shutdown
	// This ensures temporary S3 access keys are deleted when the provider exits
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	// Cleanup goroutine
	go func() {
		<-sigChan
		log.Println("Received shutdown signal, cleaning up...")
		utils.CleanupAllClients()
		os.Exit(0)
	}()

	// Also cleanup on normal exit using defer
	defer func() {
		log.Println("Provider shutting down, cleaning up...")
		utils.CleanupAllClients()
	}()

	opts := providerserver.ServeOpts{
		// TODO: Update this string with the published name of your provider.
		// Also update the tfplugindocs generate command to either remove the
		// -provider-name flag or set its value to the updated provider name.
		Address: "registry.terraform.io/team-fenrir/storagegrid",
		Debug:   debug,
	}

	err := providerserver.Serve(context.Background(), provider.New(version), opts)

	if err != nil {
		log.Fatal(err.Error())
	}
}
