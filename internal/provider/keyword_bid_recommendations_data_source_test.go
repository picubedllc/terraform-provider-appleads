// Copyright (c) 2026 Pi Cubed LLC
// SPDX-License-Identifier: MIT

package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/picubedllc/terraform-provider-appleads/internal/client"
)

func TestKeywordBidRecommendationsDataSource_UnitRead(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/reports/campaigns/10/adgroups/20/keywords" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"reportingDataResponse": map[string]any{
					"row": []map[string]any{{
						"metadata": map[string]any{
							"keywordId": 99,
							"keyword":   "party games",
							"matchType": "EXACT",
							"bidAmount": map[string]string{"amount": "0.05", "currency": "USD"},
						},
						"insights": map[string]any{
							"bidRecommendation": map[string]any{
								"suggestedBidAmount": map[string]string{"amount": "1.50", "currency": "USD"},
								"bidMin":             map[string]string{"amount": "0.80", "currency": "USD"},
								"bidMax":             map[string]string{"amount": "2.40", "currency": "USD"},
							},
						},
					}},
				},
			},
		})
	}))
	t.Cleanup(srv.Close)

	apiClient, err := client.New(client.WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	ds, ok := NewKeywordBidRecommendationsDataSource().(*keywordBidRecommendationsDataSource)
	if !ok {
		t.Fatalf("type %T", NewKeywordBidRecommendationsDataSource())
	}
	ds.client = apiClient

	ctx := context.Background()
	schemaResp := &datasource.SchemaResponse{}
	ds.Schema(ctx, datasource.SchemaRequest{}, schemaResp)

	objType := tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"id":          tftypes.String,
			"campaign_id": tftypes.String,
			"ad_group_id": tftypes.String,
			"start_time":  tftypes.String,
			"end_time":    tftypes.String,
			"time_zone":   tftypes.String,
			"keywords": tftypes.List{ElementType: tftypes.Object{
				AttributeTypes: map[string]tftypes.Type{
					"keyword_id":             tftypes.String,
					"text":                   tftypes.String,
					"match_type":             tftypes.String,
					"bid_amount":             tftypes.String,
					"bid_currency":           tftypes.String,
					"suggested_bid_amount":   tftypes.String,
					"suggested_bid_currency": tftypes.String,
					"bid_min_amount":         tftypes.String,
					"bid_max_amount":         tftypes.String,
				},
			}},
		},
	}
	config := tfsdk.Config{
		Schema: schemaResp.Schema,
		Raw: tftypes.NewValue(objType, map[string]tftypes.Value{
			"id":          tftypes.NewValue(tftypes.String, nil),
			"campaign_id": tftypes.NewValue(tftypes.String, "10"),
			"ad_group_id": tftypes.NewValue(tftypes.String, "20"),
			"start_time":  tftypes.NewValue(tftypes.String, "2026-09-01"),
			"end_time":    tftypes.NewValue(tftypes.String, "2026-09-14"),
			"time_zone":   tftypes.NewValue(tftypes.String, nil),
			"keywords":    tftypes.NewValue(objType.AttributeTypes["keywords"], nil),
		}),
	}
	resp := &datasource.ReadResponse{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: tftypes.NewValue(objType, nil)},
	}
	ds.Read(ctx, datasource.ReadRequest{Config: config}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("diagnostics: %v", resp.Diagnostics)
	}

	var state keywordBidRecommendationsModel
	diags := resp.State.Get(ctx, &state)
	if diags.HasError() {
		t.Fatalf("state get: %v", diags)
	}
	if state.TimeZone.ValueString() != "UTC" {
		t.Fatalf("timezone = %q", state.TimeZone.ValueString())
	}
	if len(state.Keywords) != 1 {
		t.Fatalf("keywords = %#v", state.Keywords)
	}
	got := state.Keywords[0]
	if got.Text.ValueString() != "party games" || got.BidAmount.ValueString() != "0.05" {
		t.Fatalf("keyword = %#v", got)
	}
	if got.SuggestedBidAmount.ValueString() != "1.50" {
		t.Fatalf("suggested = %#v", got)
	}
	if got.SuggestedBidAmount.ValueString() == got.BidAmount.ValueString() {
		t.Fatal("must not treat Apple's suggestion as the configured bid")
	}
}
