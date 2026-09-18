// Copyright (c) 2026 Pi Cubed LLC
// SPDX-License-Identifier: MIT

package provider

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/picubedllc/terraform-provider-appleads/internal/client"
)

func searchTermReportMetricAttrs() map[string]tftypes.Type {
	return map[string]tftypes.Type{
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
}

func searchTermReportRowAttrs() map[string]tftypes.Type {
	rowAttrs := map[string]tftypes.Type{
		"search_term_text": tftypes.String,
		"keyword_id":       tftypes.String,
		"keyword":          tftypes.String,
		"match_type":       tftypes.String,
		"ad_group_id":      tftypes.String,
		"ad_group_name":    tftypes.String,
		"bid_amount":       tftypes.String,
		"bid_currency":     tftypes.String,
		"deleted":          tftypes.Bool,
	}
	for k, v := range searchTermReportMetricAttrs() {
		rowAttrs[k] = v
	}
	return rowAttrs
}

func searchTermReportObjType() tftypes.Object {
	return tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"id":           tftypes.String,
			"campaign_id":  tftypes.String,
			"ad_group_id":  tftypes.String,
			"start_time":   tftypes.String,
			"end_time":     tftypes.String,
			"time_zone":    tftypes.String,
			"search_terms": tftypes.List{ElementType: tftypes.Object{AttributeTypes: searchTermReportRowAttrs()}},
		},
	}
}

func mustSearchTermReportDS(t *testing.T) *searchTermReportDataSource {
	t.Helper()
	ds, ok := NewSearchTermReportDataSource().(*searchTermReportDataSource)
	if !ok {
		t.Fatalf("type %T", NewSearchTermReportDataSource())
	}
	return ds
}

func searchTermReportSampleRow() map[string]any {
	return map[string]any{
		"metadata": map[string]any{
			"searchTermText": "word party game",
			"keywordId":      5,
			"keyword":        "word game",
			"matchType":      "BROAD",
			"adGroupId":      20,
			"adGroupName":    "Broad Match",
			"bidAmount":      map[string]string{"amount": "1.50", "currency": "USD"},
			"deleted":        false,
		},
		"total": map[string]any{
			"impressions":    11,
			"taps":           2,
			"ttr":            0.18,
			"localSpend":     map[string]string{"amount": "0.30", "currency": "USD"},
			"avgCPT":         map[string]string{"amount": "0.15", "currency": "USD"},
			"totalInstalls":  1,
			"conversionRate": 0.5,
			"totalAvgCPI":    map[string]string{"amount": "0.30", "currency": "USD"},
		},
	}
}

func TestSearchTermReportDataSource_UnitReadCampaignScoped(t *testing.T) {
	t.Parallel()

	var gotBody client.ReportingRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/reports/campaigns/10/searchterms" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"reportingDataResponse": map[string]any{
					"row": []map[string]any{searchTermReportSampleRow()},
				},
			},
		})
	}))
	t.Cleanup(srv.Close)

	apiClient, err := client.New(client.WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	ds := mustSearchTermReportDS(t)
	ds.client = apiClient

	ctx := context.Background()
	schemaResp := &datasource.SchemaResponse{}
	ds.Schema(ctx, datasource.SchemaRequest{}, schemaResp)
	objType := searchTermReportObjType()

	config := tfsdk.Config{
		Schema: schemaResp.Schema,
		Raw: tftypes.NewValue(objType, map[string]tftypes.Value{
			"id":           tftypes.NewValue(tftypes.String, nil),
			"campaign_id":  tftypes.NewValue(tftypes.String, "10"),
			"ad_group_id":  tftypes.NewValue(tftypes.String, nil),
			"start_time":   tftypes.NewValue(tftypes.String, "2026-09-01"),
			"end_time":     tftypes.NewValue(tftypes.String, "2026-09-14"),
			"time_zone":    tftypes.NewValue(tftypes.String, nil),
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

	if gotBody.TimeZone != "ORTZ" {
		t.Fatalf("default timeZone = %q, want ORTZ", gotBody.TimeZone)
	}

	var state searchTermReportModel
	diags := resp.State.Get(ctx, &state)
	if diags.HasError() {
		t.Fatalf("state get: %v", diags)
	}
	if state.ID.ValueString() != "10/2026-09-01/2026-09-14" {
		t.Fatalf("id = %q", state.ID.ValueString())
	}
	if state.TimeZone.ValueString() != "ORTZ" {
		t.Fatalf("time_zone = %q", state.TimeZone.ValueString())
	}
	if !state.AdGroupID.IsNull() {
		t.Fatalf("ad_group_id should be null, got %v", state.AdGroupID)
	}
	if len(state.SearchTerms) != 1 {
		t.Fatalf("search_terms len = %d", len(state.SearchTerms))
	}
	row := state.SearchTerms[0]
	if row.SearchTermText.ValueString() != "word party game" || row.Keyword.ValueString() != "word game" {
		t.Fatalf("row = %#v", row)
	}
	if row.Impressions.ValueInt64() != 11 || row.BidAmount.ValueString() != "1.50" {
		t.Fatalf("metrics/bid = %#v", row)
	}
}

func TestSearchTermReportDataSource_UnitReadAdGroupScoped(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/reports/campaigns/10/adgroups/20/searchterms" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"reportingDataResponse": map[string]any{
					"row": []map[string]any{searchTermReportSampleRow()},
				},
			},
		})
	}))
	t.Cleanup(srv.Close)

	apiClient, err := client.New(client.WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	ds := mustSearchTermReportDS(t)
	ds.client = apiClient

	ctx := context.Background()
	schemaResp := &datasource.SchemaResponse{}
	ds.Schema(ctx, datasource.SchemaRequest{}, schemaResp)
	objType := searchTermReportObjType()

	config := tfsdk.Config{
		Schema: schemaResp.Schema,
		Raw: tftypes.NewValue(objType, map[string]tftypes.Value{
			"id":           tftypes.NewValue(tftypes.String, nil),
			"campaign_id":  tftypes.NewValue(tftypes.String, "10"),
			"ad_group_id":  tftypes.NewValue(tftypes.String, "20"),
			"start_time":   tftypes.NewValue(tftypes.String, "2026-09-01"),
			"end_time":     tftypes.NewValue(tftypes.String, "2026-09-14"),
			"time_zone":    tftypes.NewValue(tftypes.String, "ORTZ"),
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
	if state.ID.ValueString() != "10/20/2026-09-01/2026-09-14" {
		t.Fatalf("id = %q", state.ID.ValueString())
	}
	if state.AdGroupID.ValueString() != "20" {
		t.Fatalf("ad_group_id = %q", state.AdGroupID.ValueString())
	}
}

func TestSearchTermReportDataSource_ORTZTimezoneNote(t *testing.T) {
	t.Parallel()

	ds := mustSearchTermReportDS(t)
	schemaResp := &datasource.SchemaResponse{}
	ds.Schema(context.Background(), datasource.SchemaRequest{}, schemaResp)

	desc := schemaResp.Schema.MarkdownDescription
	if !strings.Contains(desc, "ORTZ") {
		t.Fatalf("schema description missing ORTZ note: %q", desc)
	}
	tzAttr, ok := schemaResp.Schema.Attributes["time_zone"]
	if !ok {
		t.Fatal("missing time_zone")
	}
	tzDesc := tzAttr.GetMarkdownDescription()
	if !strings.Contains(tzDesc, "ORTZ") {
		t.Fatalf("time_zone description missing ORTZ: %q", tzDesc)
	}
	if searchTermReportTimeZone(types.StringNull()) != "ORTZ" {
		t.Fatal("default time zone must be ORTZ")
	}
}

func TestSearchTermReportDataSource_MissingStartEnd(t *testing.T) {
	t.Parallel()

	apiClient, err := client.New(client.WithBaseURL("http://127.0.0.1:1"))
	if err != nil {
		t.Fatal(err)
	}
	ds := mustSearchTermReportDS(t)
	ds.client = apiClient

	ctx := context.Background()
	schemaResp := &datasource.SchemaResponse{}
	ds.Schema(ctx, datasource.SchemaRequest{}, schemaResp)
	objType := searchTermReportObjType()

	config := tfsdk.Config{
		Schema: schemaResp.Schema,
		Raw: tftypes.NewValue(objType, map[string]tftypes.Value{
			"id":           tftypes.NewValue(tftypes.String, nil),
			"campaign_id":  tftypes.NewValue(tftypes.String, "10"),
			"ad_group_id":  tftypes.NewValue(tftypes.String, nil),
			"start_time":   tftypes.NewValue(tftypes.String, ""),
			"end_time":     tftypes.NewValue(tftypes.String, ""),
			"time_zone":    tftypes.NewValue(tftypes.String, nil),
			"search_terms": tftypes.NewValue(objType.AttributeTypes["search_terms"], nil),
		}),
	}
	resp := &datasource.ReadResponse{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: tftypes.NewValue(objType, nil)},
	}
	ds.Read(ctx, datasource.ReadRequest{Config: config}, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for missing start/end")
	}
	detail := resp.Diagnostics.Errors()[0].Detail()
	if !regexp.MustCompile(`startTime and endTime are required`).MatchString(detail) {
		t.Fatalf("diags = %v", resp.Diagnostics)
	}
}

func TestSearchTermReportDataSource_EmptyRows(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"reportingDataResponse": map[string]any{
					"row": []map[string]any{},
				},
			},
		})
	}))
	t.Cleanup(srv.Close)

	apiClient, err := client.New(client.WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	ds := mustSearchTermReportDS(t)
	ds.client = apiClient

	ctx := context.Background()
	schemaResp := &datasource.SchemaResponse{}
	ds.Schema(ctx, datasource.SchemaRequest{}, schemaResp)
	objType := searchTermReportObjType()

	config := tfsdk.Config{
		Schema: schemaResp.Schema,
		Raw: tftypes.NewValue(objType, map[string]tftypes.Value{
			"id":           tftypes.NewValue(tftypes.String, nil),
			"campaign_id":  tftypes.NewValue(tftypes.String, "10"),
			"ad_group_id":  tftypes.NewValue(tftypes.String, nil),
			"start_time":   tftypes.NewValue(tftypes.String, "2026-09-01"),
			"end_time":     tftypes.NewValue(tftypes.String, "2026-09-14"),
			"time_zone":    tftypes.NewValue(tftypes.String, nil),
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
	if len(state.SearchTerms) != 0 {
		t.Fatalf("expected empty search_terms, got %#v", state.SearchTerms)
	}
}

func TestSearchTermReportDataSource_APIError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, `{"error":{"errors":[{"messageCode":"INVALID_INPUT","message":"bad request","field":"timeZone"}]}}`)
	}))
	t.Cleanup(srv.Close)

	apiClient, err := client.New(client.WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	ds := mustSearchTermReportDS(t)
	ds.client = apiClient

	ctx := context.Background()
	schemaResp := &datasource.SchemaResponse{}
	ds.Schema(ctx, datasource.SchemaRequest{}, schemaResp)
	objType := searchTermReportObjType()

	config := tfsdk.Config{
		Schema: schemaResp.Schema,
		Raw: tftypes.NewValue(objType, map[string]tftypes.Value{
			"id":           tftypes.NewValue(tftypes.String, nil),
			"campaign_id":  tftypes.NewValue(tftypes.String, "10"),
			"ad_group_id":  tftypes.NewValue(tftypes.String, nil),
			"start_time":   tftypes.NewValue(tftypes.String, "2026-09-01"),
			"end_time":     tftypes.NewValue(tftypes.String, "2026-09-14"),
			"time_zone":    tftypes.NewValue(tftypes.String, "UTC"),
			"search_terms": tftypes.NewValue(objType.AttributeTypes["search_terms"], nil),
		}),
	}
	resp := &datasource.ReadResponse{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: tftypes.NewValue(objType, nil)},
	}
	ds.Read(ctx, datasource.ReadRequest{Config: config}, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected API error")
	}
	if resp.Diagnostics.Errors()[0].Summary() != "Unable to read Apple Ads search term report" {
		t.Fatalf("summary = %q", resp.Diagnostics.Errors()[0].Summary())
	}
}

func TestSearchTermReportDataSource_InvalidIDs(t *testing.T) {
	t.Parallel()

	apiClient, err := client.New(client.WithBaseURL("http://127.0.0.1:1"))
	if err != nil {
		t.Fatal(err)
	}
	ds := mustSearchTermReportDS(t)
	ds.client = apiClient

	ctx := context.Background()
	schemaResp := &datasource.SchemaResponse{}
	ds.Schema(ctx, datasource.SchemaRequest{}, schemaResp)
	objType := searchTermReportObjType()

	t.Run("invalid_campaign_id", func(t *testing.T) {
		t.Parallel()
		config := tfsdk.Config{
			Schema: schemaResp.Schema,
			Raw: tftypes.NewValue(objType, map[string]tftypes.Value{
				"id":           tftypes.NewValue(tftypes.String, nil),
				"campaign_id":  tftypes.NewValue(tftypes.String, "not-a-number"),
				"ad_group_id":  tftypes.NewValue(tftypes.String, nil),
				"start_time":   tftypes.NewValue(tftypes.String, "2026-09-01"),
				"end_time":     tftypes.NewValue(tftypes.String, "2026-09-14"),
				"time_zone":    tftypes.NewValue(tftypes.String, nil),
				"search_terms": tftypes.NewValue(objType.AttributeTypes["search_terms"], nil),
			}),
		}
		resp := &datasource.ReadResponse{
			State: tfsdk.State{Schema: schemaResp.Schema, Raw: tftypes.NewValue(objType, nil)},
		}
		ds.Read(ctx, datasource.ReadRequest{Config: config}, resp)
		if !resp.Diagnostics.HasError() {
			t.Fatal("expected invalid campaign_id error")
		}
		if resp.Diagnostics.Errors()[0].Summary() != "Invalid campaign_id" {
			t.Fatalf("summary = %q", resp.Diagnostics.Errors()[0].Summary())
		}
	})

	t.Run("invalid_ad_group_id", func(t *testing.T) {
		t.Parallel()
		config := tfsdk.Config{
			Schema: schemaResp.Schema,
			Raw: tftypes.NewValue(objType, map[string]tftypes.Value{
				"id":           tftypes.NewValue(tftypes.String, nil),
				"campaign_id":  tftypes.NewValue(tftypes.String, "10"),
				"ad_group_id":  tftypes.NewValue(tftypes.String, "bad-id"),
				"start_time":   tftypes.NewValue(tftypes.String, "2026-09-01"),
				"end_time":     tftypes.NewValue(tftypes.String, "2026-09-14"),
				"time_zone":    tftypes.NewValue(tftypes.String, nil),
				"search_terms": tftypes.NewValue(objType.AttributeTypes["search_terms"], nil),
			}),
		}
		resp := &datasource.ReadResponse{
			State: tfsdk.State{Schema: schemaResp.Schema, Raw: tftypes.NewValue(objType, nil)},
		}
		ds.Read(ctx, datasource.ReadRequest{Config: config}, resp)
		if !resp.Diagnostics.HasError() {
			t.Fatal("expected invalid ad_group_id error")
		}
		if resp.Diagnostics.Errors()[0].Summary() != "Invalid ad_group_id" {
			t.Fatalf("summary = %q", resp.Diagnostics.Errors()[0].Summary())
		}
	})
}
