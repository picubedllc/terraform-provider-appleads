// Copyright (c) 2026 Pi Cubed LLC
// SPDX-License-Identifier: MIT

package provider

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func testECPrivateKeyPEM(t *testing.T) string {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	der, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: der}))
}

func clearAppleAdsEnv(t *testing.T) {
	t.Helper()
	for _, k := range []string{envOrgID, envClientID, envTeamID, envKeyID, envPrivateKey} {
		t.Setenv(k, "")
		_ = os.Unsetenv(k)
	}
}

func TestConfigure_MissingCredentials(t *testing.T) {
	clearAppleAdsEnv(t)

	p := New("test")()
	schemaResp := &provider.SchemaResponse{}
	p.Schema(context.Background(), provider.SchemaRequest{}, schemaResp)

	config := tfsdk.Config{
		Schema: schemaResp.Schema,
		Raw: tftypes.NewValue(schemaResp.Schema.Type().TerraformType(context.Background()), map[string]tftypes.Value{
			"org_id":                  tftypes.NewValue(tftypes.String, nil),
			"client_id":               tftypes.NewValue(tftypes.String, nil),
			"team_id":                 tftypes.NewValue(tftypes.String, nil),
			"key_id":                  tftypes.NewValue(tftypes.String, nil),
			"private_key":             tftypes.NewValue(tftypes.String, nil),
			"allow_campaign_deletion": tftypes.NewValue(tftypes.Bool, nil),
		}),
	}

	resp := &provider.ConfigureResponse{}
	p.Configure(context.Background(), provider.ConfigureRequest{Config: config}, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected missing credential diagnostics")
	}
	found := false
	for _, d := range resp.Diagnostics {
		if strings.Contains(d.Summary(), "Missing Apple Ads credential") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("diagnostics = %#v", resp.Diagnostics)
	}
}

func TestConfigure_EnvVarsUsedWhenConfigUnset(t *testing.T) {
	clearAppleAdsEnv(t)
	pemKey := testECPrivateKeyPEM(t)
	t.Setenv(envOrgID, "org-env")
	t.Setenv(envClientID, "client-env")
	t.Setenv(envTeamID, "team-env")
	t.Setenv(envKeyID, "key-env")
	t.Setenv(envPrivateKey, pemKey)

	p := New("test")()
	schemaResp := &provider.SchemaResponse{}
	p.Schema(context.Background(), provider.SchemaRequest{}, schemaResp)

	config := tfsdk.Config{
		Schema: schemaResp.Schema,
		Raw: tftypes.NewValue(schemaResp.Schema.Type().TerraformType(context.Background()), map[string]tftypes.Value{
			"org_id":                  tftypes.NewValue(tftypes.String, nil),
			"client_id":               tftypes.NewValue(tftypes.String, nil),
			"team_id":                 tftypes.NewValue(tftypes.String, nil),
			"key_id":                  tftypes.NewValue(tftypes.String, nil),
			"private_key":             tftypes.NewValue(tftypes.String, nil),
			"allow_campaign_deletion": tftypes.NewValue(tftypes.Bool, false),
		}),
	}

	resp := &provider.ConfigureResponse{}
	p.Configure(context.Background(), provider.ConfigureRequest{Config: config}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %#v", resp.Diagnostics)
	}
	data, ok := resp.ResourceData.(*ProviderData)
	if !ok || data == nil || data.Client == nil {
		t.Fatalf("ResourceData = %#v", resp.ResourceData)
	}
	if data.AllowCampaignDeletion {
		t.Fatal("expected allow_campaign_deletion default false")
	}
}

func TestConfigure_ConfigTakesPrecedenceOverEnv(t *testing.T) {
	clearAppleAdsEnv(t)
	pemKey := testECPrivateKeyPEM(t)
	t.Setenv(envOrgID, "org-env")
	t.Setenv(envClientID, "client-env")
	t.Setenv(envTeamID, "team-env")
	t.Setenv(envKeyID, "key-env")
	t.Setenv(envPrivateKey, "invalid-should-not-be-used")

	p := New("test")()
	schemaResp := &provider.SchemaResponse{}
	p.Schema(context.Background(), provider.SchemaRequest{}, schemaResp)

	config := tfsdk.Config{
		Schema: schemaResp.Schema,
		Raw: tftypes.NewValue(schemaResp.Schema.Type().TerraformType(context.Background()), map[string]tftypes.Value{
			"org_id":                  tftypes.NewValue(tftypes.String, "org-cfg"),
			"client_id":               tftypes.NewValue(tftypes.String, "client-cfg"),
			"team_id":                 tftypes.NewValue(tftypes.String, "team-cfg"),
			"key_id":                  tftypes.NewValue(tftypes.String, "key-cfg"),
			"private_key":             tftypes.NewValue(tftypes.String, pemKey),
			"allow_campaign_deletion": tftypes.NewValue(tftypes.Bool, true),
		}),
	}

	resp := &provider.ConfigureResponse{}
	p.Configure(context.Background(), provider.ConfigureRequest{Config: config}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %#v", resp.Diagnostics)
	}
	data, ok := resp.ResourceData.(*ProviderData)
	if !ok || data == nil {
		t.Fatalf("ResourceData = %#v", resp.ResourceData)
	}
	if !data.AllowCampaignDeletion {
		t.Fatal("expected allow_campaign_deletion true from config")
	}
}

func TestProviderServerFactory(t *testing.T) {
	t.Parallel()
	_, err := providerserver.NewProtocol6WithError(New("test")())()
	if err != nil {
		t.Fatal(err)
	}
}
