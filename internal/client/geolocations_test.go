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

func TestSearchGeoLocations_QueryAndFilters(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/search/geo" {
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
		q := r.URL.Query()
		if q.Get("query") != "New York" {
			t.Fatalf("query = %q", q.Get("query"))
		}
		if q.Get("entity") != GeoEntityLocality {
			t.Fatalf("entity = %q", q.Get("entity"))
		}
		if q.Get("countrycode") != "US" {
			t.Fatalf("countrycode = %q", q.Get("countrycode"))
		}
		if q.Get("limit") != "20" {
			t.Fatalf("limit = %q", q.Get("limit"))
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{"id": "US|NY|New York", "entity": GeoEntityLocality, "displayName": "New York, New York, United States"},
				{"id": "US|NY", "entity": GeoEntityAdminArea, "displayName": "New York, United States"},
			},
			"pagination": map[string]int{"totalResults": 2, "startIndex": 0, "itemsPerPage": 20},
		})
	}))
	t.Cleanup(srv.Close)

	c, err := New(WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	got, err := c.SearchGeoLocations(context.Background(), GeoSearchParams{
		Query:       "New York",
		Entity:      GeoEntityLocality,
		CountryCode: "US",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d", len(got))
	}
	if got[0].ID != "US|NY|New York" || got[0].Entity != GeoEntityLocality {
		t.Fatalf("got[0] = %#v", got[0])
	}
}

func TestSearchGeoLocations_ByID(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("id"); got != "US|NY|New York" {
			t.Fatalf("id = %q", got)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{"id": "US|NY|New York", "entity": GeoEntityLocality, "displayName": "New York, New York, United States"},
			},
		})
	}))
	t.Cleanup(srv.Close)

	c, err := New(WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	got, err := c.SearchGeoLocations(context.Background(), GeoSearchParams{ID: "US|NY|New York"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].DisplayName == "" {
		t.Fatalf("got = %#v", got)
	}
}

func TestSearchGeoLocations_RejectsShortQuery(t *testing.T) {
	t.Parallel()

	c, err := New(WithBaseURL("http://127.0.0.1:1"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.SearchGeoLocations(context.Background(), GeoSearchParams{Query: "NY"})
	if err == nil || err.Error() != "geo search query must be at least three characters" {
		t.Fatalf("err = %v", err)
	}
}
