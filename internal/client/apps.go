// Copyright (c) 2026 Pi Cubed LLC
// SPDX-License-Identifier: MIT

package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// App is an App Store app resolved via the Apple Ads search/apps API.
// Field names follow Apple Ads Campaign Management API v5.
type App struct {
	AdamID          int64  `json:"adamId"`
	AppName         string `json:"appName"`
	DeveloperName   string `json:"developerName,omitempty"`
	CountryOrRegion string `json:"countryOrRegion,omitempty"`
}

// SearchApps finds apps by query string.
// Uses GET /search/apps (Apple Ads Campaign Management API v5).
// When returnOwnedApps is true, results are limited to apps the org can promote.
func (c *Client) SearchApps(ctx context.Context, query string, returnOwnedApps bool, limit int) ([]App, error) {
	if query == "" {
		return nil, fmt.Errorf("app search query is required")
	}
	if limit <= 0 {
		limit = 20
	}

	q := url.Values{}
	q.Set("query", query)
	q.Set("limit", strconv.Itoa(limit))
	q.Set("offset", "0")
	if returnOwnedApps {
		q.Set("returnOwnedApps", "true")
	}

	var env ListResponse[App]
	if err := c.DoJSONWithQuery(ctx, http.MethodGet, "search/apps", q, nil, &env); err != nil {
		return nil, err
	}
	return env.Data, nil
}

// GetApp fetches a single app by Adam ID.
// Apple Ads exposes app metadata through search; we query by adamId string and
// require an exact adamId match in the response set.
func (c *Client) GetApp(ctx context.Context, adamID int64) (*App, error) {
	apps, err := c.SearchApps(ctx, strconv.FormatInt(adamID, 10), false, 50)
	if err != nil {
		return nil, err
	}
	for i := range apps {
		if apps[i].AdamID == adamID {
			return &apps[i], nil
		}
	}
	return nil, &APIError{
		StatusCode: http.StatusNotFound,
		Code:       "NOT_FOUND",
		Message:    fmt.Sprintf("app with adamId %d not found", adamID),
	}
}

// FindAppByName resolves an app by exact (case-insensitive) display name.
// Prefers owned apps (returnOwnedApps=true). If multiple apps match, returns an
// error listing candidates (id + name) rather than silently picking one.
func (c *Client) FindAppByName(ctx context.Context, name string) (*App, error) {
	if name == "" {
		return nil, fmt.Errorf("app name is required")
	}

	matches, err := filterAppsByName(c, ctx, name, true)
	if err != nil {
		return nil, err
	}
	if len(matches) == 0 {
		matches, err = filterAppsByName(c, ctx, name, false)
		if err != nil {
			return nil, err
		}
	}

	switch len(matches) {
	case 0:
		return nil, &APIError{
			StatusCode: http.StatusNotFound,
			Code:       "NOT_FOUND",
			Message:    fmt.Sprintf("no app found with name %q", name),
		}
	case 1:
		return &matches[0], nil
	default:
		var b strings.Builder
		fmt.Fprintf(&b, "multiple apps matched name %q; disambiguate using id:", name)
		for _, m := range matches {
			fmt.Fprintf(&b, "\n  - id=%d name=%q developer=%q", m.AdamID, m.AppName, m.DeveloperName)
		}
		return nil, &APIError{
			StatusCode: http.StatusConflict,
			Code:       "MULTIPLE_MATCHES",
			Message:    b.String(),
		}
	}
}

func filterAppsByName(c *Client, ctx context.Context, name string, ownedOnly bool) ([]App, error) {
	apps, err := c.SearchApps(ctx, name, ownedOnly, 50)
	if err != nil {
		return nil, err
	}
	var matches []App
	for _, app := range apps {
		if strings.EqualFold(strings.TrimSpace(app.AppName), strings.TrimSpace(name)) {
			matches = append(matches, app)
		}
	}
	return matches, nil
}
