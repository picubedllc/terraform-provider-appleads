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

	"github.com/picubedllc/terraform-provider-appleads/internal/client"
)

func TestCampaignImport_HydratesFullState(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"id":                 77,
				"name":               "Imported",
				"adamId":             5,
				"status":             "ENABLED",
				"deleted":            false,
				"countriesOrRegions": []string{"US"},
				"servingStatus":      "RUNNING",
				"displayStatus":      "RUNNING",
				"modificationTime":   "2026-05-01T00:00:00Z",
			},
		})
	}))
	t.Cleanup(srv.Close)

	apiClient, err := client.New(client.WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	got, err := apiClient.GetCampaign(context.Background(), 77)
	if err != nil {
		t.Fatal(err)
	}
	state, diags := campaignModelFromClient(context.Background(), got)
	if diags.HasError() {
		t.Fatal(diags)
	}
	if state.ID.ValueString() != "77" || state.Name.ValueString() != "Imported" {
		t.Fatalf("%#v", state)
	}
	if state.ServingStatus.ValueString() != "RUNNING" {
		t.Fatalf("serving=%s", state.ServingStatus.ValueString())
	}
}

func TestCampaignImport_PreservesBudgetAmount(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"id":                 88,
				"name":               "Imported Budget",
				"adamId":             5,
				"status":             "PAUSED",
				"deleted":            false,
				"countriesOrRegions": []string{"US"},
				"budgetAmount":       map[string]string{"amount": "750.00", "currency": "USD"},
				"dailyBudgetAmount":  map[string]string{"amount": "30.00", "currency": "USD"},
				"servingStatus":      "RUNNING",
				"displayStatus":      "RUNNING",
				"modificationTime":   "2026-05-01T00:00:00Z",
			},
		})
	}))
	t.Cleanup(srv.Close)

	apiClient, err := client.New(client.WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	got, err := apiClient.GetCampaign(context.Background(), 88)
	if err != nil {
		t.Fatal(err)
	}
	state, diags := campaignModelFromClient(context.Background(), got)
	if diags.HasError() {
		t.Fatal(diags)
	}
	if state.BudgetAmount.ValueString() != "750.00" || state.BudgetCurrency.ValueString() != "USD" {
		t.Fatalf("budget = %s %s", state.BudgetAmount.ValueString(), state.BudgetCurrency.ValueString())
	}
	if state.DailyBudgetAmount.ValueString() != "30.00" {
		t.Fatalf("daily = %q", state.DailyBudgetAmount.ValueString())
	}
}

func TestCampaignImport_RejectsArchived(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{"id": 77, "name": "Archived", "adamId": 1, "deleted": true},
		})
	}))
	t.Cleanup(srv.Close)

	apiClient, err := client.New(client.WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
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

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{"errors": []map[string]string{{"messageCode": "NOT_FOUND", "message": "missing"}}},
		})
	}))
	t.Cleanup(srv.Close)

	apiClient, err := client.New(client.WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	_, err = apiClient.GetCampaign(context.Background(), 404)
	if !client.IsNotFound(err) {
		t.Fatalf("%v", err)
	}
	if !strings.Contains(err.Error(), "missing") && !strings.Contains(err.Error(), "404") {
		t.Fatalf("%v", err)
	}
}
