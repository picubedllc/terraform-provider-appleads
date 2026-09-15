// Copyright (c) 2026 Pi Cubed LLC
// SPDX-License-Identifier: MIT

package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetAdGroupKeywordReport_BidRecommendation(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/reports/campaigns/10/adgroups/20/keywords" {
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
		var body ReportingRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.StartTime != "2026-09-01" || body.EndTime != "2026-09-14" || body.TimeZone != "UTC" {
			t.Fatalf("body = %#v", body)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"reportingDataResponse": map[string]any{
					"row": []map[string]any{{
						"other": false,
						"total": map[string]any{
							"impressions":    12,
							"taps":           3,
							"ttr":            0.25,
							"localSpend":     map[string]string{"amount": "0.09", "currency": "USD"},
							"avgCPT":         map[string]string{"amount": "0.03", "currency": "USD"},
							"totalInstalls":  1,
							"conversionRate": 0.3333,
							"totalAvgCPI":    map[string]string{"amount": "0.09", "currency": "USD"},
						},
						"metadata": map[string]any{
							"keywordId":     99,
							"keyword":       "party games",
							"keywordStatus": "PAUSED",
							"matchType":     "EXACT",
							"bidAmount":     map[string]string{"amount": "0.05", "currency": "USD"},
							"adGroupId":     20,
							"adGroupName":   "Generic Search",
						},
						"insights": map[string]any{
							"bidRecommendation": map[string]any{
								"suggestedBidAmount": map[string]string{"amount": "1.50", "currency": "USD"},
								"bidMin":             map[string]string{"amount": "0.80", "currency": "USD"},
								"bidMax":             map[string]string{"amount": "2.40", "currency": "USD"},
							},
						},
					}},
				},
			},
		})
	}))
	t.Cleanup(srv.Close)

	c, err := New(WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	rows, err := c.GetAdGroupKeywordReport(context.Background(), 10, 20, DefaultReportingRequest("2026-09-01", "2026-09-14", "UTC"))
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("rows = %#v", rows)
	}
	row := rows[0]
	if row.Metadata.Keyword != "party games" || row.Metadata.KeywordID != 99 {
		t.Fatalf("metadata = %#v", row.Metadata)
	}
	if row.Metadata.BidAmount == nil || row.Metadata.BidAmount.Amount != "0.05" {
		t.Fatalf("bid = %#v", row.Metadata.BidAmount)
	}
	if row.Insights == nil || row.Insights.BidRecommendation == nil {
		t.Fatal("expected bid recommendation")
	}
	rec := row.Insights.BidRecommendation
	if rec.SuggestedBidAmount.Amount != "1.50" || rec.BidMin.Amount != "0.80" || rec.BidMax.Amount != "2.40" {
		t.Fatalf("rec = %#v", rec)
	}
	if rec.SuggestedBidAmount.Amount == row.Metadata.BidAmount.Amount {
		t.Fatal("suggested bid must not be treated as the configured bid")
	}
}

func TestGetCampaignReport(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/reports/campaigns" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"reportingDataResponse": map[string]any{
					"row": []map[string]any{{
						"metadata": map[string]any{"campaignId": 7, "campaignName": "Generic Search"},
						"total":    map[string]any{"impressions": 4, "taps": 1},
					}},
				},
			},
		})
	}))
	t.Cleanup(srv.Close)

	c, err := New(WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	rows, err := c.GetCampaignReport(context.Background(), DefaultReportingRequest("2026-09-01", "2026-09-14", "UTC"))
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].Metadata.CampaignName != "Generic Search" || rows[0].Total.Impressions != 4 {
		t.Fatalf("rows = %#v", rows)
	}
}

func TestGetAdGroupSearchTermReport(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/reports/campaigns/10/adgroups/20/searchterms" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		var body ReportingRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.TimeZone != "ORTZ" {
			t.Fatalf("timeZone = %q", body.TimeZone)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"reportingDataResponse": map[string]any{
					"row": []map[string]any{{
						"metadata": map[string]any{
							"searchTermText": "word party game",
							"keywordId":      5,
							"keyword":        "word game",
							"matchType":      "BROAD",
						},
						"total": map[string]any{"impressions": 11, "taps": 2, "totalInstalls": 1},
					}},
				},
			},
		})
	}))
	t.Cleanup(srv.Close)

	c, err := New(WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	req := DefaultReportingRequest("2026-09-01", "2026-09-14", "UTC")
	rows, err := c.GetAdGroupSearchTermReport(context.Background(), 10, 20, req)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].Metadata.SearchTermText != "word party game" {
		t.Fatalf("rows = %#v", rows)
	}
}

func TestGetReport_RequiresDates(t *testing.T) {
	t.Parallel()
	c, err := New(WithBaseURL("http://127.0.0.1:1"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.GetCampaignReport(context.Background(), &ReportingRequest{})
	if err == nil {
		t.Fatal("expected error")
	}
}
