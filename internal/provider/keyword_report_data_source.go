// Copyright (c) 2026 Pi Cubed LLC
// SPDX-License-Identifier: MIT

package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/picubedllc/terraform-provider-appleads/internal/client"
)

var (
	_ datasource.DataSource              = &keywordReportDataSource{}
	_ datasource.DataSourceWithConfigure = &keywordReportDataSource{}
)

func NewKeywordReportDataSource() datasource.DataSource { return &keywordReportDataSource{} }

type keywordReportDataSource struct{ client *client.Client }

type keywordReportModel struct {
	ID         types.String            `tfsdk:"id"`
	CampaignID types.String            `tfsdk:"campaign_id"`
	AdGroupID  types.String            `tfsdk:"ad_group_id"`
	StartTime  types.String            `tfsdk:"start_time"`
	EndTime    types.String            `tfsdk:"end_time"`
	TimeZone   types.String            `tfsdk:"time_zone"`
	Keywords   []keywordReportRowModel `tfsdk:"keywords"`
}

type keywordReportRowModel struct {
	KeywordID types.String `tfsdk:"keyword_id"`
	Text      types.String `tfsdk:"text"`
	MatchType types.String `tfsdk:"match_type"`
	reportMetricsModel
}

func (d *keywordReportDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_keyword_report"
}

func (d *keywordReportDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads Apple Ads keyword-level performance within an ad group.\n\n" +
			"Read-only snapshot from `POST /api/v5/reports/campaigns/{campaignId}/adgroups/{adgroupId}/keywords`. " +
			"For Apple suggested CPT, use `appleads_keyword_bid_recommendations` instead of copying report bids into configuration.",
		Attributes: map[string]schema.Attribute{
			"id":          schema.StringAttribute{Computed: true, MarkdownDescription: "Composite id `campaign_id/ad_group_id/start_time/end_time`."},
			"campaign_id": schema.StringAttribute{Required: true, MarkdownDescription: "Parent campaign id."},
			"ad_group_id": schema.StringAttribute{Required: true, MarkdownDescription: "Parent ad group id."},
			"start_time":  schema.StringAttribute{Required: true, MarkdownDescription: "Report start date (`YYYY-MM-DD`).", Validators: []validator.String{stringvalidator.LengthAtLeast(10)}},
			"end_time":    schema.StringAttribute{Required: true, MarkdownDescription: "Report end date (`YYYY-MM-DD`).", Validators: []validator.String{stringvalidator.LengthAtLeast(10)}},
			"time_zone":   schema.StringAttribute{Optional: true, Computed: true, MarkdownDescription: "Report time zone. Defaults to `UTC`."},
			"keywords": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Keyword rows for the window.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: mergeAttributes(map[string]schema.Attribute{
						"keyword_id": schema.StringAttribute{Computed: true, MarkdownDescription: "Keyword id."},
						"text":       schema.StringAttribute{Computed: true, MarkdownDescription: "Keyword text."},
						"match_type": schema.StringAttribute{Computed: true, MarkdownDescription: "EXACT or BROAD."},
					}, reportMetricsAttributes()),
				},
			},
		},
	}
}

func (d *keywordReportDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	configureReportClient(req, resp, &d.client)
}

func (d *keywordReportDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config keywordReportModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if d.client == nil {
		resp.Diagnostics.AddError("Client not configured", "The provider client was not configured before reading appleads_keyword_report.")
		return
	}
	campaignID, err := strconv.ParseInt(config.CampaignID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid campaign_id", fmt.Sprintf("campaign_id must be numeric; got %q", config.CampaignID.ValueString()))
		return
	}
	adGroupID, err := strconv.ParseInt(config.AdGroupID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid ad_group_id", fmt.Sprintf("ad_group_id must be numeric; got %q", config.AdGroupID.ValueString()))
		return
	}
	tz := reportTimeZone(config.TimeZone)
	rows, err := d.client.GetAdGroupKeywordReport(ctx, campaignID, adGroupID, client.DefaultReportingRequest(config.StartTime.ValueString(), config.EndTime.ValueString(), tz))
	if err != nil {
		resp.Diagnostics.AddError("Unable to read Apple Ads keyword report", err.Error())
		return
	}
	out := make([]keywordReportRowModel, 0, len(rows))
	for _, row := range rows {
		out = append(out, keywordReportRowModel{
			KeywordID:          types.StringValue(strconv.FormatInt(row.Metadata.KeywordID, 10)),
			Text:               types.StringValue(row.Metadata.Keyword),
			MatchType:          types.StringValue(row.Metadata.MatchType),
			reportMetricsModel: metricsFromSpend(row.Total),
		})
	}
	state := keywordReportModel{
		ID:         types.StringValue(fmt.Sprintf("%s/%s/%s/%s", config.CampaignID.ValueString(), config.AdGroupID.ValueString(), config.StartTime.ValueString(), config.EndTime.ValueString())),
		CampaignID: config.CampaignID,
		AdGroupID:  config.AdGroupID,
		StartTime:  config.StartTime,
		EndTime:    config.EndTime,
		TimeZone:   types.StringValue(tz),
		Keywords:   out,
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
