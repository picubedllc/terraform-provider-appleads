// Copyright (c) 2026 Pi Cubed LLC
// SPDX-License-Identifier: MIT

package client

import (
	"context"
	"fmt"
	"net/http"
)

// ReportingRequest is the Apple Ads Campaign Management API v5 report body.
// Money stays on SpendRow as decimal strings; dates are YYYY-MM-DD.
type ReportingRequest struct {
	StartTime                  string    `json:"startTime"`
	EndTime                    string    `json:"endTime"`
	TimeZone                   string    `json:"timeZone,omitempty"`
	Granularity                string    `json:"granularity,omitempty"`
	ReturnRowTotals            bool      `json:"returnRowTotals"`
	ReturnGrandTotals          bool      `json:"returnGrandTotals"`
	ReturnRecordsWithNoMetrics bool      `json:"returnRecordsWithNoMetrics"`
	Selector                   *Selector `json:"selector,omitempty"`
}

// SpendRow is a reporting metrics row. Money fields use Apple's decimal-string Money.
type SpendRow struct {
	Impressions    int64   `json:"impressions,omitempty"`
	Taps           int64   `json:"taps,omitempty"`
	TTR            float64 `json:"ttr,omitempty"`
	LocalSpend     *Money  `json:"localSpend,omitempty"`
	AvgCPT         *Money  `json:"avgCPT,omitempty"`
	TotalInstalls  int64   `json:"totalInstalls,omitempty"`
	ConversionRate float64 `json:"conversionRate,omitempty"`
	TotalAvgCPI    *Money  `json:"totalAvgCPI,omitempty"`
	TapInstalls    int64   `json:"tapInstalls,omitempty"`
	TapInstallRate float64 `json:"tapInstallRate,omitempty"`
	TapInstallCPI  *Money  `json:"tapInstallCPI,omitempty"`
}

// KeywordBidRecommendation is Apple's suggested CPT for a keyword (observational).
// In API v5, suggestedBidAmount replaces deprecated bidMin/bidMax.
type KeywordBidRecommendation struct {
	SuggestedBidAmount *Money `json:"suggestedBidAmount,omitempty"`
	BidMin             *Money `json:"bidMin,omitempty"`
	BidMax             *Money `json:"bidMax,omitempty"`
}

// KeywordInsights is the insights object on a keyword report row.
type KeywordInsights struct {
	BidRecommendation *KeywordBidRecommendation `json:"bidRecommendation,omitempty"`
}

// ReportingRow is one report row with typed metadata.
type ReportingRow[M any] struct {
	Other    bool             `json:"other"`
	Total    *SpendRow        `json:"total,omitempty"`
	Metadata M                `json:"metadata"`
	Insights *KeywordInsights `json:"insights,omitempty"`
}

// ReportingCampaignMetadata is campaign report row metadata.
type ReportingCampaignMetadata struct {
	CampaignID     int64  `json:"campaignId"`
	CampaignName   string `json:"campaignName"`
	Deleted        bool   `json:"deleted"`
	CampaignStatus string `json:"campaignStatus"`
	AppName        string `json:"appName,omitempty"`
	AdamID         int64  `json:"adamId,omitempty"`
}

// ReportingAdGroupMetadata is ad group report row metadata.
type ReportingAdGroupMetadata struct {
	AdGroupID     int64  `json:"adGroupId"`
	AdGroupName   string `json:"adGroupName"`
	CampaignID    int64  `json:"campaignId"`
	CampaignName  string `json:"campaignName"`
	Deleted       bool   `json:"adGroupDeleted"`
	AdGroupStatus string `json:"adGroupStatus,omitempty"`
}

// ReportingKeywordMetadata is keyword report row metadata.
type ReportingKeywordMetadata struct {
	KeywordID     int64  `json:"keywordId"`
	Keyword       string `json:"keyword"`
	KeywordStatus string `json:"keywordStatus"`
	MatchType     string `json:"matchType"`
	BidAmount     *Money `json:"bidAmount,omitempty"`
	Deleted       bool   `json:"deleted"`
	AdGroupID     int64  `json:"adGroupId"`
	AdGroupName   string `json:"adGroupName"`
	CampaignID    int64  `json:"campaignId,omitempty"`
}

// ReportingSearchTermMetadata is search-term report row metadata.
type ReportingSearchTermMetadata struct {
	SearchTermText string `json:"searchTermText"`
	KeywordID      int64  `json:"keywordId"`
	Keyword        string `json:"keyword"`
	MatchType      string `json:"matchType"`
	BidAmount      *Money `json:"bidAmount,omitempty"`
	Deleted        bool   `json:"deleted"`
	AdGroupID      int64  `json:"adGroupId"`
	AdGroupName    string `json:"adGroupName"`
}

type reportingResponseBody[M any] struct {
	ReportingDataResponse struct {
		Row []ReportingRow[M] `json:"row"`
	} `json:"reportingDataResponse"`
}

// DefaultReportingRequest builds a totals-only request for a date window.
// Search-term reports must pass timeZone ORTZ.
func DefaultReportingRequest(startTime, endTime, timeZone string) *ReportingRequest {
	if timeZone == "" {
		timeZone = "UTC"
	}
	return &ReportingRequest{
		StartTime:                  startTime,
		EndTime:                    endTime,
		TimeZone:                   timeZone,
		ReturnRowTotals:            true,
		ReturnGrandTotals:          false,
		ReturnRecordsWithNoMetrics: true,
		Selector: &Selector{
			Pagination: &SelectorPagination{Offset: 0, Limit: 1000},
		},
	}
}

func getReport[M any](ctx context.Context, c *Client, path string, req *ReportingRequest) ([]ReportingRow[M], error) {
	if req == nil {
		return nil, fmt.Errorf("reporting request is required")
	}
	if req.StartTime == "" || req.EndTime == "" {
		return nil, fmt.Errorf("startTime and endTime are required")
	}
	var env Response[reportingResponseBody[M]]
	if err := c.DoJSON(ctx, http.MethodPost, path, req, &env); err != nil {
		return nil, err
	}
	return env.Data.ReportingDataResponse.Row, nil
}

// GetCampaignReport fetches org-level campaign reports (POST /reports/campaigns).
func (c *Client) GetCampaignReport(ctx context.Context, req *ReportingRequest) ([]ReportingRow[ReportingCampaignMetadata], error) {
	return getReport[ReportingCampaignMetadata](ctx, c, "reports/campaigns", req)
}

// GetAdGroupReport fetches ad group reports within a campaign (POST /reports/campaigns/{campaignId}/adgroups).
func (c *Client) GetAdGroupReport(ctx context.Context, campaignID int64, req *ReportingRequest) ([]ReportingRow[ReportingAdGroupMetadata], error) {
	path := fmt.Sprintf("reports/campaigns/%d/adgroups", campaignID)
	return getReport[ReportingAdGroupMetadata](ctx, c, path, req)
}

// GetAdGroupKeywordReport fetches targeting-keyword reports within an ad group,
// including insights.bidRecommendation when Apple returns it
// (POST /reports/campaigns/{campaignId}/adgroups/{adgroupId}/keywords).
func (c *Client) GetAdGroupKeywordReport(ctx context.Context, campaignID, adGroupID int64, req *ReportingRequest) ([]ReportingRow[ReportingKeywordMetadata], error) {
	path := fmt.Sprintf("reports/campaigns/%d/adgroups/%d/keywords", campaignID, adGroupID)
	return getReport[ReportingKeywordMetadata](ctx, c, path, req)
}

// GetCampaignSearchTermReport fetches search-term reports for a campaign
// (POST /reports/campaigns/{campaignId}/searchterms).
// Apple requires timeZone=ORTZ. Terms typically appear only after 10 impressions.
func (c *Client) GetCampaignSearchTermReport(ctx context.Context, campaignID int64, req *ReportingRequest) ([]ReportingRow[ReportingSearchTermMetadata], error) {
	path := fmt.Sprintf("reports/campaigns/%d/searchterms", campaignID)
	return getReport[ReportingSearchTermMetadata](ctx, c, path, req)
}

// GetAdGroupSearchTermReport fetches search-term reports within an ad group
// (POST /reports/campaigns/{campaignId}/adgroups/{adgroupId}/searchterms).
// Apple requires timeZone=ORTZ.
func (c *Client) GetAdGroupSearchTermReport(ctx context.Context, campaignID, adGroupID int64, req *ReportingRequest) ([]ReportingRow[ReportingSearchTermMetadata], error) {
	path := fmt.Sprintf("reports/campaigns/%d/adgroups/%d/searchterms", campaignID, adGroupID)
	return getReport[ReportingSearchTermMetadata](ctx, c, path, req)
}
