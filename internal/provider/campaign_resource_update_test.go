// Copyright (c) 2026 Pi Cubed LLC
// SPDX-License-Identifier: MIT

package provider

import (
	"context"
	"encoding/json"
	"net/http"
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
	if upd.DailyBudget == nil || upd.DailyBudget.Value == nil || upd.DailyBudget.Value.Amount != "15.00" {
		t.Fatalf("dailyBudget = %#v", upd.DailyBudget)
	}
	raw, err := json.Marshal(upd)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "budgetAmount") || strings.Contains(string(raw), "500") {
		t.Fatalf("update JSON must omit the lifetime budget, got %s", raw)
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
	bs := upd.BidStrategy
	if bs == nil || bs.BidStrategyType != client.BiddingStrategyMaxConversions {
		t.Fatalf("bidStrategy = %#v", bs)
	}
	if bs.Bid == nil || bs.Bid.Amount != "12.00" || bs.Bid.Currency != "USD" {
		t.Fatalf("bidStrategy.bid = %#v", bs.Bid)
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
	if upd.BidStrategy == nil || upd.BidStrategy.BidStrategyType != client.BiddingStrategyManualCPT {
		t.Fatalf("bidStrategy = %#v", upd.BidStrategy)
	}
	if upd.BidStrategy.Bid != nil {
		t.Fatalf("bid should be omitted for Manual, got %#v", upd.BidStrategy.Bid)
	}
}

func TestCampaignUpdateFromPlan_SharedBudgetsAndEndTime(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	orders, _ := types.ListValueFrom(ctx, types.StringType, []string{"31", "32"})
	upd, diags := campaignUpdateFromPlan(ctx, campaignModel{
		Name:         types.StringValue("Shared"),
		BudgetOrders: orders,
		EndTime:      types.StringValue("2026-12-31T00:00:00.000"),
	})
	if diags.HasError() {
		t.Fatalf("%v", diags)
	}
	if len(upd.SharedBudgets) != 2 || upd.SharedBudgets[0].BudgetID != 31 || upd.SharedBudgets[1].BudgetID != 32 {
		t.Fatalf("sharedBudgets = %#v", upd.SharedBudgets)
	}
	if upd.EndTime != "2026-12-31T00:00:00.000" {
		t.Fatalf("endTime = %q", upd.EndTime)
	}
}

func TestCampaignUpdate_MutableSucceeds(t *testing.T) {
	t.Parallel()

	var sawBody map[string]any
	apiClient := newCampaignTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/campaigns/9" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&sawBody); err != nil {
			t.Error(err)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"result": map[string]any{
				"id":                 9,
				"name":               sawBody["name"],
				"status":             sawBody["status"],
				"promotedObjectId":   "1",
				"promotedObjectType": "APPSTORE_APP",
				"targeting": map[string]any{
					"countryOrRegion": map[string]any{"include": []string{"US"}},
				},
				"dailyBudget":      sawBody["dailyBudget"],
				"modificationTime": "2026-04-01T00:00:00Z",
			},
		})
	})

	ctx := context.Background()
	upd, diags := campaignUpdateFromPlan(ctx, campaignModel{
		Name:                types.StringValue("New"),
		Status:              types.StringValue("PAUSED"),
		DailyBudgetAmount:   types.StringValue("10.00"),
		DailyBudgetCurrency: types.StringValue("USD"),
		BudgetOrders:        types.ListNull(types.StringType),
	})
	if diags.HasError() {
		t.Fatalf("%v", diags)
	}
	out, err := apiClient.UpdateCampaign(ctx, 9, upd)
	if err != nil {
		t.Fatal(err)
	}
	// v1 update body is bare (no "campaign" envelope).
	if _, wrapped := sawBody["campaign"]; wrapped {
		t.Fatalf("update body must not be wrapped: %#v", sawBody)
	}
	if sawBody["name"] != "New" || out.Name != "New" {
		t.Fatalf("saw=%#v out=%#v", sawBody, out)
	}
	state, diags := campaignModelFromAPI(ctx, out)
	if diags.HasError() {
		t.Fatalf("%v", diags)
	}
	if state.DailyBudgetAmount.ValueString() != "10.00" {
		t.Fatalf("daily = %q", state.DailyBudgetAmount.ValueString())
	}
}

func TestCampaignUpdate_ImmutableBlocksWithoutAPICall(t *testing.T) {
	t.Parallel()
	called := false
	_ = newCampaignTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

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
	diags := immutableCampaignChangeDiagnostics(state.ID.ValueString(), changes)
	if !diags.HasError() || !strings.Contains(diags[0].Summary(), "countries_or_regions") {
		t.Fatalf("diags = %v", diags)
	}
	// Update returns on immutable changes before any client call.
	if called {
		t.Fatal("API should not be called when immutable fields change")
	}
}
