// Copyright (c) 2026 Pi Cubed LLC
// SPDX-License-Identifier: MIT

package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCreateCampaign_SuccessPopulatesFromResponse(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/campaigns" {
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
		var body CampaignCreate
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.Name != "Demo" || body.AdamID != 42 {
			t.Fatalf("body = %#v", body)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"id":                 99,
				"name":               "Demo",
				"adamId":             42,
				"status":             "ENABLED",
				"servingStatus":      "RUNNING",
				"displayStatus":      "RUNNING",
				"countriesOrRegions": []string{"US"},
				"supplySources":      []string{"APPSTORE_SEARCH_RESULTS"},
				"adChannelType":      "SEARCH",
				"dailyBudgetAmount":  map[string]string{"amount": "50.00", "currency": "USD"},
				"modificationTime":   "2026-01-01T00:00:00Z",
				"deleted":            false,
			},
		})
	}))
	t.Cleanup(srv.Close)

	c, err := New(WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	out, err := c.CreateCampaign(context.Background(), &CampaignCreate{
		Name:               "Demo",
		AdamID:             42,
		CountriesOrRegions: []string{"US"},
		DailyBudgetAmount:  &Money{Amount: "50.00", Currency: "USD"},
		Status:             "ENABLED",
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.ID != 99 || out.ServingStatus != "RUNNING" || out.DailyBudgetAmount.Amount != "50.00" {
		t.Fatalf("out = %#v", out)
	}
}

func TestCreateCampaign_APIValidationErrorSurfaced(t *testing.T) {
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

	c, err := New(WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.CreateCampaign(context.Background(), &CampaignCreate{Name: "x", AdamID: 1, CountriesOrRegions: []string{"US"}})
	if err == nil {
		t.Fatal("expected error")
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("type = %T", err)
	}
	if !strings.Contains(apiErr.Message, "dailyBudgetAmount") {
		t.Fatalf("message = %s", apiErr.Message)
	}
}

func TestCreateCampaign_MoneyRoundTripNoFloat(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var raw map[string]any
		if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
			t.Fatal(err)
		}
		daily := raw["dailyBudgetAmount"].(map[string]any)
		if daily["amount"] != "12.34" {
			t.Fatalf("amount = %#v (float noise?)", daily["amount"])
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"id":                1,
				"name":              "m",
				"dailyBudgetAmount": map[string]string{"amount": "12.34", "currency": "USD"},
			},
		})
	}))
	t.Cleanup(srv.Close)

	c, err := New(WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	out, err := c.CreateCampaign(context.Background(), &CampaignCreate{
		Name:               "m",
		AdamID:             1,
		CountriesOrRegions: []string{"US"},
		DailyBudgetAmount:  &Money{Amount: "12.34", Currency: "USD"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.DailyBudgetAmount.Amount != "12.34" {
		t.Fatalf("amount = %q", out.DailyBudgetAmount.Amount)
	}
}
