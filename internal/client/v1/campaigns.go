// SPDX-License-Identifier: MIT

package v1

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

// Promoted object types for campaigns.
const (
	PromotedObjectTypeAppStoreApp   = "APPSTORE_APP"
	PromotedObjectTypeBusinessBrand = "BUSINESS_BRAND"
)

// BidStrategy describes Platform API campaign/ad-group bid configuration.
type BidStrategy struct {
	BidStrategyType string `json:"bidStrategyType,omitempty"`
	BidStrategyGoal string `json:"bidStrategyGoal,omitempty"`
	Bid             *Money `json:"bid,omitempty"`
}

// MoneyValue wraps a Money value object used by dailyBudget and similar fields.
type MoneyValue struct {
	Value *Money `json:"value,omitempty"`
}

// IncludeList is a targeting include list.
type IncludeList struct {
	Include []string `json:"include,omitempty"`
}

// CampaignTargeting holds campaign-level targeting (countries, supply placement).
type CampaignTargeting struct {
	CountryOrRegion *IncludeList `json:"countryOrRegion,omitempty"`
	SupplyPlacement *IncludeList `json:"supplyPlacement,omitempty"`
}

// SharedBudgetRef assigns a shared budget to a campaign.
type SharedBudgetRef struct {
	BudgetID int64 `json:"budgetId"`
}

// Campaign is a Platform API v1 campaign.
type Campaign struct {
	ID                          int64              `json:"id,omitempty"`
	AdAccountID                 int64              `json:"adAccountId,omitempty"`
	Name                        string             `json:"name,omitempty"`
	Status                      string             `json:"status,omitempty"`
	PromotedObjectType          string             `json:"promotedObjectType,omitempty"`
	PromotedObjectID            string             `json:"promotedObjectId,omitempty"`
	BillingEvent                string             `json:"billingEvent,omitempty"`
	AdChannelType               string             `json:"adChannelType,omitempty"`
	BidStrategy                 *BidStrategy       `json:"bidStrategy,omitempty"`
	DailyBudget                 *MoneyValue        `json:"dailyBudget,omitempty"`
	Targeting                   *CampaignTargeting `json:"targeting,omitempty"`
	SharedBudgets               []SharedBudgetRef  `json:"sharedBudgets,omitempty"`
	StartTime                   string             `json:"startTime,omitempty"`
	EndTime                     string             `json:"endTime,omitempty"`
	PaymentModel                string             `json:"paymentModel,omitempty"`
	SystemStatus                string             `json:"systemStatus,omitempty"`
	SystemStatusReasons         []string           `json:"systemStatusReasons,omitempty"`
	SystemStatusLimitingReasons []string           `json:"systemStatusLimitingReasons,omitempty"`
	DisplayStatus               string             `json:"displayStatus,omitempty"`
	CreationTime                string             `json:"creationTime,omitempty"`
	ModificationTime            string             `json:"modificationTime,omitempty"`
	Deleted                     bool               `json:"deleted,omitempty"`
}

// CampaignCreate is POST /campaigns.
type CampaignCreate struct {
	AdAccountID        int64              `json:"adAccountId,omitempty"`
	Name               string             `json:"name"`
	Status             string             `json:"status,omitempty"`
	PromotedObjectType string             `json:"promotedObjectType"`
	PromotedObjectID   string             `json:"promotedObjectId"`
	BillingEvent       string             `json:"billingEvent,omitempty"`
	AdChannelType      string             `json:"adChannelType,omitempty"`
	BidStrategy        *BidStrategy       `json:"bidStrategy,omitempty"`
	DailyBudget        *MoneyValue        `json:"dailyBudget,omitempty"`
	Targeting          *CampaignTargeting `json:"targeting,omitempty"`
	SharedBudgets      []SharedBudgetRef  `json:"sharedBudgets,omitempty"`
	StartTime          string             `json:"startTime,omitempty"`
	EndTime            string             `json:"endTime,omitempty"`
}

// CampaignUpdate is PUT /campaigns/{id} (partial mutable fields).
type CampaignUpdate struct {
	Name          string            `json:"name,omitempty"`
	Status        string            `json:"status,omitempty"`
	BidStrategy   *BidStrategy      `json:"bidStrategy,omitempty"`
	DailyBudget   *MoneyValue       `json:"dailyBudget,omitempty"`
	SharedBudgets []SharedBudgetRef `json:"sharedBudgets,omitempty"`
	StartTime     string            `json:"startTime,omitempty"`
	EndTime       string            `json:"endTime,omitempty"`
}

// CreateCampaign creates a campaign via POST /campaigns.
func (c *Client) CreateCampaign(ctx context.Context, in *CampaignCreate) (*Campaign, error) {
	if in == nil {
		return nil, fmt.Errorf("campaign create payload is required")
	}
	payload := *in
	if payload.AdAccountID == 0 {
		if id, err := strconv.ParseInt(strings.TrimSpace(c.adAccountID), 10, 64); err == nil && id != 0 {
			payload.AdAccountID = id
		}
	}
	var env Response[Campaign]
	if err := c.DoJSON(ctx, http.MethodPost, "campaigns", &payload, &env); err != nil {
		return nil, err
	}
	return &env.Result, nil
}

// GetCampaign fetches a campaign by ID via GET /campaigns/{id}.
func (c *Client) GetCampaign(ctx context.Context, id int64) (*Campaign, error) {
	var env Response[Campaign]
	path := fmt.Sprintf("campaigns/%d", id)
	if err := c.DoJSON(ctx, http.MethodGet, path, nil, &env); err != nil {
		return nil, err
	}
	return &env.Result, nil
}

// UpdateCampaign updates mutable fields via PUT /campaigns/{id}.
func (c *Client) UpdateCampaign(ctx context.Context, id int64, in *CampaignUpdate) (*Campaign, error) {
	if in == nil {
		return nil, fmt.Errorf("campaign update payload is required")
	}
	var env Response[Campaign]
	path := fmt.Sprintf("campaigns/%d", id)
	if err := c.DoJSON(ctx, http.MethodPut, path, in, &env); err != nil {
		return nil, err
	}
	return &env.Result, nil
}

// DeleteCampaign soft-deletes a campaign via DELETE /campaigns/{id}.
func (c *Client) DeleteCampaign(ctx context.Context, id int64) error {
	path := fmt.Sprintf("campaigns/%d", id)
	return c.DoJSON(ctx, http.MethodDelete, path, nil, nil)
}

// QueryCampaigns runs POST /campaigns/query.
func (c *Client) QueryCampaigns(ctx context.Context, q QueryRequest) (*PageResult[Campaign], error) {
	return QueryPage[Campaign](ctx, c, "campaigns/query", q)
}

// ParseCampaignID converts a Terraform campaign id string to int64.
func ParseCampaignID(id string) (int64, error) {
	v, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid campaign id %q: %w", id, err)
	}
	return v, nil
}
