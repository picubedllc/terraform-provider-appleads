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
	SearchTerms []searchTermReportRowModel `tfsdk:"search_terms"`
}

type searchTermReportRowModel struct {
	SearchTermText types.String `tfsdk:"search_term_text"`
	KeywordID      types.String `tfsdk:"keyword_id"`
	KeywordText    types.String `tfsdk:"keyword_text"`
	MatchType      types.String `tfsdk:"match_type"`
	reportMetricsModel
}

func (d *searchTermReportDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_search_term_report"
}

func (d *searchTermReportDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads Apple Ads search-term reports so Discovery terms can be inspected before a **manual** promotion into an Exact campaign.\n\n" +
			"Does **not** create keywords or negatives. Promotion remains a Terraform config change.\n\n" +
			"Apple requires `timeZone=ORTZ` and typically omits terms with fewer than 10 impressions. " +
			"Uses `POST /api/v5/reports/campaigns/{campaignId}/searchterms` or the ad-group-scoped path when `ad_group_id` is set.",
		Attributes: map[string]schema.Attribute{
			"id":          schema.StringAttribute{Computed: true, MarkdownDescription: "Composite id of the report window and scope."},
			"campaign_id": schema.StringAttribute{Required: true, MarkdownDescription: "Campaign id (usually Discovery)."},
			"ad_group_id": schema.StringAttribute{Optional: true, MarkdownDescription: "When set, scope the report to this ad group."},
			"start_time":  schema.StringAttribute{Required: true, MarkdownDescription: "Report start date (`YYYY-MM-DD`).", Validators: []validator.String{stringvalidator.LengthAtLeast(10)}},
			"end_time":    schema.StringAttribute{Required: true, MarkdownDescription: "Report end date (`YYYY-MM-DD`).", Validators: []validator.String{stringvalidator.LengthAtLeast(10)}},
			"search_terms": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Search term rows for the window.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: mergeAttributes(map[string]schema.Attribute{
						"search_term_text": schema.StringAttribute{Computed: true, MarkdownDescription: "Query Apple matched."},
						"keyword_id":       schema.StringAttribute{Computed: true, MarkdownDescription: "Matched targeting keyword id, when present."},
						"keyword_text":     schema.StringAttribute{Computed: true, MarkdownDescription: "Matched targeting keyword text, when present."},
						"match_type":       schema.StringAttribute{Computed: true, MarkdownDescription: "Match type of the matched keyword, when present."},
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
	reqBody := client.DefaultReportingRequest(config.StartTime.ValueString(), config.EndTime.ValueString(), "ORTZ")
	var rows []client.ReportingRow[client.ReportingSearchTermMetadata]
	if !config.AdGroupID.IsNull() && !config.AdGroupID.IsUnknown() && config.AdGroupID.ValueString() != "" {
		adGroupID, parseErr := strconv.ParseInt(config.AdGroupID.ValueString(), 10, 64)
		if parseErr != nil {
			resp.Diagnostics.AddError("Invalid ad_group_id", fmt.Sprintf("ad_group_id must be numeric; got %q", config.AdGroupID.ValueString()))
			return
		}
		rows, err = d.client.GetAdGroupSearchTermReport(ctx, campaignID, adGroupID, reqBody)
	} else {
		rows, err = d.client.GetCampaignSearchTermReport(ctx, campaignID, reqBody)
	}
	if err != nil {
		resp.Diagnostics.AddError("Unable to read Apple Ads search term report", err.Error())
		return
	}
	out := make([]searchTermReportRowModel, 0, len(rows))
	for _, row := range rows {
		item := searchTermReportRowModel{
			SearchTermText:     types.StringValue(row.Metadata.SearchTermText),
			KeywordText:        types.StringValue(row.Metadata.Keyword),
			MatchType:          types.StringValue(row.Metadata.MatchType),
			reportMetricsModel: metricsFromSpend(row.Total),
		}
		if row.Metadata.KeywordID != 0 {
			item.KeywordID = types.StringValue(strconv.FormatInt(row.Metadata.KeywordID, 10))
		} else {
			item.KeywordID = types.StringNull()
		}
		out = append(out, item)
	}
	id := fmt.Sprintf("%s/%s/%s", config.CampaignID.ValueString(), config.StartTime.ValueString(), config.EndTime.ValueString())
	if !config.AdGroupID.IsNull() && config.AdGroupID.ValueString() != "" {
		id = fmt.Sprintf("%s/%s/%s/%s", config.CampaignID.ValueString(), config.AdGroupID.ValueString(), config.StartTime.ValueString(), config.EndTime.ValueString())
	}
	state := searchTermReportModel{
		ID:          types.StringValue(id),
		CampaignID:  config.CampaignID,
		AdGroupID:   config.AdGroupID,
		StartTime:   config.StartTime,
		EndTime:     config.EndTime,
		SearchTerms: out,
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
