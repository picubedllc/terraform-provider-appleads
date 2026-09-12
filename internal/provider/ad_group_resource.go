// Copyright (c) 2026 Pi Cubed LLC
// SPDX-License-Identifier: MIT

package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/shopspring/decimal"

	"github.com/picubedllc/terraform-provider-appleads/internal/client"
)

var (
	_ resource.Resource                = &adGroupResource{}
	_ resource.ResourceWithConfigure   = &adGroupResource{}
	_ resource.ResourceWithImportState = &adGroupResource{}
)

func NewAdGroupResource() resource.Resource { return &adGroupResource{} }

type adGroupResource struct {
	client *client.Client
}

// adGroupModel maps apple_ads_ad_group.
//
// Immutable: campaign_id (no RequiresReplace — Update errors instead).
// Mutable: name, status, default_bid_amount, cpa_goal_amount, automated_keywords_opt_in, end_time.
type adGroupModel struct {
	ID                     types.String `tfsdk:"id"`
	CampaignID             types.String `tfsdk:"campaign_id"`
	Name                   types.String `tfsdk:"name"`
	Status                 types.String `tfsdk:"status"`
	DefaultBidAmount       types.String `tfsdk:"default_bid_amount"`
	DefaultBidCurrency     types.String `tfsdk:"default_bid_currency"`
	CPAGoalAmount          types.String `tfsdk:"cpa_goal_amount"`
	CPAGoalCurrency        types.String `tfsdk:"cpa_goal_currency"`
	AutomatedKeywordsOptIn types.Bool   `tfsdk:"automated_keywords_opt_in"`
	EndTime                types.String `tfsdk:"end_time"`
	ServingStatus          types.String `tfsdk:"serving_status"`
	DisplayStatus          types.String `tfsdk:"display_status"`
	ModificationTime       types.String `tfsdk:"modification_time"`
}

func (r *adGroupResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ad_group"
}

func (r *adGroupResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an Apple Ads ad group under a campaign.\n\n" +
			"**Immutable:** `campaign_id` — changing it returns an error (no `RequiresReplace`) so historical ad-group identity is preserved.\n\n" +
			"**Mutable:** `name`, `status`, `default_bid_amount`/`default_bid_currency`, `cpa_goal_amount`/`cpa_goal_currency`, `automated_keywords_opt_in` (Search Match), `end_time`.\n\n" +
			"**Computed:** `id`, `serving_status`, `display_status`, `modification_time`.\n\n" +
			"Audience `targetingDimensions` from Apple's API are not yet exposed in this resource schema.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Ad group identifier.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"campaign_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Parent campaign id (immutable). Changing this after create returns an error; create a new apple_ads_ad_group instead.",
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Ad group name (mutable).",
				Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"status": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "ENABLED or PAUSED (mutable).",
				Validators:          []validator.String{stringvalidator.OneOf("ENABLED", "PAUSED")},
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"default_bid_amount": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Default max CPT bid as a decimal string (mutable).",
				Validators:          []validator.String{stringvalidator.RegexMatches(moneyAmountRegexp, "must be a positive decimal string")},
			},
			"default_bid_currency": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Currency for default_bid_amount (e.g. USD).",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"cpa_goal_amount": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Optional CPA goal amount as a decimal string (mutable).",
				Validators:          []validator.String{stringvalidator.RegexMatches(moneyAmountRegexp, "must be a positive decimal string")},
			},
			"cpa_goal_currency": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Currency for cpa_goal_amount.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"automated_keywords_opt_in": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Search Match / automated keywords opt-in (mutable).",
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"end_time": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Optional end time (ISO-8601, mutable).",
			},
			"serving_status": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Effective serving status from Apple Ads.",
			},
			"display_status": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Display status from Apple Ads.",
			},
			"modification_time": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Last modification timestamp.",
			},
		},
	}
}

func (r *adGroupResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *adGroupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan adGroupModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	campaignID, err := strconv.ParseInt(plan.CampaignID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("campaign_id"), "Invalid campaign_id", err.Error())
		return
	}
	bid, diags := moneyFromStrings(plan.DefaultBidAmount, plan.DefaultBidCurrency)
	resp.Diagnostics.Append(diags...)
	if bid == nil {
		resp.Diagnostics.AddAttributeError(path.Root("default_bid_amount"), "Missing default bid", "default_bid_amount is required")
	}
	if bid != nil {
		if amt, err := decimal.NewFromString(bid.Amount); err == nil && !amt.IsPositive() {
			resp.Diagnostics.AddError("Invalid default_bid_amount", "must be greater than zero")
		}
	}
	cpa, diags := moneyFromStrings(plan.CPAGoalAmount, plan.CPAGoalCurrency)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	in := &client.AdGroupCreate{
		Name:             plan.Name.ValueString(),
		DefaultBidAmount: bid,
		CPAGoal:          cpa,
	}
	if !plan.Status.IsNull() && !plan.Status.IsUnknown() {
		in.Status = plan.Status.ValueString()
	}
	if !plan.AutomatedKeywordsOptIn.IsNull() && !plan.AutomatedKeywordsOptIn.IsUnknown() {
		in.AutomatedKeywordsOptIn = plan.AutomatedKeywordsOptIn.ValueBool()
	}
	if !plan.EndTime.IsNull() && !plan.EndTime.IsUnknown() {
		in.EndTime = plan.EndTime.ValueString()
	}
	created, err := r.client.CreateAdGroup(ctx, campaignID, in)
	if err != nil {
		resp.Diagnostics.Append(apiErrorDiagnostic("Unable to create Apple Ads ad group", err)...)
		return
	}
	state := adGroupModelFromClient(created)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *adGroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state adGroupModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	campaignID, _ := strconv.ParseInt(state.CampaignID.ValueString(), 10, 64)
	adGroupID, err := strconv.ParseInt(state.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid ad group id", err.Error())
		return
	}
	got, err := r.client.GetAdGroup(ctx, campaignID, adGroupID)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.Append(apiErrorDiagnostic("Unable to read Apple Ads ad group", err)...)
		return
	}
	if got.Deleted {
		resp.State.RemoveResource(ctx)
		return
	}
	newState := adGroupModelFromClient(got)
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *adGroupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var state, plan adGroupModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if state.CampaignID.ValueString() != plan.CampaignID.ValueString() {
		resp.Diagnostics.AddError(
			`Cannot change immutable ad group field "campaign_id"`,
			fmt.Sprintf(
				"Apple Ads does not allow moving ad group %s between campaigns. "+
					"Automatically replacing this resource would delete historical ad group identity. "+
					"Create a new apple_ads_ad_group explicitly instead.",
				state.ID.ValueString(),
			),
		)
		return
	}
	campaignID, _ := strconv.ParseInt(state.CampaignID.ValueString(), 10, 64)
	adGroupID, _ := strconv.ParseInt(state.ID.ValueString(), 10, 64)
	bid, diags := moneyFromStrings(plan.DefaultBidAmount, plan.DefaultBidCurrency)
	resp.Diagnostics.Append(diags...)
	cpa, diags := moneyFromStrings(plan.CPAGoalAmount, plan.CPAGoalCurrency)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	upd := &client.AdGroupUpdate{
		Name:             plan.Name.ValueString(),
		DefaultBidAmount: bid,
		CPAGoal:          cpa,
	}
	if !plan.Status.IsNull() && !plan.Status.IsUnknown() {
		upd.Status = plan.Status.ValueString()
	}
	if !plan.AutomatedKeywordsOptIn.IsNull() && !plan.AutomatedKeywordsOptIn.IsUnknown() {
		v := plan.AutomatedKeywordsOptIn.ValueBool()
		upd.AutomatedKeywordsOptIn = &v
	}
	if !plan.EndTime.IsNull() && !plan.EndTime.IsUnknown() {
		upd.EndTime = plan.EndTime.ValueString()
	}
	updated, err := r.client.UpdateAdGroup(ctx, campaignID, adGroupID, upd)
	if err != nil {
		resp.Diagnostics.Append(apiErrorDiagnostic("Unable to update Apple Ads ad group", err)...)
		return
	}
	newState := adGroupModelFromClient(updated)
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *adGroupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state adGroupModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	campaignID, _ := strconv.ParseInt(state.CampaignID.ValueString(), 10, 64)
	adGroupID, _ := strconv.ParseInt(state.ID.ValueString(), 10, 64)
	if err := r.client.DeleteAdGroup(ctx, campaignID, adGroupID); err != nil {
		resp.Diagnostics.Append(apiErrorDiagnostic("Unable to delete Apple Ads ad group", err)...)
		return
	}
}

func (r *adGroupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	campaignID, adGroupID, err := client.ParseAdGroupImportID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import id", err.Error())
		return
	}
	var got *client.AdGroup
	if campaignID == 0 {
		got, err = r.client.FindAdGroupByID(ctx, adGroupID)
	} else {
		got, err = r.client.GetAdGroup(ctx, campaignID, adGroupID)
	}
	if err != nil {
		resp.Diagnostics.Append(apiErrorDiagnostic("Unable to import Apple Ads ad group", err)...)
		return
	}
	if got.Deleted {
		resp.Diagnostics.AddError("Cannot import deleted ad group", fmt.Sprintf("Ad group %d is deleted.", adGroupID))
		return
	}
	state := adGroupModelFromClient(got)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func adGroupModelFromClient(a *client.AdGroup) adGroupModel {
	m := adGroupModel{
		ID:                     types.StringValue(strconv.FormatInt(a.ID, 10)),
		CampaignID:             types.StringValue(strconv.FormatInt(a.CampaignID, 10)),
		Name:                   types.StringValue(a.Name),
		Status:                 types.StringValue(a.Status),
		ServingStatus:          types.StringValue(a.ServingStatus),
		DisplayStatus:          types.StringValue(a.DisplayStatus),
		ModificationTime:       types.StringValue(a.ModificationTime),
		AutomatedKeywordsOptIn: types.BoolValue(a.AutomatedKeywordsOptIn),
	}
	if a.DefaultBidAmount != nil {
		m.DefaultBidAmount = types.StringValue(a.DefaultBidAmount.Amount)
		m.DefaultBidCurrency = types.StringValue(a.DefaultBidAmount.Currency)
	}
	if a.CPAGoal != nil {
		m.CPAGoalAmount = types.StringValue(a.CPAGoal.Amount)
		m.CPAGoalCurrency = types.StringValue(a.CPAGoal.Currency)
	} else {
		m.CPAGoalAmount = types.StringNull()
		m.CPAGoalCurrency = types.StringNull()
	}
	if a.EndTime != "" {
		m.EndTime = types.StringValue(a.EndTime)
	} else {
		m.EndTime = types.StringNull()
	}
	return m
}
