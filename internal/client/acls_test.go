// Copyright (c) 2026 Pi Cubed LLC
// SPDX-License-Identifier: MIT

package client

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCheckOrgID(t *testing.T) {
	t.Parallel()

	acls := []UserACL{{OrgID: 1234567, OrgName: "Example"}}
	if err := CheckOrgID("1234567", acls); err != nil {
		t.Fatal(err)
	}
	err := CheckOrgID("123456", acls)
	if err == nil {
		t.Fatal("expected mismatch")
	}
	var orgErr *OrgIDError
	if !errors.As(err, &orgErr) {
		t.Fatalf("err type %T", err)
	}
	if orgErr.Configured != "123456" || !strings.Contains(err.Error(), "1234567") {
		t.Fatalf("error = %v", err)
	}
}

func TestCreateCampaign_ForbiddenEnrichedWithOrgMismatch(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/acls"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{{"orgId": 1234567, "orgName": "Example"}},
			})
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/campaigns"):
			w.WriteHeader(http.StatusForbidden)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"error": map[string]any{
					"errors": []map[string]string{{
						"messageCode": "FORBIDDEN",
						"message":     "Access forbidden - feature disabled",
					}},
				},
			})
		default:
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	c, err := New(WithBaseURL(srv.URL), WithOrgID("123456"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.CreateCampaign(context.Background(), &CampaignCreate{
		Name:               "x",
		AdamID:             1,
		CountriesOrRegions: []string{"US"},
	})
	if err == nil {
		t.Fatal("expected error")
	}
	var orgErr *OrgIDError
	if !errors.As(err, &orgErr) {
		t.Fatalf("err type %T: %v", err, err)
	}
	if orgErr.Configured != "123456" || !strings.Contains(orgErr.Error(), "1234567") {
		t.Fatalf("org error = %v", orgErr)
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusForbidden {
		t.Fatalf("cause = %v", err)
	}
}

func TestDoJSON_HTML503Sanitized(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte("<html><body><h1>503 Service Temporarily Unavailable</h1><center>Apple</center></body></html>"))
	}))
	t.Cleanup(srv.Close)

	c, err := New(WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	err = c.DoJSON(context.Background(), http.MethodGet, "campaigns", nil, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("err type %T", err)
	}
	if apiErr.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("status = %d", apiErr.StatusCode)
	}
	if strings.Contains(apiErr.Message, "<html>") || strings.Contains(apiErr.Message, "<h1>") {
		t.Fatalf("message still contains HTML: %q", apiErr.Message)
	}
}

func TestCreateCampaign_ForbiddenWhenOrgMatchesACL(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/acls"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{{"orgId": 1234567, "orgName": "Example"}},
			})
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/campaigns"):
			w.WriteHeader(http.StatusForbidden)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"error": map[string]any{
					"errors": []map[string]string{{
						"messageCode": "FORBIDDEN",
						"message":     "Access forbidden - feature disabled",
					}},
				},
			})
		default:
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	c, err := New(WithBaseURL(srv.URL), WithOrgID("1234567"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.CreateCampaign(context.Background(), &CampaignCreate{
		Name:               "x",
		AdamID:             1,
		CountriesOrRegions: []string{"US"},
	})
	if err == nil {
		t.Fatal("expected error")
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("err type %T: %v", err, err)
	}
	if strings.Contains(strings.ToLower(apiErr.Message), "get /acls") {
		t.Fatalf("matched org should keep the API error: %q", apiErr.Message)
	}
	if !strings.Contains(apiErr.Message, "feature disabled") {
		t.Fatalf("message = %q", apiErr.Message)
	}
}
