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

// Product page visibility states (Apple Ads Campaign Management API v5).
const (
	ProductPageStateHidden  = "HIDDEN"
	ProductPageStateVisible = "VISIBLE"
)

// Device classes for product page locale details.
const (
	DeviceClassIPad   = "IPAD"
	DeviceClassIPhone = "IPHONE"
)

// ProductPage is custom product page metadata from App Store Connect,
// exposed read-only via Apple Ads GET /apps/{adamId}/product-pages.
type ProductPage struct {
	ID               string `json:"id"`
	Name             string `json:"name,omitempty"`
	State            string `json:"state,omitempty"`
	AdamID           int64  `json:"adamId,omitempty"`
	DeepLink         string `json:"deepLink,omitempty"`
	CreationTime     string `json:"creationTime,omitempty"`
	ModificationTime string `json:"modificationTime,omitempty"`
}

// ProductPageListParams filters GET /apps/{adamId}/product-pages.
type ProductPageListParams struct {
	Name   string
	States []string
}

// ProductPageLocale is localized product page metadata
// (GET /apps/{adamId}/product-pages/{productPageId}/locale-details).
type ProductPageLocale struct {
	AdamID           int64  `json:"adamId,omitempty"`
	AppName          string `json:"appName,omitempty"`
	DeviceClasses    string `json:"deviceClasses,omitempty"`
	Language         string `json:"language,omitempty"`
	LanguageCode     string `json:"languageCode,omitempty"`
	ProductPageID    string `json:"productPageId,omitempty"`
	PromotionalText  string `json:"promotionalText,omitempty"`
	ShortDescription string `json:"shortDescription,omitempty"`
	SubTitle         string `json:"subTitle,omitempty"`
}

// ProductPageLocaleListParams filters locale-details list requests.
type ProductPageLocaleListParams struct {
	DeviceClasses []string
	LanguageCodes []string
	Languages     []string
	Expand        *bool
}

// LocaleInfo is a language / language-code pair on a country-or-region record.
type LocaleInfo struct {
	Language     string `json:"language,omitempty"`
	LanguageCode string `json:"languageCode,omitempty"`
}

// CountryOrRegion is a supported Apple Ads country or region
// (GET /countries-or-regions).
type CountryOrRegion struct {
	CountryOrRegion    string       `json:"countryOrRegion"`
	DefaultLanguages   []LocaleInfo `json:"defaultLanguages,omitempty"`
	SupportedLanguages []LocaleInfo `json:"supportedLanguages,omitempty"`
}

// ListProductPages returns custom product pages for an app (read-only).
// Pages are owned in App Store Connect; the Ads API only lists metadata.
func (c *Client) ListProductPages(ctx context.Context, adamID int64, p ProductPageListParams) ([]ProductPage, error) {
	if adamID <= 0 {
		return nil, fmt.Errorf("adamId must be a positive integer")
	}

	q := url.Values{}
	if name := strings.TrimSpace(p.Name); name != "" {
		q.Set("name", name)
	}
	if states := normalizeCSVFilter(p.States); states != "" {
		q.Set("states", states)
	}

	path := fmt.Sprintf("apps/%d/product-pages", adamID)
	var env ListResponse[ProductPage]
	if err := c.DoJSONWithQuery(ctx, http.MethodGet, path, q, nil, &env); err != nil {
		return nil, err
	}
	return env.Data, nil
}

// GetProductPage fetches one product page by adamId and productPageId (UUID string).
func (c *Client) GetProductPage(ctx context.Context, adamID int64, productPageID string) (*ProductPage, error) {
	if adamID <= 0 {
		return nil, fmt.Errorf("adamId must be a positive integer")
	}
	productPageID = strings.TrimSpace(productPageID)
	if productPageID == "" {
		return nil, fmt.Errorf("productPageId is required")
	}

	path := fmt.Sprintf("apps/%d/product-pages/%s", adamID, url.PathEscape(productPageID))
	var env Response[ProductPage]
	if err := c.DoJSON(ctx, http.MethodGet, path, nil, &env); err != nil {
		return nil, err
	}
	return &env.Data, nil
}

// ListProductPageLocales returns locale details for a product page (read-only).
func (c *Client) ListProductPageLocales(ctx context.Context, adamID int64, productPageID string, p ProductPageLocaleListParams) ([]ProductPageLocale, error) {
	if adamID <= 0 {
		return nil, fmt.Errorf("adamId must be a positive integer")
	}
	productPageID = strings.TrimSpace(productPageID)
	if productPageID == "" {
		return nil, fmt.Errorf("productPageId is required")
	}

	q := url.Values{}
	if v := normalizeCSVFilter(p.DeviceClasses); v != "" {
		q.Set("deviceClasses", v)
	}
	if v := normalizeCSVFilter(p.LanguageCodes); v != "" {
		q.Set("languageCodes", v)
	}
	if v := normalizeCSVFilter(p.Languages); v != "" {
		q.Set("languages", v)
	}
	if p.Expand != nil {
		q.Set("expand", strconv.FormatBool(*p.Expand))
	}

	path := fmt.Sprintf("apps/%d/product-pages/%s/locale-details", adamID, url.PathEscape(productPageID))
	var env ListResponse[ProductPageLocale]
	if err := c.DoJSONWithQuery(ctx, http.MethodGet, path, q, nil, &env); err != nil {
		return nil, err
	}
	return env.Data, nil
}

// ListCountriesOrRegions returns supported countries or regions (read-only).
// When countriesOrRegions is non-empty, Apple filters to those ISO codes.
func (c *Client) ListCountriesOrRegions(ctx context.Context, countriesOrRegions []string) ([]CountryOrRegion, error) {
	q := url.Values{}
	if v := normalizeCSVFilter(countriesOrRegions); v != "" {
		q.Set("countriesOrRegions", v)
	}

	var env ListResponse[CountryOrRegion]
	if err := c.DoJSONWithQuery(ctx, http.MethodGet, "countries-or-regions", q, nil, &env); err != nil {
		return nil, err
	}
	return env.Data, nil
}

func normalizeCSVFilter(values []string) string {
	var parts []string
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v != "" {
			parts = append(parts, v)
		}
	}
	return strings.Join(parts, ",")
}
