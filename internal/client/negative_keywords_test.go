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

func TestCreateCampaignNegativeKeyword(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/campaigns/10/negativekeywords/bulk" {
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
		var body []NegativeKeywordCreate
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if len(body) != 1 || body[0].Text != "free" || body[0].MatchType != "EXACT" {
			t.Fatalf("body = %#v", body)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{{
				"id":         5,
				"campaignId": 10,
				"text":       "free",
				"matchType":  "EXACT",
				"status":     "ACTIVE",
				"deleted":    false,
			}},
		})
	}))
	t.Cleanup(srv.Close)

	c, err := New(WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	out, err := c.CreateCampaignNegativeKeyword(context.Background(), 10, &NegativeKeywordCreate{
		Text: "free", MatchType: "EXACT", Status: "ACTIVE",
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.ID != 5 || out.Text != "free" {
		t.Fatalf("out = %#v", out)
	}
}

func TestCreateAdGroupNegativeKeyword(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/campaigns/10/adgroups/20/negativekeywords/bulk" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{{
				"id":         6,
				"campaignId": 10,
				"adGroupId":  20,
				"text":       "cheap",
				"matchType":  "BROAD",
				"status":     "ACTIVE",
			}},
		})
	}))
	t.Cleanup(srv.Close)

	c, err := New(WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	out, err := c.CreateAdGroupNegativeKeyword(context.Background(), 10, 20, &NegativeKeywordCreate{
		Text: "cheap", MatchType: "BROAD",
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.ID != 6 || out.AdGroupID != 20 {
		t.Fatalf("out = %#v", out)
	}
}

func TestDeleteCampaignNegativeKeyword_Bulk(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/campaigns/10/negativekeywords/delete/bulk" {
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
		var ids []int64
		if err := json.NewDecoder(r.Body).Decode(&ids); err != nil {
			t.Fatal(err)
		}
		if len(ids) != 1 || ids[0] != 5 {
			t.Fatalf("ids = %#v", ids)
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	c, err := New(WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	if err := c.DeleteCampaignNegativeKeyword(context.Background(), 10, 5); err != nil {
		t.Fatal(err)
	}
}

func TestParseNegativeKeywordImportID(t *testing.T) {
	t.Parallel()

	scope, cid, aid, kid, err := ParseNegativeKeywordImportID("campaign/10/5")
	if err != nil || scope != "campaign" || cid != 10 || aid != 0 || kid != 5 {
		t.Fatalf("campaign: %s %d %d %d %v", scope, cid, aid, kid, err)
	}
	scope, cid, aid, kid, err = ParseNegativeKeywordImportID("adgroup/10/20/6")
	if err != nil || scope != "adgroup" || cid != 10 || aid != 20 || kid != 6 {
		t.Fatalf("adgroup3: %s %d %d %d %v", scope, cid, aid, kid, err)
	}
	scope, cid, aid, kid, err = ParseNegativeKeywordImportID("adgroup/20/6")
	if err != nil || scope != "adgroup" || cid != 0 || aid != 20 || kid != 6 {
		t.Fatalf("adgroup2: %s %d %d %d %v", scope, cid, aid, kid, err)
	}
	_, _, _, _, err = ParseNegativeKeywordImportID("nope/1")
	if err == nil || !strings.Contains(err.Error(), "scope must be") {
		t.Fatalf("err = %v", err)
	}
}
