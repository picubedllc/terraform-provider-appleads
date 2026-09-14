// Copyright (c) 2026 Pi Cubed LLC
// SPDX-License-Identifier: MIT

package client

import (
	"net/url"
	"testing"
)

func TestJoinAPIPath_PreservesV5Prefix(t *testing.T) {
	t.Parallel()

	cases := []struct {
		base, path, want string
	}{
		{DefaultBaseURL, "acls", "https://api.searchads.apple.com/api/v5/acls"},
		{"https://api.searchads.apple.com/api/v5", "acls", "https://api.searchads.apple.com/api/v5/acls"},
		{"https://api.searchads.apple.com/api/v5", "/campaigns", "https://api.searchads.apple.com/api/v5/campaigns"},
		{"https://api.searchads.apple.com/api/v5", "campaigns/1", "https://api.searchads.apple.com/api/v5/campaigns/1"},
		{"http://127.0.0.1:1234", "/campaigns", "http://127.0.0.1:1234/campaigns"},
		{"http://127.0.0.1:1234/", "campaigns", "http://127.0.0.1:1234/campaigns"},
	}
	for _, tc := range cases {
		base, err := url.Parse(tc.base)
		if err != nil {
			t.Fatalf("parse %q: %v", tc.base, err)
		}
		got := joinAPIPath(base, tc.path).String()
		if got != tc.want {
			t.Errorf("join(%q, %q) = %q, want %q", tc.base, tc.path, got, tc.want)
		}
	}
}
