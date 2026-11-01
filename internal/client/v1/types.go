// SPDX-License-Identifier: MIT

// Package v1 is an Apple Ads Platform API v1 HTTP client.
// It is independent of Terraform Plugin Framework types.
package v1

import "github.com/picubedllc/terraform-provider-appleads/internal/client"

// Money is Apple Ads' currency amount (decimal string, never float64).
type Money = client.Money

// Response is the Platform API single-object envelope (`result`).
type Response[T any] struct {
	Result T `json:"result"`
}

// ListResponse is the Platform API list envelope (`result` + pagination).
type ListResponse[T any] struct {
	Result     []T        `json:"result"`
	Pagination Pagination `json:"pagination"`
}

// Pagination matches Platform API pagination metadata.
type Pagination struct {
	Offset     int `json:"offset"`
	PageSize   int `json:"pageSize"`
	TotalCount int `json:"totalCount"`
}

// QueryRequest is the common /query body (filters, sorting, pagination, fields).
type QueryRequest struct {
	Fields     []string      `json:"fields,omitempty"`
	Filters    []QueryFilter `json:"filters,omitempty"`
	Sorting    []QuerySort   `json:"sorting,omitempty"`
	Pagination *QueryPagination `json:"pagination,omitempty"`
	Include    []string         `json:"include,omitempty"`
}

// QueryFilter is one Platform API query filter.
// Value may be a scalar or array depending on the operator.
type QueryFilter struct {
	Field    string `json:"field"`
	Operator string `json:"operator"`
	Value    any    `json:"value"`
}

// QuerySort is one sort clause (`order`: ASC | DESC).
type QuerySort struct {
	Field string `json:"field"`
	Order string `json:"order"`
}

// QueryPagination controls query pagination in request bodies.
type QueryPagination struct {
	Offset   int `json:"offset"`
	PageSize int `json:"pageSize"`
}

// EqualsFilter builds an EQUALS filter.
func EqualsFilter(field string, value any) QueryFilter {
	return QueryFilter{Field: field, Operator: "EQUALS", Value: value}
}

// PageResult is a decoded page of query/list results.
type PageResult[T any] struct {
	Data       []T
	Offset     int
	PageSize   int
	TotalCount int
}
