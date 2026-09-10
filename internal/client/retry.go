// Copyright (c) 2026 Pi Cubed LLC
// SPDX-License-Identifier: MIT

package client

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	defaultMaxAttempts = 5
	defaultMaxElapsed  = 60 * time.Second
	defaultBaseDelay   = 500 * time.Millisecond
	defaultMaxDelay    = 30 * time.Second
)

// RetryConfig controls retry/backoff behavior for transient Apple Ads API failures.
type RetryConfig struct {
	MaxAttempts int
	MaxElapsed  time.Duration
	BaseDelay   time.Duration
	MaxDelay    time.Duration
	// Now is an injectable clock for tests. Defaults to time.Now.
	Now func() time.Time
	// After waits for a duration (or until ctx is done). Defaults to sleepWithContext.
	After func(ctx context.Context, d time.Duration) error
	// RandFloat64 returns a jitter factor in [0,1). Defaults to rand.Float64.
	RandFloat64 func() float64
}

func (c RetryConfig) withDefaults() RetryConfig {
	if c.MaxAttempts <= 0 {
		c.MaxAttempts = defaultMaxAttempts
	}
	if c.MaxElapsed <= 0 {
		c.MaxElapsed = defaultMaxElapsed
	}
	if c.BaseDelay <= 0 {
		c.BaseDelay = defaultBaseDelay
	}
	if c.MaxDelay <= 0 {
		c.MaxDelay = defaultMaxDelay
	}
	if c.Now == nil {
		c.Now = time.Now
	}
	if c.After == nil {
		c.After = sleepWithContext
	}
	if c.RandFloat64 == nil {
		c.RandFloat64 = rand.Float64
	}
	return c
}

// RetryTransport wraps a base RoundTripper with retries for 429, transient 5xx,
// and common network errors. Permanent 4xx responses are not retried.
type RetryTransport struct {
	Base   http.RoundTripper
	Config RetryConfig
}

func (t *RetryTransport) base() http.RoundTripper {
	if t.Base != nil {
		return t.Base
	}
	return http.DefaultTransport
}

// RoundTrip implements http.RoundTripper.
func (t *RetryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	cfg := t.Config.withDefaults()
	start := cfg.Now()

	var bodyBytes []byte
	if req.Body != nil {
		var err error
		bodyBytes, err = io.ReadAll(req.Body)
		_ = req.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("read request body for retry: %w", err)
		}
	}

	var lastErr error
	for attempt := 1; attempt <= cfg.MaxAttempts; attempt++ {
		if err := req.Context().Err(); err != nil {
			return nil, err
		}
		if attempt > 1 && cfg.Now().Sub(start) >= cfg.MaxElapsed {
			if lastErr != nil {
				return nil, lastErr
			}
			return nil, fmt.Errorf("retry budget exhausted after %s", cfg.MaxElapsed)
		}

		r := req.Clone(req.Context())
		if bodyBytes != nil {
			r.Body = io.NopCloser(bytes.NewReader(bodyBytes))
			r.GetBody = func() (io.ReadCloser, error) {
				return io.NopCloser(bytes.NewReader(bodyBytes)), nil
			}
			r.ContentLength = int64(len(bodyBytes))
		}

		resp, err := t.base().RoundTrip(r)
		if err != nil {
			lastErr = err
			if !isRetryableNetErr(err) || attempt == cfg.MaxAttempts {
				return nil, err
			}
			delay := cfg.backoff(attempt, nil)
			if err := cfg.After(req.Context(), delay); err != nil {
				return nil, err
			}
			continue
		}

		if !isRetryableStatus(resp.StatusCode) || attempt == cfg.MaxAttempts {
			return resp, nil
		}

		// Drain and close so the connection can be reused before retrying.
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()

		delay := cfg.backoff(attempt, resp)
		lastErr = fmt.Errorf("retryable status %d", resp.StatusCode)
		if err := cfg.After(req.Context(), delay); err != nil {
			return nil, err
		}
	}

	if lastErr != nil {
		return nil, lastErr
	}
	return nil, fmt.Errorf("retry attempts exhausted")
}

func (c RetryConfig) backoff(attempt int, resp *http.Response) time.Duration {
	if resp != nil {
		if ra := resp.Header.Get("Retry-After"); ra != "" {
			if secs, err := strconv.Atoi(ra); err == nil && secs >= 0 {
				d := time.Duration(secs) * time.Second
				if d > c.MaxDelay {
					return c.MaxDelay
				}
				return d
			}
			if when, err := http.ParseTime(ra); err == nil {
				d := when.Sub(c.Now())
				if d < 0 {
					d = 0
				}
				if d > c.MaxDelay {
					return c.MaxDelay
				}
				return d
			}
		}
	}

	// Exponential backoff with full jitter: delay = random(0, min(max, base*2^(attempt-1)))
	exp := math.Min(float64(c.MaxDelay), float64(c.BaseDelay)*math.Pow(2, float64(attempt-1)))
	return time.Duration(c.RandFloat64() * exp)
}

func isRetryableStatus(code int) bool {
	switch code {
	case http.StatusTooManyRequests,
		http.StatusInternalServerError,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout:
		return true
	default:
		return false
	}
}

func isRetryableNetErr(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		return netErr.Timeout()
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "connection reset") ||
		strings.Contains(msg, "connection refused") ||
		strings.Contains(msg, "broken pipe") ||
		strings.Contains(msg, "i/o timeout")
}

func sleepWithContext(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return ctx.Err()
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// WithRetry wraps an existing HTTP client transport with retry middleware.
func WithRetry(httpClient *http.Client, cfg RetryConfig) *http.Client {
	if httpClient == nil {
		httpClient = &http.Client{}
	}
	cloned := *httpClient
	cloned.Transport = &RetryTransport{
		Base:   httpClient.Transport,
		Config: cfg,
	}
	return &cloned
}
