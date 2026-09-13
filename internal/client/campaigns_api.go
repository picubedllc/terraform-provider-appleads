// Copyright (c) 2026 Pi Cubed LLC
// SPDX-License-Identifier: MIT

package client

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
)

// CreateCampaign creates a campaign via POST /campaigns.
func (c *Client) CreateCampaign(ctx context.Context, in *CampaignCreate) (*Campaign, error) {
	if in == nil {
		return nil, fmt.Errorf("campaign create payload is required")
	}
	var env Response[Campaign]
	if err := c.DoJSON(ctx, http.MethodPost, "campaigns", in, &env); err != nil {
		return nil, err
	}
	return &env.Data, nil
}

// GetCampaign fetches a campaign by ID via GET /campaigns/{id}.
// Apple returns soft-deleted campaigns with Deleted=true rather than 404.
func (c *Client) GetCampaign(ctx context.Context, id int64) (*Campaign, error) {
	var env Response[Campaign]
	path := fmt.Sprintf("campaigns/%d", id)
	if err := c.DoJSON(ctx, http.MethodGet, path, nil, &env); err != nil {
		return nil, err
	}
	return &env.Data, nil
}

// UpdateCampaign updates mutable campaign fields via PUT /campaigns/{id}.
// Request body is wrapped as {"campaign":{...}} per Apple Ads API v5.
func (c *Client) UpdateCampaign(ctx context.Context, id int64, in *CampaignUpdate) (*Campaign, error) {
	if in == nil {
		return nil, fmt.Errorf("campaign update payload is required")
	}
	var env Response[Campaign]
	path := fmt.Sprintf("campaigns/%d", id)
	if err := c.DoJSON(ctx, http.MethodPut, path, campaignUpdateEnvelope{Campaign: in}, &env); err != nil {
		return nil, err
	}
	return &env.Data, nil
}

// DeleteCampaign archives a campaign via DELETE /campaigns/{id} (Apple soft-delete).
func (c *Client) DeleteCampaign(ctx context.Context, id int64) error {
	path := fmt.Sprintf("campaigns/%d", id)
	return c.DoJSON(ctx, http.MethodDelete, path, nil, nil)
}

// ParseCampaignID converts a Terraform campaign id string to int64.
func ParseCampaignID(id string) (int64, error) {
	v, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid campaign id %q: %w", id, err)
	}
	return v, nil
}
