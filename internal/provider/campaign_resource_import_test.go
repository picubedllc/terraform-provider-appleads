// Copyright (c) 2026 Pi Cubed LLC
// SPDX-License-Identifier: MIT

package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/picubedllc/terraform-provider-appleads/internal/client"
)

func TestCampaignImport_HydratesFullState(t *testing.T) {
	t.Parallel()

	apiClient := newCampaignTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/campaigns/77" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"result": map[string]any{
				"id":                 77,
				"name":               "Imported",
				"promotedObjectId":   "5",
				"promotedObjectType": "APPSTORE_APP",
				"status":             "ENABLED",
				"deleted":            false,
				"targeting": map[string]any{
					"countryOrRegion": map[string]any{"include": []string{"US"}},
					"supplyPlacement": map[string]any{"include": []string{"APPSTORE_SEARCH_RESULTS"}},
				},
				"bidStrategy":      map[string]any{"bidStrategyType": "MANUAL_CPT"},
				"sharedBudgets":    []map[string]any{{"budgetId": 12}},
				"systemStatus":     "RUNNING",
				"displayStatus":    "RUNNING",
				"modificationTime": "2026-05-01T00:00:00Z",
			},
		})
	})
	got, err := apiClient.GetCampaign(context.Background(), 77)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	state, diags := campaignModelFromAPI(ctx, got)
	if diags.HasError() {
		t.Fatal(diags)
	}
	if state.ID.ValueString() != "77" || state.Name.ValueString() != "Imported" || state.AdamID.ValueString() != "5" {
		t.Fatalf("%#v", state)
	}
	if state.ServingStatus.ValueString() != "RUNNING" {
		t.Fatalf("serving=%s", state.ServingStatus.ValueString())
	}
	if state.AdChannelType.ValueString() != "SEARCH" || state.BiddingStrategy.ValueString() != "MANUAL_CPT" {
		t.Fatalf("channel=%s bidding=%s", state.AdChannelType.ValueString(), state.BiddingStrategy.ValueString())
	}
	orders, d := stringList(ctx, state.BudgetOrders)
	if d.HasError() || len(orders) != 1 || orders[0] != "12" {
		t.Fatalf("budget_orders = %#v (%v)", orders, d)
	}
}

func TestCampaignImport_DailyBudgetAndNoLifetimeBudget(t *testing.T) {
	t.Parallel()

	apiClient := newCampaignTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"result": map[string]any{
				"id":                 88,
				"name":               "Imported Budget",
				"promotedObjectId":   "5",
				"promotedObjectType": "APPSTORE_APP",
				"status":             "PAUSED",
				"deleted":            false,
				"targeting": map[string]any{
					"countryOrRegion": map[string]any{"include": []string{"US"}},
				},
				"dailyBudget":      map[string]any{"value": map[string]string{"amount": "30.00", "currency": "USD"}},
				"systemStatus":     "RUNNING",
				"displayStatus":    "RUNNING",
				"modificationTime": "2026-05-01T00:00:00Z",
			},
		})
	})
	got, err := apiClient.GetCampaign(context.Background(), 88)
	if err != nil {
		t.Fatal(err)
	}
	state, diags := campaignModelFromAPI(context.Background(), got)
	if diags.HasError() {
		t.Fatal(diags)
	}
	if !state.BudgetAmount.IsNull() || !state.BudgetCurrency.IsNull() {
		t.Fatalf("lifetime budget must be null on import, got %q %q", state.BudgetAmount.ValueString(), state.BudgetCurrency.ValueString())
	}
	if state.DailyBudgetAmount.ValueString() != "30.00" || state.DailyBudgetCurrency.ValueString() != "USD" {
		t.Fatalf("daily = %q %q", state.DailyBudgetAmount.ValueString(), state.DailyBudgetCurrency.ValueString())
	}
}

func TestCampaignImport_RejectsArchived(t *testing.T) {
	t.Parallel()

	apiClient := newCampaignTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"result": map[string]any{"id": 77, "name": "Archived", "promotedObjectId": "1", "deleted": true},
		})
	})
	got, err := apiClient.GetCampaign(context.Background(), 77)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Deleted {
		t.Fatal("expected deleted")
	}
}

func TestCampaignImport_NotFound(t *testing.T) {
	t.Parallel()

	apiClient := newCampaignTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{"code": "NOT_FOUND", "message": "missing"},
		})
	})
	_, err := apiClient.GetCampaign(context.Background(), 404)
	if !client.IsNotFound(err) {
		t.Fatalf("%v", err)
	}
	if !strings.Contains(err.Error(), "missing") && !strings.Contains(err.Error(), "404") {
		t.Fatalf("%v", err)
	}
}
