// SPDX-License-Identifier: MIT

package v1

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
)

// CreativeDestination identifies where a creative points.
type CreativeDestination struct {
	DestinationType string            `json:"destinationType,omitempty"`
	Parameters      map[string]string `json:"parameters,omitempty"`
}

// Creative is a Platform API v1 creative.
type Creative struct {
	ID                  int64                `json:"id,omitempty"`
	AdAccountID         int64                `json:"adAccountId,omitempty"`
	Name                string               `json:"name,omitempty"`
	CreativeType        string               `json:"creativeType,omitempty"`
	Destination         *CreativeDestination `json:"destination,omitempty"`
	SystemStatus        string               `json:"systemStatus,omitempty"`
	SystemStatusReasons []string             `json:"systemStatusReasons,omitempty"`
	CreationTime        string               `json:"creationTime,omitempty"`
	ModificationTime    string               `json:"modificationTime,omitempty"`
	Deleted             bool                 `json:"deleted,omitempty"`
}

type CreativeCreate struct {
	Name         string               `json:"name"`
	CreativeType string               `json:"creativeType"`
	Destination  *CreativeDestination `json:"destination,omitempty"`
}

func (c *Client) CreateCreative(ctx context.Context, in *CreativeCreate) (*Creative, error) {
	if in == nil {
		return nil, fmt.Errorf("creative create payload is required")
	}
	var env Response[Creative]
	if err := c.DoJSON(ctx, http.MethodPost, "creatives", in, &env); err != nil {
		return nil, err
	}
	return &env.Result, nil
}

func (c *Client) GetCreative(ctx context.Context, id int64) (*Creative, error) {
	var env Response[Creative]
	path := fmt.Sprintf("creatives/%d", id)
	if err := c.DoJSON(ctx, http.MethodGet, path, nil, &env); err != nil {
		return nil, err
	}
	return &env.Result, nil
}

func (c *Client) QueryCreatives(ctx context.Context, q QueryRequest) (*PageResult[Creative], error) {
	return QueryPage[Creative](ctx, c, "creatives/query", q)
}

func (c *Client) FindCreativeByID(ctx context.Context, id int64) (*Creative, error) {
	page, err := c.QueryCreatives(ctx, QueryRequest{
		Filters:    []QueryFilter{EqualsFilter("id", id)},
		Pagination: &QueryPagination{PageSize: 1},
	})
	if err != nil {
		return nil, err
	}
	if len(page.Data) == 0 {
		return nil, &APIError{StatusCode: http.StatusNotFound, Code: "NOT_FOUND", Message: fmt.Sprintf("creative %d not found", id)}
	}
	return &page.Data[0], nil
}

func ParseCreativeID(id string) (int64, error) {
	v, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid creative id %q: %w", id, err)
	}
	return v, nil
}
