// SPDX-License-Identifier: MIT

package v1

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/picubedllc/terraform-provider-appleads/internal/client"
)

// AuthTransport injects bearer auth and Platform API account context.
type AuthTransport struct {
	Tokens      client.TokenSource
	AdAccountID string
	Base        http.RoundTripper
}

func (t *AuthTransport) base() http.RoundTripper {
	if t.Base != nil {
		return t.Base
	}
	return http.DefaultTransport
}

// RoundTrip implements http.RoundTripper.
func (t *AuthTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.Tokens == nil {
		return nil, fmt.Errorf("apple ads platform api auth transport: TokenSource is required")
	}

	token, err := t.Tokens.Token(req.Context())
	if err != nil {
		return nil, fmt.Errorf("apple ads platform api auth transport: get access token: %w", err)
	}
	if token == "" {
		return nil, fmt.Errorf("apple ads platform api auth transport: empty access token")
	}

	r := req.Clone(req.Context())
	r.Header = req.Header.Clone()
	r.Header.Set("Authorization", "Bearer "+token)
	if t.AdAccountID != "" && adAccountContextRequired(r.URL.Path) {
		r.Header.Set(client.HeaderAPContext, "adAccountId="+t.AdAccountID)
	}
	if r.Header.Get("Accept") == "" {
		r.Header.Set("Accept", "application/json")
	}
	if r.Body != nil && r.Header.Get("Content-Type") == "" {
		r.Header.Set("Content-Type", "application/json")
	}

	return t.base().RoundTrip(r)
}

// NewAuthenticatedHTTPClient returns an HTTP client with Platform API auth headers.
func NewAuthenticatedHTTPClient(tokens client.TokenSource, adAccountID string, base http.RoundTripper) *http.Client {
	return &http.Client{
		Transport: &AuthTransport{
			Tokens:      tokens,
			AdAccountID: adAccountID,
			Base:        base,
		},
	}
}

func adAccountContextRequired(path string) bool {
	p := strings.TrimSuffix(path, "/")
	// Me/ACL-style discovery endpoints are unscoped; ad-account listing may be too.
	for _, suffix := range []string{"/acls", "/me", "/adaccounts"} {
		if strings.HasSuffix(p, suffix) {
			return false
		}
	}
	return true
}
