// Copyright (c) 2026 Pi Cubed LLC
// SPDX-License-Identifier: MIT

package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/picubedllc/terraform-provider-appleads/internal/client"
)

// TestMaxConversionsAutoAdGroup_ListThenImportFormats covers the documented
// happy path without live Apple credentials: after a Max Conversions campaign
// exists, list the auto-created ad group and import it with both existing
// appleads_ad_group import id formats.
func TestMaxConversionsAutoAdGroup_ListThenImportFormats(t *testing.T) {
	t.Parallel()

	const (
		campaignID int64 = 42
		adGroupID  int64 = 99
	)
	autoAG := map[string]any{
		"id":                     adGroupID,
		"campaignId":             campaignID,
		"name":                   "Automated Ad Group",
		"status":                 "ENABLED",
		"servingStatus":          "RUNNING",
		"displayStatus":          "RUNNING",
		"pricingModel":           "CPC",
		"startTime":              "2026-01-01T00:00:00.000",
		"defaultBidAmount":       map[string]string{"amount": "0.01", "currency": "USD"},
		"automatedKeywordsOptIn": true,
		"modificationTime":       "2026-05-01T00:00:00Z",
		"deleted":                false,
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/campaigns/42/adgroups":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data":       []map[string]any{autoAG},
				"pagination": map[string]any{"totalResults": 1, "startIndex": 0, "itemsPerPage": 50},
			})
		case r.Method == http.MethodGet && r.URL.Path == "/campaigns/42/adgroups/99":
			_ = json.NewEncoder(w).Encode(map[string]any{"data": autoAG})
		case r.Method == http.MethodPost && r.URL.Path == "/adgroups/find":
			_ = json.NewEncoder(w).Encode(map[string]any{"data": []map[string]any{autoAG}})
		default:
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	apiClient, err := client.New(client.WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}

	page, err := apiClient.ListAdGroups(context.Background(), campaignID, client.PageParams{Limit: 50, Offset: 0})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Data) != 1 {
		t.Fatalf("expected 1 auto ad group, got %d", len(page.Data))
	}
	found := page.Data[0]
	if found.ID != adGroupID || !found.AutomatedKeywordsOptIn || found.PricingModel != client.PricingModelCPC {
		t.Fatalf("auto AG = %#v", found)
	}

	compoundID := "42/99"
	cid, aid, err := client.ParseAdGroupImportID(compoundID)
	if err != nil || cid != campaignID || aid != adGroupID {
		t.Fatalf("compound parse: %d %d %v", cid, aid, err)
	}
	bareID := "99"
	cid, aid, err = client.ParseAdGroupImportID(bareID)
	if err != nil || cid != 0 || aid != adGroupID {
		t.Fatalf("bare parse: %d %d %v", cid, aid, err)
	}

	r := &adGroupResource{client: apiClient}
	schemaResp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("schema: %v", schemaResp.Diagnostics)
	}

	for _, importID := range []string{compoundID, bareID} {
		t.Run("import_"+strings.ReplaceAll(importID, "/", "_"), func(t *testing.T) {
			t.Parallel()

			state := tfsdk.State{Schema: schemaResp.Schema, Raw: tftypes.NewValue(schemaResp.Schema.Type().TerraformType(context.Background()), nil)}
			resp := &resource.ImportStateResponse{State: state}
			r.ImportState(context.Background(), resource.ImportStateRequest{ID: importID}, resp)
			if resp.Diagnostics.HasError() {
				t.Fatalf("ImportState(%q): %v", importID, resp.Diagnostics)
			}

			var model adGroupModel
			diags := resp.State.Get(context.Background(), &model)
			if diags.HasError() {
				t.Fatalf("state get: %v", diags)
			}
			if model.ID.ValueString() != "99" || model.CampaignID.ValueString() != "42" {
				t.Fatalf("ids = %#v", model)
			}
			if !model.AutomatedKeywordsOptIn.ValueBool() {
				t.Fatal("expected automated_keywords_opt_in true on imported auto AG")
			}
			if model.Name.ValueString() != "Automated Ad Group" {
				t.Fatalf("name = %q", model.Name.ValueString())
			}
			if model.DefaultBidAmount.ValueString() != "0.01" {
				t.Fatalf("default_bid_amount = %q", model.DefaultBidAmount.ValueString())
			}
		})
	}
}

func TestMaxConversionsAutoAdGroup_DocsCoverImportWorkflow(t *testing.T) {
	t.Parallel()

	r := NewCampaignResource()
	schemaResp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("schema: %v", schemaResp.Diagnostics)
	}
	campaignDesc := schemaResp.Schema.GetMarkdownDescription()
	for _, want := range []string{
		"Maximize Conversions automated ad group",
		"outside Terraform state",
		"appleads_ad_group",
		"campaign_id/ad_group_id",
		"default_bid_amount",
		"Display channel",
	} {
		if !strings.Contains(campaignDesc, want) {
			t.Fatalf("campaign schema missing %q", want)
		}
	}

	ag := NewAdGroupResource()
	agSchema := &resource.SchemaResponse{}
	ag.Schema(context.Background(), resource.SchemaRequest{}, agSchema)
	if agSchema.Diagnostics.HasError() {
		t.Fatalf("ad group schema: %v", agSchema.Diagnostics)
	}
	agDesc := agSchema.Schema.GetMarkdownDescription()
	for _, want := range []string{
		"Maximize Conversions auto-created ad groups",
		"campaign_id/ad_group_id",
		"default_bid_amount",
	} {
		if !strings.Contains(agDesc, want) {
			t.Fatalf("ad group schema missing %q", want)
		}
	}

	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "../.."))
	for _, rel := range []string{
		"examples/resources/appleads_ad_group/README.md",
		"examples/resources/appleads_ad_group/import.sh",
		"examples/resources/appleads_campaign/README.md",
	} {
		body, err := os.ReadFile(filepath.Join(repoRoot, rel))
		if err != nil {
			t.Fatalf("read %s: %v", rel, err)
		}
		text := string(body)
		for _, want := range []string{
			"MAX_CONVERSIONS",
			"import",
		} {
			if !strings.Contains(strings.ToLower(text), strings.ToLower(want)) {
				t.Fatalf("%s missing %q", rel, want)
			}
		}
	}

	adGroupREADME, err := os.ReadFile(filepath.Join(repoRoot, "examples/resources/appleads_ad_group/README.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"Automated Ad Group",
		"campaign_id/ad_group_id",
		"default_bid_amount",
		"Keywords",
		"Display",
	} {
		if !strings.Contains(string(adGroupREADME), want) {
			t.Fatalf("ad group README missing %q", want)
		}
	}
}
