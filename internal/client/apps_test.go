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

func TestSearchApps(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/search/apps" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("query"); got != "Screenbase" {
			t.Fatalf("query = %q", got)
		}
		if got := r.URL.Query().Get("returnOwnedApps"); got != "true" {
			t.Fatalf("returnOwnedApps = %q", got)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{"adamId": 123, "appName": "Screenbase", "developerName": "Pi Cubed"},
			},
			"pagination": map[string]int{"totalResults": 1, "startIndex": 0, "itemsPerPage": 20},
		})
	}))
	t.Cleanup(srv.Close)

	c, err := New(WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	apps, err := c.SearchApps(context.Background(), "Screenbase", true, 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(apps) != 1 || apps[0].AdamID != 123 || apps[0].AppName != "Screenbase" {
		t.Fatalf("apps = %#v", apps)
	}
}

func TestFindAppByName_ExactMatch(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{"adamId": 1, "appName": "Screenbase", "developerName": "A"},
				{"adamId": 2, "appName": "Screenbase Pro", "developerName": "A"},
			},
		})
	}))
	t.Cleanup(srv.Close)

	c, err := New(WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	app, err := c.FindAppByName(context.Background(), "Screenbase")
	if err != nil {
		t.Fatal(err)
	}
	if app.AdamID != 1 {
		t.Fatalf("adamId = %d", app.AdamID)
	}
}

func TestFindAppByName_MultipleMatches(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{"adamId": 1, "appName": "Demo", "developerName": "A"},
				{"adamId": 2, "appName": "demo", "developerName": "B"},
			},
		})
	}))
	t.Cleanup(srv.Close)

	c, err := New(WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.FindAppByName(context.Background(), "Demo")
	if err == nil {
		t.Fatal("expected error")
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("err type = %T", err)
	}
	if apiErr.Code != "MULTIPLE_MATCHES" {
		t.Fatalf("code = %s", apiErr.Code)
	}
	if !strings.Contains(apiErr.Message, "id=1") || !strings.Contains(apiErr.Message, "id=2") {
		t.Fatalf("message = %s", apiErr.Message)
	}
}

func TestFindAppByName_NotFound(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"data": []any{}})
	}))
	t.Cleanup(srv.Close)

	c, err := New(WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.FindAppByName(context.Background(), "Nope")
	if !IsNotFound(err) {
		t.Fatalf("err = %v", err)
	}
}

func TestGetApp_ByAdamID(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{"adamId": 999, "appName": "Other"},
				{"adamId": 42, "appName": "Target"},
			},
		})
	}))
	t.Cleanup(srv.Close)

	c, err := New(WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	app, err := c.GetApp(context.Background(), 42)
	if err != nil {
		t.Fatal(err)
	}
	if app.AppName != "Target" {
		t.Fatalf("app = %#v", app)
	}
}
