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
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/picubedllc/terraform-provider-appleads/internal/client"
)

func campaignKeywordReportMetricAttrs() map[string]tftypes.Type {
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

func campaignKeywordReportRowAttrs() map[string]tftypes.Type {
	rowAttrs := map[string]tftypes.Type{
		"keyword_id":    tftypes.String,
		"text":          tftypes.String,
		"match_type":    tftypes.String,
		"ad_group_id":   tftypes.String,
		"ad_group_name": tftypes.String,
	}
	for k, v := range campaignKeywordReportMetricAttrs() {
		rowAttrs[k] = v
	}
	return rowAttrs
}

func campaignKeywordReportObjType() tftypes.Object {
	return tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"id":          tftypes.String,
			"campaign_id": tftypes.String,
			"start_time":  tftypes.String,
			"end_time":    tftypes.String,
			"time_zone":   tftypes.String,
			"keywords":    tftypes.List{ElementType: tftypes.Object{AttributeTypes: campaignKeywordReportRowAttrs()}},
		},
	}
}

func mustCampaignKeywordReportDS(t *testing.T) *campaignKeywordReportDataSource {
	t.Helper()
	ds, ok := NewCampaignKeywordReportDataSource().(*campaignKeywordReportDataSource)
	if !ok {
		t.Fatalf("type %T", NewCampaignKeywordReportDataSource())
	}
	return ds
}

func campaignKeywordReportSampleRow() map[string]any {
	return map[string]any{
		"metadata": map[string]any{
			"keywordId":   99,
			"keyword":     "party games",
			"matchType":   "EXACT",
			"adGroupId":   20,
			"adGroupName": "Generic Search",
		},
		"total": map[string]any{
			"impressions":    12,
			"taps":           3,
			"ttr":            0.25,
			"localSpend":     map[string]string{"amount": "0.09", "currency": "USD"},
			"avgCPT":         map[string]string{"amount": "0.03", "currency": "USD"},
			"totalInstalls":  1,
			"conversionRate": 0.3333,
			"totalAvgCPI":    map[string]string{"amount": "0.09", "currency": "USD"},
		},
	}
}

func TestCampaignKeywordReportDataSource_UnitRead(t *testing.T) {
	t.Parallel()

	var gotBody client.ReportingRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/reports/campaigns/10/keywords" {
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"reportingDataResponse": map[string]any{
					"row": []map[string]any{campaignKeywordReportSampleRow()},
				},
			},
		})
	}))
	t.Cleanup(srv.Close)

	apiClient, err := client.New(client.WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	ds := mustCampaignKeywordReportDS(t)
	ds.client = apiClient

	ctx := context.Background()
	schemaResp := &datasource.SchemaResponse{}
	ds.Schema(ctx, datasource.SchemaRequest{}, schemaResp)
	objType := campaignKeywordReportObjType()

	config := tfsdk.Config{
		Schema: schemaResp.Schema,
		Raw: tftypes.NewValue(objType, map[string]tftypes.Value{
			"id":          tftypes.NewValue(tftypes.String, nil),
			"campaign_id": tftypes.NewValue(tftypes.String, "10"),
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
	if gotBody.TimeZone != "UTC" {
		t.Fatalf("default timeZone = %q", gotBody.TimeZone)
	}

	var state campaignKeywordReportModel
	diags := resp.State.Get(ctx, &state)
	if diags.HasError() {
		t.Fatalf("state get: %v", diags)
	}
	if state.ID.ValueString() != "10/2026-09-01/2026-09-14" {
		t.Fatalf("id = %q", state.ID.ValueString())
	}
	if state.TimeZone.ValueString() != "UTC" {
		t.Fatalf("time_zone = %q", state.TimeZone.ValueString())
	}
	if len(state.Keywords) != 1 {
		t.Fatalf("keywords = %#v", state.Keywords)
	}
	row := state.Keywords[0]
	if row.Text.ValueString() != "party games" || row.AdGroupID.ValueString() != "20" {
		t.Fatalf("row = %#v", row)
	}
	if row.Impressions.ValueInt64() != 12 || row.LocalSpendAmount.ValueString() != "0.09" {
		t.Fatalf("metrics = %#v", row)
	}
}

func TestCampaignKeywordReportDataSource_MissingStartEnd(t *testing.T) {
	t.Parallel()

	apiClient, err := client.New(client.WithBaseURL("http://127.0.0.1:1"))
	if err != nil {
		t.Fatal(err)
	}
	ds := mustCampaignKeywordReportDS(t)
	ds.client = apiClient

	ctx := context.Background()
	schemaResp := &datasource.SchemaResponse{}
	ds.Schema(ctx, datasource.SchemaRequest{}, schemaResp)
	objType := campaignKeywordReportObjType()

	config := tfsdk.Config{
		Schema: schemaResp.Schema,
		Raw: tftypes.NewValue(objType, map[string]tftypes.Value{
			"id":          tftypes.NewValue(tftypes.String, nil),
			"campaign_id": tftypes.NewValue(tftypes.String, "10"),
			"start_time":  tftypes.NewValue(tftypes.String, ""),
			"end_time":    tftypes.NewValue(tftypes.String, ""),
			"time_zone":   tftypes.NewValue(tftypes.String, nil),
			"keywords":    tftypes.NewValue(objType.AttributeTypes["keywords"], nil),
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

func TestCampaignKeywordReportDataSource_EmptyRows(t *testing.T) {
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
	ds := mustCampaignKeywordReportDS(t)
	ds.client = apiClient

	ctx := context.Background()
	schemaResp := &datasource.SchemaResponse{}
	ds.Schema(ctx, datasource.SchemaRequest{}, schemaResp)
	objType := campaignKeywordReportObjType()

	config := tfsdk.Config{
		Schema: schemaResp.Schema,
		Raw: tftypes.NewValue(objType, map[string]tftypes.Value{
			"id":          tftypes.NewValue(tftypes.String, nil),
			"campaign_id": tftypes.NewValue(tftypes.String, "10"),
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
	var state campaignKeywordReportModel
	diags := resp.State.Get(ctx, &state)
	if diags.HasError() {
		t.Fatalf("state get: %v", diags)
	}
	if len(state.Keywords) != 0 {
		t.Fatalf("expected empty keywords, got %#v", state.Keywords)
	}
}

func TestCampaignKeywordReportDataSource_APIError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, `{"error":{"errors":[{"messageCode":"INVALID_INPUT","message":"bad request","field":"startTime"}]}}`)
	}))
	t.Cleanup(srv.Close)

	apiClient, err := client.New(client.WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	ds := mustCampaignKeywordReportDS(t)
	ds.client = apiClient

	ctx := context.Background()
	schemaResp := &datasource.SchemaResponse{}
	ds.Schema(ctx, datasource.SchemaRequest{}, schemaResp)
	objType := campaignKeywordReportObjType()

	config := tfsdk.Config{
		Schema: schemaResp.Schema,
		Raw: tftypes.NewValue(objType, map[string]tftypes.Value{
			"id":          tftypes.NewValue(tftypes.String, nil),
			"campaign_id": tftypes.NewValue(tftypes.String, "10"),
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
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected API error")
	}
	if resp.Diagnostics.Errors()[0].Summary() != "Unable to read Apple Ads campaign keyword report" {
		t.Fatalf("summary = %q", resp.Diagnostics.Errors()[0].Summary())
	}
}

func TestCampaignKeywordReportDataSource_InvalidCampaignID(t *testing.T) {
	t.Parallel()

	apiClient, err := client.New(client.WithBaseURL("http://127.0.0.1:1"))
	if err != nil {
		t.Fatal(err)
	}
	ds := mustCampaignKeywordReportDS(t)
	ds.client = apiClient

	ctx := context.Background()
	schemaResp := &datasource.SchemaResponse{}
	ds.Schema(ctx, datasource.SchemaRequest{}, schemaResp)
	objType := campaignKeywordReportObjType()

	config := tfsdk.Config{
		Schema: schemaResp.Schema,
		Raw: tftypes.NewValue(objType, map[string]tftypes.Value{
			"id":          tftypes.NewValue(tftypes.String, nil),
			"campaign_id": tftypes.NewValue(tftypes.String, "not-a-number"),
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
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected invalid campaign_id error")
	}
	if resp.Diagnostics.Errors()[0].Summary() != "Invalid campaign_id" {
		t.Fatalf("summary = %q", resp.Diagnostics.Errors()[0].Summary())
	}
}
