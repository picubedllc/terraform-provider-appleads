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

func TestCampaignCreateFromPlan_AndStateFromResponse(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body client.CampaignCreate
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.DailyBudgetAmount == nil || body.DailyBudgetAmount.Amount != "25.50" {
			t.Fatalf("daily budget = %#v", body.DailyBudgetAmount)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"id":                 55,
				"name":               body.Name,
				"adamId":             body.AdamID,
				"status":             "ENABLED",
				"servingStatus":      "RUNNING",
				"displayStatus":      "RUNNING",
				"countriesOrRegions": body.CountriesOrRegions,
				"supplySources":      []string{"APPSTORE_SEARCH_RESULTS"},
				"adChannelType":      "SEARCH",
				"dailyBudgetAmount":  map[string]string{"amount": "25.50", "currency": "USD"},
				"modificationTime":   "2026-02-01T00:00:00Z",
			},
		})
	}))
	t.Cleanup(srv.Close)

	apiClient, err := client.New(client.WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	countries, diags := types.ListValueFrom(ctx, types.StringType, []string{"US", "CA"})
	if diags.HasError() {
		t.Fatal(diags)
	}
	plan := campaignModel{
		Name:                types.StringValue("Launch"),
		AdamID:              types.StringValue("9"),
		Status:              types.StringValue("ENABLED"),
		CountriesOrRegions:  countries,
		DailyBudgetAmount:   types.StringValue("25.50"),
		DailyBudgetCurrency: types.StringValue("USD"),
		SupplySources:       types.ListNull(types.StringType),
		BudgetOrders:        types.ListNull(types.StringType),
	}

	in, diags := campaignCreateFromPlan(ctx, plan)
	if diags.HasError() {
		t.Fatalf("%v", diags)
	}
	created, err := apiClient.CreateCampaign(ctx, in)
	if err != nil {
		t.Fatal(err)
	}
	state, diags := campaignModelFromClient(ctx, created)
	if diags.HasError() {
		t.Fatalf("%v", diags)
	}
	if state.ID.ValueString() != "55" || state.ServingStatus.ValueString() != "RUNNING" {
		t.Fatalf("state = %#v", state)
	}
	if state.DailyBudgetAmount.ValueString() != "25.50" {
		t.Fatalf("daily = %q", state.DailyBudgetAmount.ValueString())
	}
}

func TestCampaignCreate_WithBudgetAmount(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body client.CampaignCreate
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.BudgetAmount == nil || body.BudgetAmount.Amount != "500.00" || body.BudgetAmount.Currency != "USD" {
			t.Fatalf("budgetAmount = %#v", body.BudgetAmount)
		}
		if body.DailyBudgetAmount == nil || body.DailyBudgetAmount.Amount != "25.00" {
			t.Fatalf("dailyBudgetAmount = %#v", body.DailyBudgetAmount)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"id":                 56,
				"name":               body.Name,
				"adamId":             body.AdamID,
				"status":             "PAUSED",
				"countriesOrRegions": body.CountriesOrRegions,
				"budgetAmount":       map[string]string{"amount": "500.00", "currency": "USD"},
				"dailyBudgetAmount":  map[string]string{"amount": "25.00", "currency": "USD"},
				"modificationTime":   "2026-02-01T00:00:00Z",
			},
		})
	}))
	t.Cleanup(srv.Close)

	apiClient, err := client.New(client.WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	countries, diags := types.ListValueFrom(ctx, types.StringType, []string{"US"})
	if diags.HasError() {
		t.Fatal(diags)
	}
	plan := campaignModel{
		Name:                types.StringValue("Lifetime Cap"),
		AdamID:              types.StringValue("9"),
		Status:              types.StringValue("PAUSED"),
		CountriesOrRegions:  countries,
		BudgetAmount:        types.StringValue("500.00"),
		BudgetCurrency:      types.StringValue("USD"),
		DailyBudgetAmount:   types.StringValue("25.00"),
		DailyBudgetCurrency: types.StringValue("USD"),
		SupplySources:       types.ListNull(types.StringType),
		BudgetOrders:        types.ListNull(types.StringType),
	}

	in, diags := campaignCreateFromPlan(ctx, plan)
	if diags.HasError() {
		t.Fatalf("%v", diags)
	}
	created, err := apiClient.CreateCampaign(ctx, in)
	if err != nil {
		t.Fatal(err)
	}
	state, diags := campaignModelFromClient(ctx, created)
	if diags.HasError() {
		t.Fatalf("%v", diags)
	}
	state, diags = overlayCampaignReported(ctx, plan, state)
	if diags.HasError() {
		t.Fatalf("%v", diags)
	}
	if state.BudgetAmount.ValueString() != "500.00" || state.BudgetCurrency.ValueString() != "USD" {
		t.Fatalf("budget = %s %s", state.BudgetAmount.ValueString(), state.BudgetCurrency.ValueString())
	}
}

func TestCampaignCreate_KeepsConfiguredCountryOrderWhenAppleReorders(t *testing.T) {
	t.Parallel()

	configured := []string{"PL", "RO", "CZ", "HU", "HR", "SK", "SI", "EE", "LV"}
	sortedByApple := []string{"CZ", "EE", "HR", "HU", "LV", "PL", "RO", "SI", "SK"}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body client.CampaignCreate
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"id":                 55,
				"name":               body.Name,
				"adamId":             body.AdamID,
				"status":             "ENABLED",
				"servingStatus":      "RUNNING",
				"displayStatus":      "RUNNING",
				"countriesOrRegions": sortedByApple,
				"supplySources":      []string{"APPSTORE_SEARCH_RESULTS"},
				"adChannelType":      "SEARCH",
				"dailyBudgetAmount":  map[string]string{"amount": "5", "currency": "USD"},
				"modificationTime":   "2026-02-01T00:00:00Z",
			},
		})
	}))
	t.Cleanup(srv.Close)

	apiClient, err := client.New(client.WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	countries, diags := types.ListValueFrom(ctx, types.StringType, configured)
	if diags.HasError() {
		t.Fatal(diags)
	}
	plan := campaignModel{
		Name:                types.StringValue("Multi country"),
		AdamID:              types.StringValue("9"),
		Status:              types.StringValue("PAUSED"),
		CountriesOrRegions:  countries,
		DailyBudgetAmount:   types.StringValue("5.00"),
		DailyBudgetCurrency: types.StringValue("USD"),
		SupplySources:       types.ListNull(types.StringType),
		BudgetOrders:        types.ListNull(types.StringType),
	}

	in, diags := campaignCreateFromPlan(ctx, plan)
	if diags.HasError() {
		t.Fatalf("%v", diags)
	}
	created, err := apiClient.CreateCampaign(ctx, in)
	if err != nil {
		t.Fatal(err)
	}
	state, diags := campaignModelFromClient(ctx, created)
	if diags.HasError() {
		t.Fatalf("%v", diags)
	}
	state, diags = overlayCampaignReported(ctx, plan, state)
	if diags.HasError() {
		t.Fatalf("%v", diags)
	}

	got, d := stringList(ctx, state.CountriesOrRegions)
	if d.HasError() {
		t.Fatal(d)
	}
	if !stringSetEqual(got, configured) {
		t.Fatalf("country set = %#v", got)
	}
	for i := range configured {
		if got[i] != configured[i] {
			t.Fatalf("countries[%d] = %q, want configured order %#v, got %#v", i, got[i], configured, got)
		}
	}
	if state.DailyBudgetAmount.ValueString() != "5.00" {
		t.Fatalf("daily = %q", state.DailyBudgetAmount.ValueString())
	}
}

func TestCampaignCreateFromPlan_SearchDefaultsMatchLivePayload(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	countries, diags := types.ListValueFrom(ctx, types.StringType, []string{"US"})
	if diags.HasError() {
		t.Fatal(diags)
	}
	in, diags := campaignCreateFromPlan(ctx, campaignModel{
		Name:                types.StringValue("Paused Search Campaign"),
		AdamID:              types.StringValue("1234567890"),
		Status:              types.StringValue("PAUSED"),
		CountriesOrRegions:  countries,
		DailyBudgetAmount:   types.StringValue("5.00"),
		DailyBudgetCurrency: types.StringValue("USD"),
		SupplySources:       types.ListNull(types.StringType),
		BudgetOrders:        types.ListNull(types.StringType),
		AdChannelType:       types.StringNull(),
		BillingEvent:        types.StringNull(),
		StartTime:           types.StringNull(),
	})
	if diags.HasError() {
		t.Fatalf("%v", diags)
	}
	if in.AdChannelType != "SEARCH" {
		t.Fatalf("adChannelType = %q", in.AdChannelType)
	}
	if len(in.SupplySources) != 1 || in.SupplySources[0] != "APPSTORE_SEARCH_RESULTS" {
		t.Fatalf("supplySources = %#v", in.SupplySources)
	}
	if in.BillingEvent != "TAPS" {
		t.Fatalf("billingEvent = %q", in.BillingEvent)
	}
	if in.BiddingStrategy != "MANUAL_CPT" {
		t.Fatalf("biddingStrategy = %q", in.BiddingStrategy)
	}
	if in.StartTime != "" {
		t.Fatalf("startTime should be omitted when unset, got %q", in.StartTime)
	}
	if in.DailyBudgetAmount == nil || in.DailyBudgetAmount.Amount != "5.00" || in.DailyBudgetAmount.Currency != "USD" {
		t.Fatalf("dailyBudgetAmount = %#v", in.DailyBudgetAmount)
	}
}

func TestCampaignCreateFromPlan_ExplicitDisplayChannelAndSupply(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	countries, diags := types.ListValueFrom(ctx, types.StringType, []string{"US"})
	if diags.HasError() {
		t.Fatal(diags)
	}
	supply, diags := types.ListValueFrom(ctx, types.StringType, []string{"APPSTORE_SEARCH_TAB"})
	if diags.HasError() {
		t.Fatal(diags)
	}
	in, diags := campaignCreateFromPlan(ctx, campaignModel{
		Name:                types.StringValue("Display Campaign"),
		AdamID:              types.StringValue("1234567890"),
		Status:              types.StringValue("PAUSED"),
		CountriesOrRegions:  countries,
		DailyBudgetAmount:   types.StringValue("5.00"),
		DailyBudgetCurrency: types.StringValue("USD"),
		AdChannelType:       types.StringValue("DISPLAY"),
		SupplySources:       supply,
		BillingEvent:        types.StringValue("TAPS"),
		BudgetOrders:        types.ListNull(types.StringType),
		StartTime:           types.StringNull(),
	})
	if diags.HasError() {
		t.Fatalf("%v", diags)
	}
	if in.AdChannelType != "DISPLAY" {
		t.Fatalf("adChannelType = %q, want DISPLAY (must not force SEARCH)", in.AdChannelType)
	}
	if len(in.SupplySources) != 1 || in.SupplySources[0] != "APPSTORE_SEARCH_TAB" {
		t.Fatalf("supplySources = %#v, must not force APPSTORE_SEARCH_RESULTS", in.SupplySources)
	}
	if in.BillingEvent != "TAPS" {
		t.Fatalf("billingEvent = %q", in.BillingEvent)
	}
}

func TestCampaignCreateFromPlan_StartTimePassThrough(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	countries, diags := types.ListValueFrom(ctx, types.StringType, []string{"US"})
	if diags.HasError() {
		t.Fatal(diags)
	}
	in, diags := campaignCreateFromPlan(ctx, campaignModel{
		Name:                types.StringValue("Scheduled"),
		AdamID:              types.StringValue("9"),
		CountriesOrRegions:  countries,
		DailyBudgetAmount:   types.StringValue("1.00"),
		DailyBudgetCurrency: types.StringValue("USD"),
		SupplySources:       types.ListNull(types.StringType),
		BudgetOrders:        types.ListNull(types.StringType),
		StartTime:           types.StringValue("2026-06-01T12:00:00.000"),
	})
	if diags.HasError() {
		t.Fatalf("%v", diags)
	}
	if in.StartTime != "2026-06-01T12:00:00.000" {
		t.Fatalf("startTime = %q", in.StartTime)
	}
}

func TestCampaignCreateFromPlan_ExplicitBiddingStrategy(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	countries, diags := types.ListValueFrom(ctx, types.StringType, []string{"US"})
	if diags.HasError() {
		t.Fatal(diags)
	}
	supply, diags := types.ListValueFrom(ctx, types.StringType, []string{client.SupplySourceSearchResults})
	if diags.HasError() {
		t.Fatal(diags)
	}
	in, diags := campaignCreateFromPlan(ctx, campaignModel{
		Name:                types.StringValue("Max Conv"),
		AdamID:              types.StringValue("1234567890"),
		CountriesOrRegions:  countries,
		SupplySources:       supply,
		AdChannelType:       types.StringValue(client.AdChannelTypeSearch),
		BillingEvent:        types.StringValue(client.BillingEventTaps),
		BiddingStrategy:     types.StringValue(client.BiddingStrategyMaxConversions),
		TargetCpaAmount:     types.StringValue("10.00"),
		TargetCpaCurrency:   types.StringValue("USD"),
		DailyBudgetAmount:   types.StringValue("5.00"),
		DailyBudgetCurrency: types.StringValue("USD"),
		BudgetOrders:        types.ListNull(types.StringType),
	})
	if diags.HasError() {
		t.Fatalf("%v", diags)
	}
	if in.BiddingStrategy != client.BiddingStrategyMaxConversions {
		t.Fatalf("biddingStrategy = %q", in.BiddingStrategy)
	}
	if in.TargetCpa == nil || in.TargetCpa.Amount != "10.00" || in.TargetCpa.Currency != "USD" {
		t.Fatalf("targetCpa = %#v", in.TargetCpa)
	}
}

func TestCampaignModelFromClient_BiddingStrategy(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	state, diags := campaignModelFromClient(ctx, &client.Campaign{
		ID:                 77,
		Name:               "Read",
		AdamID:             9,
		Status:             "PAUSED",
		AdChannelType:      client.AdChannelTypeSearch,
		BillingEvent:       client.BillingEventTaps,
		BiddingStrategy:    client.BiddingStrategyMaxConversions,
		TargetCpa:          &client.Money{Amount: "10.00", Currency: "USD"},
		CountriesOrRegions: []string{"US"},
		SupplySources:      []string{client.SupplySourceSearchResults},
	})
	if diags.HasError() {
		t.Fatalf("%v", diags)
	}
	if state.BiddingStrategy.ValueString() != client.BiddingStrategyMaxConversions {
		t.Fatalf("bidding_strategy = %q", state.BiddingStrategy.ValueString())
	}
	if state.TargetCpaAmount.ValueString() != "10.00" || state.TargetCpaCurrency.ValueString() != "USD" {
		t.Fatalf("target_cpa = %q %q", state.TargetCpaAmount.ValueString(), state.TargetCpaCurrency.ValueString())
	}
}

func TestCampaignModelFromClient_PaymentModelAndBillingStart(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	state, diags := campaignModelFromClient(ctx, &client.Campaign{
		ID:                 77,
		Name:               "Read",
		AdamID:             9,
		Status:             "PAUSED",
		AdChannelType:      "SEARCH",
		BillingEvent:       "TAPS",
		PaymentModel:       "PAYG",
		StartTime:          "2026-06-01T12:00:00.000",
		CountriesOrRegions: []string{"US"},
		SupplySources:      []string{"APPSTORE_SEARCH_RESULTS"},
		ServingStatus:      "NOT_RUNNING",
		DisplayStatus:      "PAUSED",
		ModificationTime:   "2026-06-02T00:00:00Z",
	})
	if diags.HasError() {
		t.Fatalf("%v", diags)
	}
	if state.PaymentModel.ValueString() != "PAYG" {
		t.Fatalf("payment_model = %q", state.PaymentModel.ValueString())
	}
	if state.BillingEvent.ValueString() != "TAPS" {
		t.Fatalf("billing_event = %q", state.BillingEvent.ValueString())
	}
	if state.StartTime.ValueString() != "2026-06-01T12:00:00.000" {
		t.Fatalf("start_time = %q", state.StartTime.ValueString())
	}
}

func TestOverlayCampaignStartTime_KeepsConfiguredPrecision(t *testing.T) {
	t.Parallel()

	configured := campaignModel{StartTime: types.StringValue("2026-06-01T12:00:00.000")}
	reported := campaignModel{StartTime: types.StringValue("2026-06-01T12:00:00.000Z")}
	out := overlayCampaignStartTime(configured, reported)
	if out.StartTime.ValueString() != "2026-06-01T12:00:00.000" {
		t.Fatalf("start_time = %q", out.StartTime.ValueString())
	}

	// When plan omitted start_time, keep Apple's assigned value.
	configured = campaignModel{StartTime: types.StringNull()}
	out = overlayCampaignStartTime(configured, reported)
	if out.StartTime.ValueString() != "2026-06-01T12:00:00.000Z" {
		t.Fatalf("start_time = %q", out.StartTime.ValueString())
	}
}

func TestOverlayCampaignReported_IncludesStartTime(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	countries, diags := types.ListValueFrom(ctx, types.StringType, []string{"US"})
	if diags.HasError() {
		t.Fatal(diags)
	}
	configured := campaignModel{
		CountriesOrRegions:  countries,
		DailyBudgetAmount:   types.StringValue("5.00"),
		DailyBudgetCurrency: types.StringValue("USD"),
		StartTime:           types.StringValue("2026-06-01T12:00:00.000"),
		SupplySources:       types.ListNull(types.StringType),
	}
	reportedCountries, diags := types.ListValueFrom(ctx, types.StringType, []string{"US"})
	if diags.HasError() {
		t.Fatal(diags)
	}
	reported := campaignModel{
		CountriesOrRegions:  reportedCountries,
		DailyBudgetAmount:   types.StringValue("5"),
		DailyBudgetCurrency: types.StringValue("USD"),
		StartTime:           types.StringValue("2026-06-01T12:00:00.000Z"),
		SupplySources:       types.ListNull(types.StringType),
	}
	out, diags := overlayCampaignReported(ctx, configured, reported)
	if diags.HasError() {
		t.Fatal(diags)
	}
	if out.DailyBudgetAmount.ValueString() != "5.00" {
		t.Fatalf("daily = %q", out.DailyBudgetAmount.ValueString())
	}
	if out.StartTime.ValueString() != "2026-06-01T12:00:00.000" {
		t.Fatalf("start_time = %q", out.StartTime.ValueString())
	}
}

func TestOverlayCampaignMoney_KeepsConfiguredScale(t *testing.T) {
	t.Parallel()

	configured := campaignModel{
		DailyBudgetAmount:   types.StringValue("5.00"),
		DailyBudgetCurrency: types.StringValue("USD"),
		TargetCpaAmount:     types.StringValue("10.00"),
		TargetCpaCurrency:   types.StringValue("USD"),
	}
	reported := campaignModel{
		DailyBudgetAmount:   types.StringValue("5"),
		DailyBudgetCurrency: types.StringValue("USD"),
		TargetCpaAmount:     types.StringValue("10"),
		TargetCpaCurrency:   types.StringValue("USD"),
	}
	out := overlayCampaignMoney(configured, reported)
	if out.DailyBudgetAmount.ValueString() != "5.00" {
		t.Fatalf("amount = %q", out.DailyBudgetAmount.ValueString())
	}
	if out.TargetCpaAmount.ValueString() != "10.00" {
		t.Fatalf("target_cpa_amount = %q", out.TargetCpaAmount.ValueString())
	}

	reported.DailyBudgetAmount = types.StringValue("6")
	reported.TargetCpaAmount = types.StringValue("12")
	out = overlayCampaignMoney(configured, reported)
	if out.DailyBudgetAmount.ValueString() != "6" {
		t.Fatalf("changed amount should keep API value, got %q", out.DailyBudgetAmount.ValueString())
	}
	if out.TargetCpaAmount.ValueString() != "12" {
		t.Fatalf("changed target_cpa should keep API value, got %q", out.TargetCpaAmount.ValueString())
	}
}

func TestOverlayCampaignLists_KeepsConfiguredCountryOrder(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	configured, diags := types.ListValueFrom(ctx, types.StringType, []string{"DE", "FR", "NL", "ES", "IT", "AT", "BE", "DK", "FI", "NO", "SE", "CH"})
	if diags.HasError() {
		t.Fatal(diags)
	}
	reported, diags := types.ListValueFrom(ctx, types.StringType, []string{"AT", "BE", "CH", "DE", "DK", "ES", "FI", "FR", "IT", "NL", "NO", "SE"})
	if diags.HasError() {
		t.Fatal(diags)
	}
	out, diags := overlayCampaignLists(ctx, campaignModel{CountriesOrRegions: configured}, campaignModel{CountriesOrRegions: reported})
	if diags.HasError() {
		t.Fatal(diags)
	}
	got, d := stringList(ctx, out.CountriesOrRegions)
	if d.HasError() {
		t.Fatal(d)
	}
	want := []string{"DE", "FR", "NL", "ES", "IT", "AT", "BE", "DK", "FI", "NO", "SE", "CH"}
	if len(got) != len(want) {
		t.Fatalf("got %#v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("countries[%d] = %q, want %q (%#v)", i, got[i], want[i], got)
		}
	}

	changed, diags := types.ListValueFrom(ctx, types.StringType, []string{"US"})
	if diags.HasError() {
		t.Fatal(diags)
	}
	out, diags = overlayCampaignLists(ctx, campaignModel{CountriesOrRegions: configured}, campaignModel{CountriesOrRegions: changed})
	if diags.HasError() {
		t.Fatal(diags)
	}
	got, d = stringList(ctx, out.CountriesOrRegions)
	if d.HasError() {
		t.Fatal(d)
	}
	if len(got) != 1 || got[0] != "US" {
		t.Fatalf("membership change should keep API value, got %#v", got)
	}
}

func TestCampaignCreate_APIValidationDiagnostic(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"errors": []map[string]string{{
					"messageCode": "INVALID_ATTRIBUTE_VALUE",
					"message":     "dailyBudgetAmount must be positive",
					"field":       "dailyBudgetAmount",
				}},
			},
		})
	}))
	t.Cleanup(srv.Close)

	apiClient, err := client.New(client.WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	_, err = apiClient.CreateCampaign(context.Background(), &client.CampaignCreate{
		Name:               "x",
		AdamID:             1,
		CountriesOrRegions: []string{"US"},
	})
	if err == nil {
		t.Fatal("expected error")
	}
	diags := apiErrorDiagnostic("Unable to create Apple Ads campaign", err)
	if !diags.HasError() {
		t.Fatal("expected diagnostic")
	}
	if !strings.Contains(diags[0].Detail(), "dailyBudgetAmount") {
		t.Fatalf("detail = %s", diags[0].Detail())
	}
}

func TestCampaignCreateFromPlan_RejectsNonPositiveBudget(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	countries, diags := types.ListValueFrom(ctx, types.StringType, []string{"US"})
	if diags.HasError() {
		t.Fatal(diags)
	}
	_, diags = campaignCreateFromPlan(ctx, campaignModel{
		Name:               types.StringValue("x"),
		AdamID:             types.StringValue("1"),
		CountriesOrRegions: countries,
		BudgetAmount:       types.StringValue("0"),
		BudgetCurrency:     types.StringValue("USD"),
		SupplySources:      types.ListNull(types.StringType),
		BudgetOrders:       types.ListNull(types.StringType),
	})
	if !diags.HasError() {
		t.Fatal("expected error for zero budget")
	}
}
