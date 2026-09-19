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

func TestCreateCampaign_IncludesOrgIDInBody(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body CampaignCreate
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.OrgID != 1234567 {
			t.Fatalf("orgId = %d, want 1234567", body.OrgID)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{"id": 1, "name": body.Name, "orgId": body.OrgID},
		})
	}))
	t.Cleanup(srv.Close)

	c, err := New(WithBaseURL(srv.URL), WithOrgID("1234567"))
	if err != nil {
		t.Fatal(err)
	}
	out, err := c.CreateCampaign(context.Background(), &CampaignCreate{
		Name:               "Demo",
		AdamID:             42,
		CountriesOrRegions: []string{"US"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.OrgID != 1234567 {
		t.Fatalf("orgId = %d", out.OrgID)
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
		daily, ok := raw["dailyBudgetAmount"].(map[string]any)
		if !ok {
			t.Fatalf("dailyBudgetAmount = %#v", raw["dailyBudgetAmount"])
		}
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

func TestUpdateCampaign_CountriesOrRegionsSetsClearGeoFlag(t *testing.T) {
	t.Parallel()

	var raw map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/campaigns/42" {
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"id":                 42,
				"name":               "example-campaign",
				"countriesOrRegions": []string{"US", "CA"},
			},
		})
	}))
	t.Cleanup(srv.Close)

	c, err := New(WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	dropped := []string{"US", "CA"}
	out, err := c.UpdateCampaign(context.Background(), 42, &CampaignUpdate{
		CountriesOrRegions:                       dropped,
		ClearGeoTargetingOnCountryOrRegionChange: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if raw["clearGeoTargetingOnCountryOrRegionChange"] != true {
		t.Fatalf("envelope flag = %#v", raw["clearGeoTargetingOnCountryOrRegionChange"])
	}
	campaign, ok := raw["campaign"].(map[string]any)
	if !ok {
		t.Fatalf("campaign = %#v", raw["campaign"])
	}
	if _, nested := campaign["clearGeoTargetingOnCountryOrRegionChange"]; nested {
		t.Fatal("clearGeoTargetingOnCountryOrRegionChange must be on the envelope, not the campaign object")
	}
	got, ok := campaign["countriesOrRegions"].([]any)
	if !ok {
		t.Fatalf("countriesOrRegions = %#v", campaign["countriesOrRegions"])
	}
	if len(got) != len(dropped) {
		t.Fatalf("countriesOrRegions = %#v", got)
	}
	for i, code := range dropped {
		if got[i] != code {
			t.Fatalf("countriesOrRegions[%d] = %#v, want %q", i, got[i], code)
		}
	}
	if out.ID != 42 {
		t.Fatalf("id = %d", out.ID)
	}
}

func TestUpdateCampaign_OmitsClearGeoFlagWhenUnchanged(t *testing.T) {
	t.Parallel()

	var raw map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{"id": 9, "name": "New"},
		})
	}))
	t.Cleanup(srv.Close)

	c, err := New(WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.UpdateCampaign(context.Background(), 9, &CampaignUpdate{Name: "New"})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := raw["clearGeoTargetingOnCountryOrRegionChange"]; ok {
		t.Fatalf("flag should be omitted when countries are unchanged, got %#v", raw)
	}
	campaign, ok := raw["campaign"].(map[string]any)
	if !ok {
		t.Fatalf("campaign = %#v", raw["campaign"])
	}
	if _, ok := campaign["countriesOrRegions"]; ok {
		t.Fatalf("countriesOrRegions should be omitted, got %#v", campaign)
	}
}
