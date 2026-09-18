// Copyright (c) 2026 Pi Cubed LLC
// SPDX-License-Identifier: MIT

package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

// AdGroup is an Apple Ads ad group (API v5).
//
// Mutability:
//
//	Immutable: CampaignID, PricingModel
//	Mutable: Name, Status, DefaultBidAmount, CPAGoal, AutomatedKeywordsOptIn,
//	         StartTime, EndTime, TargetingDimensions
//	Computed: ID, ServingStatus, DisplayStatus, ModificationTime, Deleted
type AdGroup struct {
	ID                     int64                `json:"id,omitempty"`
	CampaignID             int64                `json:"campaignId,omitempty"`
	Name                   string               `json:"name,omitempty"`
	Status                 string               `json:"status,omitempty"`
	ServingStatus          string               `json:"servingStatus,omitempty"`
	DisplayStatus          string               `json:"displayStatus,omitempty"`
	DefaultBidAmount       *Money               `json:"defaultBidAmount,omitempty"`
	CPAGoal                *Money               `json:"cpaGoal,omitempty"`
	AutomatedKeywordsOptIn bool                 `json:"automatedKeywordsOptIn,omitempty"`
	StartTime              string               `json:"startTime,omitempty"`
	EndTime                string               `json:"endTime,omitempty"`
	ModificationTime       string               `json:"modificationTime,omitempty"`
	Deleted                bool                 `json:"deleted,omitempty"`
	TargetingDimensions    *TargetingDimensions `json:"targetingDimensions,omitempty"`
	ServingStateReasons    []string             `json:"servingStateReasons,omitempty"`
	PricingModel           string               `json:"pricingModel,omitempty"`
}

// PricingModel values for AdGroup.PricingModel (Apple Ads Campaign Management API v5).
const (
	PricingModelCPC = "CPC" // cost per tap
	PricingModelCPM = "CPM" // cost per thousand impressions
)

// TargetingDimensions holds ad group audience targeting (Apple Ads API v5).
//
// Geo targeting (country, adminArea, locality) only works on campaigns with a
// single countriesOrRegions value. Resolve IDs with SearchGeoLocations:
// country is ISO alpha-2 (US), adminArea is Country|AdminArea (US|NY),
// locality is Country|AdminArea|Locality (US|NY|New York).
type TargetingDimensions struct {
	Age            *AgeTarget            `json:"age,omitempty"`
	Gender         *GenderTarget         `json:"gender,omitempty"`
	DeviceClass    *DeviceClassTarget    `json:"deviceClass,omitempty"`
	Country        *LocalityTarget       `json:"country,omitempty"`
	AdminArea      *LocalityTarget       `json:"adminArea,omitempty"`
	Locality       *LocalityTarget       `json:"locality,omitempty"`
	Daypart        *DaypartTarget        `json:"daypart,omitempty"`
	AppDownloaders *AppDownloadersTarget `json:"appDownloaders,omitempty"`
}

type AgeTarget struct {
	Included []AgeRange `json:"included,omitempty"`
}
type AgeRange struct {
	MinAge int `json:"minAge,omitempty"`
	MaxAge int `json:"maxAge,omitempty"`
}
type GenderTarget struct {
	Included []string `json:"included,omitempty"`
}
type DeviceClassTarget struct {
	Included []string `json:"included,omitempty"`
}
type LocalityTarget struct {
	Included []string `json:"included,omitempty"`
}

// DaypartTarget limits serving to hours of the week in the user's time zone.
// Hours are 0–167 starting Sunday 12:00 AM (Monday 1:00 AM is 25).
type DaypartTarget struct {
	UserTime *DaypartUserTime `json:"userTime,omitempty"`
}

type DaypartUserTime struct {
	Included []int `json:"included,omitempty"`
}

// AppDownloadersTarget includes or excludes users by owned-app Adam IDs.
type AppDownloadersTarget struct {
	Included []string `json:"included,omitempty"`
	Excluded []string `json:"excluded,omitempty"`
}

// targetingDimensionsUpdate always serializes every TargetingDimensions key.
// Apple requires all dimensions on ad group update; unspecified keys are JSON null.
type targetingDimensionsUpdate struct {
	Age            *AgeTarget            `json:"age"`
	Gender         *GenderTarget         `json:"gender"`
	DeviceClass    *DeviceClassTarget    `json:"deviceClass"`
	Country        *LocalityTarget       `json:"country"`
	AdminArea      *LocalityTarget       `json:"adminArea"`
	Locality       *LocalityTarget       `json:"locality"`
	Daypart        *DaypartTarget        `json:"daypart"`
	AppDownloaders *AppDownloadersTarget `json:"appDownloaders"`
}

func (t *TargetingDimensions) updateWire() targetingDimensionsUpdate {
	if t == nil {
		return targetingDimensionsUpdate{}
	}
	return targetingDimensionsUpdate{
		Age:            t.Age,
		Gender:         t.Gender,
		DeviceClass:    t.DeviceClass,
		Country:        t.Country,
		AdminArea:      t.AdminArea,
		Locality:       t.Locality,
		Daypart:        t.Daypart,
		AppDownloaders: t.AppDownloaders,
	}
}

// AdGroupCreate is POST /campaigns/{id}/adgroups body.
//
// Apple requires name, defaultBidAmount, pricingModel, and startTime
// (HTTP 400 REQUIRED_VALUE / START_TIME_IS_REQUIRED). This client does not
// default omitted fields.
type AdGroupCreate struct {
	Name                   string               `json:"name"`
	DefaultBidAmount       *Money               `json:"defaultBidAmount"`
	CPAGoal                *Money               `json:"cpaGoal,omitempty"`
	AutomatedKeywordsOptIn bool                 `json:"automatedKeywordsOptIn"`
	Status                 string               `json:"status,omitempty"`
	StartTime              string               `json:"startTime,omitempty"`
	EndTime                string               `json:"endTime,omitempty"`
	TargetingDimensions    *TargetingDimensions `json:"targetingDimensions,omitempty"`
	PricingModel           string               `json:"pricingModel,omitempty"`
}

// AdGroupUpdate is the mutable subset for PUT.
//
// TargetingDimensions, when non-nil, is sent with every Apple dimension key
// present (unspecified nested dimensions are JSON null). Set
// ClearTargetingDimensions to send targetingDimensions: null and clear
// previously set audience targeting. Omit both to leave targeting unchanged.
type AdGroupUpdate struct {
	Name                     string               `json:"name,omitempty"`
	DefaultBidAmount         *Money               `json:"defaultBidAmount,omitempty"`
	CPAGoal                  *Money               `json:"cpaGoal,omitempty"`
	AutomatedKeywordsOptIn   *bool                `json:"automatedKeywordsOptIn,omitempty"`
	Status                   string               `json:"status,omitempty"`
	StartTime                string               `json:"startTime,omitempty"`
	EndTime                  string               `json:"endTime,omitempty"`
	TargetingDimensions      *TargetingDimensions `json:"targetingDimensions,omitempty"`
	ClearTargetingDimensions bool                 `json:"-"`
}

// MarshalJSON encodes the update payload. When TargetingDimensions is set,
// every targetingDimensions key is included (null for unset nested dimensions)
// because Apple requires a full object on targeting updates. ClearTargetingDimensions
// serializes targetingDimensions as JSON null.
func (u AdGroupUpdate) MarshalJSON() ([]byte, error) {
	type wire struct {
		Name                   string          `json:"name,omitempty"`
		DefaultBidAmount       *Money          `json:"defaultBidAmount,omitempty"`
		CPAGoal                *Money          `json:"cpaGoal,omitempty"`
		AutomatedKeywordsOptIn *bool           `json:"automatedKeywordsOptIn,omitempty"`
		Status                 string          `json:"status,omitempty"`
		StartTime              string          `json:"startTime,omitempty"`
		EndTime                string          `json:"endTime,omitempty"`
		TargetingDimensions    json.RawMessage `json:"targetingDimensions,omitempty"`
	}
	w := wire{
		Name:                   u.Name,
		DefaultBidAmount:       u.DefaultBidAmount,
		CPAGoal:                u.CPAGoal,
		AutomatedKeywordsOptIn: u.AutomatedKeywordsOptIn,
		Status:                 u.Status,
		StartTime:              u.StartTime,
		EndTime:                u.EndTime,
	}
	switch {
	case u.ClearTargetingDimensions:
		w.TargetingDimensions = json.RawMessage("null")
	case u.TargetingDimensions != nil:
		b, err := json.Marshal(u.TargetingDimensions.updateWire())
		if err != nil {
			return nil, err
		}
		w.TargetingDimensions = b
	}
	return json.Marshal(w)
}

type adGroupUpdateEnvelope struct {
	AdGroup *AdGroupUpdate `json:"adGroup"`
}

func (c *Client) CreateAdGroup(ctx context.Context, campaignID int64, in *AdGroupCreate) (*AdGroup, error) {
	if in == nil {
		return nil, fmt.Errorf("ad group create payload is required")
	}
	var env Response[AdGroup]
	path := fmt.Sprintf("campaigns/%d/adgroups", campaignID)
	if err := c.DoJSON(ctx, http.MethodPost, path, in, &env); err != nil {
		return nil, err
	}
	return &env.Data, nil
}

func (c *Client) GetAdGroup(ctx context.Context, campaignID, adGroupID int64) (*AdGroup, error) {
	var env Response[AdGroup]
	path := fmt.Sprintf("campaigns/%d/adgroups/%d", campaignID, adGroupID)
	if err := c.DoJSON(ctx, http.MethodGet, path, nil, &env); err != nil {
		return nil, err
	}
	return &env.Data, nil
}

// ListAdGroups fetches one page of ad groups for a campaign (GET /campaigns/{id}/adgroups).
func (c *Client) ListAdGroups(ctx context.Context, campaignID int64, page PageParams) (*PageResult[AdGroup], error) {
	path := fmt.Sprintf("campaigns/%d/adgroups", campaignID)
	return FetchPage[AdGroup](ctx, c, http.MethodGet, path, page, nil)
}

func (c *Client) UpdateAdGroup(ctx context.Context, campaignID, adGroupID int64, in *AdGroupUpdate) (*AdGroup, error) {
	var env Response[AdGroup]
	path := fmt.Sprintf("campaigns/%d/adgroups/%d", campaignID, adGroupID)
	if err := c.DoJSON(ctx, http.MethodPut, path, adGroupUpdateEnvelope{AdGroup: in}, &env); err != nil {
		return nil, err
	}
	return &env.Data, nil
}

func (c *Client) DeleteAdGroup(ctx context.Context, campaignID, adGroupID int64) error {
	path := fmt.Sprintf("campaigns/%d/adgroups/%d", campaignID, adGroupID)
	return c.DoJSON(ctx, http.MethodDelete, path, nil, nil)
}

// FindAdGroupByID locates an ad group across the org when only the ad group id is known.
func (c *Client) FindAdGroupByID(ctx context.Context, adGroupID int64) (*AdGroup, error) {
	body := Selector{
		Conditions: []SelectorCondition{{
			Field:    "id",
			Operator: "EQUALS",
			Values:   []string{strconv.FormatInt(adGroupID, 10)},
		}},
		Pagination: &SelectorPagination{Offset: 0, Limit: 1},
	}
	var env ListResponse[AdGroup]
	if err := c.DoJSON(ctx, http.MethodPost, "adgroups/find", body, &env); err != nil {
		return nil, err
	}
	if len(env.Data) == 0 {
		return nil, &APIError{StatusCode: http.StatusNotFound, Code: "NOT_FOUND", Message: fmt.Sprintf("ad group %d not found", adGroupID)}
	}
	return &env.Data[0], nil
}

// ParseAdGroupImportID parses "campaignID/adGroupID" or bare adGroupID.
func ParseAdGroupImportID(id string) (campaignID, adGroupID int64, err error) {
	parts := strings.Split(id, "/")
	switch len(parts) {
	case 1:
		adGroupID, err = strconv.ParseInt(parts[0], 10, 64)
		return 0, adGroupID, err
	case 2:
		campaignID, err = strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			return 0, 0, err
		}
		adGroupID, err = strconv.ParseInt(parts[1], 10, 64)
		return campaignID, adGroupID, err
	default:
		return 0, 0, fmt.Errorf("invalid ad group import id %q; expected ad_group_id or campaign_id/ad_group_id", id)
	}
}
