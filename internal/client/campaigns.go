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
//	  - CountriesOrRegions
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
//
//	Computed / read-only from Apple:
//	  - ID, OrgID, ServingStatus, DisplayStatus, ServingStateReasons,
//	    ModificationTime, Deleted, CountryOrRegionServingStateReasons
type Campaign struct {
	ID                   int64    `json:"id,omitempty"`
	OrgID                int64    `json:"orgId,omitempty"`
	Name                 string   `json:"name,omitempty"`
	BudgetAmount         *Money   `json:"budgetAmount,omitempty"`
	DailyBudgetAmount    *Money   `json:"dailyBudgetAmount,omitempty"`
	AdamID               int64    `json:"adamId,omitempty"`
	PaymentModel         string   `json:"paymentModel,omitempty"`
	Status               string   `json:"status,omitempty"`
	ServingStatus        string   `json:"servingStatus,omitempty"`
	ServingStateReasons  []string `json:"servingStateReasons,omitempty"`
	DisplayStatus        string   `json:"displayStatus,omitempty"`
	CountriesOrRegions   []string `json:"countriesOrRegions,omitempty"`
	SupplySources        []string `json:"supplySources,omitempty"`
	AdChannelType        string   `json:"adChannelType,omitempty"`
	BillingEvent         string   `json:"billingEvent,omitempty"`
	BudgetOrders         []int64  `json:"budgetOrders,omitempty"`
	StartTime            string   `json:"startTime,omitempty"`
	EndTime              string   `json:"endTime,omitempty"`
	ModificationTime     string   `json:"modificationTime,omitempty"`
	Deleted              bool     `json:"deleted,omitempty"`
	CountryOrRegionServingStateReasons map[string][]string `json:"countryOrRegionServingStateReasons,omitempty"`
}

// CampaignCreate is the POST /campaigns body.
type CampaignCreate struct {
	Name               string   `json:"name"`
	AdamID             int64    `json:"adamId"`
	CountriesOrRegions []string `json:"countriesOrRegions"`
	Status             string   `json:"status,omitempty"`
	BudgetAmount       *Money   `json:"budgetAmount,omitempty"`
	DailyBudgetAmount  *Money   `json:"dailyBudgetAmount,omitempty"`
	SupplySources      []string `json:"supplySources,omitempty"`
	AdChannelType      string   `json:"adChannelType,omitempty"`
	BillingEvent       string   `json:"billingEvent,omitempty"`
	BudgetOrders       []int64  `json:"budgetOrders,omitempty"`
	StartTime          string   `json:"startTime,omitempty"`
	EndTime            string   `json:"endTime,omitempty"`
}

// CampaignUpdate is the mutable subset for PUT /campaigns/{id}.
// Wrapped as {"campaign": {...}} by UpdateCampaign.
type CampaignUpdate struct {
	Name              string  `json:"name,omitempty"`
	Status            string  `json:"status,omitempty"`
	BudgetAmount      *Money  `json:"budgetAmount,omitempty"`
	DailyBudgetAmount *Money  `json:"dailyBudgetAmount,omitempty"`
	BudgetOrders      []int64 `json:"budgetOrders,omitempty"`
	EndTime           string  `json:"endTime,omitempty"`
	ClearEndTime      bool    `json:"-"` // sentinel handled by marshal helper when needed
}

type campaignUpdateEnvelope struct {
	Campaign *CampaignUpdate `json:"campaign"`
}
