// Copyright (c) 2026 Pi Cubed LLC
// SPDX-License-Identifier: MIT

package client

// Campaign is an Apple Ads campaign (API v5).
//
// Mutability (Terraform classification — confirmed against Apple Ads Campaign
// Management API v5 update semantics):
//
//	Immutable (cannot change in place; Terraform must NOT RequiresReplace):
//	  - AdamID
//	  - SupplySources
//	  - AdChannelType
//
//	Mutable (in-place update supported):
//	  - Name
//	  - Status
//	  - BudgetAmount
//	  - DailyBudgetAmount
//	  - BudgetOrders
//	  - EndTime
//	  - CountriesOrRegions (PUT /campaigns/{id} with
//	    clearGeoTargetingOnCountryOrRegionChange=true; this clears ad-group
//	    geo targeting on the campaign)
//
//	Computed / read-only from Apple:
//	  - ID, OrgID, ServingStatus, DisplayStatus, ServingStateReasons,
//	    ModificationTime, Deleted, CountryOrRegionServingStateReasons
type Campaign struct {
	ID                                 int64               `json:"id,omitempty"`
	OrgID                              int64               `json:"orgId,omitempty"`
	Name                               string              `json:"name,omitempty"`
	BudgetAmount                       *Money              `json:"budgetAmount,omitempty"`
	DailyBudgetAmount                  *Money              `json:"dailyBudgetAmount,omitempty"`
	AdamID                             int64               `json:"adamId,omitempty"`
	PaymentModel                       string              `json:"paymentModel,omitempty"`
	Status                             string              `json:"status,omitempty"`
	ServingStatus                      string              `json:"servingStatus,omitempty"`
	ServingStateReasons                []string            `json:"servingStateReasons,omitempty"`
	DisplayStatus                      string              `json:"displayStatus,omitempty"`
	CountriesOrRegions                 []string            `json:"countriesOrRegions,omitempty"`
	SupplySources                      []string            `json:"supplySources,omitempty"`
	AdChannelType                      string              `json:"adChannelType,omitempty"`
	BillingEvent                       string              `json:"billingEvent,omitempty"`
	BiddingStrategy                    string              `json:"biddingStrategy,omitempty"`
	BudgetOrders                       []int64             `json:"budgetOrders,omitempty"`
	StartTime                          string              `json:"startTime,omitempty"`
	EndTime                            string              `json:"endTime,omitempty"`
	ModificationTime                   string              `json:"modificationTime,omitempty"`
	Deleted                            bool                `json:"deleted,omitempty"`
	CountryOrRegionServingStateReasons map[string][]string `json:"countryOrRegionServingStateReasons,omitempty"`
}

// CampaignCreate is the POST /campaigns body.
// Field set matches Apple's Create a Campaign example plus the SEARCH
// defaults proven against API v5 (billingEvent, biddingStrategy, supplySources).
type CampaignCreate struct {
	OrgID              int64    `json:"orgId,omitempty"`
	Name               string   `json:"name"`
	AdamID             int64    `json:"adamId"`
	CountriesOrRegions []string `json:"countriesOrRegions"`
	Status             string   `json:"status,omitempty"`
	BudgetAmount       *Money   `json:"budgetAmount,omitempty"`
	DailyBudgetAmount  *Money   `json:"dailyBudgetAmount,omitempty"`
	SupplySources      []string `json:"supplySources,omitempty"`
	AdChannelType      string   `json:"adChannelType,omitempty"`
	BillingEvent       string   `json:"billingEvent,omitempty"`
	BiddingStrategy    string   `json:"biddingStrategy,omitempty"`
	BudgetOrders       []int64  `json:"budgetOrders,omitempty"`
	StartTime          string   `json:"startTime,omitempty"`
	EndTime            string   `json:"endTime,omitempty"`
}

// CampaignUpdate is the mutable subset for PUT /campaigns/{id}.
// Wrapped as {"campaign": {...}} by UpdateCampaign. When CountriesOrRegions
// membership changes, set ClearGeoTargetingOnCountryOrRegionChange so the
// envelope includes Apple's required flag (see Update a Campaign).
type CampaignUpdate struct {
	Name                                     string   `json:"name,omitempty"`
	Status                                   string   `json:"status,omitempty"`
	BudgetAmount                             *Money   `json:"budgetAmount,omitempty"`
	DailyBudgetAmount                        *Money   `json:"dailyBudgetAmount,omitempty"`
	BudgetOrders                             []int64  `json:"budgetOrders,omitempty"`
	EndTime                                  string   `json:"endTime,omitempty"`
	CountriesOrRegions                       []string `json:"countriesOrRegions,omitempty"`
	ClearEndTime                             bool     `json:"-"` // sentinel handled by marshal helper when needed
	ClearGeoTargetingOnCountryOrRegionChange bool     `json:"-"`
}

type campaignUpdateEnvelope struct {
	ClearGeoTargetingOnCountryOrRegionChange bool            `json:"clearGeoTargetingOnCountryOrRegionChange,omitempty"`
	Campaign                                 *CampaignUpdate `json:"campaign"`
}

// Referenced so the schema-stage package stays unused-clean until Update uses it.
var _ = campaignUpdateEnvelope{}
