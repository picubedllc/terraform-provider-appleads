// SPDX-License-Identifier: MIT

package v1

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
)

// AdGroup is a Platform API v1 ad group.
type AdGroup struct {
	ID                        int64          `json:"id,omitempty"`
	AdAccountID               int64          `json:"adAccountId,omitempty"`
	CampaignID                int64          `json:"campaignId,omitempty"`
	Name                      string         `json:"name,omitempty"`
	Status                    string         `json:"status,omitempty"`
	PricingModel              string         `json:"pricingModel,omitempty"`
	AutomatedKeywordsOptIn    bool           `json:"automatedKeywordsOptIn,omitempty"`
	AutomatedKeywordsRequired bool           `json:"automatedKeywordsRequired,omitempty"`
	StartTime                 string         `json:"startTime,omitempty"`
	EndTime                   string         `json:"endTime,omitempty"`
	BidStrategy               *BidStrategy   `json:"bidStrategy,omitempty"`
	Targeting                 map[string]any `json:"targeting,omitempty"`
	DisplayStatus             string         `json:"displayStatus,omitempty"`
	SystemStatus              string         `json:"systemStatus,omitempty"`
	SystemStatusReasons       []string       `json:"systemStatusReasons,omitempty"`
	CreationTime              string         `json:"creationTime,omitempty"`
	ModificationTime          string         `json:"modificationTime,omitempty"`
	Deleted                   bool           `json:"deleted,omitempty"`
}

// AdGroupCreate is POST /adgroups.
type AdGroupCreate struct {
	CampaignID             int64          `json:"campaignId"`
	Name                   string         `json:"name"`
	Status                 string         `json:"status,omitempty"`
	PricingModel           string         `json:"pricingModel,omitempty"`
	AutomatedKeywordsOptIn bool           `json:"automatedKeywordsOptIn,omitempty"`
	StartTime              string         `json:"startTime,omitempty"`
	EndTime                string         `json:"endTime,omitempty"`
	BidStrategy            *BidStrategy   `json:"bidStrategy,omitempty"`
	Targeting              map[string]any `json:"targeting,omitempty"`
}

// AdGroupUpdate is PUT /adgroups/{id}.
type AdGroupUpdate struct {
	Name                   string         `json:"name,omitempty"`
	Status                 string         `json:"status,omitempty"`
	AutomatedKeywordsOptIn *bool          `json:"automatedKeywordsOptIn,omitempty"`
	StartTime              string         `json:"startTime,omitempty"`
	EndTime                string         `json:"endTime,omitempty"`
	BidStrategy            *BidStrategy   `json:"bidStrategy,omitempty"`
	Targeting              map[string]any `json:"targeting,omitempty"`
}

func (c *Client) CreateAdGroup(ctx context.Context, in *AdGroupCreate) (*AdGroup, error) {
	if in == nil {
		return nil, fmt.Errorf("ad group create payload is required")
	}
	var env Response[AdGroup]
	if err := c.DoJSON(ctx, http.MethodPost, "adgroups", in, &env); err != nil {
		return nil, err
	}
	return &env.Result, nil
}

func (c *Client) GetAdGroup(ctx context.Context, id int64) (*AdGroup, error) {
	var env Response[AdGroup]
	path := fmt.Sprintf("adgroups/%d", id)
	if err := c.DoJSON(ctx, http.MethodGet, path, nil, &env); err != nil {
		return nil, err
	}
	return &env.Result, nil
}

func (c *Client) UpdateAdGroup(ctx context.Context, id int64, in *AdGroupUpdate) (*AdGroup, error) {
	if in == nil {
		return nil, fmt.Errorf("ad group update payload is required")
	}
	var env Response[AdGroup]
	path := fmt.Sprintf("adgroups/%d", id)
	if err := c.DoJSON(ctx, http.MethodPut, path, in, &env); err != nil {
		return nil, err
	}
	return &env.Result, nil
}

func (c *Client) DeleteAdGroup(ctx context.Context, id int64) error {
	path := fmt.Sprintf("adgroups/%d", id)
	return c.DoJSON(ctx, http.MethodDelete, path, nil, nil)
}

func (c *Client) QueryAdGroups(ctx context.Context, q QueryRequest) (*PageResult[AdGroup], error) {
	return QueryPage[AdGroup](ctx, c, "adgroups/query", q)
}

func (c *Client) FindAdGroupByID(ctx context.Context, adGroupID int64) (*AdGroup, error) {
	page, err := c.QueryAdGroups(ctx, QueryRequest{
		Filters: []QueryFilter{EqualsFilter("id", adGroupID)},
		Pagination: &QueryPagination{Offset: 0, PageSize: 1},
	})
	if err != nil {
		return nil, err
	}
	if len(page.Data) == 0 {
		return nil, &APIError{StatusCode: http.StatusNotFound, Code: "NOT_FOUND", Message: fmt.Sprintf("ad group %d not found", adGroupID)}
	}
	return &page.Data[0], nil
}

func ParseAdGroupID(id string) (int64, error) {
	v, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid ad group id %q: %w", id, err)
	}
	return v, nil
}
