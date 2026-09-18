// Copyright (c) 2026 Pi Cubed LLC
// SPDX-License-Identifier: MIT

package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"

	"github.com/picubedllc/terraform-provider-appleads/internal/client"
)

// campaignComboMarkdown documents Apple Ads channel/supply/billing/bidding
// pairing rules enforced at plan time. Display delivery still needs creatives
// and ads resources (Phase 3) before ads can serve.
const campaignComboMarkdown = `

## Channel / supply / billing / bidding matrix

Apple Ads Campaign Management API v5 allows these combinations only:

| Channel | Supply sources | Billing | Bidding |
|---|---|---|---|
| ` + "`SEARCH`" + ` | ` + "`APPSTORE_SEARCH_RESULTS`" + ` | ` + "`TAPS`" + ` | ` + "`MANUAL_CPT`" + ` or ` + "`MAX_CONVERSIONS`" + ` |
| ` + "`DISPLAY`" + ` | ` + "`APPSTORE_TODAY_TAB`" + ` / ` + "`APPSTORE_SEARCH_TAB`" + ` / ` + "`APPSTORE_PRODUCT_PAGES_BROWSE`" + ` | ` + "`TAPS`" + ` | ` + "`MANUAL_CPT`" + ` only |

` + "`MAX_CONVERSIONS`" + ` requires Search Results supply. Target CPA schema support lands with bidding follow-up work (#44); until then Max Conversions may still need Apple-side target CPA.

**Display note:** Configuring a Display campaign is supported for create/read, but end-to-end delivery still requires creatives/ads (not yet managed by this provider).
`

func (r *campaignResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config campaignModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	channel, supply, billing, bidding, skip, diags := resolveCampaignComboConfig(ctx, config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() || skip {
		return
	}

	resp.Diagnostics.Append(validateCampaignCombo(channel, supply, billing, bidding)...)
}

// resolveCampaignComboConfig applies the same create-time defaults used by
// campaignCreateFromPlan so omitted Optional/Computed values are validated as
// Apple will receive them. Returns skip=true when any input is still unknown.
func resolveCampaignComboConfig(ctx context.Context, config campaignModel) (channel string, supply []string, billing, bidding string, skip bool, diags diag.Diagnostics) {
	if config.AdChannelType.IsUnknown() || config.SupplySources.IsUnknown() ||
		config.BillingEvent.IsUnknown() || config.BiddingStrategy.IsUnknown() {
		return "", nil, "", "", true, diags
	}

	channel = client.AdChannelTypeSearch
	if !config.AdChannelType.IsNull() && config.AdChannelType.ValueString() != "" {
		channel = config.AdChannelType.ValueString()
	}

	supply, d := stringList(ctx, config.SupplySources)
	diags.Append(d...)
	if diags.HasError() {
		return "", nil, "", "", false, diags
	}
	if len(supply) == 0 {
		supply = []string{client.SupplySourceSearchResults}
	}

	billing = client.BillingEventTaps
	if !config.BillingEvent.IsNull() && config.BillingEvent.ValueString() != "" {
		billing = config.BillingEvent.ValueString()
	}

	bidding = client.BiddingStrategyManualCPT
	if !config.BiddingStrategy.IsNull() && config.BiddingStrategy.ValueString() != "" {
		bidding = config.BiddingStrategy.ValueString()
	}

	return channel, supply, billing, bidding, false, diags
}

// validateCampaignCombo enforces the Apple Ads channel/supply/billing/bidding
// matrix. bidding may be empty (treated as MANUAL_CPT).
//
// TODO(PR4/#44): when target_cpa_amount / target_cpa_currency land on the
// schema, require a positive target CPA whenever bidding is MAX_CONVERSIONS.
func validateCampaignCombo(channel string, supply []string, billing, bidding string) diag.Diagnostics {
	var diags diag.Diagnostics

	if bidding == "" {
		bidding = client.BiddingStrategyManualCPT
	}
	if billing == "" {
		billing = client.BillingEventTaps
	}
	if channel == "" {
		channel = client.AdChannelTypeSearch
	}
	if len(supply) == 0 {
		supply = []string{client.SupplySourceSearchResults}
	}

	if billing != client.BillingEventTaps {
		diags.AddAttributeError(
			path.Root("billing_event"),
			"Invalid campaign billing_event",
			fmt.Sprintf("billing_event must be %q; got %q.", client.BillingEventTaps, billing),
		)
	}

	hasSearchResults := false
	hasDisplaySupply := false
	unknownSupply := make([]string, 0)
	for _, s := range supply {
		switch s {
		case client.SupplySourceSearchResults:
			hasSearchResults = true
		case client.SupplySourceTodayTab, client.SupplySourceSearchTab, client.SupplySourceProductPagesBrowse:
			hasDisplaySupply = true
		default:
			unknownSupply = append(unknownSupply, s)
		}
	}
	if len(unknownSupply) > 0 {
		diags.AddAttributeError(
			path.Root("supply_sources"),
			"Invalid campaign supply_sources",
			fmt.Sprintf(
				"Unsupported supply source(s): %s. Allowed values: %s, %s, %s, %s.",
				strings.Join(unknownSupply, ", "),
				client.SupplySourceSearchResults,
				client.SupplySourceTodayTab,
				client.SupplySourceSearchTab,
				client.SupplySourceProductPagesBrowse,
			),
		)
	}

	switch channel {
	case client.AdChannelTypeSearch:
		if hasDisplaySupply || !hasSearchResults {
			diags.AddAttributeError(
				path.Root("supply_sources"),
				"Invalid SEARCH campaign supply_sources",
				fmt.Sprintf(
					"SEARCH campaigns require supply_sources = [%q]. Display supplies (%s, %s, %s) require ad_channel_type = %q.",
					client.SupplySourceSearchResults,
					client.SupplySourceTodayTab,
					client.SupplySourceSearchTab,
					client.SupplySourceProductPagesBrowse,
					client.AdChannelTypeDisplay,
				),
			)
		}
		switch bidding {
		case client.BiddingStrategyManualCPT, client.BiddingStrategyMaxConversions:
			// ok
		default:
			diags.AddAttributeError(
				path.Root("bidding_strategy"),
				"Invalid SEARCH campaign bidding_strategy",
				fmt.Sprintf(
					"SEARCH campaigns support %q or %q; got %q.",
					client.BiddingStrategyManualCPT,
					client.BiddingStrategyMaxConversions,
					bidding,
				),
			)
		}
	case client.AdChannelTypeDisplay:
		if hasSearchResults || !hasDisplaySupply {
			diags.AddAttributeError(
				path.Root("supply_sources"),
				"Invalid DISPLAY campaign supply_sources",
				fmt.Sprintf(
					"DISPLAY campaigns require one or more of %s, %s, %s. %q requires ad_channel_type = %q.",
					client.SupplySourceTodayTab,
					client.SupplySourceSearchTab,
					client.SupplySourceProductPagesBrowse,
					client.SupplySourceSearchResults,
					client.AdChannelTypeSearch,
				),
			)
		}
		if bidding != client.BiddingStrategyManualCPT {
			diags.AddAttributeError(
				path.Root("bidding_strategy"),
				"Invalid DISPLAY campaign bidding_strategy",
				fmt.Sprintf(
					"DISPLAY campaigns support only %q; got %q. %q requires SEARCH + %q.",
					client.BiddingStrategyManualCPT,
					bidding,
					client.BiddingStrategyMaxConversions,
					client.SupplySourceSearchResults,
				),
			)
		}
	default:
		diags.AddAttributeError(
			path.Root("ad_channel_type"),
			"Invalid campaign ad_channel_type",
			fmt.Sprintf(
				"ad_channel_type must be %q or %q; got %q.",
				client.AdChannelTypeSearch,
				client.AdChannelTypeDisplay,
				channel,
			),
		)
	}

	// Soft check independent of channel: Max Conversions always needs Search Results.
	if bidding == client.BiddingStrategyMaxConversions && !hasSearchResults {
		diags.AddAttributeError(
			path.Root("bidding_strategy"),
			"MAX_CONVERSIONS requires Search Results supply",
			fmt.Sprintf(
				"%q requires supply_sources to include %q (Search channel).",
				client.BiddingStrategyMaxConversions,
				client.SupplySourceSearchResults,
			),
		)
	}

	// TODO(PR4/#44): reject MAX_CONVERSIONS when target_cpa_amount is unset once
	// that attribute exists on appleads_campaign.

	return diags
}

// campaignSupplySourceValues are allowed supply_sources list element values.
func campaignSupplySourceValues() []string {
	return []string{
		client.SupplySourceSearchResults,
		client.SupplySourceTodayTab,
		client.SupplySourceSearchTab,
		client.SupplySourceProductPagesBrowse,
	}
}
