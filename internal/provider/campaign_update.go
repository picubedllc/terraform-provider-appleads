// Copyright (c) 2026 Pi Cubed LLC
// SPDX-License-Identifier: MIT

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/picubedllc/terraform-provider-appleads/internal/client"
)

// immutableCampaignFieldChange describes a forbidden in-place change.
type immutableCampaignFieldChange struct {
	Field string
}

func detectImmutableCampaignChanges(ctx context.Context, state, plan campaignModel) []immutableCampaignFieldChange {
	var changes []immutableCampaignFieldChange

	if !stringAttrEqual(state.AdamID, plan.AdamID) {
		changes = append(changes, immutableCampaignFieldChange{Field: "adam_id"})
	}
	if !stringAttrEqual(state.AdChannelType, plan.AdChannelType) {
		changes = append(changes, immutableCampaignFieldChange{Field: "ad_channel_type"})
	}
	if !listAttrEqual(ctx, state.CountriesOrRegions, plan.CountriesOrRegions) {
		changes = append(changes, immutableCampaignFieldChange{Field: "countries_or_regions"})
	}
	if !listAttrEqual(ctx, state.SupplySources, plan.SupplySources) {
		changes = append(changes, immutableCampaignFieldChange{Field: "supply_sources"})
	}
	return changes
}

func immutableCampaignChangeDiagnostics(campaignID string, changes []immutableCampaignFieldChange) diag.Diagnostics {
	var diags diag.Diagnostics
	for _, ch := range changes {
		diags.AddError(
			fmt.Sprintf("Cannot change immutable campaign field %q", ch.Field),
			fmt.Sprintf(
				"Apple Ads does not allow changing %s on an existing campaign. "+
					"Automatically replacing this Terraform resource would permanently archive campaign %s "+
					"and create a new campaign with a new historical identity. "+
					"Create a new apple_ads_campaign resource explicitly instead.",
				ch.Field, campaignID,
			),
		)
	}
	return diags
}

func stringAttrEqual(a, b types.String) bool {
	// Treat null and empty as equivalent for optional computed immutables.
	as := ""
	bs := ""
	if !a.IsNull() && !a.IsUnknown() {
		as = a.ValueString()
	}
	if !b.IsNull() && !b.IsUnknown() {
		bs = b.ValueString()
	}
	return as == bs
}

func listAttrEqual(ctx context.Context, a, b types.List) bool {
	if a.IsUnknown() || b.IsUnknown() {
		return true
	}
	as, da := stringList(ctx, a)
	bs, db := stringList(ctx, b)
	if da.HasError() || db.HasError() {
		return false
	}
	if len(as) != len(bs) {
		return false
	}
	for i := range as {
		if as[i] != bs[i] {
			return false
		}
	}
	return true
}

func campaignUpdateFromPlan(ctx context.Context, plan campaignModel) (*client.CampaignUpdate, diag.Diagnostics) {
	var diags diag.Diagnostics
	upd := &client.CampaignUpdate{}
	if !plan.Name.IsNull() && !plan.Name.IsUnknown() {
		upd.Name = plan.Name.ValueString()
	}
	if !plan.Status.IsNull() && !plan.Status.IsUnknown() {
		upd.Status = plan.Status.ValueString()
	}
	budget, d := moneyFromStrings(plan.BudgetAmount, plan.BudgetCurrency)
	diags.Append(d...)
	upd.BudgetAmount = budget
	daily, d := moneyFromStrings(plan.DailyBudgetAmount, plan.DailyBudgetCurrency)
	diags.Append(d...)
	upd.DailyBudgetAmount = daily
	orders, d := int64ListFromStrings(ctx, plan.BudgetOrders)
	diags.Append(d...)
	upd.BudgetOrders = orders
	if !plan.EndTime.IsNull() && !plan.EndTime.IsUnknown() {
		upd.EndTime = plan.EndTime.ValueString()
	}
	return upd, diags
}
