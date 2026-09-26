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

// Ad statuses (Apple Ads Campaign Management API v5).
const (
	AdStatusEnabled = "ENABLED"
	AdStatusPaused  = "PAUSED"
)

// Ad is an Apple Ads ad — the assignment of a creative to an ad group (API v5).
//
// Mutability:
//
//	Immutable: CampaignID, AdGroupID (parent identity)
//	Mutable: Name, Status, CreativeID (update may rotate the creative)
//	Computed: ID, OrgID, CreativeType, ServingStatus, ServingStateReasons,
//	  CreationTime, ModificationTime, Deleted
type Ad struct {
	ID                  int64    `json:"id,omitempty"`
	OrgID               int64    `json:"orgId,omitempty"`
	CampaignID          int64    `json:"campaignId,omitempty"`
	AdGroupID           int64    `json:"adGroupId,omitempty"`
	Name                string   `json:"name,omitempty"`
	CreativeID          int64    `json:"creativeId,omitempty"`
	CreativeType        string   `json:"creativeType,omitempty"`
	Status              string   `json:"status,omitempty"`
	ServingStatus       string   `json:"servingStatus,omitempty"`
	ServingStateReasons []string `json:"servingStateReasons,omitempty"`
	CreationTime        string   `json:"creationTime,omitempty"`
	ModificationTime    string   `json:"modificationTime,omitempty"`
	Deleted             bool     `json:"deleted,omitempty"`
}

// AdCreate is the POST .../ads body.
type AdCreate struct {
	Name       string `json:"name"`
	CreativeID int64  `json:"creativeId"`
	Status     string `json:"status"`
}

// AdUpdate is the PUT .../ads/{adId} body (mutable subset).
type AdUpdate struct {
	Name       string `json:"name,omitempty"`
	Status     string `json:"status,omitempty"`
	CreativeID int64  `json:"creativeId,omitempty"`
}

// CreateAd creates an ad under an ad group via POST .../ads.
func (c *Client) CreateAd(ctx context.Context, campaignID, adGroupID int64, in *AdCreate) (*Ad, error) {
	if in == nil {
		return nil, fmt.Errorf("ad create payload is required")
	}
	if campaignID <= 0 || adGroupID <= 0 {
		return nil, fmt.Errorf("campaignId and adGroupId must be positive integers")
	}
	if strings.TrimSpace(in.Name) == "" {
		return nil, fmt.Errorf("name is required")
	}
	if in.CreativeID <= 0 {
		return nil, fmt.Errorf("creativeId must be a positive integer")
	}
	if strings.TrimSpace(in.Status) == "" {
		return nil, fmt.Errorf("status is required")
	}

	path := fmt.Sprintf("campaigns/%d/adgroups/%d/ads", campaignID, adGroupID)
	var env Response[Ad]
	if err := c.DoJSON(ctx, http.MethodPost, path, in, &env); err != nil {
		return nil, err
	}
	return &env.Data, nil
}

// GetAd fetches an ad via GET .../ads/{adId}.
func (c *Client) GetAd(ctx context.Context, campaignID, adGroupID, adID int64) (*Ad, error) {
	path := fmt.Sprintf("campaigns/%d/adgroups/%d/ads/%d", campaignID, adGroupID, adID)
	var env Response[Ad]
	if err := c.DoJSON(ctx, http.MethodGet, path, nil, &env); err != nil {
		return nil, err
	}
	return &env.Data, nil
}

// ListAds lists ads in an ad group via GET .../ads (paginated).
func (c *Client) ListAds(ctx context.Context, campaignID, adGroupID int64) ([]Ad, error) {
	path := fmt.Sprintf("campaigns/%d/adgroups/%d/ads", campaignID, adGroupID)
	return FetchAllPages[Ad](ctx, c, http.MethodGet, path, 1000, nil)
}

// UpdateAd updates mutable ad fields via PUT .../ads/{adId}.
func (c *Client) UpdateAd(ctx context.Context, campaignID, adGroupID, adID int64, in *AdUpdate) (*Ad, error) {
	if in == nil {
		return nil, fmt.Errorf("ad update payload is required")
	}
	path := fmt.Sprintf("campaigns/%d/adgroups/%d/ads/%d", campaignID, adGroupID, adID)
	var env Response[Ad]
	if err := c.DoJSON(ctx, http.MethodPut, path, in, &env); err != nil {
		return nil, err
	}
	return &env.Data, nil
}

// DeleteAd deletes an ad assignment via DELETE .../ads/{adId}.
func (c *Client) DeleteAd(ctx context.Context, campaignID, adGroupID, adID int64) error {
	path := fmt.Sprintf("campaigns/%d/adgroups/%d/ads/%d", campaignID, adGroupID, adID)
	return c.DoJSON(ctx, http.MethodDelete, path, nil, nil)
}

// FindAdsInCampaign finds ads via POST /campaigns/{campaignId}/ads/find.
func (c *Client) FindAdsInCampaign(ctx context.Context, campaignID int64, sel Selector) ([]Ad, error) {
	path := fmt.Sprintf("campaigns/%d/ads/find", campaignID)
	var env ListResponse[Ad]
	if err := c.DoJSON(ctx, http.MethodPost, path, sel, &env); err != nil {
		return nil, err
	}
	return env.Data, nil
}

// FindAds finds ads org-wide via POST /ads/find.
func (c *Client) FindAds(ctx context.Context, sel Selector) ([]Ad, error) {
	var env ListResponse[Ad]
	if err := c.DoJSON(ctx, http.MethodPost, "ads/find", sel, &env); err != nil {
		return nil, err
	}
	return env.Data, nil
}

// FindAdByID locates an ad by id within a campaign (all ad groups).
func (c *Client) FindAdByID(ctx context.Context, campaignID, adID int64) (*Ad, error) {
	body := Selector{
		Conditions: []SelectorCondition{{
			Field:    "id",
			Operator: "EQUALS",
			Values:   []string{strconv.FormatInt(adID, 10)},
		}},
		Pagination: &SelectorPagination{Offset: 0, Limit: 1},
	}
	found, err := c.FindAdsInCampaign(ctx, campaignID, body)
	if err != nil {
		return nil, err
	}
	if len(found) == 0 {
		return nil, &APIError{
			StatusCode: http.StatusNotFound,
			Code:       "NOT_FOUND",
			Message:    fmt.Sprintf("ad %d not found in campaign %d", adID, campaignID),
		}
	}
	return &found[0], nil
}

// ParseAdImportID parses:
//   - "campaignID/adGroupID/adID" (preferred)
//   - "adGroupID/adID" (campaign resolved via FindAdGroupByID)
func ParseAdImportID(id string) (campaignID, adGroupID, adID int64, err error) {
	parts := strings.Split(id, "/")
	switch len(parts) {
	case 2:
		adGroupID, err = strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			return 0, 0, 0, err
		}
		adID, err = strconv.ParseInt(parts[1], 10, 64)
		return 0, adGroupID, adID, err
	case 3:
		campaignID, err = strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			return 0, 0, 0, err
		}
		adGroupID, err = strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			return 0, 0, 0, err
		}
		adID, err = strconv.ParseInt(parts[2], 10, 64)
		return campaignID, adGroupID, adID, err
	default:
		return 0, 0, 0, fmt.Errorf("invalid ad import id %q; expected ad_group_id/ad_id or campaign_id/ad_group_id/ad_id", id)
	}
}
