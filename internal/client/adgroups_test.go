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
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.ID != 77 || out.CampaignID != 10 || out.DefaultBidAmount.Amount != "1.25" {
		t.Fatalf("out = %#v", out)
	}
}

func TestCreateAdGroup_MoneyRoundTripNoFloat(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var raw map[string]any
		if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
			t.Fatal(err)
		}
		bid := raw["defaultBidAmount"].(map[string]any)
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
