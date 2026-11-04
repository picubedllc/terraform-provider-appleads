// SPDX-License-Identifier: MIT

package v1

import (
	"context"
)

// BidSuggestion is a Platform API keyword bid suggestion.
type BidSuggestion struct {
	KeywordID    int64  `json:"keywordId,omitempty"`
	Text         string `json:"text,omitempty"`
	SuggestedBid *Money `json:"suggestedBid,omitempty"`
	BidFloor     *Money `json:"bidFloor,omitempty"`
	BidCeiling   *Money `json:"bidCeiling,omitempty"`
}

// QueryBidSuggestions runs the suggestions/recommendations query surface.
// Exact path may be refined against live docs during Phase 4 acceptance.
func (c *Client) QueryBidSuggestions(ctx context.Context, q QueryRequest) (*PageResult[BidSuggestion], error) {
	page, err := QueryPage[BidSuggestion](ctx, c, "suggestions/keywords/bids/query", q)
	if err == nil {
		return page, nil
	}
	return QueryPage[BidSuggestion](ctx, c, "recommendations/keywords/bids/query", q)
}
