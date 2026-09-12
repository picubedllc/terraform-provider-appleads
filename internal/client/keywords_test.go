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

func TestCreateKeyword_BulkSuccess(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/campaigns/10/adgroups/20/targetingkeywords/bulk" {
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
		var body []KeywordCreate
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if len(body) != 1 || body[0].Text != "screenshot organizer" || body[0].MatchType != "EXACT" {
			t.Fatalf("body = %#v", body)
		}
		if body[0].BidAmount == nil || body[0].BidAmount.Amount != "1.50" {
			t.Fatalf("bid = %#v", body[0].BidAmount)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{{
				"id":         99,
				"campaignId": 10,
				"adGroupId":  20,
				"text":       body[0].Text,
				"matchType":  body[0].MatchType,
				"status":     "ACTIVE",
				"bidAmount":  map[string]string{"amount": "1.50", "currency": "USD"},
				"deleted":    false,
			}},
		})
	}))
	t.Cleanup(srv.Close)

	c, err := New(WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	out, err := c.CreateKeyword(context.Background(), 10, 20, &KeywordCreate{
		Text:      "screenshot organizer",
		MatchType: "EXACT",
		Status:    "ACTIVE",
		BidAmount: &Money{Amount: "1.50", Currency: "USD"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.ID != 99 || out.Text != "screenshot organizer" || out.BidAmount.Amount != "1.50" {
		t.Fatalf("out = %#v", out)
	}
}

func TestCreateKeyword_MoneyNoFloat(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var raw []map[string]any
		if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
			t.Fatal(err)
		}
		bid := raw[0]["bidAmount"].(map[string]any)
		if bid["amount"] != "0.99" {
			t.Fatalf("amount = %#v", bid["amount"])
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{{
				"id":        1,
				"text":      "x",
				"matchType": "BROAD",
				"bidAmount": map[string]string{"amount": "0.99", "currency": "USD"},
			}},
		})
	}))
	t.Cleanup(srv.Close)

	c, err := New(WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	out, err := c.CreateKeyword(context.Background(), 1, 2, &KeywordCreate{
		Text: "x", MatchType: "BROAD", BidAmount: &Money{Amount: "0.99", Currency: "USD"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.BidAmount.Amount != "0.99" {
		t.Fatalf("amount = %q", out.BidAmount.Amount)
	}
}

func TestUpdateKeyword_Bulk(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Fatalf("method = %s", r.Method)
		}
		var body []KeywordUpdate
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if len(body) != 1 || body[0].ID != 99 || body[0].Status != "PAUSED" {
			t.Fatalf("body = %#v", body)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{{
				"id":        99,
				"status":    "PAUSED",
				"bidAmount": map[string]string{"amount": "2.00", "currency": "USD"},
				"text":      "screenshot organizer",
				"matchType": "EXACT",
			}},
		})
	}))
	t.Cleanup(srv.Close)

	c, err := New(WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	out, err := c.UpdateKeyword(context.Background(), 10, 20, &KeywordUpdate{
		ID: 99, Status: "PAUSED", BidAmount: &Money{Amount: "2.00", Currency: "USD"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.Status != "PAUSED" || out.BidAmount.Amount != "2.00" {
		t.Fatalf("out = %#v", out)
	}
}

func TestGetKeyword_SoftDeleted(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{"id": 5, "deleted": true, "text": "gone"},
		})
	}))
	t.Cleanup(srv.Close)

	c, err := New(WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	got, err := c.GetKeyword(context.Background(), 1, 2, 5)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Deleted {
		t.Fatal("expected deleted")
	}
}

func TestParseKeywordImportID(t *testing.T) {
	t.Parallel()

	cid, aid, kid, err := ParseKeywordImportID("20/99")
	if err != nil || cid != 0 || aid != 20 || kid != 99 {
		t.Fatalf("2-part: %d %d %d %v", cid, aid, kid, err)
	}
	cid, aid, kid, err = ParseKeywordImportID("10/20/99")
	if err != nil || cid != 10 || aid != 20 || kid != 99 {
		t.Fatalf("3-part: %d %d %d %v", cid, aid, kid, err)
	}
	_, _, _, err = ParseKeywordImportID("only")
	if err == nil || !strings.Contains(err.Error(), "invalid keyword import id") {
		t.Fatalf("err = %v", err)
	}
}
