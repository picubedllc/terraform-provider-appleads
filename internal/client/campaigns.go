// Copyright (c) 2026 Pi Cubed LLC
// SPDX-License-Identifier: MIT

package client

// Bidding strategy values for Campaign.BiddingStrategy (API v5).
const (
	BiddingStrategyManualCPT      = "MANUAL_CPT"
	BiddingStrategyMaxConversions = "MAX_CONVERSIONS"
)

// Ad channel type values for Campaign.AdChannelType (API v5).
const (
	AdChannelTypeSearch  = "SEARCH"
	AdChannelTypeDisplay = "DISPLAY"
)

// Supply source values for Campaign.SupplySources (API v5).
const (
	SupplySourceSearchResults      = "APPSTORE_SEARCH_RESULTS"
	SupplySourceTodayTab           = "APPSTORE_TODAY_TAB"
	SupplySourceSearchTab          = "APPSTORE_SEARCH_TAB"
	SupplySourceProductPagesBrowse = "APPSTORE_PRODUCT_PAGES_BROWSE"
)

// Billing event values for Campaign.BillingEvent (API v5).
const (
	BillingEventTaps = "TAPS"
)

// Campaign is an Apple Ads campaign (API v5).
//
// Mutability (Terraform classification — Apple Ads Campaign Management API v5;
// budgetAmount treated as create-only per Apple docs / safe default from live probes):
//
//	Immutable / create-only (cannot change in place; Terraform must NOT RequiresReplace):
//	  - AdamID
//	  - CountriesOrRegions
//	  - SupplySources
//	  - AdChannelType
//	  - BillingEvent (tied to channel/supply at create)
//	  - BudgetAmount (lifetime total; set on create only)
//
//	Mutable (in-place update supported):
//	  - Name
//	  - Status
//	  - DailyBudgetAmount
//	  - BudgetOrders
//	  - EndTime
//	  - BiddingStrategy
//	  - TargetCpa (required when BiddingStrategy is MAX_CONVERSIONS)
//
//	Create optional / probe-dependent:
//	  - StartTime — settable on create; update support is probe-dependent
//
//	Computed / read-only from Apple:
//	  - ID, OrgID, PaymentModel, ServingStatus, DisplayStatus, ServingStateReasons,
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
	TargetCpa                          *Money              `json:"targetCpa,omitempty"`
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
	TargetCpa          *Money   `json:"targetCpa,omitempty"`
	BudgetOrders       []int64  `json:"budgetOrders,omitempty"`
	StartTime          string   `json:"startTime,omitempty"`
	EndTime            string   `json:"endTime,omitempty"`
}

// CampaignUpdate is the mutable subset for PUT /campaigns/{id}.
// Wrapped as {"campaign": {...}} by UpdateCampaign.
//
// BudgetAmount is intentionally absent: Apple documents lifetime budget as
// create-only. Terraform never sends budgetAmount on update. Live probes that
// still need to exercise a rejected update may POST a raw JSON body.
type CampaignUpdate struct {
	Name              string  `json:"name,omitempty"`
	Status            string  `json:"status,omitempty"`
	DailyBudgetAmount *Money  `json:"dailyBudgetAmount,omitempty"`
	BudgetOrders      []int64 `json:"budgetOrders,omitempty"`
	BiddingStrategy   string  `json:"biddingStrategy,omitempty"`
	TargetCpa         *Money  `json:"targetCpa,omitempty"`
	StartTime         string  `json:"startTime,omitempty"`
	EndTime           string  `json:"endTime,omitempty"`
	ClearEndTime      bool    `json:"-"` // sentinel handled by marshal helper when needed
}

type campaignUpdateEnvelope struct {
	Campaign *CampaignUpdate `json:"campaign"`
}

// Referenced so the schema-stage package stays unused-clean until Update uses it.
var _ = campaignUpdateEnvelope{}
