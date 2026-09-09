// Copyright (c) 2026 Pi Cubed LLC
// SPDX-License-Identifier: MIT

package client_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/picubedllc/terraform-provider-appleads/internal/client"
)

func TestRetryTransport_Retries429ThenSucceeds(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		body, _ := io.ReadAll(r.Body)
		if string(body) != `{"ok":true}` {
			t.Errorf("body = %q", body)
		}
		if n < 3 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":"ok"}`))
	}))
	t.Cleanup(srv.Close)

	var sleeps []time.Duration
	httpClient := &http.Client{
		Transport: &client.RetryTransport{
			Base: srv.Client().Transport,
			Config: client.RetryConfig{
				MaxAttempts: 5,
				BaseDelay:   time.Millisecond,
				MaxDelay:    time.Second,
				RandFloat64: func() float64 { return 0 },
				After: func(ctx context.Context, d time.Duration) error {
					sleeps = append(sleeps, d)
					return nil
				},
			},
		},
	}

	c, err := client.New(client.WithBaseURL(srv.URL), client.WithHTTPClient(httpClient))
	if err != nil {
		t.Fatal(err)
	}

	var out map[string]string
	err = c.DoJSON(context.Background(), http.MethodPost, "/campaigns", map[string]bool{"ok": true}, &out)
	if err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 3 {
		t.Fatalf("calls = %d, want 3", calls.Load())
	}
	if len(sleeps) != 2 {
		t.Fatalf("sleeps = %d, want 2", len(sleeps))
	}
}

func TestRetryTransport_DoesNotRetry400(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"errors":[{"messageCode":"BAD","message":"nope"}]}}`))
	}))
	t.Cleanup(srv.Close)

	httpClient := &http.Client{
		Transport: &client.RetryTransport{
			Base: srv.Client().Transport,
			Config: client.RetryConfig{
				MaxAttempts: 5,
				After:       func(context.Context, time.Duration) error { t.Fatal("should not sleep"); return nil },
			},
		},
	}
	c, err := client.New(client.WithBaseURL(srv.URL), client.WithHTTPClient(httpClient))
	if err != nil {
		t.Fatal(err)
	}

	err = c.DoJSON(context.Background(), http.MethodGet, "/x", nil, nil)
	var apiErr *client.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("want APIError, got %T: %v", err, err)
	}
	if calls.Load() != 1 {
		t.Fatalf("calls = %d, want 1", calls.Load())
	}
}

func TestRetryTransport_StopsOnContextCancel(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	t.Cleanup(srv.Close)

	ctx, cancel := context.WithCancel(context.Background())
	httpClient := &http.Client{
		Transport: &client.RetryTransport{
			Base: srv.Client().Transport,
			Config: client.RetryConfig{
				MaxAttempts: 10,
				After: func(ctx context.Context, d time.Duration) error {
					cancel()
					return ctx.Err()
				},
			},
		},
	}
	c, err := client.New(client.WithBaseURL(srv.URL), client.WithHTTPClient(httpClient))
	if err != nil {
		t.Fatal(err)
	}

	err = c.DoJSON(ctx, http.MethodGet, "/x", nil, nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
	if calls.Load() != 1 {
		t.Fatalf("calls = %d, want 1", calls.Load())
	}
}

func TestRetryTransport_HonorsRetryAfter(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		if n == 1 {
			w.Header().Set("Retry-After", "7")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(srv.Close)

	var slept time.Duration
	httpClient := &http.Client{
		Transport: &client.RetryTransport{
			Base: srv.Client().Transport,
			Config: client.RetryConfig{
				MaxAttempts: 3,
				BaseDelay:   time.Hour, // would be used if Retry-After ignored
				After: func(ctx context.Context, d time.Duration) error {
					slept = d
					return nil
				},
			},
		},
	}
	c, err := client.New(client.WithBaseURL(srv.URL), client.WithHTTPClient(httpClient))
	if err != nil {
		t.Fatal(err)
	}

	err = c.DoJSON(context.Background(), http.MethodGet, "/x", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if slept != 7*time.Second {
		t.Fatalf("slept = %s, want 7s", slept)
	}
}
