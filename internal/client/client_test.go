// Copyright (c) 2026 Pi Cubed LLC
// SPDX-License-Identifier: MIT

package client_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/picubedllc/terraform-provider-appleads/internal/client"
)

type staticToken string

func (s staticToken) Token(ctx context.Context) (string, error) {
	return string(s), nil
}

func TestDoJSON_SuccessRoundTrip(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/campaigns" {
			t.Errorf("path = %s, want /campaigns", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Errorf("Authorization = %q", got)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("Content-Type = %q", got)
		}
		body, _ := io.ReadAll(r.Body)
		var in map[string]string
		if err := json.Unmarshal(body, &in); err != nil {
			t.Fatalf("unmarshal request: %v", err)
		}
		if in["name"] != "demo" {
			t.Errorf("name = %q", in["name"])
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]string{"id": "42", "name": "demo"},
		})
	}))
	t.Cleanup(srv.Close)

	c, err := client.New(
		client.WithBaseURL(srv.URL),
		client.WithTokenSource(staticToken("test-token")),
	)
	if err != nil {
		t.Fatal(err)
	}

	var out struct {
		Data struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"data"`
	}
	err = c.DoJSON(context.Background(), http.MethodPost, "/campaigns", map[string]string{"name": "demo"}, &out)
	if err != nil {
		t.Fatal(err)
	}
	if out.Data.ID != "42" || out.Data.Name != "demo" {
		t.Fatalf("unexpected response: %+v", out.Data)
	}
}

func TestDoJSON_APIErrorParsing(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Request-Id", "req-123")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"errors":[{"messageCode":"INVALID_NAME","message":"name is required","field":"name"}]}}`))
	}))
	t.Cleanup(srv.Close)

	c, err := client.New(client.WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}

	err = c.DoJSON(context.Background(), http.MethodGet, "/campaigns/1", nil, nil)
	var apiErr *client.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error type = %T, want *APIError; err=%v", err, err)
	}
	if apiErr.StatusCode != http.StatusBadRequest {
		t.Errorf("StatusCode = %d", apiErr.StatusCode)
	}
	if apiErr.Code != "INVALID_NAME" {
		t.Errorf("Code = %q", apiErr.Code)
	}
	if !strings.Contains(apiErr.Message, "name is required") {
		t.Errorf("Message = %q", apiErr.Message)
	}
	if apiErr.RequestID != "req-123" {
		t.Errorf("RequestID = %q", apiErr.RequestID)
	}
}

func TestFetchAllPages_MultiplePages(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		offset := r.URL.Query().Get("offset")
		w.Header().Set("Content-Type", "application/json")
		switch offset {
		case "", "0":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data":       []map[string]string{{"id": "1"}, {"id": "2"}},
				"pagination": map[string]int{"totalResults": 3, "startIndex": 0, "itemsPerPage": 2},
			})
		case "2":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data":       []map[string]string{{"id": "3"}},
				"pagination": map[string]int{"totalResults": 3, "startIndex": 2, "itemsPerPage": 2},
			})
		default:
			t.Errorf("unexpected offset %q on call %d", offset, n)
			w.WriteHeader(http.StatusBadRequest)
		}
	}))
	t.Cleanup(srv.Close)

	c, err := client.New(client.WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}

	type item struct {
		ID string `json:"id"`
	}
	items, err := client.FetchAllPages[item](context.Background(), c, http.MethodGet, "/campaigns", 2, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 3 {
		t.Fatalf("len(items) = %d, want 3", len(items))
	}
	if calls.Load() < 2 {
		t.Fatalf("expected at least 2 page fetches, got %d", calls.Load())
	}
}

func TestDoJSON_ContextCancellation(t *testing.T) {
	t.Parallel()

	started := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		select {
		case <-r.Context().Done():
			return
		case <-time.After(5 * time.Second):
			w.WriteHeader(http.StatusOK)
		}
	}))
	t.Cleanup(srv.Close)

	c, err := client.New(client.WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-started
		cancel()
	}()

	err = c.DoJSON(ctx, http.MethodGet, "/slow", nil, nil)
	if err == nil {
		t.Fatal("expected context cancellation error")
	}
	if !errors.Is(err, context.Canceled) && !strings.Contains(err.Error(), "context canceled") {
		t.Fatalf("unexpected error: %v", err)
	}
}
