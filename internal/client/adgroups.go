// Copyright (c) 2026 Pi Cubed LLC
// SPDX-License-Identifier: MIT

package client

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

// AdGroup is an Apple Ads ad group (API v5).
//
// Mutability:
//
//	Immutable: CampaignID
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
}

// TargetingDimensions holds ad group audience targeting.
type TargetingDimensions struct {
	Age         *AgeTarget         `json:"age,omitempty"`
	Gender      *GenderTarget      `json:"gender,omitempty"`
	DeviceClass *DeviceClassTarget `json:"deviceClass,omitempty"`
	Country     *LocalityTarget    `json:"country,omitempty"`
	AdminArea   *LocalityTarget    `json:"adminArea,omitempty"`
	Locality    *LocalityTarget    `json:"locality,omitempty"`
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

// AdGroupCreate is POST /campaigns/{id}/adgroups body.
type AdGroupCreate struct {
	Name                   string               `json:"name"`
	DefaultBidAmount       *Money               `json:"defaultBidAmount"`
	CPAGoal                *Money               `json:"cpaGoal,omitempty"`
	AutomatedKeywordsOptIn bool                 `json:"automatedKeywordsOptIn"`
	Status                 string               `json:"status,omitempty"`
	StartTime              string               `json:"startTime,omitempty"`
	EndTime                string               `json:"endTime,omitempty"`
	TargetingDimensions    *TargetingDimensions `json:"targetingDimensions,omitempty"`
}

// AdGroupUpdate is the mutable subset for PUT.
type AdGroupUpdate struct {
	Name                   string               `json:"name,omitempty"`
	DefaultBidAmount       *Money               `json:"defaultBidAmount,omitempty"`
	CPAGoal                *Money               `json:"cpaGoal,omitempty"`
	AutomatedKeywordsOptIn *bool                `json:"automatedKeywordsOptIn,omitempty"`
	Status                 string               `json:"status,omitempty"`
	StartTime              string               `json:"startTime,omitempty"`
	EndTime                string               `json:"endTime,omitempty"`
	TargetingDimensions    *TargetingDimensions `json:"targetingDimensions,omitempty"`
}

type adGroupUpdateEnvelope struct {
	AdGroup *AdGroupUpdate `json:"adGroup"`
}

func (c *Client) CreateAdGroup(ctx context.Context, campaignID int64, in *AdGroupCreate) (*AdGroup, error) {
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
