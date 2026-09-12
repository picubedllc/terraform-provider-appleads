// Copyright (c) 2026 Pi Cubed LLC
// SPDX-License-Identifier: MIT

package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/picubedllc/terraform-provider-appleads/internal/client"
)

func TestCampaignRead_SoftDeletedRemovesFromState(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"id":      7,
				"name":    "Gone",
				"adamId":  1,
				"deleted": true,
				"status":  "ENABLED",
			},
		})
	}))
	t.Cleanup(srv.Close)

	apiClient, err := client.New(client.WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	got, err := apiClient.GetCampaign(context.Background(), 7)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Deleted {
		t.Fatal("expected deleted=true")
	}
}

func TestCampaignRead_DriftUpdatesModel(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"id":                 7,
				"name":               "Renamed Externally",
				"adamId":             1,
				"status":             "PAUSED",
				"deleted":            false,
				"countriesOrRegions": []string{"US"},
				"servingStatus":      "NOT_RUNNING",
				"displayStatus":      "PAUSED",
				"modificationTime":   "2026-03-01T00:00:00Z",
			},
		})
	}))
	t.Cleanup(srv.Close)

	apiClient, err := client.New(client.WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	got, err := apiClient.GetCampaign(context.Background(), 7)
	if err != nil {
		t.Fatal(err)
	}
	state, diags := campaignModelFromClient(context.Background(), got)
	if diags.HasError() {
		t.Fatal(diags)
	}
	if state.Name.ValueString() != "Renamed Externally" {
		t.Fatalf("name = %s", state.Name.ValueString())
	}
	if state.Status.ValueString() != "PAUSED" {
		t.Fatalf("status = %s", state.Status.ValueString())
	}
}

func TestCampaignRead_NotFound404(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"errors": []map[string]string{{"messageCode": "NOT_FOUND", "message": "campaign not found"}},
			},
		})
	}))
	t.Cleanup(srv.Close)

	apiClient, err := client.New(client.WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	_, err = apiClient.GetCampaign(context.Background(), 404)
	if !client.IsNotFound(err) {
		t.Fatalf("err = %v", err)
	}
}

func TestCampaignModelFromClient_PreservesID(t *testing.T) {
	t.Parallel()
	m, diags := campaignModelFromClient(context.Background(), &client.Campaign{
		ID:                 42,
		Name:               "n",
		AdamID:             1,
		CountriesOrRegions: []string{"US"},
	})
	if diags.HasError() {
		t.Fatal(diags)
	}
	if m.ID.ValueString() != "42" {
		t.Fatalf("id = %s", m.ID.ValueString())
	}
	_ = types.String{}
}
