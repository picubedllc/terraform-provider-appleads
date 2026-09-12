// Copyright (c) 2026 Pi Cubed LLC
// SPDX-License-Identifier: MIT

package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/picubedllc/terraform-provider-appleads/internal/acctest"
)

// testAccProtoV6ProviderFactories are used to instantiate a provider during
// acceptance testing. The factory function will be invoked for every Terraform
// CLI command executed to create a provider server to which the CLI can
// reattach.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"apple-ads": providerserver.NewProtocol6WithError(New("test")()),
}

func testAccPreCheck(t *testing.T) {
	t.Helper()
	acctest.PreCheck(t)
	required := []string{
		"APPLEADS_TEST_ADAM_ID",
	}
	for _, k := range required {
		if os.Getenv(k) == "" {
			t.Fatalf("%s must be set for campaign acceptance tests", k)
		}
	}
}

func testAccProviderConfig(allowDeletion bool) string {
	allow := "false"
	if allowDeletion {
		allow = "true"
	}
	return `
provider "apple-ads" {
  allow_campaign_deletion = ` + allow + `
}
`
}

func TestAccProvider_ConfigureFromEnvAndReadACL(t *testing.T) {
	acctest.PreCheck(t)

	p := New("test")()
	schemaResp := &provider.SchemaResponse{}
	p.Schema(t.Context(), provider.SchemaRequest{}, schemaResp)

	config := tfsdk.Config{
		Schema: schemaResp.Schema,
		Raw: tftypes.NewValue(schemaResp.Schema.Type().TerraformType(t.Context()), map[string]tftypes.Value{
			"org_id":                  tftypes.NewValue(tftypes.String, nil),
			"client_id":               tftypes.NewValue(tftypes.String, nil),
			"team_id":                 tftypes.NewValue(tftypes.String, nil),
			"key_id":                  tftypes.NewValue(tftypes.String, nil),
			"private_key":             tftypes.NewValue(tftypes.String, nil),
			"allow_campaign_deletion": tftypes.NewValue(tftypes.Bool, nil),
		}),
	}

	resp := &provider.ConfigureResponse{}
	p.Configure(t.Context(), provider.ConfigureRequest{Config: config}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("configure diagnostics: %#v", resp.Diagnostics)
	}
	data, ok := resp.ResourceData.(*ProviderData)
	if !ok || data == nil || data.Client == nil {
		t.Fatalf("ResourceData = %#v", resp.ResourceData)
	}

	acls := acctest.RequireUserACLs(t, data.Client)
	acctest.RequireOrgAccess(t, acls, acctest.Credentials(t).OrgID)
}
