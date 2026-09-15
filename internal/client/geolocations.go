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

// Geo entity types returned by Search for Geolocations (GET /search/geo).
const (
	GeoEntityCountry   = "Country"
	GeoEntityAdminArea = "AdminArea"
	GeoEntityLocality  = "Locality"
)

// GeoLocation is a targetable Apple Ads location.
//
// ID format is CountryCode, CountryCode|AdminArea, or
// CountryCode|AdminArea|Locality (for example US, US|NY, US|NY|New York).
type GeoLocation struct {
	ID          string `json:"id"`
	Entity      string `json:"entity"`
	DisplayName string `json:"displayName"`
}

// GeoSearchParams filters GET /search/geo.
// Query is prefix-matched and must be at least three characters when set.
type GeoSearchParams struct {
	Query       string
	Entity      string
	CountryCode string
	ID          string
	Limit       int
	Offset      int
}

// SearchGeoLocations finds targetable locations (Search for Geolocations).
// Returned IDs are used in ad group targetingDimensions.country, adminArea, and locality.
func (c *Client) SearchGeoLocations(ctx context.Context, p GeoSearchParams) ([]GeoLocation, error) {
	query := strings.TrimSpace(p.Query)
	id := strings.TrimSpace(p.ID)
	if query != "" && len([]rune(query)) < 3 {
		return nil, fmt.Errorf("geo search query must be at least three characters")
	}
	if p.Limit <= 0 {
		p.Limit = 20
	}

	q := url.Values{}
	if query != "" {
		q.Set("query", query)
	}
	if entity := strings.TrimSpace(p.Entity); entity != "" {
		q.Set("entity", entity)
	}
	if cc := strings.TrimSpace(p.CountryCode); cc != "" {
		q.Set("countrycode", cc)
	}
	if id != "" {
		q.Set("id", id)
	}
	q.Set("limit", strconv.Itoa(p.Limit))
	if p.Offset > 0 {
		q.Set("offset", strconv.Itoa(p.Offset))
	}

	var env ListResponse[GeoLocation]
	if err := c.DoJSONWithQuery(ctx, http.MethodGet, "search/geo", q, nil, &env); err != nil {
		return nil, err
	}
	return env.Data, nil
}
