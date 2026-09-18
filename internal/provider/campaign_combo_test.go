// Copyright (c) 2026 Pi Cubed LLC
// SPDX-License-Identifier: MIT

package provider

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/picubedllc/terraform-provider-appleads/internal/client"
)

func TestValidateCampaignCombo_ValidSearchManual(t *testing.T) {
	t.Parallel()

	diags := validateCampaignCombo(
		client.AdChannelTypeSearch,
		[]string{client.SupplySourceSearchResults},
		client.BillingEventTaps,
		client.BiddingStrategyManualCPT,
	)
	if diags.HasError() {
		t.Fatalf("expected valid Search+Manual: %v", diags)
	}
}

func TestValidateCampaignCombo_ValidSearchMaxConversions(t *testing.T) {
	t.Parallel()

	// target_cpa is not on schema yet; combo validation still accepts Max Conv + Search.
	diags := validateCampaignCombo(
		client.AdChannelTypeSearch,
		[]string{client.SupplySourceSearchResults},
		client.BillingEventTaps,
		client.BiddingStrategyMaxConversions,
	)
	if diags.HasError() {
		t.Fatalf("expected valid Search+MaxConversions: %v", diags)
	}
}

func TestValidateCampaignCombo_ValidDisplayCombos(t *testing.T) {
	t.Parallel()

	for _, supply := range []string{
		client.SupplySourceTodayTab,
		client.SupplySourceSearchTab,
		client.SupplySourceProductPagesBrowse,
	} {
		supply := supply
		t.Run(supply, func(t *testing.T) {
			t.Parallel()
			diags := validateCampaignCombo(
				client.AdChannelTypeDisplay,
				[]string{supply},
				client.BillingEventTaps,
				client.BiddingStrategyManualCPT,
			)
			if diags.HasError() {
				t.Fatalf("expected valid Display+%s: %v", supply, diags)
			}
		})
	}
}

func TestValidateCampaignCombo_InvalidCrossCombos(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		channel string
		supply  []string
		billing string
		bidding string
		wantSub string
	}{
		{
			name:    "search-with-display-supply",
			channel: client.AdChannelTypeSearch,
			supply:  []string{client.SupplySourceTodayTab},
			billing: client.BillingEventTaps,
			bidding: client.BiddingStrategyManualCPT,
			wantSub: "SEARCH campaigns require",
		},
		{
			name:    "display-with-search-supply",
			channel: client.AdChannelTypeDisplay,
			supply:  []string{client.SupplySourceSearchResults},
			billing: client.BillingEventTaps,
			bidding: client.BiddingStrategyManualCPT,
			wantSub: "DISPLAY campaigns require",
		},
		{
			name:    "display-with-max-conversions",
			channel: client.AdChannelTypeDisplay,
			supply:  []string{client.SupplySourceSearchTab},
			billing: client.BillingEventTaps,
			bidding: client.BiddingStrategyMaxConversions,
			wantSub: "DISPLAY campaigns support only",
		},
		{
			name:    "max-conversions-without-search-results",
			channel: client.AdChannelTypeSearch,
			supply:  []string{client.SupplySourceProductPagesBrowse},
			billing: client.BillingEventTaps,
			bidding: client.BiddingStrategyMaxConversions,
			wantSub: "MAX_CONVERSIONS requires Search Results",
		},
		{
			name:    "mixed-search-and-display-supply-on-search",
			channel: client.AdChannelTypeSearch,
			supply:  []string{client.SupplySourceSearchResults, client.SupplySourceTodayTab},
			billing: client.BillingEventTaps,
			bidding: client.BiddingStrategyManualCPT,
			wantSub: "SEARCH campaigns require",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			diags := validateCampaignCombo(tc.channel, tc.supply, tc.billing, tc.bidding)
			if !diags.HasError() {
				t.Fatal("expected error diagnostics")
			}
			found := false
			for _, d := range diags {
				if strings.Contains(d.Summary(), tc.wantSub) || strings.Contains(d.Detail(), tc.wantSub) {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("expected diagnostic containing %q, got %v", tc.wantSub, diags)
			}
		})
	}
}

func TestValidateCampaignCombo_DefaultsTreatOmittedAsSearchManual(t *testing.T) {
	t.Parallel()

	diags := validateCampaignCombo("", nil, "", "")
	if diags.HasError() {
		t.Fatalf("empty inputs should default to valid Search+Manual: %v", diags)
	}
}

func TestCampaignResource_ValidateConfig_ValidAndInvalid(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	r := NewCampaignResource().(*campaignResource)

	countries, diags := types.ListValueFrom(ctx, types.StringType, []string{"US"})
	if diags.HasError() {
		t.Fatal(diags)
	}
	searchSupply, diags := types.ListValueFrom(ctx, types.StringType, []string{client.SupplySourceSearchResults})
	if diags.HasError() {
		t.Fatal(diags)
	}
	displaySupply, diags := types.ListValueFrom(ctx, types.StringType, []string{client.SupplySourceTodayTab})
	if diags.HasError() {
		t.Fatal(diags)
	}

	cases := []struct {
		name    string
		model   campaignModel
		wantErr bool
	}{
		{
			name: "valid-search-manual",
			model: campaignModel{
				Name:               types.StringValue("ok"),
				AdamID:             types.StringValue("1"),
				CountriesOrRegions: countries,
				AdChannelType:      types.StringValue(client.AdChannelTypeSearch),
				SupplySources:      searchSupply,
				BillingEvent:       types.StringValue(client.BillingEventTaps),
				BiddingStrategy:    types.StringValue(client.BiddingStrategyManualCPT),
			},
		},
		{
			name: "valid-search-max-conversions",
			model: campaignModel{
				Name:               types.StringValue("max"),
				AdamID:             types.StringValue("1"),
				CountriesOrRegions: countries,
				AdChannelType:      types.StringValue(client.AdChannelTypeSearch),
				SupplySources:      searchSupply,
				BillingEvent:       types.StringValue(client.BillingEventTaps),
				BiddingStrategy:    types.StringValue(client.BiddingStrategyMaxConversions),
			},
		},
		{
			name: "valid-display",
			model: campaignModel{
				Name:               types.StringValue("display"),
				AdamID:             types.StringValue("1"),
				CountriesOrRegions: countries,
				AdChannelType:      types.StringValue(client.AdChannelTypeDisplay),
				SupplySources:      displaySupply,
				BillingEvent:       types.StringValue(client.BillingEventTaps),
				BiddingStrategy:    types.StringValue(client.BiddingStrategyManualCPT),
			},
		},
		{
			name: "invalid-display-max-conversions",
			model: campaignModel{
				Name:               types.StringValue("bad"),
				AdamID:             types.StringValue("1"),
				CountriesOrRegions: countries,
				AdChannelType:      types.StringValue(client.AdChannelTypeDisplay),
				SupplySources:      displaySupply,
				BillingEvent:       types.StringValue(client.BillingEventTaps),
				BiddingStrategy:    types.StringValue(client.BiddingStrategyMaxConversions),
			},
			wantErr: true,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			cfg := campaignConfigFromModel(t, ctx, r, tc.model)
			resp := &resource.ValidateConfigResponse{}
			r.ValidateConfig(ctx, resource.ValidateConfigRequest{Config: cfg}, resp)
			if tc.wantErr && !resp.Diagnostics.HasError() {
				t.Fatal("expected error diagnostics")
			}
			if !tc.wantErr && resp.Diagnostics.HasError() {
				t.Fatalf("unexpected: %v", resp.Diagnostics)
			}
		})
	}
}

func TestCampaignResource_SchemaDocumentsComboMatrix(t *testing.T) {
	t.Parallel()

	r := NewCampaignResource()
	resp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	desc := resp.Schema.GetMarkdownDescription()
	for _, want := range []string{
		"Channel / supply / billing / bidding matrix",
		"APPSTORE_SEARCH_RESULTS",
		"MAX_CONVERSIONS",
		"creatives/ads",
	} {
		if !strings.Contains(desc, want) {
			t.Fatalf("schema description missing %q", want)
		}
	}
	bidding, ok := resp.Schema.Attributes["bidding_strategy"]
	if !ok {
		t.Fatal("missing bidding_strategy attribute")
	}
	if !bidding.IsOptional() || !bidding.IsComputed() {
		t.Fatal("bidding_strategy should be Optional/Computed")
	}
	if !strings.Contains(strings.ToLower(bidding.GetMarkdownDescription()), "mutable") {
		t.Fatal("bidding_strategy should document mutability")
	}
}

func campaignConfigFromModel(t *testing.T, ctx context.Context, r *campaignResource, m campaignModel) tfsdk.Config {
	t.Helper()

	schemaResp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("schema: %v", schemaResp.Diagnostics)
	}

	// Zero-value framework attrs are invalid; null everything unset.
	m = campaignModelWithNullDefaults(m)

	state := tfsdk.State{Schema: schemaResp.Schema}
	diags := state.Set(ctx, m)
	if diags.HasError() {
		t.Fatalf("state set: %v", diags)
	}
	return tfsdk.Config{Schema: schemaResp.Schema, Raw: state.Raw}
}

func campaignModelWithNullDefaults(m campaignModel) campaignModel {
	nullEmptyString := func(v types.String) types.String {
		if v.IsNull() || v.IsUnknown() {
			if v.IsUnknown() {
				return v
			}
			return types.StringNull()
		}
		if v.ValueString() == "" {
			return types.StringNull()
		}
		return v
	}

	m.ID = nullEmptyString(m.ID)
	m.Status = nullEmptyString(m.Status)
	m.AdChannelType = nullEmptyString(m.AdChannelType)
	m.BillingEvent = nullEmptyString(m.BillingEvent)
	m.BiddingStrategy = nullEmptyString(m.BiddingStrategy)
	m.BudgetAmount = nullEmptyString(m.BudgetAmount)
	m.BudgetCurrency = nullEmptyString(m.BudgetCurrency)
	m.DailyBudgetAmount = nullEmptyString(m.DailyBudgetAmount)
	m.DailyBudgetCurrency = nullEmptyString(m.DailyBudgetCurrency)
	m.StartTime = nullEmptyString(m.StartTime)
	m.EndTime = nullEmptyString(m.EndTime)
	m.PaymentModel = nullEmptyString(m.PaymentModel)
	m.ServingStatus = nullEmptyString(m.ServingStatus)
	m.DisplayStatus = nullEmptyString(m.DisplayStatus)
	m.ModificationTime = nullEmptyString(m.ModificationTime)
	if m.SupplySources.IsNull() && !m.SupplySources.IsUnknown() {
		m.SupplySources = types.ListNull(types.StringType)
	}
	if m.BudgetOrders.IsNull() && !m.BudgetOrders.IsUnknown() {
		m.BudgetOrders = types.ListNull(types.StringType)
	}
	if m.CountriesOrRegions.IsNull() && !m.CountriesOrRegions.IsUnknown() {
		m.CountriesOrRegions = types.ListNull(types.StringType)
	}
	return m
}
