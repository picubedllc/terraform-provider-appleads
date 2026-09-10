// Copyright (c) 2026 Pi Cubed LLC
// SPDX-License-Identifier: MIT

package client_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/picubedllc/terraform-provider-appleads/internal/client"
)

type mockTokenSource struct {
	token string
	err   error
}

func (m mockTokenSource) Token(ctx context.Context) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return m.token, nil
}

func TestAuthTransport_InjectsBearerAndOrgHeader(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer tok-1" {
			t.Errorf("Authorization = %q", got)
		}
		if got := r.Header.Get(client.HeaderAPContext); got != "orgId=999" {
			t.Errorf("X-AP-Context = %q", got)
		}
		if got := r.Header.Get("Accept"); got != "application/json" {
			t.Errorf("Accept = %q", got)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("Content-Type = %q", got)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(srv.Close)

	httpClient := client.NewAuthenticatedHTTPClient(mockTokenSource{token: "tok-1"}, "999", srv.Client().Transport)
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, srv.URL+"/v5/campaigns", strings.NewReader(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status = %d", resp.StatusCode)
	}
}

func TestAuthTransport_TokenSourceError(t *testing.T) {
	t.Parallel()

	base := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		t.Fatal("base transport should not be called when token source fails")
		return nil, nil
	})

	tr := &client.AuthTransport{
		Tokens: mockTokenSource{err: errors.New("refresh failed")},
		OrgID:  "1",
		Base:   base,
	}
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://example.invalid/", nil)
	_, err := tr.RoundTrip(req)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "refresh failed") {
		t.Fatalf("error = %v", err)
	}
}

func TestClient_WithAuthTransport_APIError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Request-Id", "rid-9")
		w.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(w, `{"error":{"errors":[{"messageCode":"NOT_FOUND","message":"campaign missing"}]}}`)
	}))
	t.Cleanup(srv.Close)

	httpClient := client.NewAuthenticatedHTTPClient(mockTokenSource{token: "tok"}, "1", srv.Client().Transport)
	c, err := client.New(
		client.WithBaseURL(srv.URL),
		client.WithHTTPClient(httpClient),
	)
	if err != nil {
		t.Fatal(err)
	}

	err = c.DoJSON(context.Background(), http.MethodGet, "/campaigns/missing", nil, nil)
	var apiErr *client.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("want APIError, got %T: %v", err, err)
	}
	if apiErr.StatusCode != http.StatusNotFound || apiErr.Code != "NOT_FOUND" || apiErr.RequestID != "rid-9" {
		t.Fatalf("apiErr = %+v", apiErr)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
