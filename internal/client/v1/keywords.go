// SPDX-License-Identifier: MIT

package v1

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
)

// Keyword is a Platform API v1 targeting keyword.
type Keyword struct {
	ID               int64  `json:"id,omitempty"`
	AdAccountID      int64  `json:"adAccountId,omitempty"`
	CampaignID       int64  `json:"campaignId,omitempty"`
	AdGroupID        int64  `json:"adGroupId,omitempty"`
	Text             string `json:"text,omitempty"`
	Status           string `json:"status,omitempty"`
	MatchType        string `json:"matchType,omitempty"`
	Bid              *Money `json:"bid,omitempty"`
	DisplayStatus    string `json:"displayStatus,omitempty"`
	CreationTime     string `json:"creationTime,omitempty"`
	ModificationTime string `json:"modificationTime,omitempty"`
	Deleted          bool   `json:"deleted,omitempty"`
}

type KeywordCreate struct {
	AdGroupID int64  `json:"adGroupId"`
	Text      string `json:"text"`
	Status    string `json:"status,omitempty"`
	MatchType string `json:"matchType,omitempty"`
	Bid       *Money `json:"bid,omitempty"`
}

type KeywordUpdate struct {
	Status string `json:"status,omitempty"`
	Bid    *Money `json:"bid,omitempty"`
}

func (c *Client) CreateKeyword(ctx context.Context, in *KeywordCreate) (*Keyword, error) {
	if in == nil {
		return nil, fmt.Errorf("keyword create payload is required")
	}
	var env Response[Keyword]
	if err := c.DoJSON(ctx, http.MethodPost, "keywords", in, &env); err != nil {
		return nil, err
	}
	return &env.Result, nil
}

func (c *Client) GetKeyword(ctx context.Context, id int64) (*Keyword, error) {
	var env Response[Keyword]
	path := fmt.Sprintf("keywords/%d", id)
	if err := c.DoJSON(ctx, http.MethodGet, path, nil, &env); err != nil {
		return nil, err
	}
	return &env.Result, nil
}

func (c *Client) UpdateKeyword(ctx context.Context, id int64, in *KeywordUpdate) (*Keyword, error) {
	if in == nil {
		return nil, fmt.Errorf("keyword update payload is required")
	}
	var env Response[Keyword]
	path := fmt.Sprintf("keywords/%d", id)
	if err := c.DoJSON(ctx, http.MethodPut, path, in, &env); err != nil {
		return nil, err
	}
	return &env.Result, nil
}

func (c *Client) DeleteKeyword(ctx context.Context, id int64) error {
	path := fmt.Sprintf("keywords/%d", id)
	return c.DoJSON(ctx, http.MethodDelete, path, nil, nil)
}

func (c *Client) QueryKeywords(ctx context.Context, q QueryRequest) (*PageResult[Keyword], error) {
	return QueryPage[Keyword](ctx, c, "keywords/query", q)
}

func ParseKeywordID(id string) (int64, error) {
	v, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid keyword id %q: %w", id, err)
	}
	return v, nil
}
