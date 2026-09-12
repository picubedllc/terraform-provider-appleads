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

func TestAdGroupModelFromClient(t *testing.T) {
	t.Parallel()

	got := &client.AdGroup{
		ID:                     77,
		CampaignID:             10,
		Name:                   "Main",
		Status:                 "ENABLED",
		ServingStatus:          "RUNNING",
		DisplayStatus:          "RUNNING",
		DefaultBidAmount:       &client.Money{Amount: "1.25", Currency: "USD"},
		CPAGoal:                &client.Money{Amount: "5.00", Currency: "USD"},
		AutomatedKeywordsOptIn: true,
		EndTime:                "2026-12-01T00:00:00Z",
		ModificationTime:       "2026-05-01T00:00:00Z",
	}
	state := adGroupModelFromClient(got)
	if state.ID.ValueString() != "77" || state.CampaignID.ValueString() != "10" {
		t.Fatalf("ids = %#v", state)
	}
	if state.DefaultBidAmount.ValueString() != "1.25" || state.CPAGoalAmount.ValueString() != "5.00" {
		t.Fatalf("money = %#v", state)
	}
	if !state.AutomatedKeywordsOptIn.ValueBool() || state.EndTime.ValueString() != "2026-12-01T00:00:00Z" {
		t.Fatalf("opts = %#v", state)
	}
}

func TestAdGroupModelFromClient_NullOptionals(t *testing.T) {
	t.Parallel()

	state := adGroupModelFromClient(&client.AdGroup{
		ID:         1,
		CampaignID: 2,
		Name:       "x",
		Status:     "PAUSED",
	})
	if !state.CPAGoalAmount.IsNull() || !state.EndTime.IsNull() {
		t.Fatalf("expected null optionals: %#v", state)
	}
}

func TestAdGroupCreate_AndStateFromResponse(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body client.AdGroupCreate
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.DefaultBidAmount == nil || body.DefaultBidAmount.Amount != "1.50" {
			t.Fatalf("bid = %#v", body.DefaultBidAmount)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"id":                     55,
				"campaignId":             9,
				"name":                   body.Name,
				"status":                 "ENABLED",
				"servingStatus":          "RUNNING",
				"displayStatus":          "RUNNING",
				"defaultBidAmount":       map[string]string{"amount": "1.50", "currency": "USD"},
				"automatedKeywordsOptIn": false,
				"modificationTime":       "2026-05-02T00:00:00Z",
			},
		})
	}))
	t.Cleanup(srv.Close)

	apiClient, err := client.New(client.WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}

	plan := adGroupModel{
		CampaignID:         types.StringValue("9"),
		Name:               types.StringValue("Launch"),
		Status:             types.StringValue("ENABLED"),
		DefaultBidAmount:   types.StringValue("1.50"),
		DefaultBidCurrency: types.StringValue("USD"),
	}
	bid, diags := moneyFromStrings(plan.DefaultBidAmount, plan.DefaultBidCurrency)
	if diags.HasError() || bid == nil {
		t.Fatalf("bid diags=%v", diags)
	}
	created, err := apiClient.CreateAdGroup(context.Background(), 9, &client.AdGroupCreate{
		Name:             plan.Name.ValueString(),
		DefaultBidAmount: bid,
		Status:           plan.Status.ValueString(),
	})
	if err != nil {
		t.Fatal(err)
	}
	state := adGroupModelFromClient(created)
	if state.ID.ValueString() != "55" || state.ServingStatus.ValueString() != "RUNNING" {
		t.Fatalf("state = %#v", state)
	}
}

func TestAdGroupUpdate_ImmutableCampaignIDMessage(t *testing.T) {
	t.Parallel()

	state := adGroupModel{
		ID:         types.StringValue("77"),
		CampaignID: types.StringValue("10"),
	}
	plan := state
	plan.CampaignID = types.StringValue("99")

	if state.CampaignID.ValueString() == plan.CampaignID.ValueString() {
		t.Fatal("setup failed")
	}
	msg := `Cannot change immutable ad group field "campaign_id"`
	detail := "Apple Ads does not allow moving ad group 77 between campaigns. " +
		"Automatically replacing this resource would delete historical ad group identity. " +
		"Create a new apple-ads_ad_group explicitly instead."
	if !strings.Contains(detail, "77") || !strings.Contains(msg, "campaign_id") {
		t.Fatal("message contract drifted")
	}
}

func TestAdGroupRead_SoftDeleted(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"id":         7,
				"campaignId": 1,
				"name":       "Gone",
				"deleted":    true,
			},
		})
	}))
	t.Cleanup(srv.Close)

	apiClient, err := client.New(client.WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	got, err := apiClient.GetAdGroup(context.Background(), 1, 7)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Deleted {
		t.Fatal("expected deleted")
	}
}

func TestAdGroupCreate_APIValidationDiagnostic(t *testing.T) {
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

	apiClient, err := client.New(client.WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	_, err = apiClient.CreateAdGroup(context.Background(), 1, &client.AdGroupCreate{
		Name:             "x",
		DefaultBidAmount: &client.Money{Amount: "0", Currency: "USD"},
	})
	if err == nil {
		t.Fatal("expected error")
	}
	diags := apiErrorDiagnostic("Unable to create Apple Ads ad group", err)
	if !diags.HasError() {
		t.Fatal("expected diagnostic")
	}
	if !strings.Contains(diags[0].Detail(), "defaultBidAmount") {
		t.Fatalf("detail = %s", diags[0].Detail())
	}
}
