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

func TestCampaignReportDataSource_UnitRead(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/reports/campaigns" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"reportingDataResponse": map[string]any{
					"row": []map[string]any{{
						"metadata": map[string]any{"campaignId": 7, "campaignName": "US_wordsync_generic"},
						"total": map[string]any{
							"impressions":    10,
							"taps":           2,
							"ttr":            0.2,
							"localSpend":     map[string]string{"amount": "0.10", "currency": "USD"},
							"avgCPT":         map[string]string{"amount": "0.05", "currency": "USD"},
							"totalInstalls":  1,
							"conversionRate": 0.5,
							"totalAvgCPI":    map[string]string{"amount": "0.10", "currency": "USD"},
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
	ds, ok := NewCampaignReportDataSource().(*campaignReportDataSource)
	if !ok {
		t.Fatalf("type %T", NewCampaignReportDataSource())
	}
	ds.client = apiClient

	ctx := context.Background()
	schemaResp := &datasource.SchemaResponse{}
	ds.Schema(ctx, datasource.SchemaRequest{}, schemaResp)

	metricAttrs := map[string]tftypes.Type{
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
	rowAttrs := map[string]tftypes.Type{
		"campaign_id": tftypes.String,
		"name":        tftypes.String,
	}
	for k, v := range metricAttrs {
		rowAttrs[k] = v
	}
	objType := tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"id":         tftypes.String,
			"start_time": tftypes.String,
			"end_time":   tftypes.String,
			"time_zone":  tftypes.String,
			"campaigns":  tftypes.List{ElementType: tftypes.Object{AttributeTypes: rowAttrs}},
		},
	}
	config := tfsdk.Config{
		Schema: schemaResp.Schema,
		Raw: tftypes.NewValue(objType, map[string]tftypes.Value{
			"id":         tftypes.NewValue(tftypes.String, nil),
			"start_time": tftypes.NewValue(tftypes.String, "2026-09-01"),
			"end_time":   tftypes.NewValue(tftypes.String, "2026-09-14"),
			"time_zone":  tftypes.NewValue(tftypes.String, nil),
			"campaigns":  tftypes.NewValue(objType.AttributeTypes["campaigns"], nil),
		}),
	}
	resp := &datasource.ReadResponse{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: tftypes.NewValue(objType, nil)},
	}
	ds.Read(ctx, datasource.ReadRequest{Config: config}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("diagnostics: %v", resp.Diagnostics)
	}
	var state campaignReportModel
	diags := resp.State.Get(ctx, &state)
	if diags.HasError() {
		t.Fatalf("state get: %v", diags)
	}
	if len(state.Campaigns) != 1 || state.Campaigns[0].Name.ValueString() != "US_wordsync_generic" {
		t.Fatalf("state = %#v", state)
	}
	if state.Campaigns[0].Impressions.ValueInt64() != 10 || state.Campaigns[0].LocalSpendAmount.ValueString() != "0.10" {
		t.Fatalf("metrics = %#v", state.Campaigns[0])
	}
}
