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
)

// PreCheck skips live tests unless TF_ACC=1 and all APPLEADS_* credentials are set.
func PreCheck(t *testing.T) {
	t.Helper()

	if os.Getenv("TF_ACC") == "" {
		t.Skip("acceptance tests skipped unless TF_ACC=1")
	}

	var missing []string
	for _, name := range []string{EnvOrgID, EnvClientID, EnvTeamID, EnvKeyID, EnvPrivateKey} {
		if strings.TrimSpace(os.Getenv(name)) == "" {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		t.Skipf("acceptance tests skipped; missing %s", strings.Join(missing, ", "))
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
