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

func TestCreateAdGroup_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/campaigns/10/adgroups" {
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
		var body AdGroupCreate
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.Name != "Main" || body.DefaultBidAmount == nil || body.DefaultBidAmount.Amount != "1.25" {
			t.Fatalf("body = %#v", body)
		}
		if body.PricingModel != PricingModelCPC {
			t.Fatalf("pricingModel = %q", body.PricingModel)
		}
		if body.StartTime != "2026-01-01T00:00:00.000" {
			t.Fatalf("startTime = %q", body.StartTime)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"id":                     77,
				"campaignId":             10,
				"name":                   body.Name,
				"status":                 "ENABLED",
				"servingStatus":          "RUNNING",
				"displayStatus":          "RUNNING",
				"defaultBidAmount":       map[string]string{"amount": "1.25", "currency": "USD"},
				"automatedKeywordsOptIn": true,
				"modificationTime":       "2026-05-01T00:00:00Z",
				"deleted":                false,
			},
		})
	}))
	t.Cleanup(srv.Close)

	c, err := New(WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	out, err := c.CreateAdGroup(context.Background(), 10, &AdGroupCreate{
		Name:                   "Main",
		DefaultBidAmount:       &Money{Amount: "1.25", Currency: "USD"},
		AutomatedKeywordsOptIn: true,
		Status:                 "ENABLED",
		PricingModel:           PricingModelCPC,
		StartTime:              "2026-01-01T00:00:00.000",
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.ID != 77 || out.CampaignID != 10 || out.DefaultBidAmount.Amount != "1.25" {
		t.Fatalf("out = %#v", out)
	}
}

func TestCreateAdGroup_DoesNotDefaultRequiredFields(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body AdGroupCreate
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.PricingModel != "" {
			t.Fatalf("pricingModel = %q, want empty", body.PricingModel)
		}
		if body.StartTime != "" {
			t.Fatalf("startTime = %q, want empty", body.StartTime)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{"id": 1, "campaignId": 10, "name": body.Name},
		})
	}))
	t.Cleanup(srv.Close)

	c, err := New(WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.CreateAdGroup(context.Background(), 10, &AdGroupCreate{
		Name:             "Main",
		DefaultBidAmount: &Money{Amount: "1.00", Currency: "USD"},
	}); err != nil {
		t.Fatal(err)
	}
}

func TestCreateAdGroup_KeepsExplicitPricingModel(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body AdGroupCreate
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.PricingModel != PricingModelCPM {
			t.Fatalf("pricingModel = %q", body.PricingModel)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{"id": 1, "campaignId": 10, "name": body.Name, "pricingModel": body.PricingModel},
		})
	}))
	t.Cleanup(srv.Close)

	c, err := New(WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	out, err := c.CreateAdGroup(context.Background(), 10, &AdGroupCreate{
		Name:             "Main",
		DefaultBidAmount: &Money{Amount: "1.00", Currency: "USD"},
		PricingModel:     PricingModelCPM,
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.PricingModel != PricingModelCPM {
		t.Fatalf("out.PricingModel = %q", out.PricingModel)
	}
}

func TestCreateAdGroup_MoneyRoundTripNoFloat(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var raw map[string]any
		if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
			t.Fatal(err)
		}
		bid, ok := raw["defaultBidAmount"].(map[string]any)
		if !ok {
			t.Fatalf("defaultBidAmount = %#v", raw["defaultBidAmount"])
		}
		if bid["amount"] != "0.99" {
			t.Fatalf("amount = %#v (float noise?)", bid["amount"])
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"id":               1,
				"campaignId":       2,
				"name":             "m",
				"defaultBidAmount": map[string]string{"amount": "0.99", "currency": "USD"},
			},
		})
	}))
	t.Cleanup(srv.Close)

	c, err := New(WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	out, err := c.CreateAdGroup(context.Background(), 2, &AdGroupCreate{
		Name:             "m",
		DefaultBidAmount: &Money{Amount: "0.99", Currency: "USD"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.DefaultBidAmount.Amount != "0.99" {
		t.Fatalf("amount = %q", out.DefaultBidAmount.Amount)
	}
}

func TestUpdateAdGroup_WrapsEnvelope(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/campaigns/10/adgroups/77" {
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
		var env adGroupUpdateEnvelope
		if err := json.NewDecoder(r.Body).Decode(&env); err != nil {
			t.Fatal(err)
		}
		if env.AdGroup == nil || env.AdGroup.Name != "Renamed" {
			t.Fatalf("env = %#v", env)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"id":         77,
				"campaignId": 10,
				"name":       "Renamed",
				"status":     "PAUSED",
			},
		})
	}))
	t.Cleanup(srv.Close)

	c, err := New(WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	out, err := c.UpdateAdGroup(context.Background(), 10, 77, &AdGroupUpdate{Name: "Renamed", Status: "PAUSED"})
	if err != nil {
		t.Fatal(err)
	}
	if out.Name != "Renamed" {
		t.Fatalf("out = %#v", out)
	}
}

func TestGetAdGroup_SoftDeletedFlag(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"id":         3,
				"campaignId": 1,
				"name":       "Gone",
				"deleted":    true,
			},
		})
	}))
	t.Cleanup(srv.Close)

	c, err := New(WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	got, err := c.GetAdGroup(context.Background(), 1, 3)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Deleted {
		t.Fatal("expected deleted=true")
	}
}

func TestFindAdGroupByID(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/adgroups/find" {
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{{
				"id":         88,
				"campaignId": 9,
				"name":       "Found",
				"deleted":    false,
			}},
		})
	}))
	t.Cleanup(srv.Close)

	c, err := New(WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	got, err := c.FindAdGroupByID(context.Background(), 88)
	if err != nil {
		t.Fatal(err)
	}
	if got.CampaignID != 9 || got.Name != "Found" {
		t.Fatalf("got = %#v", got)
	}
}

func TestParseAdGroupImportID(t *testing.T) {
	t.Parallel()

	cid, aid, err := ParseAdGroupImportID("123")
	if err != nil || cid != 0 || aid != 123 {
		t.Fatalf("bare: %d %d %v", cid, aid, err)
	}
	cid, aid, err = ParseAdGroupImportID("10/77")
	if err != nil || cid != 10 || aid != 77 {
		t.Fatalf("compound: %d %d %v", cid, aid, err)
	}
	_, _, err = ParseAdGroupImportID("a/b/c")
	if err == nil || !strings.Contains(err.Error(), "invalid ad group import id") {
		t.Fatalf("err = %v", err)
	}
}

func TestCreateAdGroup_APIValidationError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"errors": []map[string]string{{
					"messageCode": "INVALID_ATTRIBUTE_VALUE",
					"message":     "defaultBidAmount must be positive",
					"field":       "defaultBidAmount",
				}},
			},
		})
	}))
	t.Cleanup(srv.Close)

	c, err := New(WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.CreateAdGroup(context.Background(), 1, &AdGroupCreate{Name: "x", DefaultBidAmount: &Money{Amount: "0", Currency: "USD"}})
	if err == nil {
		t.Fatal("expected error")
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("type = %T", err)
	}
	if !strings.Contains(apiErr.Message, "defaultBidAmount") {
		t.Fatalf("message = %s", apiErr.Message)
	}
}

func TestCreateAdGroup_SendsLocalityTargeting(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body AdGroupCreate
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.TargetingDimensions == nil || body.TargetingDimensions.Locality == nil {
			t.Fatalf("targetingDimensions = %#v", body.TargetingDimensions)
		}
		if got := body.TargetingDimensions.Locality.Included; len(got) != 1 || got[0] != "US|NY|New York" {
			t.Fatalf("locality = %#v", body.TargetingDimensions.Locality)
		}
		if body.TargetingDimensions.Age != nil || body.TargetingDimensions.Daypart != nil {
			t.Fatalf("omitted dimensions should be absent on create: %#v", body.TargetingDimensions)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"id":         77,
				"campaignId": 10,
				"name":       body.Name,
				"targetingDimensions": map[string]any{
					"age":         nil,
					"gender":      nil,
					"country":     nil,
					"adminArea":   nil,
					"locality":    map[string]any{"included": []string{"US|NY|New York"}},
					"deviceClass": map[string]any{"included": []string{"IPHONE", "IPAD"}},
					"daypart":     nil,
					"appDownloaders": map[string]any{
						"included": []any{},
						"excluded": []any{},
					},
				},
			},
		})
	}))
	t.Cleanup(srv.Close)

	c, err := New(WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	out, err := c.CreateAdGroup(context.Background(), 10, &AdGroupCreate{
		Name:             "NYC",
		DefaultBidAmount: &Money{Amount: "1.00", Currency: "USD"},
		PricingModel:     PricingModelCPC,
		StartTime:        "2026-01-01T00:00:00.000",
		TargetingDimensions: &TargetingDimensions{
			Locality: &LocalityTarget{Included: []string{"US|NY|New York"}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.TargetingDimensions == nil || out.TargetingDimensions.Locality == nil {
		t.Fatalf("out.TargetingDimensions = %#v", out.TargetingDimensions)
	}
	if got := out.TargetingDimensions.Locality.Included; len(got) != 1 || got[0] != "US|NY|New York" {
		t.Fatalf("locality = %#v", out.TargetingDimensions.Locality)
	}
}

func TestUpdateAdGroup_TargetingDimensionsSendsNullsForUnset(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var raw map[string]any
		if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
			t.Fatal(err)
		}
		ag, ok := raw["adGroup"].(map[string]any)
		if !ok {
			t.Fatalf("envelope = %#v", raw)
		}
		td, ok := ag["targetingDimensions"].(map[string]any)
		if !ok {
			t.Fatalf("targetingDimensions = %#v", ag["targetingDimensions"])
		}
		for _, key := range []string{"age", "gender", "deviceClass", "country", "adminArea", "daypart", "appDownloaders"} {
			if td[key] != nil {
				t.Fatalf("%s = %#v, want null", key, td[key])
			}
		}
		loc, ok := td["locality"].(map[string]any)
		if !ok {
			t.Fatalf("locality = %#v", td["locality"])
		}
		included, ok := loc["included"].([]any)
		if !ok || len(included) != 1 || included[0] != "US|NY|New York" {
			t.Fatalf("locality.included = %#v", loc["included"])
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{"id": 77, "campaignId": 10, "name": "NYC"},
		})
	}))
	t.Cleanup(srv.Close)

	c, err := New(WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.UpdateAdGroup(context.Background(), 10, 77, &AdGroupUpdate{
		Name: "NYC",
		TargetingDimensions: &TargetingDimensions{
			Locality: &LocalityTarget{Included: []string{"US|NY|New York"}},
		},
	}); err != nil {
		t.Fatal(err)
	}
}

func TestUpdateAdGroup_ClearTargetingDimensions(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var raw map[string]any
		if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
			t.Fatal(err)
		}
		ag, ok := raw["adGroup"].(map[string]any)
		if !ok {
			t.Fatalf("envelope = %#v", raw)
		}
		if ag["targetingDimensions"] != nil {
			t.Fatalf("targetingDimensions = %#v, want JSON null", ag["targetingDimensions"])
		}
		if _, present := ag["targetingDimensions"]; !present {
			t.Fatal("targetingDimensions key missing; want explicit null")
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{"id": 77, "campaignId": 10, "name": "NYC"},
		})
	}))
	t.Cleanup(srv.Close)

	c, err := New(WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.UpdateAdGroup(context.Background(), 10, 77, &AdGroupUpdate{
		Name:                     "NYC",
		ClearTargetingDimensions: true,
	}); err != nil {
		t.Fatal(err)
	}
}

func TestUpdateAdGroup_OmitsTargetingWhenUnset(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var raw map[string]any
		if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
			t.Fatal(err)
		}
		ag, ok := raw["adGroup"].(map[string]any)
		if !ok {
			t.Fatalf("envelope = %#v", raw)
		}
		if _, present := ag["targetingDimensions"]; present {
			t.Fatalf("targetingDimensions should be omitted, got %#v", ag["targetingDimensions"])
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{"id": 77, "campaignId": 10, "name": "Renamed"},
		})
	}))
	t.Cleanup(srv.Close)

	c, err := New(WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.UpdateAdGroup(context.Background(), 10, 77, &AdGroupUpdate{Name: "Renamed"}); err != nil {
		t.Fatal(err)
	}
}

func TestListAdGroups(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/campaigns/10/adgroups" {
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
		if r.URL.Query().Get("limit") != "50" {
			t.Fatalf("limit = %q", r.URL.Query().Get("limit"))
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{"id": 1, "campaignId": 10, "name": "Auto", "automatedKeywordsOptIn": true},
			},
			"pagination": map[string]any{"totalResults": 1, "startIndex": 0, "itemsPerPage": 50},
		})
	}))
	t.Cleanup(srv.Close)

	c, err := New(WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	page, err := c.ListAdGroups(context.Background(), 10, PageParams{Limit: 50, Offset: 0})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Data) != 1 || page.Data[0].Name != "Auto" || !page.Data[0].AutomatedKeywordsOptIn {
		t.Fatalf("page = %#v", page)
	}
}
