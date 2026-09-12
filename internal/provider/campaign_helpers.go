// Copyright (c) 2026 Pi Cubed LLC
// SPDX-License-Identifier: MIT

package provider

import (
	"context"
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
	if !plan.AdChannelType.IsNull() && !plan.AdChannelType.IsUnknown() && plan.AdChannelType.ValueString() != "" {
		in.AdChannelType = plan.AdChannelType.ValueString()
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
	m.ServingStatus = types.StringValue(c.ServingStatus)
	m.DisplayStatus = types.StringValue(c.DisplayStatus)
	m.ModificationTime = types.StringValue(c.ModificationTime)
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

	orderStrs := make([]string, 0, len(c.BudgetOrders))
	for _, id := range c.BudgetOrders {
		orderStrs = append(orderStrs, strconv.FormatInt(id, 10))
	}
	orders, d := types.ListValueFrom(ctx, types.StringType, orderStrs)
	diags.Append(d...)
	m.BudgetOrders = orders

	return m, diags
}

func apiErrorDiagnostic(summary string, err error) diag.Diagnostics {
	var diags diag.Diagnostics
	if apiErr, ok := err.(*client.APIError); ok {
		detail := apiErr.Message
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
