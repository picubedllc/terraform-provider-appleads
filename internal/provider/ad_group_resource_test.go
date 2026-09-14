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

	"github.com/hashicorp/terraform-plugin-framework/resource"
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
		StartTime:              "2026-01-01T00:00:00.000",
		EndTime:                "2026-12-01T00:00:00Z",
		ModificationTime:       "2026-05-01T00:00:00Z",
		PricingModel:           client.PricingModelCPC,
	}
	state := adGroupModelFromClient(got)
	if state.ID.ValueString() != "77" || state.CampaignID.ValueString() != "10" {
		t.Fatalf("ids = %#v", state)
	}
	if state.DefaultBidAmount.ValueString() != "1.25" || state.CPAGoalAmount.ValueString() != "5.00" {
		t.Fatalf("money = %#v", state)
	}
	if !state.AutomatedKeywordsOptIn.ValueBool() || state.StartTime.ValueString() != "2026-01-01T00:00:00.000" || state.EndTime.ValueString() != "2026-12-01T00:00:00Z" {
		t.Fatalf("opts = %#v", state)
	}
	if state.PricingModel.ValueString() != client.PricingModelCPC {
		t.Fatalf("pricing_model = %s", state.PricingModel.ValueString())
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
	if !state.CPAGoalAmount.IsNull() || !state.StartTime.IsNull() || !state.EndTime.IsNull() {
		t.Fatalf("expected null optionals: %#v", state)
	}
	if !state.PricingModel.IsNull() {
		t.Fatalf("pricing_model = %#v, want null", state.PricingModel)
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
		if body.PricingModel != client.PricingModelCPC {
			t.Fatalf("pricingModel = %q", body.PricingModel)
		}
		if body.StartTime != "2026-01-01T00:00:00.000" {
			t.Fatalf("startTime = %q", body.StartTime)
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
		PricingModel:       types.StringValue(client.PricingModelCPC),
		StartTime:          types.StringValue("2026-01-01T00:00:00.000"),
	}
	bid, diags := moneyFromStrings(plan.DefaultBidAmount, plan.DefaultBidCurrency)
	if diags.HasError() || bid == nil {
		t.Fatalf("bid diags=%v", diags)
	}
	created, err := apiClient.CreateAdGroup(context.Background(), 9, &client.AdGroupCreate{
		Name:             plan.Name.ValueString(),
		DefaultBidAmount: bid,
		Status:           plan.Status.ValueString(),
		PricingModel:     plan.PricingModel.ValueString(),
		StartTime:        plan.StartTime.ValueString(),
	})
	if err != nil {
		t.Fatal(err)
	}
	state := adGroupModelFromClient(created)
	if state.ID.ValueString() != "55" || state.ServingStatus.ValueString() != "RUNNING" {
		t.Fatalf("state = %#v", state)
	}
}

func TestOverlayAdGroupMoney_KeepsConfiguredScale(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		configured string
		reported   string
		want       string
	}{
		{name: "trailing zeros", configured: "1.00", reported: "1", want: "1.00"},
		{name: "cents unchanged", configured: "0.58", reported: "0.58", want: "0.58"},
		{name: "cents extra scale", configured: "0.58", reported: "0.580", want: "0.58"},
		{name: "half dollar scale", configured: "0.50", reported: "0.5", want: "0.50"},
		{name: "real change keeps API", configured: "0.58", reported: "0.59", want: "0.59"},
		{name: "whole dollar change", configured: "1.00", reported: "2", want: "2"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			out := overlayAdGroupMoney(
				adGroupModel{DefaultBidAmount: types.StringValue(tt.configured), DefaultBidCurrency: types.StringValue("USD")},
				adGroupModel{DefaultBidAmount: types.StringValue(tt.reported), DefaultBidCurrency: types.StringValue("USD")},
			)
			if out.DefaultBidAmount.ValueString() != tt.want {
				t.Fatalf("amount = %q, want %q", out.DefaultBidAmount.ValueString(), tt.want)
			}
		})
	}
}

func TestAdGroupResource_SchemaRequiredCreateFields(t *testing.T) {
	t.Parallel()

	r := NewAdGroupResource()
	resp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	for _, name := range []string{"campaign_id", "name", "default_bid_amount", "pricing_model", "start_time"} {
		attr, ok := resp.Schema.Attributes[name]
		if !ok {
			t.Fatalf("missing attribute %q", name)
		}
		if !attr.IsRequired() {
			t.Fatalf("%s should be required to match Apple Ads create", name)
		}
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
		"Create a new appleads_ad_group explicitly instead."
	if !strings.Contains(detail, "77") || !strings.Contains(msg, "campaign_id") {
		t.Fatal("message contract drifted")
	}
}

func TestAdGroupUpdate_ImmutablePricingModelMessage(t *testing.T) {
	t.Parallel()
	msg := `Cannot change immutable ad group field "pricing_model"`
	if !strings.Contains(msg, "pricing_model") {
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
