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

func adReportMetricAttrs() map[string]tftypes.Type {
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

func adReportRowAttrs() map[string]tftypes.Type {
	rowAttrs := map[string]tftypes.Type{
		"ad_id":          tftypes.String,
		"name":           tftypes.String,
		"ad_group_id":    tftypes.String,
		"creative_id":    tftypes.String,
		"creative_type":  tftypes.String,
		"language":       tftypes.String,
		"display_status": tftypes.String,
	}
	for k, v := range adReportMetricAttrs() {
		rowAttrs[k] = v
	}
	return rowAttrs
}

func adReportObjType() tftypes.Object {
	return tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"id":          tftypes.String,
			"campaign_id": tftypes.String,
			"start_time":  tftypes.String,
			"end_time":    tftypes.String,
			"time_zone":   tftypes.String,
			"ads":         tftypes.List{ElementType: tftypes.Object{AttributeTypes: adReportRowAttrs()}},
		},
	}
}

func mustAdReportDS(t *testing.T) *adReportDataSource {
	t.Helper()
	ds, ok := NewAdReportDataSource().(*adReportDataSource)
	if !ok {
		t.Fatalf("type %T", NewAdReportDataSource())
	}
	return ds
}

func adReportSampleRow() map[string]any {
	return map[string]any{
		"metadata": map[string]any{
			"adId":          55,
			"adName":        "CPP US",
			"adGroupId":     20,
			"campaignId":    10,
			"creativeId":    77,
			"creativeType":  "CUSTOM_PRODUCT_PAGE",
			"language":      "en",
			"displayStatus": "RUNNING",
			"deleted":       false,
		},
		"total": map[string]any{
			"impressions":    20,
			"taps":           4,
			"ttr":            0.2,
			"localSpend":     map[string]string{"amount": "0.40", "currency": "USD"},
			"avgCPT":         map[string]string{"amount": "0.10", "currency": "USD"},
			"totalInstalls":  2,
			"conversionRate": 0.5,
			"totalAvgCPI":    map[string]string{"amount": "0.20", "currency": "USD"},
		},
	}
}

func TestAdReportDataSource_UnitRead(t *testing.T) {
	t.Parallel()

	var gotBody client.ReportingRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/reports/campaigns/10/ads" {
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"reportingDataResponse": map[string]any{
					"row": []map[string]any{adReportSampleRow()},
				},
			},
		})
	}))
	t.Cleanup(srv.Close)

	apiClient, err := client.New(client.WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	ds := mustAdReportDS(t)
	ds.client = apiClient

	ctx := context.Background()
	schemaResp := &datasource.SchemaResponse{}
	ds.Schema(ctx, datasource.SchemaRequest{}, schemaResp)
	objType := adReportObjType()

	config := tfsdk.Config{
		Schema: schemaResp.Schema,
		Raw: tftypes.NewValue(objType, map[string]tftypes.Value{
			"id":          tftypes.NewValue(tftypes.String, nil),
			"campaign_id": tftypes.NewValue(tftypes.String, "10"),
			"start_time":  tftypes.NewValue(tftypes.String, "2026-09-01"),
			"end_time":    tftypes.NewValue(tftypes.String, "2026-09-14"),
			"time_zone":   tftypes.NewValue(tftypes.String, nil),
			"ads":         tftypes.NewValue(objType.AttributeTypes["ads"], nil),
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

	var state adReportModel
	diags := resp.State.Get(ctx, &state)
	if diags.HasError() {
		t.Fatalf("state get: %v", diags)
	}
	if state.ID.ValueString() != "10/2026-09-01/2026-09-14" {
		t.Fatalf("id = %q", state.ID.ValueString())
	}
	if len(state.Ads) != 1 {
		t.Fatalf("ads = %#v", state.Ads)
	}
	row := state.Ads[0]
	if row.Name.ValueString() != "CPP US" || row.AdID.ValueString() != "55" {
		t.Fatalf("row = %#v", row)
	}
	if row.CreativeID.ValueString() != "77" || row.CreativeType.ValueString() != "CUSTOM_PRODUCT_PAGE" {
		t.Fatalf("creative = %#v", row)
	}
	if row.Impressions.ValueInt64() != 20 || row.LocalSpendAmount.ValueString() != "0.40" {
		t.Fatalf("metrics = %#v", row)
	}
}

func TestAdReportDataSource_MissingStartEnd(t *testing.T) {
	t.Parallel()

	apiClient, err := client.New(client.WithBaseURL("http://127.0.0.1:1"))
	if err != nil {
		t.Fatal(err)
	}
	ds := mustAdReportDS(t)
	ds.client = apiClient

	ctx := context.Background()
	schemaResp := &datasource.SchemaResponse{}
	ds.Schema(ctx, datasource.SchemaRequest{}, schemaResp)
	objType := adReportObjType()

	config := tfsdk.Config{
		Schema: schemaResp.Schema,
		Raw: tftypes.NewValue(objType, map[string]tftypes.Value{
			"id":          tftypes.NewValue(tftypes.String, nil),
			"campaign_id": tftypes.NewValue(tftypes.String, "10"),
			"start_time":  tftypes.NewValue(tftypes.String, ""),
			"end_time":    tftypes.NewValue(tftypes.String, ""),
			"time_zone":   tftypes.NewValue(tftypes.String, nil),
			"ads":         tftypes.NewValue(objType.AttributeTypes["ads"], nil),
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

func TestAdReportDataSource_EmptyRows(t *testing.T) {
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
	ds := mustAdReportDS(t)
	ds.client = apiClient

	ctx := context.Background()
	schemaResp := &datasource.SchemaResponse{}
	ds.Schema(ctx, datasource.SchemaRequest{}, schemaResp)
	objType := adReportObjType()

	config := tfsdk.Config{
		Schema: schemaResp.Schema,
		Raw: tftypes.NewValue(objType, map[string]tftypes.Value{
			"id":          tftypes.NewValue(tftypes.String, nil),
			"campaign_id": tftypes.NewValue(tftypes.String, "10"),
			"start_time":  tftypes.NewValue(tftypes.String, "2026-09-01"),
			"end_time":    tftypes.NewValue(tftypes.String, "2026-09-14"),
			"time_zone":   tftypes.NewValue(tftypes.String, nil),
			"ads":         tftypes.NewValue(objType.AttributeTypes["ads"], nil),
		}),
	}
	resp := &datasource.ReadResponse{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: tftypes.NewValue(objType, nil)},
	}
	ds.Read(ctx, datasource.ReadRequest{Config: config}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("diagnostics: %v", resp.Diagnostics)
	}
	var state adReportModel
	diags := resp.State.Get(ctx, &state)
	if diags.HasError() {
		t.Fatalf("state get: %v", diags)
	}
	if len(state.Ads) != 0 {
		t.Fatalf("expected empty ads, got %#v", state.Ads)
	}
}

func TestAdReportDataSource_APIError(t *testing.T) {
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
	ds := mustAdReportDS(t)
	ds.client = apiClient

	ctx := context.Background()
	schemaResp := &datasource.SchemaResponse{}
	ds.Schema(ctx, datasource.SchemaRequest{}, schemaResp)
	objType := adReportObjType()

	config := tfsdk.Config{
		Schema: schemaResp.Schema,
		Raw: tftypes.NewValue(objType, map[string]tftypes.Value{
			"id":          tftypes.NewValue(tftypes.String, nil),
			"campaign_id": tftypes.NewValue(tftypes.String, "10"),
			"start_time":  tftypes.NewValue(tftypes.String, "2026-09-01"),
			"end_time":    tftypes.NewValue(tftypes.String, "2026-09-14"),
			"time_zone":   tftypes.NewValue(tftypes.String, nil),
			"ads":         tftypes.NewValue(objType.AttributeTypes["ads"], nil),
		}),
	}
	resp := &datasource.ReadResponse{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: tftypes.NewValue(objType, nil)},
	}
	ds.Read(ctx, datasource.ReadRequest{Config: config}, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected API error")
	}
	if resp.Diagnostics.Errors()[0].Summary() != "Unable to read Apple Ads ad report" {
		t.Fatalf("summary = %q", resp.Diagnostics.Errors()[0].Summary())
	}
}

func TestAdReportDataSource_InvalidCampaignID(t *testing.T) {
	t.Parallel()

	apiClient, err := client.New(client.WithBaseURL("http://127.0.0.1:1"))
	if err != nil {
		t.Fatal(err)
	}
	ds := mustAdReportDS(t)
	ds.client = apiClient

	ctx := context.Background()
	schemaResp := &datasource.SchemaResponse{}
	ds.Schema(ctx, datasource.SchemaRequest{}, schemaResp)
	objType := adReportObjType()

	config := tfsdk.Config{
		Schema: schemaResp.Schema,
		Raw: tftypes.NewValue(objType, map[string]tftypes.Value{
			"id":          tftypes.NewValue(tftypes.String, nil),
			"campaign_id": tftypes.NewValue(tftypes.String, "not-a-number"),
			"start_time":  tftypes.NewValue(tftypes.String, "2026-09-01"),
			"end_time":    tftypes.NewValue(tftypes.String, "2026-09-14"),
			"time_zone":   tftypes.NewValue(tftypes.String, nil),
			"ads":         tftypes.NewValue(objType.AttributeTypes["ads"], nil),
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
