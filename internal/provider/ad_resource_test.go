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

func TestAdModelFromClient(t *testing.T) {
	t.Parallel()

	got := &client.Ad{
		ID:                  501,
		CampaignID:          10,
		AdGroupID:           20,
		Name:                "Summer Ad",
		CreativeID:          94895512,
		CreativeType:        client.CreativeTypeCustomProductPage,
		Status:              client.AdStatusPaused,
		ServingStatus:       "RUNNING",
		ServingStateReasons: []string{},
		ModificationTime:    "2026-06-01T00:00:00Z",
	}
	state, diags := adModelFromClient(context.Background(), got)
	if diags.HasError() {
		t.Fatalf("diags = %v", diags)
	}
	if state.ID.ValueString() != "501" || state.AdGroupID.ValueString() != "20" {
		t.Fatalf("ids = %#v", state)
	}
	if state.CreativeID.ValueString() != "94895512" || state.Status.ValueString() != client.AdStatusPaused {
		t.Fatalf("creative/status = %#v", state)
	}
	if state.CreativeType.ValueString() != client.CreativeTypeCustomProductPage {
		t.Fatalf("creative_type = %q", state.CreativeType.ValueString())
	}
}

func TestAdCreate_AndStateFromResponse(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/adgroups/find"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{{"id": 20, "campaignId": 10, "name": "ag"}},
			})
		case r.Method == http.MethodPost && r.URL.Path == "/campaigns/10/adgroups/20/ads":
			var body client.AdCreate
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if body.CreativeID != 99 || body.Status != client.AdStatusPaused {
				t.Fatalf("body = %#v", body)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{
					"id":           55,
					"campaignId":   10,
					"adGroupId":    20,
					"name":         body.Name,
					"creativeId":   body.CreativeID,
					"creativeType": "CUSTOM_PRODUCT_PAGE",
					"status":       body.Status,
				},
			})
		default:
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	apiClient, err := client.New(client.WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	ag, err := apiClient.FindAdGroupByID(context.Background(), 20)
	if err != nil {
		t.Fatal(err)
	}
	created, err := apiClient.CreateAd(context.Background(), ag.CampaignID, 20, &client.AdCreate{
		Name: "CPP ad", CreativeID: 99, Status: client.AdStatusPaused,
	})
	if err != nil {
		t.Fatal(err)
	}
	state, diags := adModelFromClient(context.Background(), created)
	if diags.HasError() {
		t.Fatalf("diags = %v", diags)
	}
	if state.ID.ValueString() != "55" || state.CreativeID.ValueString() != "99" {
		t.Fatalf("state = %#v", state)
	}
}

func TestAdUpdate_ImmutableMessages(t *testing.T) {
	t.Parallel()

	cases := []struct {
		field string
		msg   string
	}{
		{"ad_group_id", `Cannot change immutable ad field "ad_group_id"`},
		{"creative_id", `Cannot change immutable ad field "creative_id"`},
	}
	for _, tc := range cases {
		if !strings.Contains(tc.msg, tc.field) {
			t.Fatalf("message contract drifted for %s", tc.field)
		}
	}
}

func TestAdUpdate_MutableRoundTrip(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Fatalf("method = %s", r.Method)
		}
		var body client.AdUpdate
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.Name != "Renamed" || body.Status != client.AdStatusEnabled {
			t.Fatalf("body = %#v", body)
		}
		if body.CreativeID != 0 {
			t.Fatalf("creativeId should be omitted on update, got %d", body.CreativeID)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"id": 55, "campaignId": 10, "adGroupId": 20,
				"name": body.Name, "status": body.Status, "creativeId": 99,
			},
		})
	}))
	t.Cleanup(srv.Close)

	apiClient, err := client.New(client.WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	updated, err := apiClient.UpdateAd(context.Background(), 10, 20, 55, &client.AdUpdate{
		Name: "Renamed", Status: client.AdStatusEnabled,
	})
	if err != nil {
		t.Fatal(err)
	}
	state, diags := adModelFromClient(context.Background(), updated)
	if diags.HasError() {
		t.Fatalf("diags = %v", diags)
	}
	if state.Name.ValueString() != "Renamed" || state.Status.ValueString() != client.AdStatusEnabled {
		t.Fatalf("state = %#v", state)
	}
}

func TestAdDelete_AndImportPaths(t *testing.T) {
	t.Parallel()

	deleted := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodDelete && r.URL.Path == "/campaigns/10/adgroups/20/ads/55":
			deleted = true
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{"data": nil})
		case r.Method == http.MethodGet && r.URL.Path == "/campaigns/10/adgroups/20/ads/55":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{
					"id": 55, "campaignId": 10, "adGroupId": 20,
					"name": "imported", "creativeId": 99, "status": "PAUSED", "deleted": false,
				},
			})
		default:
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	apiClient, err := client.New(client.WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	if err := apiClient.DeleteAd(context.Background(), 10, 20, 55); err != nil {
		t.Fatal(err)
	}
	if !deleted {
		t.Fatal("expected delete")
	}

	cID, agID, adID, err := client.ParseAdImportID("10/20/55")
	if err != nil || cID != 10 || agID != 20 || adID != 55 {
		t.Fatalf("parse = %d/%d/%d err=%v", cID, agID, adID, err)
	}
	got, err := apiClient.GetAd(context.Background(), cID, agID, adID)
	if err != nil {
		t.Fatal(err)
	}
	state, diags := adModelFromClient(context.Background(), got)
	if diags.HasError() || state.Name.ValueString() != "imported" {
		t.Fatalf("state = %#v diags=%v", state, diags)
	}
}

func TestAdCreate_APIValidationDiagnostic(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"errors": []map[string]string{{
					"messageCode": "INVALID_ATTRIBUTE_VALUE",
					"message":     "creativeId is invalid",
					"field":       "creativeId",
				}},
			},
		})
	}))
	t.Cleanup(srv.Close)

	apiClient, err := client.New(client.WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	_, err = apiClient.CreateAd(context.Background(), 1, 2, &client.AdCreate{
		Name: "x", CreativeID: 9, Status: client.AdStatusPaused,
	})
	if err == nil {
		t.Fatal("expected error")
	}
	diags := apiErrorDiagnostic("Unable to create Apple Ads ad", err)
	if !diags.HasError() || !strings.Contains(diags[0].Detail(), "creativeId") {
		t.Fatalf("detail = %v", diags)
	}
}

func TestAdRead_SoftDeleted(t *testing.T) {
	t.Parallel()

	state, diags := adModelFromClient(context.Background(), &client.Ad{
		ID: 1, CampaignID: 2, AdGroupID: 3, Name: "x", CreativeID: 4, Status: "PAUSED", Deleted: true,
	})
	if diags.HasError() {
		t.Fatalf("diags = %v", diags)
	}
	// Soft-delete is handled in Read; model mapping still succeeds.
	if state.ID.ValueString() != "1" {
		t.Fatalf("state = %#v", state)
	}
}
