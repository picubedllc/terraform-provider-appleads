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
	_ datasource.DataSource              = &searchTermReportDataSource{}
	_ datasource.DataSourceWithConfigure = &searchTermReportDataSource{}
)

func NewSearchTermReportDataSource() datasource.DataSource { return &searchTermReportDataSource{} }

type searchTermReportDataSource struct{ client *client.Client }

type searchTermReportModel struct {
	ID          types.String               `tfsdk:"id"`
	CampaignID  types.String               `tfsdk:"campaign_id"`
	AdGroupID   types.String               `tfsdk:"ad_group_id"`
	StartTime   types.String               `tfsdk:"start_time"`
	EndTime     types.String               `tfsdk:"end_time"`
	TimeZone    types.String               `tfsdk:"time_zone"`
	SearchTerms []searchTermReportRowModel `tfsdk:"search_terms"`
}

type searchTermReportRowModel struct {
	SearchTermText types.String `tfsdk:"search_term_text"`
	KeywordID      types.String `tfsdk:"keyword_id"`
	Keyword        types.String `tfsdk:"keyword"`
	MatchType      types.String `tfsdk:"match_type"`
	AdGroupID      types.String `tfsdk:"ad_group_id"`
	AdGroupName    types.String `tfsdk:"ad_group_name"`
	BidAmount      types.String `tfsdk:"bid_amount"`
	BidCurrency    types.String `tfsdk:"bid_currency"`
	Deleted        types.Bool   `tfsdk:"deleted"`
	reportMetricsModel
}

func (d *searchTermReportDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_search_term_report"
}

func (d *searchTermReportDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads Apple Ads search-term performance for a campaign (optionally scoped to an ad group).\n\n" +
			"Read-only snapshot from `POST /api/v5/reports/campaigns/{campaignId}/searchterms` " +
			"or `POST /api/v5/reports/campaigns/{campaignId}/adgroups/{adgroupId}/searchterms` when `ad_group_id` is set. " +
			"Apple requires `time_zone` `ORTZ` (organization time zone); that is the default. " +
			"Search terms typically appear only after about 10 impressions. " +
			"Observational only: do not drive bids from this data source in the same apply without care.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Composite id `campaign_id[/ad_group_id]/start_time/end_time`.",
			},
			"campaign_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Parent campaign id.",
			},
			"ad_group_id": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "When set, scopes the report to this ad group.",
			},
			"start_time": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Report start date (`YYYY-MM-DD`).",
				Validators:          []validator.String{stringvalidator.LengthAtLeast(10)},
			},
			"end_time": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Report end date (`YYYY-MM-DD`).",
				Validators:          []validator.String{stringvalidator.LengthAtLeast(10)},
			},
			"time_zone": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Report time zone. Defaults to `ORTZ` (required by Apple for search-term reports).",
			},
			"search_terms": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Search-term rows for the window.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: mergeAttributes(map[string]schema.Attribute{
						"search_term_text": schema.StringAttribute{Computed: true, MarkdownDescription: "Search term text."},
						"keyword_id":       schema.StringAttribute{Computed: true, MarkdownDescription: "Matched keyword id."},
						"keyword":          schema.StringAttribute{Computed: true, MarkdownDescription: "Matched keyword text."},
						"match_type":       schema.StringAttribute{Computed: true, MarkdownDescription: "EXACT or BROAD."},
						"ad_group_id":      schema.StringAttribute{Computed: true, MarkdownDescription: "Ad group id for the row."},
						"ad_group_name":    schema.StringAttribute{Computed: true, MarkdownDescription: "Ad group name for the row."},
						"bid_amount":       schema.StringAttribute{Computed: true, MarkdownDescription: "Keyword bid amount as a decimal string, when present."},
						"bid_currency":     schema.StringAttribute{Computed: true, MarkdownDescription: "Keyword bid currency, when present."},
						"deleted":          schema.BoolAttribute{Computed: true, MarkdownDescription: "Whether the matched keyword is deleted."},
					}, reportMetricsAttributes()),
				},
			},
		},
	}
}

func (d *searchTermReportDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	configureReportClient(req, resp, &d.client)
}

func (d *searchTermReportDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config searchTermReportModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if d.client == nil {
		resp.Diagnostics.AddError("Client not configured", "The provider client was not configured before reading appleads_search_term_report.")
		return
	}
	campaignID, err := strconv.ParseInt(config.CampaignID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid campaign_id", fmt.Sprintf("campaign_id must be numeric; got %q", config.CampaignID.ValueString()))
		return
	}

	tz := searchTermReportTimeZone(config.TimeZone)
	reportReq := client.DefaultReportingRequest(config.StartTime.ValueString(), config.EndTime.ValueString(), tz)

	var rows []client.ReportingRow[client.ReportingSearchTermMetadata]
	idParts := config.CampaignID.ValueString()
	adGroupIDAttr := types.StringNull()

	if !config.AdGroupID.IsNull() && !config.AdGroupID.IsUnknown() && config.AdGroupID.ValueString() != "" {
		adGroupID, parseErr := strconv.ParseInt(config.AdGroupID.ValueString(), 10, 64)
		if parseErr != nil {
			resp.Diagnostics.AddError("Invalid ad_group_id", fmt.Sprintf("ad_group_id must be numeric; got %q", config.AdGroupID.ValueString()))
			return
		}
		rows, err = d.client.GetAdGroupSearchTermReport(ctx, campaignID, adGroupID, reportReq)
		idParts = fmt.Sprintf("%s/%s", config.CampaignID.ValueString(), config.AdGroupID.ValueString())
		adGroupIDAttr = config.AdGroupID
	} else {
		rows, err = d.client.GetCampaignSearchTermReport(ctx, campaignID, reportReq)
	}
	if err != nil {
		resp.Diagnostics.AddError("Unable to read Apple Ads search term report", err.Error())
		return
	}

	out := make([]searchTermReportRowModel, 0, len(rows))
	for _, row := range rows {
		bidAmt, bidCur := moneyStrings(row.Metadata.BidAmount)
		item := searchTermReportRowModel{
			SearchTermText:     types.StringValue(row.Metadata.SearchTermText),
			KeywordID:          types.StringValue(strconv.FormatInt(row.Metadata.KeywordID, 10)),
			Keyword:            types.StringValue(row.Metadata.Keyword),
			MatchType:          types.StringValue(row.Metadata.MatchType),
			AdGroupID:          types.StringValue(strconv.FormatInt(row.Metadata.AdGroupID, 10)),
			AdGroupName:        types.StringValue(row.Metadata.AdGroupName),
			BidAmount:          bidAmt,
			BidCurrency:        bidCur,
			Deleted:            types.BoolValue(row.Metadata.Deleted),
			reportMetricsModel: metricsFromSpend(row.Total),
		}
		out = append(out, item)
	}

	state := searchTermReportModel{
		ID:          types.StringValue(fmt.Sprintf("%s/%s/%s", idParts, config.StartTime.ValueString(), config.EndTime.ValueString())),
		CampaignID:  config.CampaignID,
		AdGroupID:   adGroupIDAttr,
		StartTime:   config.StartTime,
		EndTime:     config.EndTime,
		TimeZone:    types.StringValue(tz),
		SearchTerms: out,
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func searchTermReportTimeZone(v types.String) string {
	if !v.IsNull() && !v.IsUnknown() && v.ValueString() != "" {
		return v.ValueString()
	}
	return "ORTZ"
}
