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

func TestCreateBudgetOrder_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/budgetorders" {
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		orgIDs, ok := body["orgIds"].([]any)
		orgID, orgIDOK := float64(0), false
		if ok && len(orgIDs) == 1 {
			orgID, orgIDOK = orgIDs[0].(float64)
		}
		if !ok || !orgIDOK || orgID != 40669820 {
			t.Fatalf("orgIds = %#v", body["orgIds"])
		}
		bo, ok := body["bo"].(map[string]any)
		if !ok {
			t.Fatalf("bo = %#v", body["bo"])
		}
		if bo["name"] != "Q1 LOC order" {
			t.Fatalf("name = %#v", bo["name"])
		}
		budget, ok := bo["budget"].(map[string]any)
		if !ok || budget["amount"] != "300.00" || budget["currency"] != "USD" {
			t.Fatalf("budget = %#v", bo["budget"])
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"orgIds": []int64{40669820},
				"bo": map[string]any{
					"id":        542370539,
					"name":      "Q1 LOC order",
					"startDate": "2026-01-01T00:00:00.000",
					"endDate":   "2026-03-31T23:59:59.999",
					"budget":    map[string]string{"amount": "300.00", "currency": "USD"},
					"status":    "ACTIVE",
				},
			},
		})
	}))
	t.Cleanup(srv.Close)

	c, err := New(WithBaseURL(srv.URL), WithOrgID("40669820"))
	if err != nil {
		t.Fatal(err)
	}
	out, err := c.CreateBudgetOrder(context.Background(), &BudgetOrderCreate{
		Name:      "Q1 LOC order",
		StartDate: "2026-01-01T00:00:00.000",
		EndDate:   "2026-03-31T23:59:59.999",
		Budget:    &Money{Amount: "300.00", Currency: "USD"},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if out.Bo == nil || out.Bo.ID != 542370539 || out.Bo.Budget.Amount != "300.00" {
		t.Fatalf("out = %#v", out)
	}
}

func TestCreateBudgetOrder_MoneyNoFloat(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var raw map[string]any
		if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
			t.Fatal(err)
		}
		bo, ok := raw["bo"].(map[string]any)
		if !ok {
			t.Fatalf("bo = %#v", raw["bo"])
		}
		budget, ok := bo["budget"].(map[string]any)
		if !ok {
			t.Fatalf("budget = %#v", bo["budget"])
		}
		if _, isFloat := budget["amount"].(float64); isFloat {
			t.Fatalf("amount encoded as float: %#v", budget["amount"])
		}
		if budget["amount"] != "99.99" {
			t.Fatalf("amount = %#v", budget["amount"])
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"orgIds": []int64{1},
				"bo": map[string]any{
					"id":     1,
					"name":   "x",
					"budget": map[string]string{"amount": "99.99", "currency": "USD"},
				},
			},
		})
	}))
	t.Cleanup(srv.Close)

	c, err := New(WithBaseURL(srv.URL), WithOrgID("1"))
	if err != nil {
		t.Fatal(err)
	}
	out, err := c.CreateBudgetOrder(context.Background(), &BudgetOrderCreate{
		Name:      "x",
		StartDate: "2026-01-01T00:00:00.000",
		EndDate:   "2026-12-31T23:59:59.999",
		Budget:    &Money{Amount: "99.99", Currency: "USD"},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if out.Bo.Budget.Amount != "99.99" {
		t.Fatalf("amount = %q", out.Bo.Budget.Amount)
	}
}

func TestCreateBudgetOrder_NilPayload(t *testing.T) {
	t.Parallel()

	c, err := New(WithBaseURL("http://example.invalid"), WithOrgID("1"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.CreateBudgetOrder(context.Background(), nil, nil)
	if err == nil || !strings.Contains(err.Error(), "required") {
		t.Fatalf("err = %v", err)
	}
}

func TestCreateBudgetOrder_MissingOrgID(t *testing.T) {
	t.Parallel()

	c, err := New(WithBaseURL("http://example.invalid"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.CreateBudgetOrder(context.Background(), &BudgetOrderCreate{
		Name: "x", StartDate: "a", EndDate: "b", Budget: &Money{Amount: "1", Currency: "USD"},
	}, nil)
	if err == nil || !strings.Contains(err.Error(), "org id") {
		t.Fatalf("err = %v", err)
	}
}

func TestCreateBudgetOrder_APIError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"errors": []map[string]string{{
					"messageCode": "INVALID_NAME",
					"message":     "name is required",
					"field":       "bo.name",
				}},
			},
		})
	}))
	t.Cleanup(srv.Close)

	c, err := New(WithBaseURL(srv.URL), WithOrgID("1"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.CreateBudgetOrder(context.Background(), &BudgetOrderCreate{
		Name: "x", StartDate: "a", EndDate: "b", Budget: &Money{Amount: "1", Currency: "USD"},
	}, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("err type = %T (%v)", err, err)
	}
	if apiErr.StatusCode != http.StatusBadRequest || apiErr.Code != "INVALID_NAME" {
		t.Fatalf("apiErr = %#v", apiErr)
	}
}

func TestGetBudgetOrder_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/budgetorders/542370539" {
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"orgIds": []int64{3761812},
				"bo": map[string]any{
					"id":     542370539,
					"name":   "get example",
					"status": "COMPLETED",
					"budget": map[string]string{"amount": "2000", "currency": "USD"},
				},
			},
		})
	}))
	t.Cleanup(srv.Close)

	c, err := New(WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	out, err := c.GetBudgetOrder(context.Background(), 542370539)
	if err != nil {
		t.Fatal(err)
	}
	if out.Bo.ID != 542370539 || out.Bo.Status != "COMPLETED" {
		t.Fatalf("out = %#v", out)
	}
}

func TestGetBudgetOrder_NotFound(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"errors": []map[string]string{{"messageCode": "NOT_FOUND", "message": "missing"}},
			},
		})
	}))
	t.Cleanup(srv.Close)

	c, err := New(WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.GetBudgetOrder(context.Background(), 1)
	if !IsNotFound(err) {
		t.Fatalf("err = %v", err)
	}
}

func TestListBudgetOrders_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/budgetorders" {
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
		if r.URL.Query().Get("limit") != "1000" {
			t.Fatalf("query = %s", r.URL.RawQuery)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{{
				"orgIds": []int64{1},
				"bo": map[string]any{
					"id":   10,
					"name": "a",
				},
			}, {
				"orgIds": []int64{1},
				"bo": map[string]any{
					"id":   11,
					"name": "b",
				},
			}},
			"pagination": map[string]int{
				"totalResults": 2,
				"startIndex":   0,
				"itemsPerPage": 1000,
			},
		})
	}))
	t.Cleanup(srv.Close)

	c, err := New(WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	out, err := c.ListBudgetOrders(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 2 || out[0].Bo.ID != 10 || out[1].Bo.Name != "b" {
		t.Fatalf("out = %#v", out)
	}
}

func TestUpdateBudgetOrder_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/budgetorders/99" {
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		bo, ok := body["bo"].(map[string]any)
		if !ok {
			t.Fatalf("bo = %#v", body["bo"])
		}
		budget, ok := bo["budget"].(map[string]any)
		if !ok {
			t.Fatalf("budget = %#v", bo["budget"])
		}
		if bo["name"] != "updated" || budget["amount"] != "400.50" {
			t.Fatalf("body = %#v", body)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"orgIds": []int64{1},
				"bo": map[string]any{
					"id":     99,
					"name":   "updated",
					"budget": map[string]string{"amount": "400.50", "currency": "USD"},
					"status": "ACTIVE",
				},
			},
		})
	}))
	t.Cleanup(srv.Close)

	c, err := New(WithBaseURL(srv.URL), WithOrgID("1"))
	if err != nil {
		t.Fatal(err)
	}
	out, err := c.UpdateBudgetOrder(context.Background(), 99, &BudgetOrderUpdate{
		Name:   "updated",
		Budget: &Money{Amount: "400.50", Currency: "USD"},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if out.Bo.Name != "updated" || out.Bo.Budget.Amount != "400.50" {
		t.Fatalf("out = %#v", out)
	}
}

func TestUpdateBudgetOrder_NilPayload(t *testing.T) {
	t.Parallel()

	c, err := New(WithBaseURL("http://example.invalid"), WithOrgID("1"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.UpdateBudgetOrder(context.Background(), 1, nil, nil)
	if err == nil || !strings.Contains(err.Error(), "required") {
		t.Fatalf("err = %v", err)
	}
}

func TestParseBudgetOrderID(t *testing.T) {
	t.Parallel()

	id, err := ParseBudgetOrderID("542370539")
	if err != nil || id != 542370539 {
		t.Fatalf("got %d %v", id, err)
	}
	id, err = ParseBudgetOrderID("  42  ")
	if err != nil || id != 42 {
		t.Fatalf("got %d %v", id, err)
	}
	_, err = ParseBudgetOrderID("not-a-number")
	if err == nil {
		t.Fatal("expected error")
	}
	_, err = ParseBudgetOrderID("")
	if err == nil {
		t.Fatal("expected error")
	}
}
