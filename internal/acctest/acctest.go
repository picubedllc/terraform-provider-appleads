// Copyright (c) 2026 Pi Cubed LLC
// SPDX-License-Identifier: MIT

// Package acctest provides helpers for live Apple Ads acceptance tests.
package acctest

import (
	"context"
	"errors"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/picubedllc/terraform-provider-appleads/internal/client"
)

// Credential environment variables. Must match the provider schema docs.
const (
	EnvOrgID      = "APPLEADS_ORG_ID"
	EnvClientID   = "APPLEADS_CLIENT_ID"
	EnvTeamID     = "APPLEADS_TEAM_ID"
	EnvKeyID      = "APPLEADS_KEY_ID"
	EnvPrivateKey = "APPLEADS_PRIVATE_KEY"
	EnvLiveTest   = "APPLEADS_LIVE_TEST"
)

// liveTestsEnabled reports whether tests that hit Apple's APIs should run.
func liveTestsEnabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(EnvLiveTest))) {
	case "1", "true", "yes":
		return true
	default:
		return false
	}
}

// PreCheck skips live Apple Ads tests unless APPLEADS_LIVE_TEST is set.
// If the flag is set but credentials are missing, it fails so CI cannot silently no-op.
func PreCheck(t *testing.T) {
	t.Helper()

	if !liveTestsEnabled() {
		t.Skip("live Apple Ads tests skipped unless APPLEADS_LIVE_TEST=1")
	}

	var missing []string
	for _, name := range []string{EnvOrgID, EnvClientID, EnvTeamID, EnvKeyID, EnvPrivateKey} {
		if strings.TrimSpace(os.Getenv(name)) == "" {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		t.Fatalf("APPLEADS_LIVE_TEST=1 but missing %s", strings.Join(missing, ", "))
	}
}

// Credentials returns Apple Ads OAuth credentials from the environment.
func Credentials(t *testing.T) client.Credentials {
	t.Helper()
	PreCheck(t)
	return client.Credentials{
		OrgID:      os.Getenv(EnvOrgID),
		ClientID:   os.Getenv(EnvClientID),
		TeamID:     os.Getenv(EnvTeamID),
		KeyID:      os.Getenv(EnvKeyID),
		PrivateKey: os.Getenv(EnvPrivateKey),
	}
}

// UserACL is the subset of GET /acls used to prove live authentication.
type UserACL struct {
	OrgID int64 `json:"orgId"`
}

// RequireUserACLs calls GET /acls, retrying Apple 503/429/5xx blips.
func RequireUserACLs(t *testing.T, c *client.Client) []UserACL {
	t.Helper()

	ctx, cancel := context.WithTimeout(t.Context(), 90*time.Second)
	defer cancel()

	deadline := time.Now().Add(45 * time.Second)
	var last error
	for attempt := 1; ; attempt++ {
		acls, err := client.FetchAllPages[UserACL](ctx, c, http.MethodGet, "/acls", 1000, nil)
		if err == nil {
			if len(acls) == 0 {
				t.Fatal("GET /acls returned no organizations")
			}
			return acls
		}
		last = err
		if time.Now().After(deadline) {
			if isTransientAPIError(err) {
				t.Skipf("Apple Ads API unavailable after retries; OAuth succeeded. last error: %v", err)
			}
			t.Fatalf("GET /acls: %v", err)
		}
		if !isTransientAPIError(err) {
			t.Fatalf("GET /acls: %v", err)
		}
		t.Logf("GET /acls attempt %d transient error, retrying", attempt)
		select {
		case <-ctx.Done():
			t.Fatalf("GET /acls: %v", last)
		case <-time.After(2 * time.Second):
		}
	}
}

// RequireOrgAccess asserts the configured org ID is present in the ACL list.
func RequireOrgAccess(t *testing.T, acls []UserACL, orgID string) {
	t.Helper()
	for _, acl := range acls {
		if strconv.FormatInt(acl.OrgID, 10) == orgID {
			return
		}
	}
	t.Fatalf("APPLEADS_ORG_ID %q was not in the authenticated ACL list", orgID)
}

// LiveClient builds the same authenticated, retrying Apple Ads client the provider uses.
func LiveClient(t *testing.T) (*client.Client, client.Credentials) {
	t.Helper()
	creds := Credentials(t)

	src, err := client.NewOAuthTokenSource(client.OAuthConfig{Credentials: creds})
	if err != nil {
		t.Fatalf("NewOAuthTokenSource: %v", err)
	}

	httpClient := client.WithRetry(
		client.NewAuthenticatedHTTPClient(src, creds.OrgID, nil),
		client.RetryConfig{},
	)
	c, err := client.New(client.WithHTTPClient(httpClient))
	if err != nil {
		t.Fatal(err)
	}
	return c, creds
}

// RequireCampaignsPage fetches one page of campaigns (read-only).
// Org-wide listing is temporary; campaign allowlisting is tracked in PI-31.
func RequireCampaignsPage(t *testing.T, c *client.Client) *client.PageResult[client.Campaign] {
	t.Helper()

	ctx, cancel := context.WithTimeout(t.Context(), 90*time.Second)
	defer cancel()

	deadline := time.Now().Add(45 * time.Second)
	var last error
	for attempt := 1; ; attempt++ {
		page, err := client.FetchPage[client.Campaign](ctx, c, http.MethodGet, "/campaigns", client.PageParams{Limit: 1}, nil)
		if err == nil {
			if page == nil {
				t.Fatal("GET /campaigns returned nil page")
			}
			return page
		}
		last = err
		if time.Now().After(deadline) {
			if isTransientAPIError(err) {
				t.Skipf("Apple Ads API unavailable after retries; last error: %v", err)
			}
			t.Fatalf("GET /campaigns: %v", err)
		}
		if !isTransientAPIError(err) {
			t.Fatalf("GET /campaigns: %v", err)
		}
		t.Logf("GET /campaigns attempt %d transient error, retrying", attempt)
		select {
		case <-ctx.Done():
			t.Fatalf("GET /campaigns: %v", last)
		case <-time.After(2 * time.Second):
		}
	}
}

// RequireSearchApps calls GET /search/apps with transient retries.
func RequireSearchApps(t *testing.T, c *client.Client, query string, ownedOnly bool) []client.App {
	t.Helper()

	ctx, cancel := context.WithTimeout(t.Context(), 90*time.Second)
	defer cancel()

	deadline := time.Now().Add(45 * time.Second)
	var last error
	for attempt := 1; ; attempt++ {
		apps, err := c.SearchApps(ctx, query, ownedOnly, 5)
		if err == nil {
			return apps
		}
		last = err
		if time.Now().After(deadline) {
			if isTransientAPIError(err) {
				t.Skipf("Apple Ads API unavailable after retries; last error: %v", err)
			}
			t.Fatalf("SearchApps(%q): %v", query, err)
		}
		if !isTransientAPIError(err) {
			t.Fatalf("SearchApps(%q): %v", query, err)
		}
		t.Logf("SearchApps attempt %d transient error, retrying", attempt)
		select {
		case <-ctx.Done():
			t.Fatalf("SearchApps(%q): %v", query, last)
		case <-time.After(2 * time.Second):
		}
	}
}

func isTransientAPIError(err error) bool {
	var apiErr *client.APIError
	if !errors.As(err, &apiErr) {
		return false
	}
	switch apiErr.StatusCode {
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
