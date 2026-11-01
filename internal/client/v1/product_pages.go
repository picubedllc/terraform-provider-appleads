// SPDX-License-Identifier: MIT

package v1

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// ProductPage is a custom product page summary.
type ProductPage struct {
	ID     string `json:"id"`
	AdamID int64  `json:"adamId,omitempty"`
	Name   string `json:"name,omitempty"`
	State  string `json:"state,omitempty"`
}

// ProductPageLocale is locale detail for a product page.
type ProductPageLocale struct {
	LanguageCode string   `json:"languageCode,omitempty"`
	Language     string   `json:"language,omitempty"`
	AssetIDs     []string `json:"assetIds,omitempty"`
}

// CountryOrRegion metadata.
type CountryOrRegion struct {
	CountryOrRegion string `json:"countryOrRegion"`
	Name            string `json:"name,omitempty"`
}

func (c *Client) QueryProductPages(ctx context.Context, q QueryRequest) (*PageResult[ProductPage], error) {
	return QueryPage[ProductPage](ctx, c, "product-pages/query", q)
}

func (c *Client) GetProductPage(ctx context.Context, productPageID string) (*ProductPage, error) {
	var env Response[ProductPage]
	path := fmt.Sprintf("product-pages/%s", url.PathEscape(productPageID))
	if err := c.DoJSON(ctx, http.MethodGet, path, nil, &env); err != nil {
		return nil, err
	}
	return &env.Result, nil
}

func (c *Client) QueryProductPageLocales(ctx context.Context, q QueryRequest) (*PageResult[ProductPageLocale], error) {
	return QueryPage[ProductPageLocale](ctx, c, "product-pages/locale-details/query", q)
}

func (c *Client) ListCountriesOrRegions(ctx context.Context, countries []string) ([]CountryOrRegion, error) {
	q := url.Values{}
	for _, ctry := range countries {
		q.Add("countriesOrRegions", ctry)
	}
	var env ListResponse[CountryOrRegion]
	if err := c.DoJSONWithQuery(ctx, http.MethodGet, "countries-or-regions", q, nil, &env); err != nil {
		return nil, err
	}
	return env.Result, nil
}
