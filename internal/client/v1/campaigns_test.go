// SPDX-License-Identifier: MIT

package v1_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/picubedllc/terraform-provider-appleads/internal/client/v1"
)

func TestCreateCampaign_AdAccountIDInBody(t *testing.T) {
	t.Parallel()

	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/campaigns" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		b, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(b, &got); err != nil {
			t.Errorf("decode body: %v", err)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"result": map[string]any{
				"id":                 77,
				"adAccountId":        4242,
				"name":               "demo",
				"promotedObjectId":   "123456",
				"promotedObjectType": "APPSTORE_APP",
				"dailyBudget":        map[string]any{"value": map[string]any{"amount": "10.00", "currency": "USD"}},
				"bidStrategy":        map[string]any{"bidStrategyType": "MAX_CONVERSIONS", "bid": map[string]any{"amount": "2.50", "currency": "USD"}},
				"targeting": map[string]any{
					"countryOrRegion": map[string]any{"include": []string{"US", "CA"}},
					"supplyPlacement": map[string]any{"include": []string{"APPSTORE_SEARCH_RESULTS"}},
				},
				"systemStatus": "RUNNING",
				"deleted":      false,
			},
		})
	}))
	t.Cleanup(srv.Close)

	c, err := v1.New(v1.WithBaseURL(srv.URL), v1.WithAdAccountID("4242"))
	if err != nil {
		t.Fatal(err)
	}
	camp, err := c.CreateCampaign(context.Background(), &v1.CampaignCreate{
		Name:               "demo",
		PromotedObjectID:   "123456",
		PromotedObjectType: v1.PromotedObjectTypeAppStoreApp,
		DailyBudget:        &v1.MoneyValue{Value: &v1.Money{Amount: "10.00", Currency: "USD"}},
		BidStrategy:        &v1.BidStrategy{BidStrategyType: "MAX_CONVERSIONS", Bid: &v1.Money{Amount: "2.50", Currency: "USD"}},
		Targeting: &v1.CampaignTargeting{
			CountryOrRegion: &v1.IncludeList{Include: []string{"US", "CA"}},
			SupplyPlacement: &v1.IncludeList{Include: []string{"APPSTORE_SEARCH_RESULTS"}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if v, ok := got["adAccountId"].(float64); !ok || int64(v) != 4242 {
		t.Fatalf("adAccountId in body = %#v", got["adAccountId"])
	}
	if got["promotedObjectType"] != "APPSTORE_APP" || got["promotedObjectId"] != "123456" {
		t.Fatalf("promoted object in body = %#v", got)
	}
	budget := got["dailyBudget"].(map[string]any)["value"].(map[string]any)
	if budget["amount"] != "10.00" || budget["currency"] != "USD" {
		t.Fatalf("dailyBudget money must be a decimal string: %#v", budget)
	}
	tgt := got["targeting"].(map[string]any)
	cor := tgt["countryOrRegion"].(map[string]any)["include"].([]any)
	sp := tgt["supplyPlacement"].(map[string]any)["include"].([]any)
	if len(cor) != 2 || len(sp) != 1 {
		t.Fatalf("targeting = %#v", tgt)
	}

	if camp.ID != 77 || camp.AdAccountID != 4242 || camp.SystemStatus != "RUNNING" || camp.Deleted {
		t.Fatalf("campaign = %#v", camp)
	}
	if camp.BidStrategy == nil || camp.BidStrategy.Bid == nil || camp.BidStrategy.Bid.Amount != "2.50" {
		t.Fatalf("bidStrategy = %#v", camp.BidStrategy)
	}
}

func TestCreateCampaign_ExplicitAdAccountIDOverridesClient(t *testing.T) {
	t.Parallel()

	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &got)
		_ = json.NewEncoder(w).Encode(map[string]any{"result": map[string]any{"id": 1}})
	}))
	t.Cleanup(srv.Close)

	c, _ := v1.New(v1.WithBaseURL(srv.URL), v1.WithAdAccountID("1"))
	if _, err := c.CreateCampaign(context.Background(), &v1.CampaignCreate{AdAccountID: 9, Name: "x", PromotedObjectID: "1", PromotedObjectType: v1.PromotedObjectTypeAppStoreApp}); err != nil {
		t.Fatal(err)
	}
	if got["adAccountId"].(float64) != 9 {
		t.Fatalf("adAccountId = %v", got["adAccountId"])
	}
}

func TestGetCampaign(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/campaigns/55" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"result": map[string]any{
				"id":                          55,
				"adAccountId":                 4242,
				"name":                        "get-me",
				"deleted":                     true,
				"systemStatus":                "NOT_RUNNING",
				"systemStatusLimitingReasons": []string{"BUDGET_EXHAUSTED"},
				"dailyBudget":                 map[string]any{"value": map[string]any{"amount": "1.00", "currency": "USD"}},
			},
		})
	}))
	t.Cleanup(srv.Close)

	c, _ := v1.New(v1.WithBaseURL(srv.URL))
	camp, err := c.GetCampaign(context.Background(), 55)
	if err != nil {
		t.Fatal(err)
	}
	if camp.ID != 55 || camp.Name != "get-me" || !camp.Deleted || camp.SystemStatus != "NOT_RUNNING" {
		t.Fatalf("campaign = %#v", camp)
	}
	if len(camp.SystemStatusLimitingReasons) != 1 {
		t.Fatalf("reasons = %#v", camp.SystemStatusLimitingReasons)
	}
	if camp.DailyBudget == nil || camp.DailyBudget.Value == nil || camp.DailyBudget.Value.Amount != "1.00" {
		t.Fatalf("dailyBudget = %#v", camp.DailyBudget)
	}
}

func TestUpdateCampaign_PartialBody(t *testing.T) {
	t.Parallel()

	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/campaigns/5" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &got)
		_ = json.NewEncoder(w).Encode(map[string]any{"result": map[string]any{"id": 5, "name": "renamed"}})
	}))
	t.Cleanup(srv.Close)

	c, _ := v1.New(v1.WithBaseURL(srv.URL))
	camp, err := c.UpdateCampaign(context.Background(), 5, &v1.CampaignUpdate{Name: "renamed"})
	if err != nil {
		t.Fatal(err)
	}
	if camp.Name != "renamed" {
		t.Fatalf("campaign = %#v", camp)
	}
	if len(got) != 1 || got["name"] != "renamed" {
		t.Fatalf("body should be bare and partial, got %#v", got)
	}
}

func TestDeleteCampaign(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/campaigns/8" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(srv.Close)

	c, _ := v1.New(v1.WithBaseURL(srv.URL))
	if err := c.DeleteCampaign(context.Background(), 8); err != nil {
		t.Fatal(err)
	}
}

func TestQueryCampaigns(t *testing.T) {
	t.Parallel()

	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/campaigns/query" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &got)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"result": []map[string]any{
				{"id": 1, "name": "a"},
				{"id": 2, "name": "b"},
			},
			"pagination": map[string]any{"offset": 0, "pageSize": 50, "totalCount": 2},
		})
	}))
	t.Cleanup(srv.Close)

	c, _ := v1.New(v1.WithBaseURL(srv.URL))
	page, err := c.QueryCampaigns(context.Background(), v1.QueryRequest{
		Filters:    []v1.QueryFilter{v1.EqualsFilter("adAccountId", 4242)},
		Pagination: &v1.QueryPagination{Offset: 0, PageSize: 50},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Data) != 2 || page.Data[1].ID != 2 || page.TotalCount != 2 || page.PageSize != 50 {
		t.Fatalf("page = %#v", page)
	}
	filters := got["filters"].([]any)
	if len(filters) != 1 || filters[0].(map[string]any)["field"] != "adAccountId" {
		t.Fatalf("filters = %#v", got["filters"])
	}
}
