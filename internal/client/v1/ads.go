// SPDX-License-Identifier: MIT

package v1

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
)

// Ad links a creative to an ad group.
type Ad struct {
	ID                          int64    `json:"id,omitempty"`
	AdAccountID                 int64    `json:"adAccountId,omitempty"`
	CampaignID                  int64    `json:"campaignId,omitempty"`
	AdGroupID                   int64    `json:"adGroupId,omitempty"`
	Name                        string   `json:"name,omitempty"`
	Status                      string   `json:"status,omitempty"`
	CreativeID                  int64    `json:"creativeId,omitempty"`
	DisplayStatus               string   `json:"displayStatus,omitempty"`
	SystemStatus                string   `json:"systemStatus,omitempty"`
	SystemStatusReasons         []string `json:"systemStatusReasons,omitempty"`
	SystemStatusLimitingReasons []string `json:"systemStatusLimitingReasons,omitempty"`
	CreationTime                string   `json:"creationTime,omitempty"`
	ModificationTime            string   `json:"modificationTime,omitempty"`
	Deleted                     bool     `json:"deleted,omitempty"`
}

type AdCreate struct {
	AdAccountID int64  `json:"adAccountId,omitempty"`
	CampaignID  int64  `json:"campaignId"`
	AdGroupID   int64  `json:"adGroupId"`
	Name        string `json:"name"`
	CreativeID  int64  `json:"creativeId"`
	Status      string `json:"status,omitempty"`
}

type AdUpdate struct {
	Name   string `json:"name,omitempty"`
	Status string `json:"status,omitempty"`
}

func (c *Client) CreateAd(ctx context.Context, in *AdCreate) (*Ad, error) {
	if in == nil {
		return nil, fmt.Errorf("ad create payload is required")
	}
	payload := *in
	if payload.AdAccountID == 0 {
		if id, err := strconv.ParseInt(c.adAccountID, 10, 64); err == nil {
			payload.AdAccountID = id
		}
	}
	var env Response[Ad]
	if err := c.DoJSON(ctx, http.MethodPost, "ads", &payload, &env); err != nil {
		return nil, err
	}
	return &env.Result, nil
}

func (c *Client) GetAd(ctx context.Context, id int64) (*Ad, error) {
	var env Response[Ad]
	path := fmt.Sprintf("ads/%d", id)
	if err := c.DoJSON(ctx, http.MethodGet, path, nil, &env); err != nil {
		return nil, err
	}
	return &env.Result, nil
}

func (c *Client) UpdateAd(ctx context.Context, id int64, in *AdUpdate) (*Ad, error) {
	if in == nil {
		return nil, fmt.Errorf("ad update payload is required")
	}
	var env Response[Ad]
	path := fmt.Sprintf("ads/%d", id)
	if err := c.DoJSON(ctx, http.MethodPut, path, in, &env); err != nil {
		return nil, err
	}
	return &env.Result, nil
}

func (c *Client) DeleteAd(ctx context.Context, id int64) error {
	path := fmt.Sprintf("ads/%d", id)
	return c.DoJSON(ctx, http.MethodDelete, path, nil, nil)
}

func (c *Client) QueryAds(ctx context.Context, q QueryRequest) (*PageResult[Ad], error) {
	return QueryPage[Ad](ctx, c, "ads/query", q)
}

func ParseAdID(id string) (int64, error) {
	v, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid ad id %q: %w", id, err)
	}
	return v, nil
}
