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

func TestCreateCreative_CustomProductPage(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/creatives" {
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
		var body CreativeCreate
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.AdamID != 899247664 || body.Type != CreativeTypeCustomProductPage {
			t.Fatalf("body = %#v", body)
		}
		if body.ProductPageID != "76659d7a-d146-43d3-b6b8-b7a12f74bf6b" {
			t.Fatalf("productPageId = %q", body.ProductPageID)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"id":            573408745,
				"orgId":         39872140,
				"adamId":        body.AdamID,
				"name":          body.Name,
				"type":          body.Type,
				"state":         "VALID",
				"stateReasons":  []string{},
				"productPageId": body.ProductPageID,
			},
		})
	}))
	t.Cleanup(srv.Close)

	c, err := New(WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	out, err := c.CreateCreative(context.Background(), &CreativeCreate{
		AdamID:        899247664,
		Name:          "Trip Trek custom product page variation 1",
		Type:          CreativeTypeCustomProductPage,
		ProductPageID: "76659d7a-d146-43d3-b6b8-b7a12f74bf6b",
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.ID != 573408745 || out.ProductPageID == "" || out.State != "VALID" {
		t.Fatalf("out = %#v", out)
	}
}

func TestCreateCreative_RequiresProductPageID(t *testing.T) {
	t.Parallel()

	c, err := New(WithBaseURL("http://example.invalid"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.CreateCreative(context.Background(), &CreativeCreate{
		AdamID: 1,
		Name:   "x",
		Type:   CreativeTypeCustomProductPage,
	})
	if err == nil || !strings.Contains(err.Error(), "productPageId") {
		t.Fatalf("err = %v", err)
	}
}

func TestCreateCreative_DefaultProductPage(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body CreativeCreate
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.Type != CreativeTypeDefaultProductPage || body.ProductPageID != "" {
			t.Fatalf("body = %#v", body)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"id":     11,
				"adamId": body.AdamID,
				"name":   body.Name,
				"type":   body.Type,
				"state":  "VALID",
			},
		})
	}))
	t.Cleanup(srv.Close)

	c, err := New(WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	out, err := c.CreateCreative(context.Background(), &CreativeCreate{
		AdamID: 100, Name: "Default PP", Type: CreativeTypeDefaultProductPage,
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.ID != 11 {
		t.Fatalf("out = %#v", out)
	}
}

func TestGetCreative_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/creatives/42" {
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{"id": 42, "name": "n", "type": "CUSTOM_PRODUCT_PAGE", "state": "VALID"},
		})
	}))
	t.Cleanup(srv.Close)

	c, err := New(WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	out, err := c.GetCreative(context.Background(), 42)
	if err != nil {
		t.Fatal(err)
	}
	if out.ID != 42 || out.Name != "n" {
		t.Fatalf("out = %#v", out)
	}
}

func TestGetCreative_NotFound(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"errors": []map[string]string{{
					"messageCode": "NOT_FOUND",
					"message":     "creative not found",
				}},
			},
		})
	}))
	t.Cleanup(srv.Close)

	c, err := New(WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.GetCreative(context.Background(), 99)
	if !IsNotFound(err) {
		t.Fatalf("err = %v", err)
	}
}

func TestFindCreatives_Selector(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/creatives/find" {
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
		var body Selector
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if len(body.Conditions) != 1 || body.Conditions[0].Field != "name" {
			t.Fatalf("body = %#v", body)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{{"id": 1, "name": "Trip"}},
			"pagination": map[string]int{
				"totalResults": 1, "startIndex": 0, "itemsPerPage": 1,
			},
		})
	}))
	t.Cleanup(srv.Close)

	c, err := New(WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	out, err := c.FindCreatives(context.Background(), Selector{
		Conditions: []SelectorCondition{{
			Field: "name", Operator: "CONTAINS", Values: []string{"Trip"},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out[0].ID != 1 {
		t.Fatalf("out = %#v", out)
	}
}

func TestFindCreativeByID_Empty(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"data": []any{}})
	}))
	t.Cleanup(srv.Close)

	c, err := New(WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.FindCreativeByID(context.Background(), 7)
	if !IsNotFound(err) {
		t.Fatalf("err = %v", err)
	}
}

func TestListCreatives_Paginated(t *testing.T) {
	t.Parallel()

	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		offset := r.URL.Query().Get("offset")
		switch offset {
		case "", "0":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{{"id": 1}},
				"pagination": map[string]int{
					"totalResults": 2, "startIndex": 0, "itemsPerPage": 1,
				},
			})
		case "1":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{{"id": 2}},
				"pagination": map[string]int{
					"totalResults": 2, "startIndex": 1, "itemsPerPage": 1,
				},
			})
		default:
			t.Fatalf("unexpected offset %q", offset)
		}
	}))
	t.Cleanup(srv.Close)

	c, err := New(WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	// Override page size via direct FetchAllPages path used by ListCreatives (1000).
	// Exercise path with Find instead isn't needed; ListCreatives uses 1000 so
	// simulate multi-page via FindAllPages manually with page size 1.
	items, err := FetchAllPages[Creative](context.Background(), c, http.MethodGet, "creatives", 1, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 || items[0].ID != 1 || items[1].ID != 2 {
		t.Fatalf("items = %#v", items)
	}
	if calls < 2 {
		t.Fatalf("calls = %d", calls)
	}
}

func TestParseCreativeID(t *testing.T) {
	t.Parallel()

	id, err := ParseCreativeID("573408745")
	if err != nil || id != 573408745 {
		t.Fatalf("id=%d err=%v", id, err)
	}
	if _, err := ParseCreativeID(""); err == nil {
		t.Fatal("expected error")
	}
	if _, err := ParseCreativeID("0"); err == nil {
		t.Fatal("expected error")
	}
}
