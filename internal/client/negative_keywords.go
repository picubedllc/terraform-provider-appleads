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

// NegativeKeyword is an Apple Ads negative keyword (campaign- or ad-group-scoped).
//
// Mutability:
//
//	Immutable: CampaignID/AdGroupID scope, Text, MatchType
//	Mutable: Status (where Apple allows)
//	Computed: ID, ModificationTime, Deleted
type NegativeKeyword struct {
	ID               int64  `json:"id,omitempty"`
	CampaignID       int64  `json:"campaignId,omitempty"`
	AdGroupID        int64  `json:"adGroupId,omitempty"`
	Text             string `json:"text,omitempty"`
	MatchType        string `json:"matchType,omitempty"`
	Status           string `json:"status,omitempty"`
	ModificationTime string `json:"modificationTime,omitempty"`
	Deleted          bool   `json:"deleted,omitempty"`
}

// NegativeKeywordCreate is one element of a negative-keyword bulk create body.
type NegativeKeywordCreate struct {
	Text      string `json:"text"`
	MatchType string `json:"matchType"`
	Status    string `json:"status,omitempty"`
}

// NegativeKeywordUpdate is one element of a negative-keyword bulk update body.
// Text and MatchType are omitted — change them by creating a new negative keyword.
type NegativeKeywordUpdate struct {
	ID     int64  `json:"id"`
	Status string `json:"status,omitempty"`
}

// CreateCampaignNegativeKeyword creates a campaign-scoped negative keyword.
func (c *Client) CreateCampaignNegativeKeyword(ctx context.Context, campaignID int64, in *NegativeKeywordCreate) (*NegativeKeyword, error) {
	var env ListResponse[NegativeKeyword]
	path := fmt.Sprintf("campaigns/%d/negativekeywords/bulk", campaignID)
	if err := c.DoJSON(ctx, http.MethodPost, path, []*NegativeKeywordCreate{in}, &env); err != nil {
		return nil, err
	}
	if len(env.Data) == 0 {
		return nil, fmt.Errorf("create campaign negative keyword: empty response data")
	}
	return &env.Data[0], nil
}

// CreateAdGroupNegativeKeyword creates an ad-group-scoped negative keyword.
func (c *Client) CreateAdGroupNegativeKeyword(ctx context.Context, campaignID, adGroupID int64, in *NegativeKeywordCreate) (*NegativeKeyword, error) {
	var env ListResponse[NegativeKeyword]
	path := fmt.Sprintf("campaigns/%d/adgroups/%d/negativekeywords/bulk", campaignID, adGroupID)
	if err := c.DoJSON(ctx, http.MethodPost, path, []*NegativeKeywordCreate{in}, &env); err != nil {
		return nil, err
	}
	if len(env.Data) == 0 {
		return nil, fmt.Errorf("create ad group negative keyword: empty response data")
	}
	return &env.Data[0], nil
}

// GetCampaignNegativeKeyword retrieves a campaign-scoped negative keyword.
func (c *Client) GetCampaignNegativeKeyword(ctx context.Context, campaignID, keywordID int64) (*NegativeKeyword, error) {
	var env Response[NegativeKeyword]
	path := fmt.Sprintf("campaigns/%d/negativekeywords/%d", campaignID, keywordID)
	if err := c.DoJSON(ctx, http.MethodGet, path, nil, &env); err != nil {
		return nil, err
	}
	return &env.Data, nil
}

// GetAdGroupNegativeKeyword retrieves an ad-group-scoped negative keyword.
func (c *Client) GetAdGroupNegativeKeyword(ctx context.Context, campaignID, adGroupID, keywordID int64) (*NegativeKeyword, error) {
	var env Response[NegativeKeyword]
	path := fmt.Sprintf("campaigns/%d/adgroups/%d/negativekeywords/%d", campaignID, adGroupID, keywordID)
	if err := c.DoJSON(ctx, http.MethodGet, path, nil, &env); err != nil {
		return nil, err
	}
	return &env.Data, nil
}

// UpdateCampaignNegativeKeyword updates mutable fields on a campaign-scoped negative keyword.
func (c *Client) UpdateCampaignNegativeKeyword(ctx context.Context, campaignID int64, in *NegativeKeywordUpdate) (*NegativeKeyword, error) {
	var env ListResponse[NegativeKeyword]
	path := fmt.Sprintf("campaigns/%d/negativekeywords/bulk", campaignID)
	if err := c.DoJSON(ctx, http.MethodPut, path, []*NegativeKeywordUpdate{in}, &env); err != nil {
		return nil, err
	}
	if len(env.Data) == 0 {
		return nil, fmt.Errorf("update campaign negative keyword: empty response data")
	}
	return &env.Data[0], nil
}

// UpdateAdGroupNegativeKeyword updates mutable fields on an ad-group-scoped negative keyword.
func (c *Client) UpdateAdGroupNegativeKeyword(ctx context.Context, campaignID, adGroupID int64, in *NegativeKeywordUpdate) (*NegativeKeyword, error) {
	var env ListResponse[NegativeKeyword]
	path := fmt.Sprintf("campaigns/%d/adgroups/%d/negativekeywords/bulk", campaignID, adGroupID)
	if err := c.DoJSON(ctx, http.MethodPut, path, []*NegativeKeywordUpdate{in}, &env); err != nil {
		return nil, err
	}
	if len(env.Data) == 0 {
		return nil, fmt.Errorf("update ad group negative keyword: empty response data")
	}
	return &env.Data[0], nil
}

// DeleteCampaignNegativeKeyword soft-deletes a campaign-scoped negative keyword via bulk delete.
func (c *Client) DeleteCampaignNegativeKeyword(ctx context.Context, campaignID, keywordID int64) error {
	path := fmt.Sprintf("campaigns/%d/negativekeywords/delete/bulk", campaignID)
	return c.DoJSON(ctx, http.MethodPost, path, []int64{keywordID}, nil)
}

// DeleteAdGroupNegativeKeyword soft-deletes an ad-group-scoped negative keyword via bulk delete.
func (c *Client) DeleteAdGroupNegativeKeyword(ctx context.Context, campaignID, adGroupID, keywordID int64) error {
	path := fmt.Sprintf("campaigns/%d/adgroups/%d/negativekeywords/delete/bulk", campaignID, adGroupID)
	return c.DoJSON(ctx, http.MethodPost, path, []int64{keywordID}, nil)
}

// FindCampaignNegativeKeyword locates a campaign-scoped negative keyword by id.
func (c *Client) FindCampaignNegativeKeyword(ctx context.Context, campaignID, keywordID int64) (*NegativeKeyword, error) {
	body := Selector{
		Conditions: []SelectorCondition{{
			Field:    "id",
			Operator: "EQUALS",
			Values:   []string{strconv.FormatInt(keywordID, 10)},
		}},
		Pagination: &SelectorPagination{Offset: 0, Limit: 1},
	}
	var env ListResponse[NegativeKeyword]
	path := fmt.Sprintf("campaigns/%d/negativekeywords/find", campaignID)
	if err := c.DoJSON(ctx, http.MethodPost, path, body, &env); err != nil {
		return nil, err
	}
	if len(env.Data) == 0 {
		return nil, &APIError{
			StatusCode: http.StatusNotFound,
			Code:       "NOT_FOUND",
			Message:    fmt.Sprintf("campaign negative keyword %d not found in campaign %d", keywordID, campaignID),
		}
	}
	return &env.Data[0], nil
}

// ParseNegativeKeywordImportID parses import ids:
//   - "campaign/<campaignID>/<keywordID>" for campaign-scoped
//   - "adgroup/<campaignID>/<adGroupID>/<keywordID>" for ad-group-scoped
//   - "adgroup/<adGroupID>/<keywordID>" (campaign resolved via FindAdGroupByID)
func ParseNegativeKeywordImportID(id string) (scope string, campaignID, adGroupID, keywordID int64, err error) {
	parts := strings.Split(id, "/")
	if len(parts) < 2 {
		return "", 0, 0, 0, fmt.Errorf("invalid negative keyword import id %q; expected campaign/<campaign_id>/<id> or adgroup/[campaign_id/]<ad_group_id>/<id>", id)
	}
	scope = strings.ToLower(parts[0])
	switch scope {
	case "campaign":
		if len(parts) != 3 {
			return "", 0, 0, 0, fmt.Errorf("invalid campaign negative keyword import id %q; expected campaign/<campaign_id>/<id>", id)
		}
		campaignID, err = strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			return "", 0, 0, 0, err
		}
		keywordID, err = strconv.ParseInt(parts[2], 10, 64)
		return scope, campaignID, 0, keywordID, err
	case "adgroup", "ad_group":
		scope = "adgroup"
		switch len(parts) {
		case 3:
			adGroupID, err = strconv.ParseInt(parts[1], 10, 64)
			if err != nil {
				return "", 0, 0, 0, err
			}
			keywordID, err = strconv.ParseInt(parts[2], 10, 64)
			return scope, 0, adGroupID, keywordID, err
		case 4:
			campaignID, err = strconv.ParseInt(parts[1], 10, 64)
			if err != nil {
				return "", 0, 0, 0, err
			}
			adGroupID, err = strconv.ParseInt(parts[2], 10, 64)
			if err != nil {
				return "", 0, 0, 0, err
			}
			keywordID, err = strconv.ParseInt(parts[3], 10, 64)
			return scope, campaignID, adGroupID, keywordID, err
		default:
			return "", 0, 0, 0, fmt.Errorf("invalid ad group negative keyword import id %q", id)
		}
	default:
		return "", 0, 0, 0, fmt.Errorf("invalid negative keyword import id %q; scope must be campaign or adgroup", id)
	}
}
