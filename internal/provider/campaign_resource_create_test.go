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
	clientv1 "github.com/picubedllc/terraform-provider-appleads/internal/client/v1"
)

func newCampaignTestClient(t *testing.T, h http.HandlerFunc) *clientv1.Client {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	c, err := clientv1.New(clientv1.WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func campaignTestList(t *testing.T, vals ...string) types.List {
	t.Helper()
	l, diags := types.ListValueFrom(context.Background(), types.StringType, vals)
	if diags.HasError() {
		t.Fatal(diags)
	}
	return l
}

func TestCampaignCreateFromPlan_AndStateFromResponse(t *testing.T) {
	t.Parallel()

	apiClient := newCampaignTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/campaigns" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		var body clientv1.CampaignCreate
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		if body.DailyBudget == nil || body.DailyBudget.Value == nil || body.DailyBudget.Value.Amount != "25.50" {
			t.Errorf("daily budget = %#v", body.DailyBudget)
		}
		if body.PromotedObjectType != "APPSTORE_APP" || body.PromotedObjectID != "9" {
			t.Errorf("promoted object = %s %s", body.PromotedObjectType, body.PromotedObjectID)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"result": map[string]any{
				"id":                 55,
				"name":               body.Name,
				"promotedObjectId":   body.PromotedObjectID,
				"promotedObjectType": body.PromotedObjectType,
				"status":             "ENABLED",
				"systemStatus":       "RUNNING",
				"displayStatus":      "RUNNING",
				"adChannelType":      "SEARCH",
				"targeting": map[string]any{
					"countryOrRegion": map[string]any{"include": body.Targeting.CountryOrRegion.Include},
					"supplyPlacement": map[string]any{"include": []string{"APPSTORE_SEARCH_RESULTS"}},
				},
				"dailyBudget":      map[string]any{"value": map[string]string{"amount": "25.50", "currency": "USD"}},
				"modificationTime": "2026-02-01T00:00:00Z",
			},
		})
	})

	ctx := context.Background()
	plan := campaignModel{
		Name:                types.StringValue("Launch"),
		AdamID:              types.StringValue("9"),
		Status:              types.StringValue("ENABLED"),
		CountriesOrRegions:  campaignTestList(t, "US", "CA"),
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
	state, diags := campaignModelFromAPI(ctx, created)
	if diags.HasError() {
		t.Fatalf("%v", diags)
	}
	if state.ID.ValueString() != "55" || state.ServingStatus.ValueString() != "RUNNING" {
		t.Fatalf("state = %#v", state)
	}
	if state.AdamID.ValueString() != "9" {
		t.Fatalf("adam_id = %q", state.AdamID.ValueString())
	}
	if state.DailyBudgetAmount.ValueString() != "25.50" {
		t.Fatalf("daily = %q", state.DailyBudgetAmount.ValueString())
	}
	got, d := stringList(ctx, state.CountriesOrRegions)
	if d.HasError() || !stringSetEqual(got, []string{"US", "CA"}) {
		t.Fatalf("countries = %#v (%v)", got, d)
	}
}

func TestCampaignCreate_WithBudgetAmountWarnsAndPreservesState(t *testing.T) {
	t.Parallel()

	apiClient := newCampaignTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		raw := map[string]any{}
		if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
			t.Error(err)
		}
		for _, k := range []string{"budgetAmount", "budget", "lifetimeBudget"} {
			if _, ok := raw[k]; ok {
				t.Errorf("lifetime budget must not be sent, found %q in %#v", k, raw)
			}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"result": map[string]any{
				"id":                 56,
				"name":               raw["name"],
				"promotedObjectId":   raw["promotedObjectId"],
				"promotedObjectType": raw["promotedObjectType"],
				"status":             "PAUSED",
				"targeting": map[string]any{
					"countryOrRegion": map[string]any{"include": []string{"US"}},
				},
				"dailyBudget":      map[string]any{"value": map[string]string{"amount": "25.00", "currency": "USD"}},
				"modificationTime": "2026-02-01T00:00:00Z",
			},
		})
	})

	ctx := context.Background()
	plan := campaignModel{
		Name:                types.StringValue("Lifetime Cap"),
		AdamID:              types.StringValue("9"),
		Status:              types.StringValue("PAUSED"),
		CountriesOrRegions:  campaignTestList(t, "US"),
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
	if len(diags) != 1 || !strings.Contains(diags[0].Summary(), "budget_amount") {
		t.Fatalf("expected a single budget_amount warning, got %v", diags)
	}
	created, err := apiClient.CreateCampaign(ctx, in)
	if err != nil {
		t.Fatal(err)
	}
	state, diags := campaignModelFromAPI(ctx, created)
	if diags.HasError() {
		t.Fatalf("%v", diags)
	}
	if !state.BudgetAmount.IsNull() {
		t.Fatalf("API-derived budget_amount should be null, got %q", state.BudgetAmount.ValueString())
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

	apiClient := newCampaignTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		var body clientv1.CampaignCreate
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"result": map[string]any{
				"id":                 55,
				"name":               body.Name,
				"promotedObjectId":   body.PromotedObjectID,
				"promotedObjectType": body.PromotedObjectType,
				"status":             "ENABLED",
				"systemStatus":       "RUNNING",
				"displayStatus":      "RUNNING",
				"adChannelType":      "SEARCH",
				"targeting": map[string]any{
					"countryOrRegion": map[string]any{"include": sortedByApple},
					"supplyPlacement": map[string]any{"include": []string{"APPSTORE_SEARCH_RESULTS"}},
				},
				"dailyBudget":      map[string]any{"value": map[string]string{"amount": "5", "currency": "USD"}},
				"modificationTime": "2026-02-01T00:00:00Z",
			},
		})
	})

	ctx := context.Background()
	plan := campaignModel{
		Name:                types.StringValue("Multi country"),
		AdamID:              types.StringValue("9"),
		Status:              types.StringValue("PAUSED"),
		CountriesOrRegions:  campaignTestList(t, configured...),
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
	state, diags := campaignModelFromAPI(ctx, created)
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
	in, diags := campaignCreateFromPlan(ctx, campaignModel{
		Name:                types.StringValue("Paused Search Campaign"),
		AdamID:              types.StringValue("1234567890"),
		Status:              types.StringValue("PAUSED"),
		CountriesOrRegions:  campaignTestList(t, "US"),
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
	if in.PromotedObjectType != clientv1.PromotedObjectTypeAppStoreApp || in.PromotedObjectID != "1234567890" {
		t.Fatalf("promoted object = %s %s", in.PromotedObjectType, in.PromotedObjectID)
	}
	if in.AdChannelType != "SEARCH" {
		t.Fatalf("adChannelType = %q", in.AdChannelType)
	}
	supply := includeValues(in.Targeting.SupplyPlacement)
	if len(supply) != 1 || supply[0] != "APPSTORE_SEARCH_RESULTS" {
		t.Fatalf("supplyPlacement = %#v", supply)
	}
	cor := includeValues(in.Targeting.CountryOrRegion)
	if len(cor) != 1 || cor[0] != "US" {
		t.Fatalf("countryOrRegion = %#v", cor)
	}
	if in.BillingEvent != "TAPS" {
		t.Fatalf("billingEvent = %q", in.BillingEvent)
	}
	if in.BidStrategy == nil || in.BidStrategy.BidStrategyType != "MANUAL_CPT" || in.BidStrategy.Bid != nil {
		t.Fatalf("bidStrategy = %#v", in.BidStrategy)
	}
	if in.StartTime != "" {
		t.Fatalf("startTime should be omitted when unset, got %q", in.StartTime)
	}
	if in.DailyBudget == nil || in.DailyBudget.Value == nil || in.DailyBudget.Value.Amount != "5.00" || in.DailyBudget.Value.Currency != "USD" {
		t.Fatalf("dailyBudget = %#v", in.DailyBudget)
	}
	if len(in.SharedBudgets) != 0 {
		t.Fatalf("sharedBudgets = %#v", in.SharedBudgets)
	}
}

func TestCampaignCreateFromPlan_ExplicitDisplayChannelAndSupply(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	in, diags := campaignCreateFromPlan(ctx, campaignModel{
		Name:                types.StringValue("Display Campaign"),
		AdamID:              types.StringValue("1234567890"),
		Status:              types.StringValue("PAUSED"),
		CountriesOrRegions:  campaignTestList(t, "US"),
		DailyBudgetAmount:   types.StringValue("5.00"),
		DailyBudgetCurrency: types.StringValue("USD"),
		AdChannelType:       types.StringValue("DISPLAY"),
		SupplySources:       campaignTestList(t, "APPSTORE_SEARCH_TAB"),
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
	supply := includeValues(in.Targeting.SupplyPlacement)
	if len(supply) != 1 || supply[0] != "APPSTORE_SEARCH_TAB" {
		t.Fatalf("supplyPlacement = %#v, must not force APPSTORE_SEARCH_RESULTS", supply)
	}
	if in.BillingEvent != "TAPS" {
		t.Fatalf("billingEvent = %q", in.BillingEvent)
	}
}

func TestCampaignCreateFromPlan_StartTimePassThrough(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	in, diags := campaignCreateFromPlan(ctx, campaignModel{
		Name:                types.StringValue("Scheduled"),
		AdamID:              types.StringValue("9"),
		CountriesOrRegions:  campaignTestList(t, "US"),
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
	in, diags := campaignCreateFromPlan(ctx, campaignModel{
		Name:                types.StringValue("Max Conv"),
		AdamID:              types.StringValue("1234567890"),
		CountriesOrRegions:  campaignTestList(t, "US"),
		SupplySources:       campaignTestList(t, client.SupplySourceSearchResults),
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
	bs := in.BidStrategy
	if bs == nil || bs.BidStrategyType != client.BiddingStrategyMaxConversions {
		t.Fatalf("bidStrategy = %#v", bs)
	}
	if bs.Bid == nil || bs.Bid.Amount != "10.00" || bs.Bid.Currency != "USD" {
		t.Fatalf("bidStrategy.bid = %#v", bs.Bid)
	}
}

func TestCampaignCreateFromPlan_BudgetOrdersToSharedBudgets(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	in, diags := campaignCreateFromPlan(ctx, campaignModel{
		Name:               types.StringValue("Shared"),
		AdamID:             types.StringValue("9"),
		CountriesOrRegions: campaignTestList(t, "US"),
		SupplySources:      types.ListNull(types.StringType),
		BudgetOrders:       campaignTestList(t, "11", "22"),
	})
	if diags.HasError() {
		t.Fatalf("%v", diags)
	}
	if len(in.SharedBudgets) != 2 || in.SharedBudgets[0].BudgetID != 11 || in.SharedBudgets[1].BudgetID != 22 {
		t.Fatalf("sharedBudgets = %#v", in.SharedBudgets)
	}
	raw, err := json.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"sharedBudgets":[{"budgetId":11},{"budgetId":22}]`) {
		t.Fatalf("json = %s", raw)
	}
}

func TestCampaignCreateFromPlan_RejectsInvalidBudgetOrder(t *testing.T) {
	t.Parallel()

	_, diags := campaignCreateFromPlan(context.Background(), campaignModel{
		Name:               types.StringValue("Shared"),
		AdamID:             types.StringValue("9"),
		CountriesOrRegions: campaignTestList(t, "US"),
		SupplySources:      types.ListNull(types.StringType),
		BudgetOrders:       campaignTestList(t, "abc"),
	})
	if !diags.HasError() {
		t.Fatal("expected error for non-numeric budget order")
	}
}

func TestCampaignModelFromAPI_BiddingStrategy(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	state, diags := campaignModelFromAPI(ctx, &clientv1.Campaign{
		ID:                 77,
		Name:               "Read",
		PromotedObjectID:   "9",
		PromotedObjectType: clientv1.PromotedObjectTypeAppStoreApp,
		Status:             "PAUSED",
		AdChannelType:      client.AdChannelTypeSearch,
		BillingEvent:       client.BillingEventTaps,
		BidStrategy: &clientv1.BidStrategy{
			BidStrategyType: client.BiddingStrategyMaxConversions,
			Bid:             &clientv1.Money{Amount: "10.00", Currency: "USD"},
		},
		Targeting: &clientv1.CampaignTargeting{
			CountryOrRegion: &clientv1.IncludeList{Include: []string{"US"}},
			SupplyPlacement: &clientv1.IncludeList{Include: []string{client.SupplySourceSearchResults}},
		},
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

func TestCampaignModelFromAPI_SystemStatusAndComputed(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	state, diags := campaignModelFromAPI(ctx, &clientv1.Campaign{
		ID:                 77,
		Name:               "Read",
		PromotedObjectID:   "9",
		PromotedObjectType: clientv1.PromotedObjectTypeAppStoreApp,
		Status:             "PAUSED",
		AdChannelType:      "SEARCH",
		BillingEvent:       "TAPS",
		PaymentModel:       "PAYG",
		StartTime:          "2026-06-01T12:00:00.000",
		SystemStatus:       "NOT_RUNNING",
		DisplayStatus:      "PAUSED",
		ModificationTime:   "2026-06-02T00:00:00Z",
		Targeting: &clientv1.CampaignTargeting{
			CountryOrRegion: &clientv1.IncludeList{Include: []string{"US"}},
			SupplyPlacement: &clientv1.IncludeList{Include: []string{"APPSTORE_SEARCH_RESULTS"}},
		},
		SharedBudgets: []clientv1.SharedBudgetRef{{BudgetID: 5}, {BudgetID: 6}},
	})
	if diags.HasError() {
		t.Fatalf("%v", diags)
	}
	if state.PaymentModel.ValueString() != "PAYG" {
		t.Fatalf("payment_model = %q", state.PaymentModel.ValueString())
	}
	if state.ServingStatus.ValueString() != "NOT_RUNNING" {
		t.Fatalf("serving_status = %q", state.ServingStatus.ValueString())
	}
	if state.BillingEvent.ValueString() != "TAPS" {
		t.Fatalf("billing_event = %q", state.BillingEvent.ValueString())
	}
	if state.StartTime.ValueString() != "2026-06-01T12:00:00.000" {
		t.Fatalf("start_time = %q", state.StartTime.ValueString())
	}
	orders, d := stringList(ctx, state.BudgetOrders)
	if d.HasError() || len(orders) != 2 || orders[0] != "5" || orders[1] != "6" {
		t.Fatalf("budget_orders = %#v (%v)", orders, d)
	}

	// serving_status stays null when systemStatus is absent.
	state, diags = campaignModelFromAPI(ctx, &clientv1.Campaign{ID: 78, PromotedObjectID: "9"})
	if diags.HasError() {
		t.Fatalf("%v", diags)
	}
	if !state.ServingStatus.IsNull() {
		t.Fatalf("serving_status = %#v, want null", state.ServingStatus)
	}
}

func TestCampaignModelFromAPI_InfersChannelFromSupply(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	state, diags := campaignModelFromAPI(ctx, &clientv1.Campaign{
		ID:               1,
		PromotedObjectID: "9",
		Targeting: &clientv1.CampaignTargeting{
			SupplyPlacement: &clientv1.IncludeList{Include: []string{"APPSTORE_TODAY_TAB"}},
		},
	})
	if diags.HasError() {
		t.Fatalf("%v", diags)
	}
	if state.AdChannelType.ValueString() != "DISPLAY" {
		t.Fatalf("ad_channel_type = %q", state.AdChannelType.ValueString())
	}
}

func TestCampaignModelFromAPI_RejectsNonAppPromotedObject(t *testing.T) {
	t.Parallel()

	_, diags := campaignModelFromAPI(context.Background(), &clientv1.Campaign{
		ID:                 1,
		PromotedObjectID:   "b1",
		PromotedObjectType: clientv1.PromotedObjectTypeBusinessBrand,
	})
	if !diags.HasError() {
		t.Fatal("expected diagnostic for BUSINESS_BRAND campaign")
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

	apiClient := newCampaignTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"code":    "INVALID_ARGUMENT",
				"message": "validation failed",
				"details": []map[string]any{{
					"code":    "INVALID_ATTRIBUTE_VALUE",
					"message": "must be positive",
					"info":    map[string]string{"field": "dailyBudget"},
				}},
			},
		})
	})
	_, err := apiClient.CreateCampaign(context.Background(), &clientv1.CampaignCreate{
		Name:               "x",
		PromotedObjectID:   "1",
		PromotedObjectType: clientv1.PromotedObjectTypeAppStoreApp,
	})
	if err == nil {
		t.Fatal("expected error")
	}
	diags := apiErrorDiagnostic("Unable to create Apple Ads campaign", err)
	if !diags.HasError() {
		t.Fatal("expected diagnostic")
	}
	if !strings.Contains(diags[0].Detail(), "dailyBudget") {
		t.Fatalf("detail = %s", diags[0].Detail())
	}
}

func TestCampaignCreateFromPlan_RejectsNonPositiveBudget(t *testing.T) {
	t.Parallel()
	_, diags := campaignCreateFromPlan(context.Background(), campaignModel{
		Name:               types.StringValue("x"),
		AdamID:             types.StringValue("1"),
		CountriesOrRegions: campaignTestList(t, "US"),
		BudgetAmount:       types.StringValue("0"),
		BudgetCurrency:     types.StringValue("USD"),
		SupplySources:      types.ListNull(types.StringType),
		BudgetOrders:       types.ListNull(types.StringType),
	})
	if !diags.HasError() {
		t.Fatal("expected error for zero budget")
	}
}
