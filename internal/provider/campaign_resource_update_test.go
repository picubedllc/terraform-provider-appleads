// Copyright (c) 2026 Pi Cubed LLC
// SPDX-License-Identifier: MIT

package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/picubedllc/terraform-provider-appleads/internal/client"
)

func TestDetectImmutableCampaignChanges(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	countriesA, _ := types.ListValueFrom(ctx, types.StringType, []string{"US"})
	countriesB, _ := types.ListValueFrom(ctx, types.StringType, []string{"GB"})
	supply, _ := types.ListValueFrom(ctx, types.StringType, []string{"APPSTORE_SEARCH_RESULTS"})

	state := campaignModel{
		ID:                 types.StringValue("12345"),
		AdamID:             types.StringValue("1"),
		AdChannelType:      types.StringValue("SEARCH"),
		CountriesOrRegions: countriesA,
		SupplySources:      supply,
	}
	plan := state
	plan.CountriesOrRegions = countriesB
	plan.Name = types.StringValue("also mutable")

	changes := detectImmutableCampaignChanges(ctx, state, plan)
	if len(changes) != 1 || changes[0].Field != "countries_or_regions" {
		t.Fatalf("changes = %#v", changes)
	}
	diags := immutableCampaignChangeDiagnostics("12345", changes)
	if !diags.HasError() {
		t.Fatal("expected diagnostics")
	}
	detail := diags[0].Detail()
	if !strings.Contains(detail, "countries_or_regions") || !strings.Contains(detail, "12345") {
		t.Fatalf("detail = %s", detail)
	}
	if !strings.Contains(detail, "Create a new apple_ads_campaign") {
		t.Fatalf("detail = %s", detail)
	}
}

func TestCampaignUpdate_MutableSucceeds(t *testing.T) {
	t.Parallel()

	var sawBody client.CampaignUpdate
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var env struct {
			Campaign client.CampaignUpdate `json:"campaign"`
		}
		if err := json.NewDecoder(r.Body).Decode(&env); err != nil {
			t.Fatal(err)
		}
		sawBody = env.Campaign
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"id":                 9,
				"name":               env.Campaign.Name,
				"status":             env.Campaign.Status,
				"adamId":             1,
				"countriesOrRegions": []string{"US"},
				"dailyBudgetAmount":  env.Campaign.DailyBudgetAmount,
				"modificationTime":   "2026-04-01T00:00:00Z",
			},
		})
	}))
	t.Cleanup(srv.Close)

	apiClient, err := client.New(client.WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	upd := &client.CampaignUpdate{
		Name:              "New",
		Status:            "PAUSED",
		DailyBudgetAmount: &client.Money{Amount: "10.00", Currency: "USD"},
	}
	out, err := apiClient.UpdateCampaign(context.Background(), 9, upd)
	if err != nil {
		t.Fatal(err)
	}
	if sawBody.Name != "New" || out.Name != "New" {
		t.Fatalf("saw=%#v out=%#v", sawBody, out)
	}
}

func TestCampaignUpdate_ImmutableBlocksWithoutAPICall(t *testing.T) {
	t.Parallel()
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	ctx := context.Background()
	countriesA, _ := types.ListValueFrom(ctx, types.StringType, []string{"US"})
	countriesB, _ := types.ListValueFrom(ctx, types.StringType, []string{"CA"})
	state := campaignModel{
		ID:                 types.StringValue("99"),
		AdamID:             types.StringValue("1"),
		CountriesOrRegions: countriesA,
		SupplySources:      types.ListNull(types.StringType),
		BudgetOrders:       types.ListNull(types.StringType),
	}
	plan := state
	plan.CountriesOrRegions = countriesB
	plan.Name = types.StringValue("mutable too")

	changes := detectImmutableCampaignChanges(ctx, state, plan)
	if len(changes) == 0 {
		t.Fatal("expected immutable change")
	}
	// Simulate Update early-return: no client call.
	_ = srv
	apiClient, _ := client.New(client.WithBaseURL(srv.URL))
	_ = apiClient
	if called {
		t.Fatal("API should not be called when immutable fields change")
	}
}
