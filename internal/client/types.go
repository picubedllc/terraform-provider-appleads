// Copyright (c) 2026 Pi Cubed LLC
// SPDX-License-Identifier: MIT

package client

// Money is Apple Ads' currency amount representation.
// Amount is a decimal string (never float64) to preserve precision.
type Money struct {
	Amount   string `json:"amount"`
	Currency string `json:"currency"`
}

// Response is the standard single-object Apple Ads API envelope.
type Response[T any] struct {
	Data T `json:"data"`
}

// ListResponse is the standard list Apple Ads API envelope.
type ListResponse[T any] struct {
	Data       []T `json:"data"`
	Pagination struct {
		TotalResults int `json:"totalResults"`
		StartIndex   int `json:"startIndex"`
		ItemsPerPage int `json:"itemsPerPage"`
	} `json:"pagination"`
}

// Selector filters find endpoints.
type Selector struct {
	Conditions []SelectorCondition `json:"conditions,omitempty"`
	Pagination *SelectorPagination `json:"pagination,omitempty"`
	OrderBy    []SelectorOrder     `json:"orderBy,omitempty"`
}

// SelectorCondition is one find filter.
type SelectorCondition struct {
	Field    string   `json:"field"`
	Operator string   `json:"operator"`
	Values   []string `json:"values"`
}

// SelectorPagination controls find pagination.
type SelectorPagination struct {
	Offset int `json:"offset"`
	Limit  int `json:"limit"`
}

// SelectorOrder controls find sort order.
type SelectorOrder struct {
	Field     string `json:"field"`
	SortOrder string `json:"sortOrder"`
}
