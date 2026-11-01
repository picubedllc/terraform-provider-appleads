// SPDX-License-Identifier: MIT

package v1

import (
	"context"
	"fmt"
	"net/http"
)

// ReportTimeRange is the Platform API reporting time window.
type ReportTimeRange struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

// ReportRequest is a Platform API apps reporting request body.
type ReportRequest struct {
	TimeRange   *ReportTimeRange `json:"timeRange,omitempty"`
	TimeZone    string           `json:"timeZone,omitempty"`
	Granularity string           `json:"granularity,omitempty"`
	Filters     []QueryFilter    `json:"filters,omitempty"`
	Sorting     []QuerySort      `json:"sorting,omitempty"`
	Pagination  *QueryPagination `json:"pagination,omitempty"`
	Fields      []string         `json:"fields,omitempty"`
}

// ReportMetrics holds aggregate/granular metric maps as decimal-friendly strings/numbers.
type ReportMetrics map[string]any

// ReportRow is one reporting row with metadata + metrics.
type ReportRow struct {
	Metadata        map[string]any `json:"metadata,omitempty"`
	TotalMetrics    ReportMetrics  `json:"totalMetrics,omitempty"`
	GranularMetrics []ReportMetrics `json:"granularMetrics,omitempty"`
}

// ReportResponse wraps reporting rows.
type ReportResponse struct {
	Rows []ReportRow `json:"rows,omitempty"`
}

func (c *Client) postReport(ctx context.Context, path string, req *ReportRequest) (*ReportResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("report request is required")
	}
	var env Response[ReportResponse]
	if err := c.DoJSON(ctx, http.MethodPost, path, req, &env); err != nil {
		return nil, err
	}
	return &env.Result, nil
}

// GetCampaignReport posts to the apps campaign reporting endpoint.
func (c *Client) GetCampaignReport(ctx context.Context, req *ReportRequest) (*ReportResponse, error) {
	return c.postReport(ctx, "reports/apps/campaigns", req)
}

func (c *Client) GetAdGroupReport(ctx context.Context, req *ReportRequest) (*ReportResponse, error) {
	return c.postReport(ctx, "reports/apps/adgroups", req)
}

func (c *Client) GetKeywordReport(ctx context.Context, req *ReportRequest) (*ReportResponse, error) {
	return c.postReport(ctx, "reports/apps/keywords", req)
}

func (c *Client) GetAdReport(ctx context.Context, req *ReportRequest) (*ReportResponse, error) {
	return c.postReport(ctx, "reports/apps/ads", req)
}

func (c *Client) GetSearchTermReport(ctx context.Context, req *ReportRequest) (*ReportResponse, error) {
	return c.postReport(ctx, "reports/apps/searchterms", req)
}
