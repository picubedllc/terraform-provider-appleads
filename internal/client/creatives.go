// Copyright (c) 2026 Pi Cubed LLC
// SPDX-License-Identifier: MIT

package client

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

// Creative types (Apple Ads Campaign Management API v5).
const (
	CreativeTypeCustomProductPage  = "CUSTOM_PRODUCT_PAGE"
	CreativeTypeDefaultProductPage = "DEFAULT_PRODUCT_PAGE"
	CreativeTypeCreativeSet        = "CREATIVE_SET" // legacy; create via product-page types in v5
)

// Creative is an Apple Ads creative (API v5).
//
// Mutability:
//
//	Immutable / create-only: AdamID, Name, Type, ProductPageID
//	  (API v5 has no creative update or delete endpoints)
//	Computed: ID, OrgID, State, StateReasons, CreationTime, ModificationTime,
//	  LanguageCode (legacy creative sets)
type Creative struct {
	ID               int64    `json:"id,omitempty"`
	OrgID            int64    `json:"orgId,omitempty"`
	AdamID           int64    `json:"adamId,omitempty"`
	Name             string   `json:"name,omitempty"`
	Type             string   `json:"type,omitempty"`
	State            string   `json:"state,omitempty"`
	StateReasons     []string `json:"stateReasons,omitempty"`
	ProductPageID    string   `json:"productPageId,omitempty"`
	LanguageCode     string   `json:"languageCode,omitempty"`
	CreationTime     string   `json:"creationTime,omitempty"`
	ModificationTime string   `json:"modificationTime,omitempty"`
}

// CreativeCreate is the POST /creatives body.
// For CUSTOM_PRODUCT_PAGE, ProductPageID is required.
// For DEFAULT_PRODUCT_PAGE, ProductPageID is optional.
type CreativeCreate struct {
	AdamID        int64  `json:"adamId"`
	Name          string `json:"name"`
	Type          string `json:"type"`
	ProductPageID string `json:"productPageId,omitempty"`
}

// CreateCreative creates a creative via POST /creatives.
func (c *Client) CreateCreative(ctx context.Context, in *CreativeCreate) (*Creative, error) {
	if in == nil {
		return nil, fmt.Errorf("creative create payload is required")
	}
	if in.AdamID <= 0 {
		return nil, fmt.Errorf("adamId must be a positive integer")
	}
	if strings.TrimSpace(in.Name) == "" {
		return nil, fmt.Errorf("name is required")
	}
	if strings.TrimSpace(in.Type) == "" {
		return nil, fmt.Errorf("type is required")
	}
	if in.Type == CreativeTypeCustomProductPage && strings.TrimSpace(in.ProductPageID) == "" {
		return nil, fmt.Errorf("productPageId is required when type is CUSTOM_PRODUCT_PAGE")
	}

	var env Response[Creative]
	if err := c.DoJSON(ctx, http.MethodPost, "creatives", in, &env); err != nil {
		return nil, err
	}
	return &env.Data, nil
}

// GetCreative fetches a creative by ID via GET /creatives/{creativeId}.
func (c *Client) GetCreative(ctx context.Context, creativeID int64) (*Creative, error) {
	if creativeID <= 0 {
		return nil, fmt.Errorf("creativeId must be a positive integer")
	}
	path := fmt.Sprintf("creatives/%d", creativeID)
	var env Response[Creative]
	if err := c.DoJSON(ctx, http.MethodGet, path, nil, &env); err != nil {
		return nil, err
	}
	return &env.Data, nil
}

// ListCreatives fetches all creatives via GET /creatives (paginated).
func (c *Client) ListCreatives(ctx context.Context) ([]Creative, error) {
	return FetchAllPages[Creative](ctx, c, http.MethodGet, "creatives", 1000, nil)
}

// FindCreatives finds creatives via POST /creatives/find.
func (c *Client) FindCreatives(ctx context.Context, sel Selector) ([]Creative, error) {
	var env ListResponse[Creative]
	if err := c.DoJSON(ctx, http.MethodPost, "creatives/find", sel, &env); err != nil {
		return nil, err
	}
	return env.Data, nil
}

// FindCreativeByID locates a creative by id within the organization.
func (c *Client) FindCreativeByID(ctx context.Context, creativeID int64) (*Creative, error) {
	body := Selector{
		Conditions: []SelectorCondition{{
			Field:    "id",
			Operator: "EQUALS",
			Values:   []string{strconv.FormatInt(creativeID, 10)},
		}},
		Pagination: &SelectorPagination{Offset: 0, Limit: 1},
	}
	found, err := c.FindCreatives(ctx, body)
	if err != nil {
		return nil, err
	}
	if len(found) == 0 {
		return nil, &APIError{
			StatusCode: http.StatusNotFound,
			Code:       "NOT_FOUND",
			Message:    fmt.Sprintf("creative %d not found", creativeID),
		}
	}
	return &found[0], nil
}

// ParseCreativeID parses a creative id string.
func ParseCreativeID(id string) (int64, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return 0, fmt.Errorf("creative id is required")
	}
	v, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid creative id %q: %w", id, err)
	}
	if v <= 0 {
		return 0, fmt.Errorf("creative id must be a positive integer")
	}
	return v, nil
}
