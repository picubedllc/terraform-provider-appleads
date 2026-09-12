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

// Keyword is an Apple Ads targeting keyword (API v5).
//
// Mutability:
//
//	Immutable: AdGroupID, Text, MatchType (changing text/match type is a different keyword)
//	Mutable: Status, BidAmount
//	Computed: ID, CampaignID, ModificationTime, Deleted, MatchType (readback), Text (readback)
type Keyword struct {
	ID               int64  `json:"id,omitempty"`
	CampaignID       int64  `json:"campaignId,omitempty"`
	AdGroupID        int64  `json:"adGroupId,omitempty"`
	Text             string `json:"text,omitempty"`
	MatchType        string `json:"matchType,omitempty"`
	Status           string `json:"status,omitempty"`
	BidAmount        *Money `json:"bidAmount,omitempty"`
	ModificationTime string `json:"modificationTime,omitempty"`
	Deleted          bool   `json:"deleted,omitempty"`
}

// KeywordCreate is one element of POST .../targetingkeywords/bulk.
type KeywordCreate struct {
	Text      string `json:"text"`
	MatchType string `json:"matchType"`
	Status    string `json:"status,omitempty"`
	BidAmount *Money `json:"bidAmount,omitempty"`
}

// KeywordUpdate is one element of PUT .../targetingkeywords/bulk.
// Text and MatchType are omitted — Apple does not allow changing them in place.
type KeywordUpdate struct {
	ID        int64  `json:"id"`
	Status    string `json:"status,omitempty"`
	BidAmount *Money `json:"bidAmount,omitempty"`
}

// CreateKeyword creates a single targeting keyword via the bulk endpoint.
func (c *Client) CreateKeyword(ctx context.Context, campaignID, adGroupID int64, in *KeywordCreate) (*Keyword, error) {
	var env ListResponse[Keyword]
	path := fmt.Sprintf("campaigns/%d/adgroups/%d/targetingkeywords/bulk", campaignID, adGroupID)
	if err := c.DoJSON(ctx, http.MethodPost, path, []*KeywordCreate{in}, &env); err != nil {
		return nil, err
	}
	if len(env.Data) == 0 {
		return nil, fmt.Errorf("create keyword: empty response data")
	}
	return &env.Data[0], nil
}

// GetKeyword retrieves a targeting keyword.
func (c *Client) GetKeyword(ctx context.Context, campaignID, adGroupID, keywordID int64) (*Keyword, error) {
	var env Response[Keyword]
	path := fmt.Sprintf("campaigns/%d/adgroups/%d/targetingkeywords/%d", campaignID, adGroupID, keywordID)
	if err := c.DoJSON(ctx, http.MethodGet, path, nil, &env); err != nil {
		return nil, err
	}
	return &env.Data, nil
}

// UpdateKeyword updates mutable fields (status, bid) via the bulk update endpoint.
func (c *Client) UpdateKeyword(ctx context.Context, campaignID, adGroupID int64, in *KeywordUpdate) (*Keyword, error) {
	var env ListResponse[Keyword]
	path := fmt.Sprintf("campaigns/%d/adgroups/%d/targetingkeywords/bulk", campaignID, adGroupID)
	if err := c.DoJSON(ctx, http.MethodPut, path, []*KeywordUpdate{in}, &env); err != nil {
		return nil, err
	}
	if len(env.Data) == 0 {
		return nil, fmt.Errorf("update keyword: empty response data")
	}
	return &env.Data[0], nil
}

// DeleteKeyword soft-deletes a targeting keyword.
func (c *Client) DeleteKeyword(ctx context.Context, campaignID, adGroupID, keywordID int64) error {
	path := fmt.Sprintf("campaigns/%d/adgroups/%d/targetingkeywords/%d", campaignID, adGroupID, keywordID)
	return c.DoJSON(ctx, http.MethodDelete, path, nil, nil)
}

// FindKeywordInCampaign locates a keyword by id within a campaign (all ad groups).
func (c *Client) FindKeywordInCampaign(ctx context.Context, campaignID, keywordID int64) (*Keyword, error) {
	body := Selector{
		Conditions: []SelectorCondition{{
			Field:    "id",
			Operator: "EQUALS",
			Values:   []string{strconv.FormatInt(keywordID, 10)},
		}},
		Pagination: &SelectorPagination{Offset: 0, Limit: 1},
	}
	var env ListResponse[Keyword]
	path := fmt.Sprintf("campaigns/%d/adgroups/targetingkeywords/find", campaignID)
	if err := c.DoJSON(ctx, http.MethodPost, path, body, &env); err != nil {
		return nil, err
	}
	if len(env.Data) == 0 {
		return nil, &APIError{
			StatusCode: http.StatusNotFound,
			Code:       "NOT_FOUND",
			Message:    fmt.Sprintf("keyword %d not found in campaign %d", keywordID, campaignID),
		}
	}
	return &env.Data[0], nil
}

// ParseKeywordImportID parses:
//   - "campaignID/adGroupID/keywordID" (preferred)
//   - "adGroupID/keywordID" (campaign resolved via FindAdGroupByID)
func ParseKeywordImportID(id string) (campaignID, adGroupID, keywordID int64, err error) {
	parts := strings.Split(id, "/")
	switch len(parts) {
	case 2:
		adGroupID, err = strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			return 0, 0, 0, err
		}
		keywordID, err = strconv.ParseInt(parts[1], 10, 64)
		return 0, adGroupID, keywordID, err
	case 3:
		campaignID, err = strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			return 0, 0, 0, err
		}
		adGroupID, err = strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			return 0, 0, 0, err
		}
		keywordID, err = strconv.ParseInt(parts[2], 10, 64)
		return campaignID, adGroupID, keywordID, err
	default:
		return 0, 0, 0, fmt.Errorf("invalid keyword import id %q; expected ad_group_id/keyword_id or campaign_id/ad_group_id/keyword_id", id)
	}
}
