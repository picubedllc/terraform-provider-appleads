// Copyright (c) 2026 Pi Cubed LLC
// SPDX-License-Identifier: MIT

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/picubedllc/terraform-provider-appleads/internal/acctest"
)

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
