// Copyright (c) 2026 Pi Cubed LLC
// SPDX-License-Identifier: MIT

package client_test

import (
	"context"
	"testing"
	"time"

	"github.com/picubedllc/terraform-provider-appleads/internal/acctest"
	"github.com/picubedllc/terraform-provider-appleads/internal/client"
)

func TestLiveOAuthTokenExchange(t *testing.T) {
	creds := acctest.Credentials(t)

	src, err := client.NewOAuthTokenSource(client.OAuthConfig{Credentials: creds})
	if err != nil {
		t.Fatalf("NewOAuthTokenSource: %v", err)
	}

	ctx, cancel := context.WithTimeout(t.Context(), 45*time.Second)
	defer cancel()

	tok, err := src.Token(ctx)
	if err != nil {
		t.Fatalf("token exchange: %v", err)
	}
	if tok == "" {
		t.Fatal("empty access token")
	}

	tok2, err := src.Token(ctx)
	if err != nil {
		t.Fatalf("cached token: %v", err)
	}
	if tok2 != tok {
		t.Fatal("second Token() should return the cached access token")
	}
}

func TestLiveAuthenticatedUserACL(t *testing.T) {
	creds := acctest.Credentials(t)

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

	acls := acctest.RequireUserACLs(t, c)
	acctest.RequireOrgAccess(t, acls, creds.OrgID)
}
