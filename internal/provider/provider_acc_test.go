// Copyright (c) 2026 Pi Cubed LLC
// SPDX-License-Identifier: MIT

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
	"appleads": providerserver.NewProtocol6WithError(New("test")()),
}

func testAccPreCheck(t *testing.T) {
	t.Helper()
	required := []string{
		"APPLEADS_ORG_ID",
		"APPLEADS_CLIENT_ID",
		"APPLEADS_TEAM_ID",
		"APPLEADS_KEY_ID",
		"APPLEADS_PRIVATE_KEY",
		"APPLEADS_TEST_ADAM_ID",
	}
	for _, k := range required {
		if os.Getenv(k) == "" {
			t.Fatalf("%s must be set for acceptance tests", k)
		}
	}
}

func testAccProviderConfig(allowDeletion bool) string {
	allow := "false"
	if allowDeletion {
		allow = "true"
	}
	return `
provider "appleads" {
  allow_campaign_deletion = ` + allow + `
}
`
}
