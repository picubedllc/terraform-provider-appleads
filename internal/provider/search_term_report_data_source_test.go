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

func TestSearchTermReportDataSource_UnitReadAdGroup(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/reports/campaigns/10/adgroups/20/searchterms" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"reportingDataResponse": map[string]any{
					"row": []map[string]any{{
						"metadata": map[string]any{
							"searchTermText": "blank slate party",
							"keywordId":      5,
							"keyword":        "blank slate",
							"matchType":      "BROAD",
						},
						"total": map[string]any{"impressions": 11, "taps": 2, "totalInstalls": 1},
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
	ds, ok := NewSearchTermReportDataSource().(*searchTermReportDataSource)
	if !ok {
		t.Fatalf("type %T", NewSearchTermReportDataSource())
	}
	ds.client = apiClient

	ctx := context.Background()
	schemaResp := &datasource.SchemaResponse{}
	ds.Schema(ctx, datasource.SchemaRequest{}, schemaResp)

	rowAttrs := map[string]tftypes.Type{
		"search_term_text":     tftypes.String,
		"keyword_id":           tftypes.String,
		"keyword_text":         tftypes.String,
		"match_type":           tftypes.String,
		"impressions":          tftypes.Number,
		"taps":                 tftypes.Number,
		"ttr":                  tftypes.Number,
		"local_spend_amount":   tftypes.String,
		"local_spend_currency": tftypes.String,
		"avg_cpt_amount":       tftypes.String,
		"avg_cpt_currency":     tftypes.String,
		"installs":             tftypes.Number,
		"conversion_rate":      tftypes.Number,
		"avg_cpa_amount":       tftypes.String,
		"avg_cpa_currency":     tftypes.String,
	}
	objType := tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"id":           tftypes.String,
			"campaign_id":  tftypes.String,
			"ad_group_id":  tftypes.String,
			"start_time":   tftypes.String,
			"end_time":     tftypes.String,
			"search_terms": tftypes.List{ElementType: tftypes.Object{AttributeTypes: rowAttrs}},
		},
	}
	config := tfsdk.Config{
		Schema: schemaResp.Schema,
		Raw: tftypes.NewValue(objType, map[string]tftypes.Value{
			"id":           tftypes.NewValue(tftypes.String, nil),
			"campaign_id":  tftypes.NewValue(tftypes.String, "10"),
			"ad_group_id":  tftypes.NewValue(tftypes.String, "20"),
			"start_time":   tftypes.NewValue(tftypes.String, "2026-09-01"),
			"end_time":     tftypes.NewValue(tftypes.String, "2026-09-14"),
			"search_terms": tftypes.NewValue(objType.AttributeTypes["search_terms"], nil),
		}),
	}
	resp := &datasource.ReadResponse{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: tftypes.NewValue(objType, nil)},
	}
	ds.Read(ctx, datasource.ReadRequest{Config: config}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("diagnostics: %v", resp.Diagnostics)
	}
	var state searchTermReportModel
	diags := resp.State.Get(ctx, &state)
	if diags.HasError() {
		t.Fatalf("state get: %v", diags)
	}
	if len(state.SearchTerms) != 1 || state.SearchTerms[0].SearchTermText.ValueString() != "blank slate party" {
		t.Fatalf("state = %#v", state)
	}
	if state.SearchTerms[0].Installs.ValueInt64() != 1 {
		t.Fatalf("installs = %#v", state.SearchTerms[0])
	}
}
