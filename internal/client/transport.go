// Copyright (c) 2026 Pi Cubed LLC
// SPDX-License-Identifier: MIT

package client

import (
	"fmt"
	"net/http"
)

// AuthTransport is an http.RoundTripper that injects Apple Ads auth and account headers.
// Terraform resources should never construct these headers themselves.
type AuthTransport struct {
	// Tokens provides bearer access tokens. Required.
	Tokens TokenSource
	// OrgID is sent as X-AP-Context: orgId=<OrgID> on every request.
	OrgID string
	// Base is the next RoundTripper. Defaults to http.DefaultTransport.
	Base http.RoundTripper
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
		return nil, fmt.Errorf("apple ads auth transport: TokenSource is required")
	}

	token, err := t.Tokens.Token(req.Context())
	if err != nil {
		return nil, fmt.Errorf("apple ads auth transport: get access token: %w", err)
	}
	if token == "" {
		return nil, fmt.Errorf("apple ads auth transport: empty access token")
	}

	// Clone to avoid mutating the caller's request.
	r := req.Clone(req.Context())
	r.Header = req.Header.Clone()
	r.Header.Set("Authorization", "Bearer "+token)
	if t.OrgID != "" {
		r.Header.Set(HeaderAPContext, "orgId="+t.OrgID)
	}
	if r.Header.Get("Accept") == "" {
		r.Header.Set("Accept", "application/json")
	}
	if r.Body != nil && r.Header.Get("Content-Type") == "" {
		r.Header.Set("Content-Type", "application/json")
	}

	return t.base().RoundTrip(r)
}

// NewAuthenticatedHTTPClient returns an *http.Client whose transport injects
// bearer auth and organization context for Apple Ads API calls.
func NewAuthenticatedHTTPClient(tokens TokenSource, orgID string, base http.RoundTripper) *http.Client {
	return &http.Client{
		Transport: &AuthTransport{
			Tokens: tokens,
			OrgID:  orgID,
			Base:   base,
		},
	}
}
