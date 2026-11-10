// SPDX-License-Identifier: MIT

package v1

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
)

// NegativeKeyword is campaign- or ad-group-scoped.
type NegativeKeyword struct {
	ID               int64  `json:"id,omitempty"`
	AdAccountID      int64  `json:"adAccountId,omitempty"`
	CampaignID       int64  `json:"campaignId,omitempty"`
	AdGroupID        int64  `json:"adGroupId,omitempty"`
	Text             string `json:"text,omitempty"`
	Status           string `json:"status,omitempty"`
	MatchType        string `json:"matchType,omitempty"`
	CreationTime     string `json:"creationTime,omitempty"`
	ModificationTime string `json:"modificationTime,omitempty"`
	Deleted          bool   `json:"deleted,omitempty"`
}

type NegativeKeywordCreate struct {
	CampaignID int64  `json:"campaignId,omitempty"`
	AdGroupID  int64  `json:"adGroupId,omitempty"`
	Text       string `json:"text"`
	Status     string `json:"status,omitempty"`
	MatchType  string `json:"matchType,omitempty"`
}

type NegativeKeywordUpdate struct {
	Status string `json:"status,omitempty"`
}

func (c *Client) CreateNegativeKeyword(ctx context.Context, in *NegativeKeywordCreate) (*NegativeKeyword, error) {
	if in == nil {
		return nil, fmt.Errorf("negative keyword create payload is required")
	}
	var env Response[NegativeKeyword]
	if err := c.DoJSON(ctx, http.MethodPost, "negative-keywords", in, &env); err != nil {
		return nil, err
	}
	return &env.Result, nil
}

func (c *Client) GetNegativeKeyword(ctx context.Context, id int64) (*NegativeKeyword, error) {
	var env Response[NegativeKeyword]
	path := fmt.Sprintf("negative-keywords/%d", id)
	if err := c.DoJSON(ctx, http.MethodGet, path, nil, &env); err != nil {
		return nil, err
	}
	return &env.Result, nil
}

func (c *Client) UpdateNegativeKeyword(ctx context.Context, id int64, in *NegativeKeywordUpdate) (*NegativeKeyword, error) {
	if in == nil {
		return nil, fmt.Errorf("negative keyword update payload is required")
	}
	var env Response[NegativeKeyword]
	path := fmt.Sprintf("negative-keywords/%d", id)
	if err := c.DoJSON(ctx, http.MethodPut, path, in, &env); err != nil {
		return nil, err
	}
	return &env.Result, nil
}

func (c *Client) DeleteNegativeKeyword(ctx context.Context, id int64) error {
	path := fmt.Sprintf("negative-keywords/%d", id)
	return c.DoJSON(ctx, http.MethodDelete, path, nil, nil)
}

func (c *Client) QueryNegativeKeywords(ctx context.Context, q QueryRequest) (*PageResult[NegativeKeyword], error) {
	return QueryPage[NegativeKeyword](ctx, c, "negative-keywords/query", q)
}

func ParseNegativeKeywordID(id string) (int64, error) {
	v, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid negative keyword id %q: %w", id, err)
	}
	return v, nil
}
