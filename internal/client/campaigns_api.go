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

// CreateCampaign creates a campaign via POST /campaigns.
func (c *Client) CreateCampaign(ctx context.Context, in *CampaignCreate) (*Campaign, error) {
	if in == nil {
		return nil, fmt.Errorf("campaign create payload is required")
	}
	payload := *in
	if payload.OrgID == 0 {
		if id, err := strconv.ParseInt(strings.TrimSpace(c.orgID), 10, 64); err == nil && id != 0 {
			payload.OrgID = id
		}
	}
	var env Response[Campaign]
	if err := c.DoJSON(ctx, http.MethodPost, "campaigns", &payload, &env); err != nil {
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
// When in.ClearGeoTargetingOnCountryOrRegionChange is set (required to change
// countriesOrRegions), the flag is sent on the envelope, not inside campaign.
func (c *Client) UpdateCampaign(ctx context.Context, id int64, in *CampaignUpdate) (*Campaign, error) {
	if in == nil {
		return nil, fmt.Errorf("campaign update payload is required")
	}
	var env Response[Campaign]
	path := fmt.Sprintf("campaigns/%d", id)
	body := campaignUpdateEnvelope{Campaign: in}
	if in.ClearGeoTargetingOnCountryOrRegionChange {
		body.ClearGeoTargetingOnCountryOrRegionChange = true
	}
	if err := c.DoJSON(ctx, http.MethodPut, path, body, &env); err != nil {
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
