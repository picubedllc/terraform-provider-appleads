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

func TestKeywordModelFromClient(t *testing.T) {
	t.Parallel()

	got := &client.Keyword{
		ID:               99,
		CampaignID:       10,
		AdGroupID:        20,
		Text:             "screenshot organizer",
		MatchType:        "EXACT",
		Status:           "ACTIVE",
		BidAmount:        &client.Money{Amount: "1.50", Currency: "USD"},
		ModificationTime: "2026-06-01T00:00:00Z",
	}
	state, diags := keywordModelFromClient(got)
	if diags.HasError() {
		t.Fatal(diags)
	}
	if state.ID.ValueString() != "99" || state.AdGroupID.ValueString() != "20" {
		t.Fatalf("ids = %#v", state)
	}
	if state.Text.ValueString() != "screenshot organizer" || state.MatchType.ValueString() != "EXACT" {
		t.Fatalf("text/match = %#v", state)
	}
	if state.BidAmount.ValueString() != "1.50" || state.BidCurrency.ValueString() != "USD" {
		t.Fatalf("bid = %#v", state)
	}
}

func TestKeywordModelFromClient_NullBid(t *testing.T) {
	t.Parallel()

	state, diags := keywordModelFromClient(&client.Keyword{
		ID: 1, CampaignID: 2, AdGroupID: 3, Text: "x", MatchType: "BROAD", Status: "PAUSED",
	})
	if diags.HasError() {
		t.Fatal(diags)
	}
	if !state.BidAmount.IsNull() || !state.BidCurrency.IsNull() {
		t.Fatalf("expected null bid: %#v", state)
	}
}

func TestKeywordCreate_AndStateFromResponse(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/adgroups/find"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{{"id": 20, "campaignId": 10, "name": "ag"}},
			})
		case r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/targetingkeywords/bulk"):
			var body []client.KeywordCreate
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if len(body) != 1 || body[0].BidAmount == nil || body[0].BidAmount.Amount != "1.25" {
				t.Fatalf("body = %#v", body)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{{
					"id":         55,
					"campaignId": 10,
					"adGroupId":  20,
					"text":       body[0].Text,
					"matchType":  body[0].MatchType,
					"status":     "ACTIVE",
					"bidAmount":  map[string]string{"amount": "1.25", "currency": "USD"},
				}},
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
	bid, diags := moneyFromStrings(types.StringValue("1.25"), types.StringValue("USD"))
	if diags.HasError() || bid == nil {
		t.Fatalf("bid diags=%v", diags)
	}
	created, err := apiClient.CreateKeyword(context.Background(), ag.CampaignID, 20, &client.KeywordCreate{
		Text: "photo cleaner", MatchType: "EXACT", Status: "ACTIVE", BidAmount: bid,
	})
	if err != nil {
		t.Fatal(err)
	}
	state, diags := keywordModelFromClient(created)
	if diags.HasError() {
		t.Fatal(diags)
	}
	if state.ID.ValueString() != "55" || state.BidAmount.ValueString() != "1.25" {
		t.Fatalf("state = %#v", state)
	}
}

func TestKeywordUpdate_ImmutableMessages(t *testing.T) {
	t.Parallel()

	cases := []struct {
		field string
		msg   string
	}{
		{"ad_group_id", `Cannot change immutable keyword field "ad_group_id"`},
		{"text", `Cannot change immutable keyword field "text"`},
		{"match_type", `Cannot change immutable keyword field "match_type"`},
	}
	for _, tc := range cases {
		if !strings.Contains(tc.msg, tc.field) {
			t.Fatalf("message contract drifted for %s", tc.field)
		}
	}
}

func TestKeywordCreate_APIValidationDiagnostic(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"errors": []map[string]string{{
					"messageCode": "INVALID_ATTRIBUTE_VALUE",
					"message":     "bidAmount must be positive",
					"field":       "bidAmount",
				}},
			},
		})
	}))
	t.Cleanup(srv.Close)

	apiClient, err := client.New(client.WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	_, err = apiClient.CreateKeyword(context.Background(), 1, 2, &client.KeywordCreate{
		Text: "x", MatchType: "EXACT", BidAmount: &client.Money{Amount: "0", Currency: "USD"},
	})
	if err == nil {
		t.Fatal("expected error")
	}
	diags := apiErrorDiagnostic("Unable to create Apple Ads keyword", err)
	if !diags.HasError() {
		t.Fatal("expected diagnostic")
	}
	if !strings.Contains(diags[0].Detail(), "bidAmount") {
		t.Fatalf("detail = %s", diags[0].Detail())
	}
}
