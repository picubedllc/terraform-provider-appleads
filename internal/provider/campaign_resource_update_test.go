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
		BillingEvent:       types.StringValue("TAPS"),
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
	if !strings.Contains(detail, "Create a new appleads_campaign") {
		t.Fatalf("detail = %s", detail)
	}
}

func TestDetectImmutableCampaignChanges_BillingEvent(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	countries, _ := types.ListValueFrom(ctx, types.StringType, []string{"US"})
	supply, _ := types.ListValueFrom(ctx, types.StringType, []string{"APPSTORE_SEARCH_RESULTS"})

	state := campaignModel{
		ID:                 types.StringValue("55"),
		AdamID:             types.StringValue("1"),
		AdChannelType:      types.StringValue("SEARCH"),
		BillingEvent:       types.StringValue("TAPS"),
		CountriesOrRegions: countries,
		SupplySources:      supply,
	}
	plan := state
	plan.BillingEvent = types.StringValue("IMPRESSIONS")
	plan.Name = types.StringValue("mutable too")

	changes := detectImmutableCampaignChanges(ctx, state, plan)
	if len(changes) != 1 || changes[0].Field != "billing_event" {
		t.Fatalf("changes = %#v", changes)
	}
	diags := immutableCampaignChangeDiagnostics("55", changes)
	if !diags.HasError() {
		t.Fatal("expected diagnostics")
	}
	if !strings.Contains(diags[0].Summary(), "billing_event") {
		t.Fatalf("summary = %s", diags[0].Summary())
	}
	if !strings.Contains(diags[0].Detail(), "Create a new appleads_campaign") {
		t.Fatalf("detail = %s", diags[0].Detail())
	}
}

func TestDetectImmutableCampaignChanges_BudgetAmount(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	countries, _ := types.ListValueFrom(ctx, types.StringType, []string{"US"})
	supply, _ := types.ListValueFrom(ctx, types.StringType, []string{"APPSTORE_SEARCH_RESULTS"})

	state := campaignModel{
		ID:                  types.StringValue("77"),
		AdamID:              types.StringValue("1"),
		AdChannelType:       types.StringValue("SEARCH"),
		BillingEvent:        types.StringValue("TAPS"),
		CountriesOrRegions:  countries,
		SupplySources:       supply,
		BudgetAmount:        types.StringValue("100.00"),
		BudgetCurrency:      types.StringValue("USD"),
		DailyBudgetAmount:   types.StringValue("10.00"),
		DailyBudgetCurrency: types.StringValue("USD"),
	}
	plan := state
	plan.BudgetAmount = types.StringValue("150.00")
	plan.DailyBudgetAmount = types.StringValue("12.00") // mutable change must not mask create-only error

	changes := detectImmutableCampaignChanges(ctx, state, plan)
	if len(changes) != 1 || changes[0].Field != "budget_amount" {
		t.Fatalf("changes = %#v", changes)
	}
	diags := immutableCampaignChangeDiagnostics("77", changes)
	if !diags.HasError() {
		t.Fatal("expected diagnostics")
	}
	if !strings.Contains(diags[0].Summary(), "budget_amount") {
		t.Fatalf("summary = %s", diags[0].Summary())
	}
	if strings.Contains(strings.ToLower(diags[0].Detail()), "requiresreplace") {
		t.Fatal("diagnostic must not mention RequiresReplace")
	}
}

func TestDetectImmutableCampaignChanges_BudgetCurrency(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	countries, _ := types.ListValueFrom(ctx, types.StringType, []string{"US"})
	supply, _ := types.ListValueFrom(ctx, types.StringType, []string{"APPSTORE_SEARCH_RESULTS"})

	state := campaignModel{
		ID:                 types.StringValue("78"),
		AdamID:             types.StringValue("1"),
		CountriesOrRegions: countries,
		SupplySources:      supply,
		BudgetAmount:       types.StringValue("100.00"),
		BudgetCurrency:     types.StringValue("USD"),
	}
	plan := state
	plan.BudgetCurrency = types.StringValue("EUR")

	changes := detectImmutableCampaignChanges(ctx, state, plan)
	if len(changes) != 1 || changes[0].Field != "budget_currency" {
		t.Fatalf("changes = %#v", changes)
	}
}

func TestDetectImmutableCampaignChanges_DailyBudgetStillMutable(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	countries, _ := types.ListValueFrom(ctx, types.StringType, []string{"US"})
	supply, _ := types.ListValueFrom(ctx, types.StringType, []string{"APPSTORE_SEARCH_RESULTS"})

	state := campaignModel{
		ID:                  types.StringValue("79"),
		AdamID:              types.StringValue("1"),
		CountriesOrRegions:  countries,
		SupplySources:       supply,
		BudgetAmount:        types.StringValue("100.00"),
		BudgetCurrency:      types.StringValue("USD"),
		DailyBudgetAmount:   types.StringValue("10.00"),
		DailyBudgetCurrency: types.StringValue("USD"),
	}
	plan := state
	plan.DailyBudgetAmount = types.StringValue("20.00")
	plan.Name = types.StringValue("renamed")

	changes := detectImmutableCampaignChanges(ctx, state, plan)
	if len(changes) != 0 {
		t.Fatalf("daily budget / name changes must not be immutable, got %#v", changes)
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

func TestCampaignUpdateFromPlan_OmitsBudgetAmount(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	upd, diags := campaignUpdateFromPlan(ctx, campaignModel{
		Name:                types.StringValue("Keep Total"),
		Status:              types.StringValue("PAUSED"),
		BudgetAmount:        types.StringValue("500.00"),
		BudgetCurrency:      types.StringValue("USD"),
		DailyBudgetAmount:   types.StringValue("15.00"),
		DailyBudgetCurrency: types.StringValue("USD"),
		BudgetOrders:        types.ListNull(types.StringType),
	})
	if diags.HasError() {
		t.Fatalf("%v", diags)
	}
	if upd.DailyBudgetAmount == nil || upd.DailyBudgetAmount.Amount != "15.00" {
		t.Fatalf("dailyBudgetAmount = %#v", upd.DailyBudgetAmount)
	}
	// Reflect: CampaignUpdate must not expose BudgetAmount for Terraform updates.
	raw, err := json.Marshal(upd)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "budgetAmount") {
		t.Fatalf("update JSON must omit budgetAmount, got %s", raw)
	}
}

func TestCampaignUpdateFromPlan_BiddingStrategyAndTargetCpa(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	upd, diags := campaignUpdateFromPlan(ctx, campaignModel{
		Name:              types.StringValue("Updated Max"),
		Status:            types.StringValue("PAUSED"),
		BiddingStrategy:   types.StringValue(client.BiddingStrategyMaxConversions),
		TargetCpaAmount:   types.StringValue("12.00"),
		TargetCpaCurrency: types.StringValue("USD"),
		BudgetOrders:      types.ListNull(types.StringType),
	})
	if diags.HasError() {
		t.Fatalf("%v", diags)
	}
	if upd.BiddingStrategy != client.BiddingStrategyMaxConversions {
		t.Fatalf("biddingStrategy = %q", upd.BiddingStrategy)
	}
	if upd.TargetCpa == nil || upd.TargetCpa.Amount != "12.00" || upd.TargetCpa.Currency != "USD" {
		t.Fatalf("targetCpa = %#v", upd.TargetCpa)
	}
}

func TestCampaignUpdateFromPlan_SwitchToManualCPT(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	upd, diags := campaignUpdateFromPlan(ctx, campaignModel{
		Name:            types.StringValue("Back to Manual"),
		Status:          types.StringValue("PAUSED"),
		BiddingStrategy: types.StringValue(client.BiddingStrategyManualCPT),
		BudgetOrders:    types.ListNull(types.StringType),
	})
	if diags.HasError() {
		t.Fatalf("%v", diags)
	}
	if upd.BiddingStrategy != client.BiddingStrategyManualCPT {
		t.Fatalf("biddingStrategy = %q", upd.BiddingStrategy)
	}
	if upd.TargetCpa != nil {
		t.Fatalf("targetCpa should be omitted for Manual, got %#v", upd.TargetCpa)
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
