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
	_ datasource.DataSource              = &campaignReportDataSource{}
	_ datasource.DataSourceWithConfigure = &campaignReportDataSource{}
)

func NewCampaignReportDataSource() datasource.DataSource { return &campaignReportDataSource{} }

type campaignReportDataSource struct{ client *client.Client }

type campaignReportModel struct {
	ID        types.String             `tfsdk:"id"`
	StartTime types.String             `tfsdk:"start_time"`
	EndTime   types.String             `tfsdk:"end_time"`
	TimeZone  types.String             `tfsdk:"time_zone"`
	Campaigns []campaignReportRowModel `tfsdk:"campaigns"`
}

type campaignReportRowModel struct {
	CampaignID types.String `tfsdk:"campaign_id"`
	Name       types.String `tfsdk:"name"`
	reportMetricsModel
}

func (d *campaignReportDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_campaign_report"
}

func (d *campaignReportDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads Apple Ads campaign-level performance for a date window.\n\n" +
			"Read-only snapshot from `POST /api/v5/reports/campaigns`. Not a substitute for a warehouse pipeline.",
		Attributes: map[string]schema.Attribute{
			"id":         schema.StringAttribute{Computed: true, MarkdownDescription: "Composite id `start_time/end_time`."},
			"start_time": schema.StringAttribute{Required: true, MarkdownDescription: "Report start date (`YYYY-MM-DD`).", Validators: []validator.String{stringvalidator.LengthAtLeast(10)}},
			"end_time":   schema.StringAttribute{Required: true, MarkdownDescription: "Report end date (`YYYY-MM-DD`).", Validators: []validator.String{stringvalidator.LengthAtLeast(10)}},
			"time_zone":  schema.StringAttribute{Optional: true, Computed: true, MarkdownDescription: "Report time zone. Defaults to `UTC`."},
			"campaigns": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Campaign rows for the window.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: mergeAttributes(map[string]schema.Attribute{
						"campaign_id": schema.StringAttribute{Computed: true, MarkdownDescription: "Campaign id."},
						"name":        schema.StringAttribute{Computed: true, MarkdownDescription: "Campaign name."},
					}, reportMetricsAttributes()),
				},
			},
		},
	}
}

func (d *campaignReportDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	configureReportClient(req, resp, &d.client)
}

func (d *campaignReportDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config campaignReportModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if d.client == nil {
		resp.Diagnostics.AddError("Client not configured", "The provider client was not configured before reading appleads_campaign_report.")
		return
	}
	tz := reportTimeZone(config.TimeZone)
	rows, err := d.client.GetCampaignReport(ctx, client.DefaultReportingRequest(config.StartTime.ValueString(), config.EndTime.ValueString(), tz))
	if err != nil {
		resp.Diagnostics.AddError("Unable to read Apple Ads campaign report", err.Error())
		return
	}
	out := make([]campaignReportRowModel, 0, len(rows))
	for _, row := range rows {
		item := campaignReportRowModel{
			CampaignID:         types.StringValue(strconv.FormatInt(row.Metadata.CampaignID, 10)),
			Name:               types.StringValue(row.Metadata.CampaignName),
			reportMetricsModel: metricsFromSpend(row.Total),
		}
		out = append(out, item)
	}
	state := campaignReportModel{
		ID:        types.StringValue(fmt.Sprintf("%s/%s", config.StartTime.ValueString(), config.EndTime.ValueString())),
		StartTime: config.StartTime,
		EndTime:   config.EndTime,
		TimeZone:  types.StringValue(tz),
		Campaigns: out,
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func reportTimeZone(v types.String) string {
	if !v.IsNull() && !v.IsUnknown() && v.ValueString() != "" {
		return v.ValueString()
	}
	return "UTC"
}

func configureReportClient(req datasource.ConfigureRequest, resp *datasource.ConfigureResponse, dest **client.Client) {
	if req.ProviderData == nil {
		return
	}
	data, ok := req.ProviderData.(*ProviderData)
	if !ok {
		resp.Diagnostics.AddError("Unexpected data source configure type", fmt.Sprintf("Expected *ProviderData, got: %T", req.ProviderData))
		return
	}
	*dest = data.Client
}
