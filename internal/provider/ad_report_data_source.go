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
	_ datasource.DataSource              = &adReportDataSource{}
	_ datasource.DataSourceWithConfigure = &adReportDataSource{}
)

func NewAdReportDataSource() datasource.DataSource { return &adReportDataSource{} }

type adReportDataSource struct{ client *client.Client }

type adReportModel struct {
	ID         types.String       `tfsdk:"id"`
	CampaignID types.String       `tfsdk:"campaign_id"`
	StartTime  types.String       `tfsdk:"start_time"`
	EndTime    types.String       `tfsdk:"end_time"`
	TimeZone   types.String       `tfsdk:"time_zone"`
	Ads        []adReportRowModel `tfsdk:"ads"`
}

type adReportRowModel struct {
	AdID          types.String `tfsdk:"ad_id"`
	Name          types.String `tfsdk:"name"`
	AdGroupID     types.String `tfsdk:"ad_group_id"`
	CreativeID    types.String `tfsdk:"creative_id"`
	CreativeType  types.String `tfsdk:"creative_type"`
	Language      types.String `tfsdk:"language"`
	DisplayStatus types.String `tfsdk:"display_status"`
	reportMetricsModel
}

func (d *adReportDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ad_report"
}

func (d *adReportDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads Apple Ads ad-level performance within a campaign.\n\n" +
			"Read-only snapshot from `POST /api/v5/reports/campaigns/{campaignId}/ads`.",
		Attributes: map[string]schema.Attribute{
			"id":          schema.StringAttribute{Computed: true, MarkdownDescription: "Composite id `campaign_id/start_time/end_time`."},
			"campaign_id": schema.StringAttribute{Required: true, MarkdownDescription: "Parent campaign id."},
			"start_time":  schema.StringAttribute{Required: true, MarkdownDescription: "Report start date (`YYYY-MM-DD`).", Validators: []validator.String{stringvalidator.LengthAtLeast(10)}},
			"end_time":    schema.StringAttribute{Required: true, MarkdownDescription: "Report end date (`YYYY-MM-DD`).", Validators: []validator.String{stringvalidator.LengthAtLeast(10)}},
			"time_zone":   schema.StringAttribute{Optional: true, Computed: true, MarkdownDescription: "Report time zone. Defaults to `UTC`."},
			"ads": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Ad rows for the window.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: mergeAttributes(map[string]schema.Attribute{
						"ad_id":          schema.StringAttribute{Computed: true, MarkdownDescription: "Ad id."},
						"name":           schema.StringAttribute{Computed: true, MarkdownDescription: "Ad name."},
						"ad_group_id":    schema.StringAttribute{Computed: true, MarkdownDescription: "Parent ad group id."},
						"creative_id":    schema.StringAttribute{Computed: true, MarkdownDescription: "Creative id, when present."},
						"creative_type":  schema.StringAttribute{Computed: true, MarkdownDescription: "Creative type (for example `CUSTOM_PRODUCT_PAGE`)."},
						"language":       schema.StringAttribute{Computed: true, MarkdownDescription: "Creative language, when present."},
						"display_status": schema.StringAttribute{Computed: true, MarkdownDescription: "Ad display status, when present."},
					}, reportMetricsAttributes()),
				},
			},
		},
	}
}

func (d *adReportDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	configureReportClient(req, resp, &d.client)
}

func (d *adReportDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config adReportModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if d.client == nil {
		resp.Diagnostics.AddError("Client not configured", "The provider client was not configured before reading appleads_ad_report.")
		return
	}
	campaignID, err := strconv.ParseInt(config.CampaignID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid campaign_id", fmt.Sprintf("campaign_id must be numeric; got %q", config.CampaignID.ValueString()))
		return
	}
	tz := reportTimeZone(config.TimeZone)
	rows, err := d.client.GetAdReport(ctx, campaignID, client.DefaultReportingRequest(config.StartTime.ValueString(), config.EndTime.ValueString(), tz))
	if err != nil {
		resp.Diagnostics.AddError("Unable to read Apple Ads ad report", err.Error())
		return
	}
	out := make([]adReportRowModel, 0, len(rows))
	for _, row := range rows {
		creativeID := types.StringNull()
		if row.Metadata.CreativeID != 0 {
			creativeID = types.StringValue(strconv.FormatInt(row.Metadata.CreativeID, 10))
		}
		out = append(out, adReportRowModel{
			AdID:               types.StringValue(strconv.FormatInt(row.Metadata.AdID, 10)),
			Name:               types.StringValue(row.Metadata.AdName),
			AdGroupID:          types.StringValue(strconv.FormatInt(row.Metadata.AdGroupID, 10)),
			CreativeID:         creativeID,
			CreativeType:       stringOrNull(row.Metadata.CreativeType),
			Language:           stringOrNull(row.Metadata.Language),
			DisplayStatus:      stringOrNull(row.Metadata.DisplayStatus),
			reportMetricsModel: metricsFromSpend(row.Total),
		})
	}
	state := adReportModel{
		ID:         types.StringValue(fmt.Sprintf("%s/%s/%s", config.CampaignID.ValueString(), config.StartTime.ValueString(), config.EndTime.ValueString())),
		CampaignID: config.CampaignID,
		StartTime:  config.StartTime,
		EndTime:    config.EndTime,
		TimeZone:   types.StringValue(tz),
		Ads:        out,
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func stringOrNull(v string) types.String {
	if v == "" {
		return types.StringNull()
	}
	return types.StringValue(v)
}
