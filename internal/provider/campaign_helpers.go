// Copyright (c) 2026 Pi Cubed LLC
// SPDX-License-Identifier: MIT

package provider

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/shopspring/decimal"

	"github.com/picubedllc/terraform-provider-appleads/internal/client"
)

func moneyFromStrings(amount, currency types.String) (*client.Money, diag.Diagnostics) {
	var diags diag.Diagnostics
	if amount.IsNull() || amount.IsUnknown() || amount.ValueString() == "" {
		return nil, diags
	}
	if _, err := decimal.NewFromString(amount.ValueString()); err != nil {
		diags.AddError("Invalid money amount", fmt.Sprintf("%q is not a valid decimal amount: %s", amount.ValueString(), err))
		return nil, diags
	}
	cur := "USD"
	if !currency.IsNull() && !currency.IsUnknown() && currency.ValueString() != "" {
		cur = currency.ValueString()
	}
	return &client.Money{Amount: amount.ValueString(), Currency: cur}, diags
}

func stringList(ctx context.Context, list types.List) ([]string, diag.Diagnostics) {
	var diags diag.Diagnostics
	if list.IsNull() || list.IsUnknown() {
		return nil, diags
	}
	var out []string
	diags.Append(list.ElementsAs(ctx, &out, false)...)
	return out, diags
}

func int64ListFromStrings(ctx context.Context, list types.List) ([]int64, diag.Diagnostics) {
	var diags diag.Diagnostics
	strs, d := stringList(ctx, list)
	diags.Append(d...)
	if diags.HasError() {
		return nil, diags
	}
	out := make([]int64, 0, len(strs))
	for _, s := range strs {
		v, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			diags.AddError("Invalid budget order id", fmt.Sprintf("%q is not a valid integer id: %s", s, err))
			return nil, diags
		}
		out = append(out, v)
	}
	return out, diags
}

func campaignCreateFromPlan(ctx context.Context, plan campaignModel) (*client.CampaignCreate, diag.Diagnostics) {
	var diags diag.Diagnostics

	if plan.Name.IsNull() || plan.Name.ValueString() == "" {
		diags.AddError("Invalid campaign name", "name must be a non-empty string")
		return nil, diags
	}
	adamID, err := strconv.ParseInt(plan.AdamID.ValueString(), 10, 64)
	if err != nil {
		diags.AddError("Invalid adam_id", fmt.Sprintf("adam_id must be a numeric Adam ID; got %q", plan.AdamID.ValueString()))
		return nil, diags
	}
	countries, d := stringList(ctx, plan.CountriesOrRegions)
	diags.Append(d...)
	if len(countries) == 0 {
		diags.AddError("Invalid countries_or_regions", "at least one country or region is required")
	}

	budget, d := moneyFromStrings(plan.BudgetAmount, plan.BudgetCurrency)
	diags.Append(d...)
	daily, d := moneyFromStrings(plan.DailyBudgetAmount, plan.DailyBudgetCurrency)
	diags.Append(d...)
	if budget != nil {
		if amt, err := decimal.NewFromString(budget.Amount); err == nil && !amt.IsPositive() {
			diags.AddError("Invalid budget_amount", "budget_amount must be greater than zero")
		}
	}
	if daily != nil {
		if amt, err := decimal.NewFromString(daily.Amount); err == nil && !amt.IsPositive() {
			diags.AddError("Invalid daily_budget_amount", "daily_budget_amount must be greater than zero")
		}
	}

	supply, d := stringList(ctx, plan.SupplySources)
	diags.Append(d...)
	orders, d := int64ListFromStrings(ctx, plan.BudgetOrders)
	diags.Append(d...)
	if diags.HasError() {
		return nil, diags
	}

	in := &client.CampaignCreate{
		Name:               plan.Name.ValueString(),
		AdamID:             adamID,
		CountriesOrRegions: countries,
		BudgetAmount:       budget,
		DailyBudgetAmount:  daily,
		SupplySources:      supply,
		BudgetOrders:       orders,
	}
	if !plan.Status.IsNull() && !plan.Status.IsUnknown() && plan.Status.ValueString() != "" {
		in.Status = plan.Status.ValueString()
	}

	// Defaults apply only when omitted. When the plan sets channel or supply,
	// do not overwrite with Search Results values.
	if !plan.AdChannelType.IsNull() && !plan.AdChannelType.IsUnknown() && plan.AdChannelType.ValueString() != "" {
		in.AdChannelType = plan.AdChannelType.ValueString()
	} else {
		in.AdChannelType = client.AdChannelTypeSearch
	}
	if len(in.SupplySources) == 0 {
		in.SupplySources = []string{client.SupplySourceSearchResults}
	}
	if !plan.BillingEvent.IsNull() && !plan.BillingEvent.IsUnknown() && plan.BillingEvent.ValueString() != "" {
		in.BillingEvent = plan.BillingEvent.ValueString()
	} else {
		in.BillingEvent = client.BillingEventTaps
	}
	if !plan.BiddingStrategy.IsNull() && !plan.BiddingStrategy.IsUnknown() && plan.BiddingStrategy.ValueString() != "" {
		in.BiddingStrategy = plan.BiddingStrategy.ValueString()
	} else {
		in.BiddingStrategy = client.BiddingStrategyManualCPT
	}

	if !plan.StartTime.IsNull() && !plan.StartTime.IsUnknown() && plan.StartTime.ValueString() != "" {
		in.StartTime = plan.StartTime.ValueString()
	}
	if !plan.EndTime.IsNull() && !plan.EndTime.IsUnknown() {
		in.EndTime = plan.EndTime.ValueString()
	}
	return in, diags
}

func campaignModelFromClient(ctx context.Context, c *client.Campaign) (campaignModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	var m campaignModel
	m.ID = types.StringValue(strconv.FormatInt(c.ID, 10))
	m.Name = types.StringValue(c.Name)
	m.AdamID = types.StringValue(strconv.FormatInt(c.AdamID, 10))
	m.Status = types.StringValue(c.Status)
	m.AdChannelType = types.StringValue(c.AdChannelType)
	m.BillingEvent = types.StringValue(c.BillingEvent)
	m.BiddingStrategy = types.StringValue(c.BiddingStrategy)
	m.PaymentModel = types.StringValue(c.PaymentModel)
	m.ServingStatus = types.StringValue(c.ServingStatus)
	m.DisplayStatus = types.StringValue(c.DisplayStatus)
	m.ModificationTime = types.StringValue(c.ModificationTime)
	if c.StartTime != "" {
		m.StartTime = types.StringValue(c.StartTime)
	} else {
		m.StartTime = types.StringNull()
	}
	if c.EndTime != "" {
		m.EndTime = types.StringValue(c.EndTime)
	} else {
		m.EndTime = types.StringNull()
	}

	countries, d := types.ListValueFrom(ctx, types.StringType, c.CountriesOrRegions)
	diags.Append(d...)
	m.CountriesOrRegions = countries

	supply, d := types.ListValueFrom(ctx, types.StringType, c.SupplySources)
	diags.Append(d...)
	m.SupplySources = supply

	if c.BudgetAmount != nil {
		m.BudgetAmount = types.StringValue(c.BudgetAmount.Amount)
		m.BudgetCurrency = types.StringValue(c.BudgetAmount.Currency)
	} else {
		m.BudgetAmount = types.StringNull()
		m.BudgetCurrency = types.StringNull()
	}
	if c.DailyBudgetAmount != nil {
		m.DailyBudgetAmount = types.StringValue(c.DailyBudgetAmount.Amount)
		m.DailyBudgetCurrency = types.StringValue(c.DailyBudgetAmount.Currency)
	} else {
		m.DailyBudgetAmount = types.StringNull()
		m.DailyBudgetCurrency = types.StringNull()
	}

	if len(c.BudgetOrders) == 0 {
		m.BudgetOrders = types.ListNull(types.StringType)
	} else {
		orderStrs := make([]string, 0, len(c.BudgetOrders))
		for _, id := range c.BudgetOrders {
			orderStrs = append(orderStrs, strconv.FormatInt(id, 10))
		}
		orders, d := types.ListValueFrom(ctx, types.StringType, orderStrs)
		diags.Append(d...)
		m.BudgetOrders = orders
	}

	return m, diags
}

// overlayCampaignReported keeps configured money scale, list order, and
// start_time when Apple only reformats those values, so Create/Read/Update
// do not fail Terraform's after-apply consistency check.
func overlayCampaignReported(ctx context.Context, configured, reported campaignModel) (campaignModel, diag.Diagnostics) {
	reported = overlayCampaignMoney(configured, reported)
	reported = overlayCampaignStartTime(configured, reported)
	return overlayCampaignLists(ctx, configured, reported)
}

// overlayCampaignStartTime keeps the configured start_time string when the
// plan set one, so millisecond-precision create values survive Apple's
// response formatting.
func overlayCampaignStartTime(configured, reported campaignModel) campaignModel {
	if configured.StartTime.IsNull() || configured.StartTime.IsUnknown() || configured.StartTime.ValueString() == "" {
		return reported
	}
	if reported.StartTime.IsNull() || reported.StartTime.IsUnknown() {
		return reported
	}
	reported.StartTime = configured.StartTime
	return reported
}

// overlayCampaignMoney keeps configured decimal strings when Apple normalizes
// them (e.g. "5.00" → "5") so Terraform does not report a perpetual diff.
func overlayCampaignMoney(configured, reported campaignModel) campaignModel {
	reported.BudgetAmount = preferAmount(configured.BudgetAmount, reported.BudgetAmount)
	reported.BudgetCurrency = preferAmount(configured.BudgetCurrency, reported.BudgetCurrency)
	reported.DailyBudgetAmount = preferAmount(configured.DailyBudgetAmount, reported.DailyBudgetAmount)
	reported.DailyBudgetCurrency = preferAmount(configured.DailyBudgetCurrency, reported.DailyBudgetCurrency)
	return reported
}

// overlayAdGroupMoney keeps configured decimal strings when Apple normalizes
// them (e.g. "1.00" → "1") so Create does not fail Terraform's after-apply
// consistency check and taint the ad group.
func overlayAdGroupMoney(configured, reported adGroupModel) adGroupModel {
	reported.DefaultBidAmount = preferAmount(configured.DefaultBidAmount, reported.DefaultBidAmount)
	reported.DefaultBidCurrency = preferAmount(configured.DefaultBidCurrency, reported.DefaultBidCurrency)
	reported.CPAGoalAmount = preferAmount(configured.CPAGoalAmount, reported.CPAGoalAmount)
	reported.CPAGoalCurrency = preferAmount(configured.CPAGoalCurrency, reported.CPAGoalCurrency)
	return reported
}

// overlayAdGroupReported keeps configured money scale and targeting dimensions
// when Apple only reformats those values, so Create/Read/Update do not fail
// Terraform's after-apply consistency check.
func overlayAdGroupReported(ctx context.Context, configured, reported adGroupModel) (adGroupModel, diag.Diagnostics) {
	reported = overlayAdGroupMoney(configured, reported)
	return overlayAdGroupTargeting(ctx, configured, reported)
}

// overlayKeywordMoney keeps a configured bid's decimal scale. When bid_amount
// is omitted, state stays null even if Apple returns the ad group default.
func overlayKeywordMoney(configured, reported keywordModel) keywordModel {
	if configured.BidAmount.IsNull() || (!configured.BidAmount.IsUnknown() && configured.BidAmount.ValueString() == "") {
		reported.BidAmount = types.StringNull()
	} else {
		reported.BidAmount = preferAmount(configured.BidAmount, reported.BidAmount)
	}
	reported.BidCurrency = preferAmount(configured.BidCurrency, reported.BidCurrency)
	return reported
}

// overlayCampaignLists keeps configured countries_or_regions and
// supply_sources order when Apple returns the same set in a different order.
func overlayCampaignLists(ctx context.Context, configured, reported campaignModel) (campaignModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	countries, d := preferConfiguredStringList(ctx, configured.CountriesOrRegions, reported.CountriesOrRegions)
	diags.Append(d...)
	reported.CountriesOrRegions = countries
	supply, d := preferConfiguredStringList(ctx, configured.SupplySources, reported.SupplySources)
	diags.Append(d...)
	reported.SupplySources = supply
	return reported, diags
}

// preferConfiguredStringList returns the configured list when it is the same
// set as the API response. Membership changes keep the reported values.
func preferConfiguredStringList(ctx context.Context, configured, reported types.List) (types.List, diag.Diagnostics) {
	if configured.IsNull() || configured.IsUnknown() || reported.IsNull() || reported.IsUnknown() {
		return reported, nil
	}
	cfg, d := stringList(ctx, configured)
	if d.HasError() {
		return reported, d
	}
	rep, d2 := stringList(ctx, reported)
	d.Append(d2...)
	if d.HasError() {
		return reported, d
	}
	if stringSetEqual(cfg, rep) {
		return configured, d
	}
	return reported, d
}

func stringSetEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	counts := make(map[string]int, len(a))
	for _, s := range a {
		counts[s]++
	}
	for _, s := range b {
		counts[s]--
		if counts[s] < 0 {
			return false
		}
	}
	for _, n := range counts {
		if n != 0 {
			return false
		}
	}
	return true
}

func preferAmount(configured, reported types.String) types.String {
	if configured.IsNull() || configured.IsUnknown() || reported.IsNull() || reported.IsUnknown() {
		return reported
	}
	cfg := configured.ValueString()
	rep := reported.ValueString()
	if cfg == "" || cfg == rep {
		return reported
	}
	a, err1 := decimal.NewFromString(cfg)
	b, err2 := decimal.NewFromString(rep)
	if err1 != nil || err2 != nil {
		return reported
	}
	if a.Equal(b) {
		return configured
	}
	return reported
}

func apiErrorDiagnostic(summary string, err error) diag.Diagnostics {
	var diags diag.Diagnostics
	var orgErr *client.OrgIDError
	if errors.As(err, &orgErr) {
		diags.AddError("Invalid Apple Ads org_id", orgErr.Error())
		return diags
	}
	var apiErr *client.APIError
	if errors.As(err, &apiErr) {
		detail := apiErr.Message
		if apiErr.StatusCode != 0 {
			detail = fmt.Sprintf("HTTP %d: %s", apiErr.StatusCode, detail)
		}
		if apiErr.Code != "" {
			detail = apiErr.Code + ": " + detail
		}
		if apiErr.RequestID != "" {
			detail += " (request_id=" + apiErr.RequestID + ")"
		}
		diags.AddError(summary, detail)
		return diags
	}
	diags.AddError(summary, err.Error())
	return diags
}
