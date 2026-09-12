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
