// Copyright (c) 2026 Pi Cubed LLC
// SPDX-License-Identifier: MIT

package provider

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/shopspring/decimal"

	"github.com/picubedllc/terraform-provider-appleads/internal/client"
	clientv1 "github.com/picubedllc/terraform-provider-appleads/internal/client/v1"
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

func campaignCreateFromPlan(ctx context.Context, plan campaignModel) (*clientv1.CampaignCreate, diag.Diagnostics) {
	var diags diag.Diagnostics

	if plan.Name.IsNull() || plan.Name.ValueString() == "" {
		diags.AddError("Invalid campaign name", "name must be a non-empty string")
		return nil, diags
	}
	adamID := strings.TrimSpace(plan.AdamID.ValueString())
	if _, err := strconv.ParseInt(adamID, 10, 64); err != nil {
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
	targetCpa, d := moneyFromStrings(plan.TargetCpaAmount, plan.TargetCpaCurrency)
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
	if targetCpa != nil {
		if amt, err := decimal.NewFromString(targetCpa.Amount); err == nil && !amt.IsPositive() {
			diags.AddError("Invalid target_cpa_amount", "target_cpa_amount must be greater than zero")
		}
	}

	supply, d := stringList(ctx, plan.SupplySources)
	diags.Append(d...)
	orders, d := int64ListFromStrings(ctx, plan.BudgetOrders)
	diags.Append(d...)
	if diags.HasError() {
		return nil, diags
	}

	if budget != nil {
		// The Platform API has no lifetime budget field. The value stays in
		// Terraform state as create-only configuration but is not sent.
		diags.AddWarning(
			"budget_amount is not sent to the Platform API",
			"The Apple Ads Platform API no longer supports a lifetime campaign budget. "+
				"budget_amount is kept in Terraform state but is not applied to the campaign; "+
				"use daily_budget_amount or budget_orders to cap spend.",
		)
	}

	in := &clientv1.CampaignCreate{
		Name:               plan.Name.ValueString(),
		PromotedObjectType: clientv1.PromotedObjectTypeAppStoreApp,
		PromotedObjectID:   adamID,
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
	if len(supply) == 0 {
		supply = []string{client.SupplySourceSearchResults}
	}
	if !plan.BillingEvent.IsNull() && !plan.BillingEvent.IsUnknown() && plan.BillingEvent.ValueString() != "" {
		in.BillingEvent = plan.BillingEvent.ValueString()
	} else {
		in.BillingEvent = client.BillingEventTaps
	}

	bidding := client.BiddingStrategyManualCPT
	if !plan.BiddingStrategy.IsNull() && !plan.BiddingStrategy.IsUnknown() && plan.BiddingStrategy.ValueString() != "" {
		bidding = plan.BiddingStrategy.ValueString()
	}
	in.BidStrategy = bidStrategyFromPlan(bidding, targetCpa)

	in.Targeting = &clientv1.CampaignTargeting{
		CountryOrRegion: &clientv1.IncludeList{Include: countries},
		SupplyPlacement: &clientv1.IncludeList{Include: supply},
	}
	if daily != nil {
		in.DailyBudget = &clientv1.MoneyValue{Value: daily}
	}
	in.SharedBudgets = sharedBudgetRefs(orders)

	if !plan.StartTime.IsNull() && !plan.StartTime.IsUnknown() && plan.StartTime.ValueString() != "" {
		in.StartTime = plan.StartTime.ValueString()
	}
	if !plan.EndTime.IsNull() && !plan.EndTime.IsUnknown() {
		in.EndTime = plan.EndTime.ValueString()
	}
	return in, diags
}

// bidStrategyFromPlan maps bidding_strategy / target_cpa_* to bidStrategy.
// Returns nil when neither a strategy nor a target CPA is set.
func bidStrategyFromPlan(strategy string, targetCpa *client.Money) *clientv1.BidStrategy {
	if strategy == "" && targetCpa == nil {
		return nil
	}
	return &clientv1.BidStrategy{BidStrategyType: strategy, Bid: targetCpa}
}

func sharedBudgetRefs(ids []int64) []clientv1.SharedBudgetRef {
	if len(ids) == 0 {
		return nil
	}
	out := make([]clientv1.SharedBudgetRef, 0, len(ids))
	for _, id := range ids {
		out = append(out, clientv1.SharedBudgetRef{BudgetID: id})
	}
	return out
}

func includeValues(l *clientv1.IncludeList) []string {
	if l == nil {
		return nil
	}
	return l.Include
}

// campaignModelFromAPI maps a Platform API v1 campaign to the Terraform model.
func campaignModelFromAPI(ctx context.Context, c *clientv1.Campaign) (campaignModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	var m campaignModel
	m.ID = types.StringValue(strconv.FormatInt(c.ID, 10))
	m.Name = types.StringValue(c.Name)
	m.Status = types.StringValue(c.Status)
	m.BillingEvent = stringOrNull(c.BillingEvent)
	m.PaymentModel = types.StringValue(c.PaymentModel)
	m.ServingStatus = stringOrNull(c.SystemStatus)
	m.DisplayStatus = types.StringValue(c.DisplayStatus)
	m.ModificationTime = types.StringValue(c.ModificationTime)

	m.AdamID = types.StringValue(c.PromotedObjectID)
	switch c.PromotedObjectType {
	case "", clientv1.PromotedObjectTypeAppStoreApp:
	default:
		diags.AddError(
			"Unsupported promoted object type",
			fmt.Sprintf("Campaign %d promotes a %q object; appleads_campaign only supports %s.", c.ID, c.PromotedObjectType, clientv1.PromotedObjectTypeAppStoreApp),
		)
	}

	var countryList, supplyList []string
	if c.Targeting != nil {
		countryList = includeValues(c.Targeting.CountryOrRegion)
		supplyList = includeValues(c.Targeting.SupplyPlacement)
	}

	m.AdChannelType = stringOrNull(c.AdChannelType)
	if c.AdChannelType == "" && len(supplyList) > 0 {
		m.AdChannelType = types.StringValue(client.AdChannelTypeDisplay)
		for _, s := range supplyList {
			if s == client.SupplySourceSearchResults {
				m.AdChannelType = types.StringValue(client.AdChannelTypeSearch)
				break
			}
		}
	}

	if c.BidStrategy != nil {
		m.BiddingStrategy = stringOrNull(c.BidStrategy.BidStrategyType)
		if c.BidStrategy.Bid != nil {
			m.TargetCpaAmount = types.StringValue(c.BidStrategy.Bid.Amount)
			m.TargetCpaCurrency = types.StringValue(c.BidStrategy.Bid.Currency)
		} else {
			m.TargetCpaAmount = types.StringNull()
			m.TargetCpaCurrency = types.StringNull()
		}
	} else {
		m.BiddingStrategy = types.StringNull()
		m.TargetCpaAmount = types.StringNull()
		m.TargetCpaCurrency = types.StringNull()
	}

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

	countries, d := types.ListValueFrom(ctx, types.StringType, countryList)
	diags.Append(d...)
	m.CountriesOrRegions = countries

	supply, d := types.ListValueFrom(ctx, types.StringType, supplyList)
	diags.Append(d...)
	m.SupplySources = supply

	// Lifetime budget is not part of the Platform API.
	m.BudgetAmount = types.StringNull()
	m.BudgetCurrency = types.StringNull()

	if c.DailyBudget != nil && c.DailyBudget.Value != nil {
		m.DailyBudgetAmount = types.StringValue(c.DailyBudget.Value.Amount)
		m.DailyBudgetCurrency = types.StringValue(c.DailyBudget.Value.Currency)
	} else {
		m.DailyBudgetAmount = types.StringNull()
		m.DailyBudgetCurrency = types.StringNull()
	}

	if len(c.SharedBudgets) == 0 {
		m.BudgetOrders = types.ListNull(types.StringType)
	} else {
		orderStrs := make([]string, 0, len(c.SharedBudgets))
		for _, ref := range c.SharedBudgets {
			orderStrs = append(orderStrs, strconv.FormatInt(ref.BudgetID, 10))
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
	reported = overlayCampaignOmitted(configured, reported)
	reported = overlayCampaignMoney(configured, reported)
	reported = overlayCampaignStartTime(configured, reported)
	return overlayCampaignLists(ctx, configured, reported)
}

// overlayCampaignOmitted keeps configured values for fields the Platform API
// does not echo back: the create-only lifetime budget, and the channel,
// billing, and bidding fields when a response omits them.
func overlayCampaignOmitted(configured, reported campaignModel) campaignModel {
	reported.BudgetAmount = keepConfiguredWhenOmitted(configured.BudgetAmount, reported.BudgetAmount)
	reported.BudgetCurrency = keepConfiguredWhenOmitted(configured.BudgetCurrency, reported.BudgetCurrency)
	reported.AdChannelType = keepConfiguredWhenOmitted(configured.AdChannelType, reported.AdChannelType)
	reported.BillingEvent = keepConfiguredWhenOmitted(configured.BillingEvent, reported.BillingEvent)
	reported.BiddingStrategy = keepConfiguredWhenOmitted(configured.BiddingStrategy, reported.BiddingStrategy)
	return reported
}

func keepConfiguredWhenOmitted(configured, reported types.String) types.String {
	if reported.IsNull() && !configured.IsNull() && !configured.IsUnknown() {
		return configured
	}
	return reported
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
	reported.TargetCpaAmount = preferAmount(configured.TargetCpaAmount, reported.TargetCpaAmount)
	reported.TargetCpaCurrency = preferAmount(configured.TargetCpaCurrency, reported.TargetCpaCurrency)
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
	if configured.IsNull() || configured.IsUnknown() {
		return reported, nil
	}
	if reported.IsNull() {
		// Platform API omitted the list; keep the configured value.
		return configured, nil
	}
	if reported.IsUnknown() {
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
