// Copyright (c) 2026 Pi Cubed LLC
// SPDX-License-Identifier: MIT

package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/picubedllc/terraform-provider-appleads/internal/client"
)

type reportMetricsModel struct {
	Impressions        types.Int64   `tfsdk:"impressions"`
	Taps               types.Int64   `tfsdk:"taps"`
	TTR                types.Float64 `tfsdk:"ttr"`
	LocalSpendAmount   types.String  `tfsdk:"local_spend_amount"`
	LocalSpendCurrency types.String  `tfsdk:"local_spend_currency"`
	AvgCPTAmount       types.String  `tfsdk:"avg_cpt_amount"`
	AvgCPTCurrency     types.String  `tfsdk:"avg_cpt_currency"`
	Installs           types.Int64   `tfsdk:"installs"`
	ConversionRate     types.Float64 `tfsdk:"conversion_rate"`
	AvgCPAAmount       types.String  `tfsdk:"avg_cpa_amount"`
	AvgCPACurrency     types.String  `tfsdk:"avg_cpa_currency"`
}

func reportMetricsAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"impressions": schema.Int64Attribute{
			Computed:            true,
			MarkdownDescription: "Impressions in the report window.",
		},
		"taps": schema.Int64Attribute{
			Computed:            true,
			MarkdownDescription: "Taps in the report window.",
		},
		"ttr": schema.Float64Attribute{
			Computed:            true,
			MarkdownDescription: "Tap-through rate.",
		},
		"local_spend_amount": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "Spend as a decimal string.",
		},
		"local_spend_currency": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "Spend currency.",
		},
		"avg_cpt_amount": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "Average CPT as a decimal string.",
		},
		"avg_cpt_currency": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "Average CPT currency.",
		},
		"installs": schema.Int64Attribute{
			Computed:            true,
			MarkdownDescription: "Total installs (tap + view) in the report window.",
		},
		"conversion_rate": schema.Float64Attribute{
			Computed:            true,
			MarkdownDescription: "Conversion rate.",
		},
		"avg_cpa_amount": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "Average CPA / CPI as a decimal string (`totalAvgCPI`).",
		},
		"avg_cpa_currency": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "Average CPA currency.",
		},
	}
}

func mergeAttributes(parts ...map[string]schema.Attribute) map[string]schema.Attribute {
	out := map[string]schema.Attribute{}
	for _, part := range parts {
		for k, v := range part {
			out[k] = v
		}
	}
	return out
}

func metricsFromSpend(s *client.SpendRow) reportMetricsModel {
	if s == nil {
		return reportMetricsModel{
			Impressions:        types.Int64Value(0),
			Taps:               types.Int64Value(0),
			TTR:                types.Float64Value(0),
			LocalSpendAmount:   types.StringNull(),
			LocalSpendCurrency: types.StringNull(),
			AvgCPTAmount:       types.StringNull(),
			AvgCPTCurrency:     types.StringNull(),
			Installs:           types.Int64Value(0),
			ConversionRate:     types.Float64Value(0),
			AvgCPAAmount:       types.StringNull(),
			AvgCPACurrency:     types.StringNull(),
		}
	}
	spendAmt, spendCur := moneyStrings(s.LocalSpend)
	cptAmt, cptCur := moneyStrings(s.AvgCPT)
	cpaAmt, cpaCur := moneyStrings(s.TotalAvgCPI)
	return reportMetricsModel{
		Impressions:        types.Int64Value(s.Impressions),
		Taps:               types.Int64Value(s.Taps),
		TTR:                types.Float64Value(s.TTR),
		LocalSpendAmount:   spendAmt,
		LocalSpendCurrency: spendCur,
		AvgCPTAmount:       cptAmt,
		AvgCPTCurrency:     cptCur,
		Installs:           types.Int64Value(s.TotalInstalls),
		ConversionRate:     types.Float64Value(s.ConversionRate),
		AvgCPAAmount:       cpaAmt,
		AvgCPACurrency:     cpaCur,
	}
}
