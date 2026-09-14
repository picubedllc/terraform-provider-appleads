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
	_ datasource.DataSource              = &keywordBidRecommendationsDataSource{}
	_ datasource.DataSourceWithConfigure = &keywordBidRecommendationsDataSource{}
)

func NewKeywordBidRecommendationsDataSource() datasource.DataSource {
	return &keywordBidRecommendationsDataSource{}
}

type keywordBidRecommendationsDataSource struct {
	client *client.Client
}

type keywordBidRecommendationsModel struct {
	ID         types.String                    `tfsdk:"id"`
	CampaignID types.String                    `tfsdk:"campaign_id"`
	AdGroupID  types.String                    `tfsdk:"ad_group_id"`
	StartTime  types.String                    `tfsdk:"start_time"`
	EndTime    types.String                    `tfsdk:"end_time"`
	TimeZone   types.String                    `tfsdk:"time_zone"`
	Keywords   []keywordBidRecommendationModel `tfsdk:"keywords"`
}

type keywordBidRecommendationModel struct {
	KeywordID            types.String `tfsdk:"keyword_id"`
	Text                 types.String `tfsdk:"text"`
	MatchType            types.String `tfsdk:"match_type"`
	BidAmount            types.String `tfsdk:"bid_amount"`
	BidCurrency          types.String `tfsdk:"bid_currency"`
	SuggestedBidAmount   types.String `tfsdk:"suggested_bid_amount"`
	SuggestedBidCurrency types.String `tfsdk:"suggested_bid_currency"`
	BidMinAmount         types.String `tfsdk:"bid_min_amount"`
	BidMaxAmount         types.String `tfsdk:"bid_max_amount"`
}

func (d *keywordBidRecommendationsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_keyword_bid_recommendations"
}

func (d *keywordBidRecommendationsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads Apple Ads suggested CPT bids for targeting keywords in an ad group.\n\n" +
			"This is **observational only**. Apple's recommendation answers roughly " +
			"\"what bid would make this keyword competitive?\". Do **not** copy `suggested_bid_amount` " +
			"into `appleads_keyword.bid_amount` automatically.\n\n" +
			"Values come from keyword-level reports (`POST /api/v5/reports/campaigns/{campaignId}/adgroups/{adgroupId}/keywords`) " +
			"as `insights.bidRecommendation`. New or paused keywords often have null suggestions until Apple has enough data. " +
			"Deprecated `bidMin`/`bidMax` are exposed when Apple still returns them.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Composite id `campaign_id/ad_group_id/start_time/end_time`.",
			},
			"campaign_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Campaign id that owns the ad group.",
			},
			"ad_group_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Ad group id whose targeting keywords are reported.",
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
				MarkdownDescription: "Report time zone. Defaults to `UTC`.",
			},
			"keywords": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Keyword rows with the current bid and Apple's suggested range when present.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"keyword_id": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Targeting keyword id.",
						},
						"text": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Keyword text.",
						},
						"match_type": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "EXACT or BROAD.",
						},
						"bid_amount": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Configured keyword Max CPT as a decimal string.",
						},
						"bid_currency": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Currency for bid_amount.",
						},
						"suggested_bid_amount": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Apple suggested CPT as a decimal string. Null when Apple omits a suggestion. Do not auto-apply.",
						},
						"suggested_bid_currency": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Currency for suggested_bid_amount.",
						},
						"bid_min_amount": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Deprecated Apple bidMin amount, when still returned.",
						},
						"bid_max_amount": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Deprecated Apple bidMax amount, when still returned.",
						},
					},
				},
			},
		},
	}
}

func (d *keywordBidRecommendationsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	data, ok := req.ProviderData.(*ProviderData)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected data source configure type",
			fmt.Sprintf("Expected *ProviderData, got: %T", req.ProviderData),
		)
		return
	}
	d.client = data.Client
}

func (d *keywordBidRecommendationsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config keywordBidRecommendationsModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if d.client == nil {
		resp.Diagnostics.AddError("Client not configured", "The provider client was not configured before reading appleads_keyword_bid_recommendations.")
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

	tz := "UTC"
	if !config.TimeZone.IsNull() && !config.TimeZone.IsUnknown() && config.TimeZone.ValueString() != "" {
		tz = config.TimeZone.ValueString()
	}

	rows, err := d.client.GetAdGroupKeywordReport(ctx, campaignID, adGroupID, client.DefaultReportingRequest(
		config.StartTime.ValueString(),
		config.EndTime.ValueString(),
		tz,
	))
	if err != nil {
		resp.Diagnostics.AddError("Unable to read Apple Ads keyword bid recommendations", err.Error())
		return
	}

	keywords := make([]keywordBidRecommendationModel, 0, len(rows))
	for _, row := range rows {
		item := keywordBidRecommendationModel{
			KeywordID: types.StringValue(strconv.FormatInt(row.Metadata.KeywordID, 10)),
			Text:      types.StringValue(row.Metadata.Keyword),
			MatchType: types.StringValue(row.Metadata.MatchType),
		}
		item.BidAmount, item.BidCurrency = moneyStrings(row.Metadata.BidAmount)
		if row.Insights != nil && row.Insights.BidRecommendation != nil {
			rec := row.Insights.BidRecommendation
			item.SuggestedBidAmount, item.SuggestedBidCurrency = moneyStrings(rec.SuggestedBidAmount)
			item.BidMinAmount, _ = moneyStrings(rec.BidMin)
			item.BidMaxAmount, _ = moneyStrings(rec.BidMax)
		} else {
			item.SuggestedBidAmount = types.StringNull()
			item.SuggestedBidCurrency = types.StringNull()
			item.BidMinAmount = types.StringNull()
			item.BidMaxAmount = types.StringNull()
		}
		keywords = append(keywords, item)
	}

	state := keywordBidRecommendationsModel{
		ID:         types.StringValue(fmt.Sprintf("%s/%s/%s/%s", config.CampaignID.ValueString(), config.AdGroupID.ValueString(), config.StartTime.ValueString(), config.EndTime.ValueString())),
		CampaignID: config.CampaignID,
		AdGroupID:  config.AdGroupID,
		StartTime:  config.StartTime,
		EndTime:    config.EndTime,
		TimeZone:   types.StringValue(tz),
		Keywords:   keywords,
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func moneyStrings(m *client.Money) (types.String, types.String) {
	if m == nil || m.Amount == "" {
		return types.StringNull(), types.StringNull()
	}
	cur := m.Currency
	if cur == "" {
		return types.StringValue(m.Amount), types.StringNull()
	}
	return types.StringValue(m.Amount), types.StringValue(cur)
}
