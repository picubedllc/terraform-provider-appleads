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
	_ datasource.DataSource              = &adGroupReportDataSource{}
	_ datasource.DataSourceWithConfigure = &adGroupReportDataSource{}
)

func NewAdGroupReportDataSource() datasource.DataSource { return &adGroupReportDataSource{} }

type adGroupReportDataSource struct{ client *client.Client }

type adGroupReportModel struct {
	ID         types.String            `tfsdk:"id"`
	CampaignID types.String            `tfsdk:"campaign_id"`
	StartTime  types.String            `tfsdk:"start_time"`
	EndTime    types.String            `tfsdk:"end_time"`
	TimeZone   types.String            `tfsdk:"time_zone"`
	AdGroups   []adGroupReportRowModel `tfsdk:"ad_groups"`
}

type adGroupReportRowModel struct {
	AdGroupID types.String `tfsdk:"ad_group_id"`
	Name      types.String `tfsdk:"name"`
	reportMetricsModel
}

func (d *adGroupReportDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ad_group_report"
}

func (d *adGroupReportDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads Apple Ads ad group-level performance within a campaign.\n\n" +
			"Read-only snapshot from `POST /api/v5/reports/campaigns/{campaignId}/adgroups`.",
		Attributes: map[string]schema.Attribute{
			"id":          schema.StringAttribute{Computed: true, MarkdownDescription: "Composite id `campaign_id/start_time/end_time`."},
			"campaign_id": schema.StringAttribute{Required: true, MarkdownDescription: "Parent campaign id."},
			"start_time":  schema.StringAttribute{Required: true, MarkdownDescription: "Report start date (`YYYY-MM-DD`).", Validators: []validator.String{stringvalidator.LengthAtLeast(10)}},
			"end_time":    schema.StringAttribute{Required: true, MarkdownDescription: "Report end date (`YYYY-MM-DD`).", Validators: []validator.String{stringvalidator.LengthAtLeast(10)}},
			"time_zone":   schema.StringAttribute{Optional: true, Computed: true, MarkdownDescription: "Report time zone. Defaults to `UTC`."},
			"ad_groups": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Ad group rows for the window.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: mergeAttributes(map[string]schema.Attribute{
						"ad_group_id": schema.StringAttribute{Computed: true, MarkdownDescription: "Ad group id."},
						"name":        schema.StringAttribute{Computed: true, MarkdownDescription: "Ad group name."},
					}, reportMetricsAttributes()),
				},
			},
		},
	}
}

func (d *adGroupReportDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	configureReportClient(req, resp, &d.client)
}

func (d *adGroupReportDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config adGroupReportModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if d.client == nil {
		resp.Diagnostics.AddError("Client not configured", "The provider client was not configured before reading appleads_ad_group_report.")
		return
	}
	campaignID, err := strconv.ParseInt(config.CampaignID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid campaign_id", fmt.Sprintf("campaign_id must be numeric; got %q", config.CampaignID.ValueString()))
		return
	}
	tz := reportTimeZone(config.TimeZone)
	rows, err := d.client.GetAdGroupReport(ctx, campaignID, client.DefaultReportingRequest(config.StartTime.ValueString(), config.EndTime.ValueString(), tz))
	if err != nil {
		resp.Diagnostics.AddError("Unable to read Apple Ads ad group report", err.Error())
		return
	}
	out := make([]adGroupReportRowModel, 0, len(rows))
	for _, row := range rows {
		out = append(out, adGroupReportRowModel{
			AdGroupID:          types.StringValue(strconv.FormatInt(row.Metadata.AdGroupID, 10)),
			Name:               types.StringValue(row.Metadata.AdGroupName),
			reportMetricsModel: metricsFromSpend(row.Total),
		})
	}
	state := adGroupReportModel{
		ID:         types.StringValue(fmt.Sprintf("%s/%s/%s", config.CampaignID.ValueString(), config.StartTime.ValueString(), config.EndTime.ValueString())),
		CampaignID: config.CampaignID,
		StartTime:  config.StartTime,
		EndTime:    config.EndTime,
		TimeZone:   types.StringValue(tz),
		AdGroups:   out,
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
