// SPDX-License-Identifier: MIT

package v1

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
)

// GeoLocation is a geo search hit.
type GeoLocation struct {
	ID              string `json:"id,omitempty"`
	Entity          string `json:"entity,omitempty"`
	DisplayName     string `json:"displayName,omitempty"`
	CountryOrRegion string `json:"countryOrRegion,omitempty"`
	AdminArea       string `json:"adminArea,omitempty"`
	Locality        string `json:"locality,omitempty"`
}

// GeoSearchParams filters geo search.
type GeoSearchParams struct {
	Query           string
	CountryOrRegion string
	Entity          string
	Limit           int
}

func (c *Client) SearchGeoLocations(ctx context.Context, p GeoSearchParams) ([]GeoLocation, error) {
	q := url.Values{}
	if p.Query != "" {
		q.Set("query", p.Query)
	}
	if p.CountryOrRegion != "" {
		q.Set("countryOrRegion", p.CountryOrRegion)
	}
	if p.Entity != "" {
		q.Set("entity", p.Entity)
	}
	if p.Limit > 0 {
		q.Set("limit", strconv.Itoa(p.Limit))
	}
	var env ListResponse[GeoLocation]
	if err := c.DoJSONWithQuery(ctx, http.MethodGet, "search/geo", q, nil, &env); err != nil {
		return nil, err
	}
	return env.Result, nil
}
