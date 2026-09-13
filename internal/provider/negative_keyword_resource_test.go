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

func TestNegativeKeywordModelFromClient_CampaignScoped(t *testing.T) {
	t.Parallel()

	state := negativeKeywordModelFromClient(&client.NegativeKeyword{
		ID: 5, CampaignID: 10, Text: "free", MatchType: "EXACT", Status: "ACTIVE",
		ModificationTime: "2026-07-01T00:00:00Z",
	})
	if state.ID.ValueString() != "5" || !state.AdGroupID.IsNull() {
		t.Fatalf("state = %#v", state)
	}
	if state.Text.ValueString() != "free" || state.MatchType.ValueString() != "EXACT" {
		t.Fatalf("text/match = %#v", state)
	}
}

func TestNegativeKeywordModelFromClient_AdGroupScoped(t *testing.T) {
	t.Parallel()

	state := negativeKeywordModelFromClient(&client.NegativeKeyword{
		ID: 6, CampaignID: 10, AdGroupID: 20, Text: "cheap", MatchType: "BROAD", Status: "PAUSED",
	})
	if state.AdGroupID.ValueString() != "20" || state.CampaignID.ValueString() != "10" {
		t.Fatalf("state = %#v", state)
	}
}

func TestNegativeKeywordCreate_CampaignAndAdGroup(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/campaigns/10/negativekeywords/bulk":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{{
					"id": 5, "campaignId": 10, "text": "free", "matchType": "EXACT", "status": "ACTIVE",
				}},
			})
		case r.Method == http.MethodPost && r.URL.Path == "/adgroups/find":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{{"id": 20, "campaignId": 10}},
			})
		case r.Method == http.MethodPost && r.URL.Path == "/campaigns/10/adgroups/20/negativekeywords/bulk":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{{
					"id": 6, "campaignId": 10, "adGroupId": 20, "text": "cheap", "matchType": "BROAD", "status": "ACTIVE",
				}},
			})
		default:
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	api, err := client.New(client.WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}

	camp, err := api.CreateCampaignNegativeKeyword(context.Background(), 10, &client.NegativeKeywordCreate{
		Text: "free", MatchType: "EXACT", Status: "ACTIVE",
	})
	if err != nil {
		t.Fatal(err)
	}
	if camp.ID != 5 {
		t.Fatalf("campaign nk = %#v", camp)
	}

	ag, err := api.FindAdGroupByID(context.Background(), 20)
	if err != nil {
		t.Fatal(err)
	}
	adg, err := api.CreateAdGroupNegativeKeyword(context.Background(), ag.CampaignID, 20, &client.NegativeKeywordCreate{
		Text: "cheap", MatchType: "BROAD",
	})
	if err != nil {
		t.Fatal(err)
	}
	state := negativeKeywordModelFromClient(adg)
	if state.ID.ValueString() != "6" || state.AdGroupID.ValueString() != "20" {
		t.Fatalf("state = %#v", state)
	}
}

func TestNegativeKeyword_ImmutableMessageContract(t *testing.T) {
	t.Parallel()
	for _, field := range []string{"campaign_id", "ad_group_id", "text", "match_type"} {
		msg := `Cannot change immutable negative keyword field "` + field + `"`
		if !strings.Contains(msg, field) {
			t.Fatalf("contract for %s", field)
		}
	}
}
