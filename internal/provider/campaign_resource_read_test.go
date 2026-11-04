// Copyright (c) 2026 Pi Cubed LLC
// SPDX-License-Identifier: MIT

package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/picubedllc/terraform-provider-appleads/internal/client"
	clientv1 "github.com/picubedllc/terraform-provider-appleads/internal/client/v1"
)

func TestCampaignRead_SoftDeletedRemovesFromState(t *testing.T) {
	t.Parallel()

	apiClient := newCampaignTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/campaigns/7" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"result": map[string]any{
				"id":                 7,
				"name":               "Gone",
				"promotedObjectId":   "1",
				"promotedObjectType": "APPSTORE_APP",
				"deleted":            true,
				"status":             "ENABLED",
			},
		})
	})
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

	apiClient := newCampaignTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"result": map[string]any{
				"id":                 7,
				"name":               "Renamed Externally",
				"promotedObjectId":   "1",
				"promotedObjectType": "APPSTORE_APP",
				"status":             "PAUSED",
				"deleted":            false,
				"targeting": map[string]any{
					"countryOrRegion": map[string]any{"include": []string{"US"}},
				},
				"systemStatus":     "NOT_RUNNING",
				"displayStatus":    "PAUSED",
				"modificationTime": "2026-03-01T00:00:00Z",
			},
		})
	})
	got, err := apiClient.GetCampaign(context.Background(), 7)
	if err != nil {
		t.Fatal(err)
	}
	state, diags := campaignModelFromAPI(context.Background(), got)
	if diags.HasError() {
		t.Fatal(diags)
	}
	if state.Name.ValueString() != "Renamed Externally" {
		t.Fatalf("name = %s", state.Name.ValueString())
	}
	if state.Status.ValueString() != "PAUSED" {
		t.Fatalf("status = %s", state.Status.ValueString())
	}
	if state.ServingStatus.ValueString() != "NOT_RUNNING" {
		t.Fatalf("serving_status = %s", state.ServingStatus.ValueString())
	}
}

func TestCampaignRead_NotFound404(t *testing.T) {
	t.Parallel()

	apiClient := newCampaignTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{"code": "NOT_FOUND", "message": "campaign not found"},
		})
	})
	_, err := apiClient.GetCampaign(context.Background(), 404)
	if !client.IsNotFound(err) {
		t.Fatalf("err = %v", err)
	}
}

func TestCampaignModelFromAPI_PreservesID(t *testing.T) {
	t.Parallel()
	m, diags := campaignModelFromAPI(context.Background(), &clientv1.Campaign{
		ID:               42,
		Name:             "n",
		PromotedObjectID: "1",
		Targeting: &clientv1.CampaignTargeting{
			CountryOrRegion: &clientv1.IncludeList{Include: []string{"US"}},
		},
	})
	if diags.HasError() {
		t.Fatal(diags)
	}
	if m.ID.ValueString() != "42" {
		t.Fatalf("id = %s", m.ID.ValueString())
	}
	if !m.BudgetOrders.IsNull() {
		t.Fatalf("budget_orders = %#v, want null", m.BudgetOrders)
	}
}
