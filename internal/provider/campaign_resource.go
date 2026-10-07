// Copyright (c) 2026 Pi Cubed LLC
// SPDX-License-Identifier: MIT

package provider

import (
	"context"
	"fmt"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/picubedllc/terraform-provider-appleads/internal/client"
)

// moneyAmountRegexp validates decimal money strings without float conversion.
var moneyAmountRegexp = regexp.MustCompile(`^(?:0|[1-9]\d*)(?:\.\d+)?$`)

var (
	_ resource.Resource                   = &campaignResource{}
	_ resource.ResourceWithImportState    = &campaignResource{}
	_ resource.ResourceWithValidateConfig = &campaignResource{}
)

func NewCampaignResource() resource.Resource {
	return &campaignResource{}
}

type campaignResource struct {
	client                *client.Client
	allowCampaignDeletion bool
}

// campaignModel is the Terraform state/plan model for appleads_campaign.
//
// Field mutability (must stay in sync with client.Campaign comments):
//
// Immutable / create-only — never use RequiresReplace; Update returns a diagnostic instead:
//   - adam_id
//   - countries_or_regions
//   - supply_sources
//   - ad_channel_type
//   - billing_event
//   - budget_amount, budget_currency (lifetime total; create-only)
//
// Mutable — in-place Update:
//   - name, status, daily_budget_amount, daily_budget_currency, budget_orders, end_time,
//     bidding_strategy, target_cpa_amount, target_cpa_currency
//
// Create optional / computed:
//   - start_time (passed on create; Apple may assign when omitted)
//
// Computed:
//   - id, payment_model, serving_status, display_status, modification_time
type campaignModel struct {
	ID                  types.String `tfsdk:"id"`
	Name                types.String `tfsdk:"name"`
	AdamID              types.String `tfsdk:"adam_id"`
	Status              types.String `tfsdk:"status"`
	CountriesOrRegions  types.List   `tfsdk:"countries_or_regions"`
	SupplySources       types.List   `tfsdk:"supply_sources"`
	AdChannelType       types.String `tfsdk:"ad_channel_type"`
	BillingEvent        types.String `tfsdk:"billing_event"`
	BiddingStrategy     types.String `tfsdk:"bidding_strategy"`
	TargetCpaAmount     types.String `tfsdk:"target_cpa_amount"`
	TargetCpaCurrency   types.String `tfsdk:"target_cpa_currency"`
	BudgetAmount        types.String `tfsdk:"budget_amount"`
	BudgetCurrency      types.String `tfsdk:"budget_currency"`
	DailyBudgetAmount   types.String `tfsdk:"daily_budget_amount"`
	DailyBudgetCurrency types.String `tfsdk:"daily_budget_currency"`
	BudgetOrders        types.List   `tfsdk:"budget_orders"`
	StartTime           types.String `tfsdk:"start_time"`
	EndTime             types.String `tfsdk:"end_time"`
	PaymentModel        types.String `tfsdk:"payment_model"`
	ServingStatus       types.String `tfsdk:"serving_status"`
	DisplayStatus       types.String `tfsdk:"display_status"`
	ModificationTime    types.String `tfsdk:"modification_time"`
}

func (r *campaignResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_campaign"
}

func (r *campaignResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an Apple Ads campaign. Immutable fields never trigger automatic replacement; " +
			"changing them returns an error diagnostic so historical campaign identity is preserved." +
			campaignComboMarkdown,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Campaign identifier assigned by Apple Ads.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},

			// Mutable
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Campaign name (mutable).",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"status": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "User-set campaign status: `ENABLED` or `PAUSED` (mutable).",
				Validators: []validator.String{
					stringvalidator.OneOf("ENABLED", "PAUSED"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"daily_budget_amount": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Daily budget amount as a decimal string (mutable).",
				Validators: []validator.String{
					stringvalidator.RegexMatches(moneyAmountRegexp, "daily_budget_amount must be a positive decimal string (e.g. \"50.00\")"),
				},
			},
			"daily_budget_currency": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Currency code for daily_budget_amount (e.g. USD).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"budget_orders": schema.ListAttribute{
				Optional:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Budget order identifiers associated with the campaign (mutable).",
			},
			"end_time": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Campaign end time in ISO-8601 format (mutable).",
			},

			// Immutable / create-only. Intentionally no RequiresReplace plan modifiers.
			"budget_amount": schema.StringAttribute{
				Optional: true,
				MarkdownDescription: "Lifetime campaign budget amount as a decimal string (create-only). " +
					"Changing this after create returns an error; create a new appleads_campaign instead. Never use floating point.",
				Validators: []validator.String{
					stringvalidator.RegexMatches(moneyAmountRegexp, "budget_amount must be a positive decimal string (e.g. \"100.00\")"),
				},
			},
			"budget_currency": schema.StringAttribute{
				Optional: true,
				Computed: true,
				MarkdownDescription: "Currency code for budget_amount (e.g. USD; create-only with budget_amount). " +
					"Changing this after create returns an error; create a new appleads_campaign instead.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"adam_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Adam ID of the promoted app (immutable). Changing this after create returns an error; create a new appleads_campaign instead.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"countries_or_regions": schema.ListAttribute{
				Required:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Country or region codes targeted by the campaign (immutable). Order is not significant; Apple may return a different order and the provider keeps the configured order when the set is unchanged. Changing membership after create returns an error; create a new appleads_campaign instead.",
				Validators: []validator.List{
					listvalidator.SizeAtLeast(1),
				},
			},
			"supply_sources": schema.ListAttribute{
				Optional:            true,
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Supply sources (immutable). `SEARCH` requires `APPSTORE_SEARCH_RESULTS`. `DISPLAY` requires one of `APPSTORE_TODAY_TAB`, `APPSTORE_SEARCH_TAB`, or `APPSTORE_PRODUCT_PAGES_BROWSE`. Order is not significant; Apple may return a different order and the provider keeps the configured order when the set is unchanged.",
				Validators: []validator.List{
					listvalidator.ValueStringsAre(stringvalidator.OneOf(campaignSupplySourceValues()...)),
				},
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
			},
			"ad_channel_type": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Ad channel type: `SEARCH` or `DISPLAY` (immutable). Defaults to `SEARCH` when omitted. Must match supply_sources and bidding_strategy (see resource docs matrix).",
				Validators: []validator.String{
					stringvalidator.OneOf(client.AdChannelTypeSearch, client.AdChannelTypeDisplay),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"billing_event": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Billing event (immutable). Only `TAPS` is supported; defaults to `TAPS` when omitted.",
				Validators: []validator.String{
					stringvalidator.OneOf(client.BillingEventTaps),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"bidding_strategy": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Bidding strategy (mutable): `MANUAL_CPT` or `MAX_CONVERSIONS`. Defaults to `MANUAL_CPT`. `MAX_CONVERSIONS` requires Search Results supply and `target_cpa_amount`.",
				Validators: []validator.String{
					stringvalidator.OneOf(client.BiddingStrategyManualCPT, client.BiddingStrategyMaxConversions),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"target_cpa_amount": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Target CPA amount as a decimal string (mutable). Required when `bidding_strategy` is `MAX_CONVERSIONS`. Never use floating point.",
				Validators: []validator.String{
					stringvalidator.RegexMatches(moneyAmountRegexp, "target_cpa_amount must be a positive decimal string (e.g. \"10.00\")"),
				},
			},
			"target_cpa_currency": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Currency code for target_cpa_amount (e.g. USD).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"start_time": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Campaign start time in ISO-8601 format. Use millisecond precision on create, e.g. `2026-01-01T00:00:00.000`. When omitted, Apple assigns the start time.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},

			// Computed read-only
			"payment_model": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Payment model inherited from the organization (e.g. `PAYG`). Read-only; not writable.",
			},
			"serving_status": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Effective serving status reported by Apple Ads.",
			},
			"display_status": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Display status reported by Apple Ads.",
			},
			"modification_time": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Last modification timestamp from Apple Ads.",
			},
		},
	}
}

func (r *campaignResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	data, ok := req.ProviderData.(*ProviderData)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected resource configure type",
			fmt.Sprintf("Expected *ProviderData, got: %T", req.ProviderData),
		)
		return
	}
	r.client = data.Client
	r.allowCampaignDeletion = data.AllowCampaignDeletion
}

func (r *campaignResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan campaignModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if r.client == nil {
		resp.Diagnostics.AddError("Client not configured", "The provider client was not configured before creating appleads_campaign.")
		return
	}

	in, diags := campaignCreateFromPlan(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.client.CreateCampaign(ctx, in)
	if err != nil {
		resp.Diagnostics.Append(apiErrorDiagnostic("Unable to create Apple Ads campaign", err)...)
		return
	}

	// Populate state from the Apple response, keeping configured money strings
	// and list order when Apple only reformats those values ("5.00" → "5",
	// or countries returned in a different order).
	state, diags := campaignModelFromClient(ctx, created)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	state, diags = overlayCampaignReported(ctx, plan, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *campaignResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state campaignModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if r.client == nil {
		resp.Diagnostics.AddError("Client not configured", "The provider client was not configured before reading appleads_campaign.")
		return
	}

	id, err := client.ParseCampaignID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid campaign id in state", err.Error())
		return
	}

	// Soft-delete handling (Apple Ads Campaign Management API v5):
	// DELETE /campaigns/{id} is a soft delete — GET /campaigns/{id} continues to
	// return the campaign object with deleted=true rather than HTTP 404.
	// Default list/find endpoints omit deleted campaigns, so a missing list hit
	// must NOT be treated as deletion. We always GET by id and trust the
	// deleted boolean (or a true 404) before removing from Terraform state.
	// Verified against Apple Ads Campaign API "Delete a Campaign" / "Get a Campaign"
	// behavior and covered by TestCampaignResource_ReadSoftDeletedRemovesState.
	got, err := r.client.GetCampaign(ctx, id)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.Append(apiErrorDiagnostic("Unable to read Apple Ads campaign", err)...)
		return
	}
	if got.Deleted {
		resp.State.RemoveResource(ctx)
		return
	}

	newState, diags := campaignModelFromClient(ctx, got)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	newState, diags = overlayCampaignReported(ctx, state, newState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *campaignResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var state, plan campaignModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if r.client == nil {
		resp.Diagnostics.AddError("Client not configured", "The provider client was not configured before updating appleads_campaign.")
		return
	}

	// Prefer failing the entire apply when any immutable field changes — even if
	// mutable fields are also present — so users never observe a partially-applied
	// update mixed with a blocked identity change.
	if changes := detectImmutableCampaignChanges(ctx, state, plan); len(changes) > 0 {
		resp.Diagnostics.Append(immutableCampaignChangeDiagnostics(state.ID.ValueString(), changes)...)
		return
	}

	id, err := client.ParseCampaignID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid campaign id in state", err.Error())
		return
	}

	upd, diags := campaignUpdateFromPlan(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	updated, err := r.client.UpdateCampaign(ctx, id, upd)
	if err != nil {
		resp.Diagnostics.Append(apiErrorDiagnostic("Unable to update Apple Ads campaign", err)...)
		return
	}

	newState, diags := campaignModelFromClient(ctx, updated)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	newState, diags = overlayCampaignReported(ctx, plan, newState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *campaignResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state campaignModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Archive guardrail: refuse before any Apple Ads call when not opted in.
	if !r.allowCampaignDeletion {
		resp.Diagnostics.Append(CampaignDeletionBlockedDiagnostics()...)
		return
	}
	if r.client == nil {
		resp.Diagnostics.AddError("Client not configured", "The provider client was not configured before deleting appleads_campaign.")
		return
	}

	id, err := client.ParseCampaignID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid campaign id in state", err.Error())
		return
	}

	// Apple Ads DELETE archives (soft-deletes) the campaign. Only remove from
	// Terraform state after Apple confirms success.
	if err := r.client.DeleteCampaign(ctx, id); err != nil {
		resp.Diagnostics.Append(apiErrorDiagnostic("Unable to archive Apple Ads campaign", err)...)
		return
	}
}

func (r *campaignResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Client not configured", "The provider client was not configured before importing appleads_campaign.")
		return
	}

	id, err := client.ParseCampaignID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import id", "Expected a numeric Apple Ads campaign id.")
		return
	}

	got, err := r.client.GetCampaign(ctx, id)
	if err != nil {
		if client.IsNotFound(err) {
			resp.Diagnostics.AddError("Campaign not found", fmt.Sprintf("No Apple Ads campaign found with id %s.", req.ID))
			return
		}
		resp.Diagnostics.Append(apiErrorDiagnostic("Unable to import Apple Ads campaign", err)...)
		return
	}
	if got.Deleted {
		resp.Diagnostics.AddError(
			"Cannot import archived campaign",
			fmt.Sprintf("Campaign %s is archived (deleted=true) in Apple Ads and cannot be imported as a managed resource.", req.ID),
		)
		return
	}

	state, diags := campaignModelFromClient(ctx, got)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
