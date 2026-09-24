// Copyright (c) 2026 Pi Cubed LLC
// SPDX-License-Identifier: MIT

package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/shopspring/decimal"

	"github.com/picubedllc/terraform-provider-appleads/internal/client"
)

var (
	_ resource.Resource                = &budgetOrderResource{}
	_ resource.ResourceWithConfigure   = &budgetOrderResource{}
	_ resource.ResourceWithImportState = &budgetOrderResource{}
)

func NewBudgetOrderResource() resource.Resource { return &budgetOrderResource{} }

type budgetOrderResource struct {
	client *client.Client
}

// budgetOrderModel maps appleads_budget_order.
//
// Mutable (always, per Apple): budget_amount/budget_currency, end_date,
// primary_buyer_name, primary_buyer_email, billing_email.
//
// Conditionally mutable (before start / no assigned campaigns): name,
// start_date, client_name, order_number.
//
// Computed / system-controlled: id, status, parent_org_id. supply_sources is
// optional on create (API v5.3+); responses typically include all values.
//
// Apple Ads API v5 has no budget-order delete endpoint. Destroy removes the
// resource from Terraform state only; the order remains in Apple Ads.
type budgetOrderModel struct {
	ID                types.String `tfsdk:"id"`
	Name              types.String `tfsdk:"name"`
	BudgetAmount      types.String `tfsdk:"budget_amount"`
	BudgetCurrency    types.String `tfsdk:"budget_currency"`
	StartDate         types.String `tfsdk:"start_date"`
	EndDate           types.String `tfsdk:"end_date"`
	PrimaryBuyerName  types.String `tfsdk:"primary_buyer_name"`
	PrimaryBuyerEmail types.String `tfsdk:"primary_buyer_email"`
	BillingEmail      types.String `tfsdk:"billing_email"`
	ClientName        types.String `tfsdk:"client_name"`
	OrderNumber       types.String `tfsdk:"order_number"`
	SupplySources     types.List   `tfsdk:"supply_sources"`
	Status            types.String `tfsdk:"status"`
	ParentOrgID       types.String `tfsdk:"parent_org_id"`
}

func (r *budgetOrderResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_budget_order"
}

func (r *budgetOrderResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an Apple Ads budget order (LOC / agency billing container).\n\n" +
			"Budget orders live at organization scope (provider `org_id`). Campaigns reference " +
			"managed orders via `appleads_campaign.budget_orders` using this resource's `id`.\n\n" +
			"**Mutable:** `budget_amount`/`budget_currency`, `end_date`, `primary_buyer_name`, " +
			"`primary_buyer_email`, `billing_email`. Also `name`, `start_date`, `client_name`, and " +
			"`order_number` when Apple still allows edits (typically before start and with no " +
			"assigned campaigns).\n\n" +
			"**Computed:** `id`, `status` (system-controlled), `parent_org_id`.\n\n" +
			"**Destroy:** Apple Ads API v5 has no delete endpoint for budget orders. Destroy " +
			"removes the resource from Terraform state only; the order remains in Apple Ads.\n\n" +
			"Money amounts are decimal strings (never floating point).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Budget order identifier assigned by Apple Ads.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Budget order name, unique within the organization (conditionally mutable before start).",
				Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"budget_amount": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Total budget amount as a decimal string (mutable). Never use floating point.",
				Validators: []validator.String{
					stringvalidator.RegexMatches(moneyAmountRegexp, "budget_amount must be a positive decimal string (e.g. \"300.00\")"),
				},
			},
			"budget_currency": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Currency code for budget_amount (e.g. USD).",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"start_date": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Scheduled start date/time (conditionally mutable before start and when no campaigns are assigned).",
			},
			"end_date": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Scheduled end date/time (mutable).",
			},
			"primary_buyer_name": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Primary buyer name (mutable).",
			},
			"primary_buyer_email": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Primary buyer email (mutable).",
			},
			"billing_email": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Billing email (mutable).",
			},
			"client_name": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Advertiser or product name. Required for agency-type accounts; conditionally mutable before start.",
			},
			"order_number": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Purchase order number. Required for agency-type accounts; conditionally mutable before start.",
			},
			"supply_sources": schema.ListAttribute{
				ElementType:         types.StringType,
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Optional supply sources on create (API v5.3+). Responses typically include all supply source values; the provider keeps configured order when the set is unchanged.",
			},
			"status": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "System-controlled budget order status (for example ACTIVE, COMPLETED, CANCELED).",
			},
			"parent_org_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Organization that owns the budget order.",
			},
		},
	}
}

func (r *budgetOrderResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	data, ok := req.ProviderData.(*ProviderData)
	if !ok {
		resp.Diagnostics.AddError("Unexpected configure type", fmt.Sprintf("Expected *ProviderData, got %T", req.ProviderData))
		return
	}
	r.client = data.Client
}

func (r *budgetOrderResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan budgetOrderModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	in, diags := budgetOrderCreateFromPlan(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.client.CreateBudgetOrder(ctx, in, nil)
	if err != nil {
		resp.Diagnostics.Append(apiErrorDiagnostic("Unable to create Apple Ads budget order", err)...)
		return
	}
	state, diags := budgetOrderModelFromInfo(ctx, created)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	state, diags = overlayBudgetOrderReported(ctx, plan, state)
	resp.Diagnostics.Append(diags...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *budgetOrderResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state budgetOrderModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := client.ParseBudgetOrderID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid budget order id", err.Error())
		return
	}
	got, err := r.client.GetBudgetOrder(ctx, id)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.Append(apiErrorDiagnostic("Unable to read Apple Ads budget order", err)...)
		return
	}
	newState, diags := budgetOrderModelFromInfo(ctx, got)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	newState, diags = overlayBudgetOrderReported(ctx, state, newState)
	resp.Diagnostics.Append(diags...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *budgetOrderResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan budgetOrderModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := client.ParseBudgetOrderID(plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid budget order id", err.Error())
		return
	}

	in, diags := budgetOrderUpdateFromPlan(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	updated, err := r.client.UpdateBudgetOrder(ctx, id, in, nil)
	if err != nil {
		resp.Diagnostics.Append(apiErrorDiagnostic("Unable to update Apple Ads budget order", err)...)
		return
	}
	state, diags := budgetOrderModelFromInfo(ctx, updated)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	state, diags = overlayBudgetOrderReported(ctx, plan, state)
	resp.Diagnostics.Append(diags...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *budgetOrderResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state budgetOrderModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.AddWarning(
		"Budget order left in Apple Ads",
		fmt.Sprintf(
			"Apple Ads API v5 has no delete endpoint for budget orders. Removed appleads_budget_order %s from Terraform state only; the order remains in Apple Ads.",
			state.ID.ValueString(),
		),
	)
}

func (r *budgetOrderResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Client not configured", "The provider client was not configured before importing appleads_budget_order.")
		return
	}

	id, err := client.ParseBudgetOrderID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import id", "Expected a numeric Apple Ads budget order id.")
		return
	}

	got, err := r.client.GetBudgetOrder(ctx, id)
	if err != nil {
		if client.IsNotFound(err) {
			resp.Diagnostics.AddError("Budget order not found", fmt.Sprintf("No Apple Ads budget order found with id %s.", req.ID))
			return
		}
		resp.Diagnostics.Append(apiErrorDiagnostic("Unable to import Apple Ads budget order", err)...)
		return
	}

	state, diags := budgetOrderModelFromInfo(ctx, got)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func budgetOrderCreateFromPlan(ctx context.Context, plan budgetOrderModel) (*client.BudgetOrderCreate, diag.Diagnostics) {
	var diags diag.Diagnostics

	budget, d := moneyFromStrings(plan.BudgetAmount, plan.BudgetCurrency)
	diags.Append(d...)
	if budget == nil {
		diags.AddError("Invalid budget_amount", "budget_amount is required")
	} else if amt, err := decimal.NewFromString(budget.Amount); err == nil && !amt.IsPositive() {
		diags.AddError("Invalid budget_amount", "must be greater than zero")
	}

	supply, d := stringList(ctx, plan.SupplySources)
	diags.Append(d...)
	if diags.HasError() {
		return nil, diags
	}

	in := &client.BudgetOrderCreate{
		Name:              plan.Name.ValueString(),
		StartDate:         plan.StartDate.ValueString(),
		EndDate:           plan.EndDate.ValueString(),
		Budget:            budget,
		OrderNumber:       optionalString(plan.OrderNumber),
		ClientName:        optionalString(plan.ClientName),
		PrimaryBuyerName:  optionalString(plan.PrimaryBuyerName),
		PrimaryBuyerEmail: optionalString(plan.PrimaryBuyerEmail),
		BillingEmail:      optionalString(plan.BillingEmail),
		SupplySources:     supply,
	}
	return in, diags
}

func budgetOrderUpdateFromPlan(ctx context.Context, plan budgetOrderModel) (*client.BudgetOrderUpdate, diag.Diagnostics) {
	var diags diag.Diagnostics

	budget, d := moneyFromStrings(plan.BudgetAmount, plan.BudgetCurrency)
	diags.Append(d...)
	if budget == nil {
		diags.AddError("Invalid budget_amount", "budget_amount is required")
	} else if amt, err := decimal.NewFromString(budget.Amount); err == nil && !amt.IsPositive() {
		diags.AddError("Invalid budget_amount", "must be greater than zero")
	}

	supply, d := stringList(ctx, plan.SupplySources)
	diags.Append(d...)
	if diags.HasError() {
		return nil, diags
	}

	in := &client.BudgetOrderUpdate{
		Name:              plan.Name.ValueString(),
		StartDate:         plan.StartDate.ValueString(),
		EndDate:           plan.EndDate.ValueString(),
		Budget:            budget,
		OrderNumber:       optionalString(plan.OrderNumber),
		ClientName:        optionalString(plan.ClientName),
		PrimaryBuyerName:  optionalString(plan.PrimaryBuyerName),
		PrimaryBuyerEmail: optionalString(plan.PrimaryBuyerEmail),
		BillingEmail:      optionalString(plan.BillingEmail),
		SupplySources:     supply,
	}
	return in, diags
}

func optionalString(v types.String) string {
	if v.IsNull() || v.IsUnknown() {
		return ""
	}
	return v.ValueString()
}

func budgetOrderModelFromInfo(ctx context.Context, info *client.BudgetOrderInfo) (budgetOrderModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	var m budgetOrderModel
	if info == nil || info.Bo == nil {
		diags.AddError("Invalid budget order response", "Apple Ads returned an empty budget order payload")
		return m, diags
	}
	bo := info.Bo
	m.ID = types.StringValue(strconv.FormatInt(bo.ID, 10))
	m.Name = types.StringValue(bo.Name)
	m.StartDate = types.StringValue(bo.StartDate)
	m.EndDate = types.StringValue(bo.EndDate)
	m.Status = types.StringValue(bo.Status)
	if bo.ParentOrgID != 0 {
		m.ParentOrgID = types.StringValue(strconv.FormatInt(bo.ParentOrgID, 10))
	} else {
		m.ParentOrgID = types.StringNull()
	}
	if bo.Budget != nil {
		m.BudgetAmount = types.StringValue(bo.Budget.Amount)
		m.BudgetCurrency = types.StringValue(bo.Budget.Currency)
	} else {
		m.BudgetAmount = types.StringNull()
		m.BudgetCurrency = types.StringNull()
	}
	m.PrimaryBuyerName = stringOrNull(bo.PrimaryBuyerName)
	m.PrimaryBuyerEmail = stringOrNull(bo.PrimaryBuyerEmail)
	m.BillingEmail = stringOrNull(bo.BillingEmail)
	m.ClientName = stringOrNull(bo.ClientName)
	m.OrderNumber = stringOrNull(bo.OrderNumber)

	if len(bo.SupplySources) == 0 {
		m.SupplySources = types.ListNull(types.StringType)
	} else {
		supply, d := types.ListValueFrom(ctx, types.StringType, bo.SupplySources)
		diags.Append(d...)
		m.SupplySources = supply
	}
	return m, diags
}

func stringOrNull(s string) types.String {
	if s == "" {
		return types.StringNull()
	}
	return types.StringValue(s)
}

func overlayBudgetOrderReported(ctx context.Context, configured, reported budgetOrderModel) (budgetOrderModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	reported.BudgetAmount = preferAmount(configured.BudgetAmount, reported.BudgetAmount)
	reported.BudgetCurrency = preferAmount(configured.BudgetCurrency, reported.BudgetCurrency)
	supply, d := preferConfiguredStringList(ctx, configured.SupplySources, reported.SupplySources)
	diags.Append(d...)
	reported.SupplySources = supply
	return reported, diags
}
