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

// BudgetOrder is an Apple Ads budget order (API v5).
//
// Mutability (per Apple Ads Campaign Management API):
//
//	Always updatable: Budget, EndDate, BillingEmail, PrimaryBuyerEmail, PrimaryBuyerName
//	Conditionally updatable (before start / no assigned campaigns): Name, StartDate, ClientName, OrderNumber
//	Computed / system-controlled: ID, Status, ParentOrgID, SupplySources (optional on create as of v5.3;
//	responses typically include all supply source values)
//
// There is no DELETE endpoint in API v5.
type BudgetOrder struct {
	ID                int64    `json:"id,omitempty"`
	Name              string   `json:"name,omitempty"`
	StartDate         string   `json:"startDate,omitempty"`
	EndDate           string   `json:"endDate,omitempty"`
	Budget            *Money   `json:"budget,omitempty"`
	OrderNumber       string   `json:"orderNumber,omitempty"`
	ClientName        string   `json:"clientName,omitempty"`
	PrimaryBuyerName  string   `json:"primaryBuyerName,omitempty"`
	PrimaryBuyerEmail string   `json:"primaryBuyerEmail,omitempty"`
	BillingEmail      string   `json:"billingEmail,omitempty"`
	Status            string   `json:"status,omitempty"`
	ParentOrgID       int64    `json:"parentOrgId,omitempty"`
	SupplySources     []string `json:"supplySources,omitempty"`
}

// BudgetOrderInfo is the Apple Ads envelope for a single budget order
// (create/get/update responses and list items).
type BudgetOrderInfo struct {
	OrgIDs []int64      `json:"orgIds,omitempty"`
	Bo     *BudgetOrder `json:"bo,omitempty"`
}

// BudgetOrderCreate is the mutable payload nested under "bo" on POST /budgetorders.
type BudgetOrderCreate struct {
	Name              string   `json:"name"`
	StartDate         string   `json:"startDate"`
	EndDate           string   `json:"endDate"`
	Budget            *Money   `json:"budget"`
	OrderNumber       string   `json:"orderNumber,omitempty"`
	ClientName        string   `json:"clientName,omitempty"`
	PrimaryBuyerName  string   `json:"primaryBuyerName,omitempty"`
	PrimaryBuyerEmail string   `json:"primaryBuyerEmail,omitempty"`
	BillingEmail      string   `json:"billingEmail,omitempty"`
	SupplySources     []string `json:"supplySources,omitempty"`
}

// BudgetOrderUpdate is the mutable payload nested under "bo" on PUT /budgetorders/{id}.
type BudgetOrderUpdate struct {
	Name              string   `json:"name,omitempty"`
	StartDate         string   `json:"startDate,omitempty"`
	EndDate           string   `json:"endDate,omitempty"`
	Budget            *Money   `json:"budget,omitempty"`
	OrderNumber       string   `json:"orderNumber,omitempty"`
	ClientName        string   `json:"clientName,omitempty"`
	PrimaryBuyerName  string   `json:"primaryBuyerName,omitempty"`
	PrimaryBuyerEmail string   `json:"primaryBuyerEmail,omitempty"`
	BillingEmail      string   `json:"billingEmail,omitempty"`
	SupplySources     []string `json:"supplySources,omitempty"`
}

type budgetOrderRequestEnvelope struct {
	OrgIDs []int64 `json:"orgIds,omitempty"`
	Bo     any     `json:"bo"`
}

// CreateBudgetOrder creates a budget order via POST /budgetorders.
// When orgIDs is empty, the client's configured org_id is used.
func (c *Client) CreateBudgetOrder(ctx context.Context, in *BudgetOrderCreate, orgIDs []int64) (*BudgetOrderInfo, error) {
	if in == nil {
		return nil, fmt.Errorf("budget order create payload is required")
	}
	ids, err := c.resolveBudgetOrderOrgIDs(orgIDs)
	if err != nil {
		return nil, err
	}
	var env Response[BudgetOrderInfo]
	if err := c.DoJSON(ctx, http.MethodPost, "budgetorders", budgetOrderRequestEnvelope{
		OrgIDs: ids,
		Bo:     in,
	}, &env); err != nil {
		return nil, err
	}
	return &env.Data, nil
}

// GetBudgetOrder fetches a budget order by ID via GET /budgetorders/{id}.
func (c *Client) GetBudgetOrder(ctx context.Context, id int64) (*BudgetOrderInfo, error) {
	var env Response[BudgetOrderInfo]
	path := fmt.Sprintf("budgetorders/%d", id)
	if err := c.DoJSON(ctx, http.MethodGet, path, nil, &env); err != nil {
		return nil, err
	}
	return &env.Data, nil
}

// ListBudgetOrders fetches all budget orders via GET /budgetorders (paginated).
func (c *Client) ListBudgetOrders(ctx context.Context) ([]BudgetOrderInfo, error) {
	return FetchAllPages[BudgetOrderInfo](ctx, c, http.MethodGet, "budgetorders", 1000, nil)
}

// UpdateBudgetOrder updates a budget order via PUT /budgetorders/{id}.
// When orgIDs is empty, the client's configured org_id is used.
func (c *Client) UpdateBudgetOrder(ctx context.Context, id int64, in *BudgetOrderUpdate, orgIDs []int64) (*BudgetOrderInfo, error) {
	if in == nil {
		return nil, fmt.Errorf("budget order update payload is required")
	}
	ids, err := c.resolveBudgetOrderOrgIDs(orgIDs)
	if err != nil {
		return nil, err
	}
	var env Response[BudgetOrderInfo]
	path := fmt.Sprintf("budgetorders/%d", id)
	if err := c.DoJSON(ctx, http.MethodPut, path, budgetOrderRequestEnvelope{
		OrgIDs: ids,
		Bo:     in,
	}, &env); err != nil {
		return nil, err
	}
	return &env.Data, nil
}

func (c *Client) resolveBudgetOrderOrgIDs(orgIDs []int64) ([]int64, error) {
	if len(orgIDs) > 0 {
		return orgIDs, nil
	}
	if c.orgID == "" {
		return nil, fmt.Errorf("org id is required for budget order requests")
	}
	id, err := strconv.ParseInt(strings.TrimSpace(c.orgID), 10, 64)
	if err != nil || id == 0 {
		return nil, fmt.Errorf("invalid org id %q for budget order request", c.orgID)
	}
	return []int64{id}, nil
}

// ParseBudgetOrderID converts a Terraform budget order id string to int64.
func ParseBudgetOrderID(id string) (int64, error) {
	v, err := strconv.ParseInt(strings.TrimSpace(id), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid budget order id %q: %w", id, err)
	}
	return v, nil
}
