// SPDX-License-Identifier: MIT

package v1

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

// App is an App Store app from Platform API search/details.
type App struct {
	AdamID          int64  `json:"adamId"`
	AppName         string `json:"appName"`
	DeveloperName   string `json:"developerName,omitempty"`
	CountryOrRegion string `json:"countryOrRegion,omitempty"`
}

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
	return env.Result, nil
}

func (c *Client) GetApp(ctx context.Context, adamID int64) (*App, error) {
	var env Response[App]
	path := fmt.Sprintf("apps/%d", adamID)
	if err := c.DoJSON(ctx, http.MethodGet, path, nil, &env); err != nil {
		// Fallback: search by adamId string (parity with v5 helper).
		apps, serr := c.SearchApps(ctx, strconv.FormatInt(adamID, 10), false, 50)
		if serr != nil {
			return nil, err
		}
		for i := range apps {
			if apps[i].AdamID == adamID {
				return &apps[i], nil
			}
		}
		return nil, &APIError{StatusCode: http.StatusNotFound, Code: "NOT_FOUND", Message: fmt.Sprintf("app with adamId %d not found", adamID)}
	}
	return &env.Result, nil
}
