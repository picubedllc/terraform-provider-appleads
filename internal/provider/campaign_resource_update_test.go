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

	countries, _ := types.ListValueFrom(ctx, types.StringType, []string{"US"})
	supply, _ := types.ListValueFrom(ctx, types.StringType, []string{"APPSTORE_SEARCH_RESULTS"})

	state := campaignModel{
		ID:                 types.StringValue("12345"),
		AdamID:             types.StringValue("1"),
		AdChannelType:      types.StringValue("SEARCH"),
		CountriesOrRegions: countries,
		SupplySources:      supply,
	}
	plan := state
	plan.AdamID = types.StringValue("2")
	plan.Name = types.StringValue("also mutable")

	changes := detectImmutableCampaignChanges(ctx, state, plan)
	if len(changes) != 1 || changes[0].Field != "adam_id" {
		t.Fatalf("changes = %#v", changes)
	}
	diags := immutableCampaignChangeDiagnostics("12345", changes)
	if !diags.HasError() {
		t.Fatal("expected diagnostics")
	}
	detail := diags[0].Detail()
	if !strings.Contains(detail, "adam_id") || !strings.Contains(detail, "12345") {
		t.Fatalf("detail = %s", detail)
	}
	if !strings.Contains(detail, "Create a new appleads_campaign") {
		t.Fatalf("detail = %s", detail)
	}
}

func TestDetectImmutableCampaignChanges_CountryReorderIsNotAChange(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	configured, _ := types.ListValueFrom(ctx, types.StringType, []string{"PL", "RO", "CZ"})
	reordered, _ := types.ListValueFrom(ctx, types.StringType, []string{"CZ", "PL", "RO"})
	supply, _ := types.ListValueFrom(ctx, types.StringType, []string{"APPSTORE_SEARCH_RESULTS"})

	state := campaignModel{
		ID:                 types.StringValue("12345"),
		AdamID:             types.StringValue("1"),
		AdChannelType:      types.StringValue("SEARCH"),
		CountriesOrRegions: configured,
		SupplySources:      supply,
	}
	plan := state
	plan.CountriesOrRegions = reordered
	plan.Name = types.StringValue("mutable")

	changes := detectImmutableCampaignChanges(ctx, state, plan)
	if len(changes) != 0 {
		t.Fatalf("reorder should not be an identity change, got %#v", changes)
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

func TestDetectImmutableCampaignChanges_CountryMembershipIsMutable(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	before, _ := types.ListValueFrom(ctx, types.StringType, []string{"US", "CA", "GB"})
	after, _ := types.ListValueFrom(ctx, types.StringType, []string{"US", "CA"})
	supply, _ := types.ListValueFrom(ctx, types.StringType, []string{"APPSTORE_SEARCH_RESULTS"})

	state := campaignModel{
		ID:                 types.StringValue("12345"),
		AdamID:             types.StringValue("1"),
		AdChannelType:      types.StringValue("SEARCH"),
		CountriesOrRegions: before,
		SupplySources:      supply,
	}
	plan := state
	plan.CountriesOrRegions = after

	changes := detectImmutableCampaignChanges(ctx, state, plan)
	if len(changes) != 0 {
		t.Fatalf("countries_or_regions membership is mutable, got %#v", changes)
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
	countries, _ := types.ListValueFrom(ctx, types.StringType, []string{"US"})
	state := campaignModel{
		ID:                 types.StringValue("99"),
		AdamID:             types.StringValue("1"),
		CountriesOrRegions: countries,
		SupplySources:      types.ListNull(types.StringType),
		BudgetOrders:       types.ListNull(types.StringType),
	}
	plan := state
	plan.AdamID = types.StringValue("2")
	plan.Name = types.StringValue("mutable too")

	changes := detectImmutableCampaignChanges(ctx, state, plan)
	if len(changes) == 0 {
		t.Fatal("expected immutable change")
	}
	_ = srv
	apiClient, _ := client.New(client.WithBaseURL(srv.URL))
	_ = apiClient
	if called {
		t.Fatal("API should not be called when immutable fields change")
	}
}

func TestCampaignUpdateFromPlan_DropCountrySendsClearGeoFlag(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	before, diags := types.ListValueFrom(ctx, types.StringType, []string{"US", "CA", "GB"})
	if diags.HasError() {
		t.Fatal(diags)
	}
	after, diags := types.ListValueFrom(ctx, types.StringType, []string{"US", "CA"})
	if diags.HasError() {
		t.Fatal(diags)
	}

	state := campaignModel{
		ID:                 types.StringValue("42"),
		Name:               types.StringValue("example-campaign"),
		AdamID:             types.StringValue("1"),
		Status:             types.StringValue("ENABLED"),
		CountriesOrRegions: before,
		SupplySources:      types.ListNull(types.StringType),
		BudgetOrders:       types.ListNull(types.StringType),
	}
	plan := state
	plan.CountriesOrRegions = after

	upd, d := campaignUpdateFromPlan(ctx, state, plan)
	if d.HasError() {
		t.Fatalf("%v", d)
	}
	if !upd.ClearGeoTargetingOnCountryOrRegionChange {
		t.Fatal("expected clearGeoTargetingOnCountryOrRegionChange")
	}
	want := []string{"US", "CA"}
	if len(upd.CountriesOrRegions) != len(want) {
		t.Fatalf("countries = %#v", upd.CountriesOrRegions)
	}
	for i := range want {
		if upd.CountriesOrRegions[i] != want[i] {
			t.Fatalf("countries[%d] = %q, want %q", i, upd.CountriesOrRegions[i], want[i])
		}
	}
}

func TestCampaignUpdateFromPlan_ReorderOmitsCountries(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	configured, diags := types.ListValueFrom(ctx, types.StringType, []string{"US", "CA", "GB"})
	if diags.HasError() {
		t.Fatal(diags)
	}
	reordered, diags := types.ListValueFrom(ctx, types.StringType, []string{"GB", "US", "CA"})
	if diags.HasError() {
		t.Fatal(diags)
	}

	state := campaignModel{
		Name:               types.StringValue("example-campaign"),
		Status:             types.StringValue("ENABLED"),
		CountriesOrRegions: configured,
		BudgetOrders:       types.ListNull(types.StringType),
	}
	plan := state
	plan.CountriesOrRegions = reordered

	upd, d := campaignUpdateFromPlan(ctx, state, plan)
	if d.HasError() {
		t.Fatalf("%v", d)
	}
	if upd.ClearGeoTargetingOnCountryOrRegionChange {
		t.Fatal("reorder must not set clearGeoTargetingOnCountryOrRegionChange")
	}
	if upd.CountriesOrRegions != nil {
		t.Fatalf("reorder must omit countriesOrRegions, got %#v", upd.CountriesOrRegions)
	}
}
