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

var _ resource.Resource = &campaignResource{}

func NewCampaignResource() resource.Resource {
	return &campaignResource{}
}

type campaignResource struct {
	client                *client.Client
	allowCampaignDeletion bool
}

// campaignModel is the Terraform state/plan model for apple-ads_campaign.
//
// Field mutability (must stay in sync with client.Campaign comments):
//
// Immutable — never use RequiresReplace; Update returns a diagnostic instead:
//   - adam_id
//   - countries_or_regions
//   - supply_sources
//   - ad_channel_type
//
// Mutable — in-place Update:
//   - name, status, budget_amount, daily_budget_amount, budget_orders, end_time
//
// Computed:
//   - id, serving_status, display_status, modification_time
type campaignModel struct {
	ID                  types.String `tfsdk:"id"`
	Name                types.String `tfsdk:"name"`
	AdamID              types.String `tfsdk:"adam_id"`
	Status              types.String `tfsdk:"status"`
	CountriesOrRegions  types.List   `tfsdk:"countries_or_regions"`
	SupplySources       types.List   `tfsdk:"supply_sources"`
	AdChannelType       types.String `tfsdk:"ad_channel_type"`
	BudgetAmount        types.String `tfsdk:"budget_amount"`
	BudgetCurrency      types.String `tfsdk:"budget_currency"`
	DailyBudgetAmount   types.String `tfsdk:"daily_budget_amount"`
	DailyBudgetCurrency types.String `tfsdk:"daily_budget_currency"`
	BudgetOrders        types.List   `tfsdk:"budget_orders"`
	EndTime             types.String `tfsdk:"end_time"`
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
			"changing them returns an error diagnostic so historical campaign identity is preserved.",
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
			"budget_amount": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Lifetime campaign budget amount as a decimal string (mutable). Never use floating point.",
				Validators: []validator.String{
					stringvalidator.RegexMatches(moneyAmountRegexp, "budget_amount must be a positive decimal string (e.g. \"100.00\")"),
				},
			},
			"budget_currency": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Currency code for budget_amount (e.g. USD).",
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

			// Immutable — intentionally NO RequiresReplace plan modifiers (PI-12).
			"adam_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Adam ID of the promoted app (immutable). Changing this after create returns an error; create a new apple-ads_campaign instead.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"countries_or_regions": schema.ListAttribute{
				Required:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Country or region codes targeted by the campaign (immutable). Changing this after create returns an error; create a new apple-ads_campaign instead.",
				Validators: []validator.List{
					listvalidator.SizeAtLeast(1),
				},
			},
			"supply_sources": schema.ListAttribute{
				Optional:            true,
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Supply sources such as `APPSTORE_SEARCH_RESULTS` (immutable).",
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
			},
			"ad_channel_type": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Ad channel type such as `SEARCH` (immutable).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},

			// Computed read-only
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
		resp.Diagnostics.AddError("Client not configured", "The provider client was not configured before creating apple-ads_campaign.")
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

	// Populate state entirely from the Apple response (not the plan) so computed
	// fields and server-side normalization are captured correctly.
	state, diags := campaignModelFromClient(ctx, created)
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
		resp.Diagnostics.AddError("Client not configured", "The provider client was not configured before reading apple-ads_campaign.")
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
		resp.Diagnostics.AddError("Client not configured", "The provider client was not configured before updating apple_ads_campaign.")
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
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *campaignResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	resp.Diagnostics.AddError(
		"Campaign Delete not implemented",
		"apple-ads_campaign Delete is implemented in PI-13.",
	)
}
