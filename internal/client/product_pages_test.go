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

func TestListProductPages_FiltersAndEmpty(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/apps/899247964/product-pages" {
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
		q := r.URL.Query()
		if q.Get("name") != "Trip Trek CPP" {
			t.Fatalf("name = %q", q.Get("name"))
		}
		if q.Get("states") != "VISIBLE,HIDDEN" {
			t.Fatalf("states = %q", q.Get("states"))
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data":       []any{},
			"pagination": map[string]int{"totalResults": 0, "startIndex": 0, "itemsPerPage": 20},
		})
	}))
	t.Cleanup(srv.Close)

	c, err := New(WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	got, err := c.ListProductPages(context.Background(), 899247964, ProductPageListParams{
		Name:   "Trip Trek CPP",
		States: []string{"VISIBLE", "HIDDEN"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("got = %#v", got)
	}
}

func TestListProductPages_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{
					"id":               "45812c9b-c296-43d3-c6a0-c5a02f74bf6e",
					"name":             "Trip Trek CPP variation",
					"state":            ProductPageStateVisible,
					"adamId":           899247964,
					"deepLink":         "triptrek://home",
					"creationTime":     "2024-10-25T23:59:59.000",
					"modificationTime": "2024-10-25T23:59:59.000",
				},
			},
		})
	}))
	t.Cleanup(srv.Close)

	c, err := New(WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	got, err := c.ListProductPages(context.Background(), 899247964, ProductPageListParams{})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("len = %d", len(got))
	}
	if got[0].ID != "45812c9b-c296-43d3-c6a0-c5a02f74bf6e" || got[0].State != ProductPageStateVisible {
		t.Fatalf("got[0] = %#v", got[0])
	}
	if got[0].DeepLink != "triptrek://home" {
		t.Fatalf("deepLink = %q", got[0].DeepLink)
	}
}

func TestListProductPages_InvalidAdamID(t *testing.T) {
	t.Parallel()

	c, err := New(WithBaseURL("http://127.0.0.1:1"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.ListProductPages(context.Background(), 0, ProductPageListParams{})
	if err == nil || err.Error() != "adamId must be a positive integer" {
		t.Fatalf("err = %v", err)
	}
}

func TestGetProductPage_Success(t *testing.T) {
	t.Parallel()

	const pageID = "45812c9b-c296-43d3-c6a0-c5a02f74bf6e"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		want := "/apps/899247964/product-pages/" + pageID
		if r.Method != http.MethodGet || r.URL.Path != want {
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"id":     pageID,
				"name":   "Trip Trek CPP variation",
				"state":  ProductPageStateVisible,
				"adamId": 899247964,
			},
		})
	}))
	t.Cleanup(srv.Close)

	c, err := New(WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	got, err := c.GetProductPage(context.Background(), 899247964, pageID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "Trip Trek CPP variation" {
		t.Fatalf("got = %#v", got)
	}
}

func TestGetProductPage_NotFound(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"errors": []map[string]any{
					{"messageCode": "NOT_FOUND", "message": "product page not found"},
				},
			},
		})
	}))
	t.Cleanup(srv.Close)

	c, err := New(WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.GetProductPage(context.Background(), 899247964, "missing-id")
	if !IsNotFound(err) {
		t.Fatalf("err = %v", err)
	}
}

func TestGetProductPage_InvalidAdamID(t *testing.T) {
	t.Parallel()

	c, err := New(WithBaseURL("http://127.0.0.1:1"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.GetProductPage(context.Background(), -1, "abc")
	if err == nil || err.Error() != "adamId must be a positive integer" {
		t.Fatalf("err = %v", err)
	}
}

func TestGetProductPage_MissingProductPageID(t *testing.T) {
	t.Parallel()

	c, err := New(WithBaseURL("http://127.0.0.1:1"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.GetProductPage(context.Background(), 1, "  ")
	if err == nil || err.Error() != "productPageId is required" {
		t.Fatalf("err = %v", err)
	}
}

func TestListProductPageLocales_Filters(t *testing.T) {
	t.Parallel()

	const pageID = "45812c9b-c296-43d3-c6a0-c5a02f74bf6e"
	expand := true
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		want := "/apps/42/product-pages/" + pageID + "/locale-details"
		if r.URL.Path != want {
			t.Fatalf("path = %s", r.URL.Path)
		}
		q := r.URL.Query()
		if q.Get("deviceClasses") != "IPHONE,IPAD" {
			t.Fatalf("deviceClasses = %q", q.Get("deviceClasses"))
		}
		if q.Get("languageCodes") != "en-US" {
			t.Fatalf("languageCodes = %q", q.Get("languageCodes"))
		}
		if q.Get("languages") != "English" {
			t.Fatalf("languages = %q", q.Get("languages"))
		}
		if q.Get("expand") != "true" {
			t.Fatalf("expand = %q", q.Get("expand"))
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{
					"adamId":           42,
					"appName":          "OrbitNote",
					"deviceClasses":    DeviceClassIPhone,
					"language":         "English",
					"languageCode":     "en-US",
					"productPageId":    pageID,
					"promotionalText":  "Try it",
					"shortDescription": "Notes app",
					"subTitle":         "Capture ideas",
				},
			},
		})
	}))
	t.Cleanup(srv.Close)

	c, err := New(WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	got, err := c.ListProductPageLocales(context.Background(), 42, pageID, ProductPageLocaleListParams{
		DeviceClasses: []string{DeviceClassIPhone, DeviceClassIPad},
		LanguageCodes: []string{"en-US"},
		Languages:     []string{"English"},
		Expand:        &expand,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].LanguageCode != "en-US" || got[0].AppName != "OrbitNote" {
		t.Fatalf("got = %#v", got)
	}
}

func TestListProductPageLocales_Empty(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"data": []any{}})
	}))
	t.Cleanup(srv.Close)

	c, err := New(WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	got, err := c.ListProductPageLocales(context.Background(), 1, "page-id", ProductPageLocaleListParams{})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("got = %#v", got)
	}
}

func TestListProductPageLocales_InvalidAdamID(t *testing.T) {
	t.Parallel()

	c, err := New(WithBaseURL("http://127.0.0.1:1"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.ListProductPageLocales(context.Background(), 0, "page", ProductPageLocaleListParams{})
	if err == nil || err.Error() != "adamId must be a positive integer" {
		t.Fatalf("err = %v", err)
	}
}

func TestListCountriesOrRegions_FilterAndEmpty(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/countries-or-regions" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("countriesOrRegions"); got != "US,CA" {
			t.Fatalf("countriesOrRegions = %q", got)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": []any{}})
	}))
	t.Cleanup(srv.Close)

	c, err := New(WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	got, err := c.ListCountriesOrRegions(context.Background(), []string{"US", "CA"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("got = %#v", got)
	}
}

func TestListCountriesOrRegions_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{
					"countryOrRegion": "US",
					"defaultLanguages": []map[string]any{
						{"language": "English", "languageCode": "en-US"},
					},
					"supportedLanguages": []map[string]any{
						{"language": "English", "languageCode": "en-US"},
						{"language": "Spanish", "languageCode": "es-MX"},
					},
				},
			},
		})
	}))
	t.Cleanup(srv.Close)

	c, err := New(WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	got, err := c.ListCountriesOrRegions(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].CountryOrRegion != "US" {
		t.Fatalf("got = %#v", got)
	}
	if len(got[0].SupportedLanguages) != 2 || got[0].SupportedLanguages[1].LanguageCode != "es-MX" {
		t.Fatalf("languages = %#v", got[0].SupportedLanguages)
	}
}
