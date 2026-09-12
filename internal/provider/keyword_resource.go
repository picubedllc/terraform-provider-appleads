// Copyright (c) 2026 Pi Cubed LLC
// SPDX-License-Identifier: MIT

package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
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
	_ resource.Resource                = &keywordResource{}
	_ resource.ResourceWithConfigure   = &keywordResource{}
	_ resource.ResourceWithImportState = &keywordResource{}
)

func NewKeywordResource() resource.Resource { return &keywordResource{} }

type keywordResource struct {
	client *client.Client
}

// keywordModel maps appleads_keyword.
//
// Immutable: ad_group_id, text, match_type (no RequiresReplace — Update errors instead).
// Mutable: status, bid_amount, bid_currency
// Computed: id, campaign_id, modification_time
type keywordModel struct {
	ID               types.String `tfsdk:"id"`
	CampaignID       types.String `tfsdk:"campaign_id"`
	AdGroupID        types.String `tfsdk:"ad_group_id"`
	Text             types.String `tfsdk:"text"`
	MatchType        types.String `tfsdk:"match_type"`
	Status           types.String `tfsdk:"status"`
	BidAmount        types.String `tfsdk:"bid_amount"`
	BidCurrency      types.String `tfsdk:"bid_currency"`
	ModificationTime types.String `tfsdk:"modification_time"`
}

func (r *keywordResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_keyword"
}

func (r *keywordResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an Apple Ads targeting keyword under an ad group.\n\n" +
			"**Immutable:** `ad_group_id`, `text`, `match_type` — changing any of these returns an error (no `RequiresReplace`) so keyword identity/history is preserved.\n\n" +
			"**Mutable:** `status`, `bid_amount`/`bid_currency`.\n\n" +
			"**Computed:** `id`, `campaign_id`, `modification_time`.\n\n" +
			"Match types follow Apple Ads API v5: `EXACT` and `BROAD`. Keyword status values are `ACTIVE` and `PAUSED`.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Keyword identifier.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"campaign_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Parent campaign id (resolved from the ad group).",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"ad_group_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Parent ad group id (immutable). Changing this after create returns an error; create a new appleads_keyword instead.",
			},
			"text": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Keyword text (immutable).",
				Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"match_type": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "EXACT or BROAD (immutable).",
				Validators:          []validator.String{stringvalidator.OneOf("EXACT", "BROAD")},
			},
			"status": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "ACTIVE or PAUSED (mutable).",
				Validators:          []validator.String{stringvalidator.OneOf("ACTIVE", "PAUSED")},
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"bid_amount": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Optional keyword CPT bid as a decimal string (mutable). When omitted, the ad group default bid applies.",
				Validators:          []validator.String{stringvalidator.RegexMatches(moneyAmountRegexp, "must be a positive decimal string")},
			},
			"bid_currency": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Currency for bid_amount (e.g. USD).",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"modification_time": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Last modification timestamp.",
			},
		},
	}
}

func (r *keywordResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *keywordResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan keywordModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	adGroupID, err := strconv.ParseInt(plan.AdGroupID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("ad_group_id"), "Invalid ad_group_id", err.Error())
		return
	}

	ag, err := r.client.FindAdGroupByID(ctx, adGroupID)
	if err != nil {
		resp.Diagnostics.Append(apiErrorDiagnostic("Unable to resolve parent ad group for keyword", err)...)
		return
	}
	campaignID := ag.CampaignID

	bid, diags := moneyFromStrings(plan.BidAmount, plan.BidCurrency)
	resp.Diagnostics.Append(diags...)
	if bid != nil {
		if amt, err := decimal.NewFromString(bid.Amount); err == nil && !amt.IsPositive() {
			resp.Diagnostics.AddError("Invalid bid_amount", "must be greater than zero")
		}
	}
	if resp.Diagnostics.HasError() {
		return
	}

	in := &client.KeywordCreate{
		Text:      plan.Text.ValueString(),
		MatchType: plan.MatchType.ValueString(),
		BidAmount: bid,
	}
	if !plan.Status.IsNull() && !plan.Status.IsUnknown() {
		in.Status = plan.Status.ValueString()
	}

	created, err := r.client.CreateKeyword(ctx, campaignID, adGroupID, in)
	if err != nil {
		resp.Diagnostics.Append(apiErrorDiagnostic("Unable to create Apple Ads keyword", err)...)
		return
	}
	// Ensure parent ids are present even if API omits them on create response.
	if created.CampaignID == 0 {
		created.CampaignID = campaignID
	}
	if created.AdGroupID == 0 {
		created.AdGroupID = adGroupID
	}
	state, diags := keywordModelFromClient(created)
	resp.Diagnostics.Append(diags...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *keywordResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state keywordModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	campaignID, _ := strconv.ParseInt(state.CampaignID.ValueString(), 10, 64)
	adGroupID, _ := strconv.ParseInt(state.AdGroupID.ValueString(), 10, 64)
	keywordID, err := strconv.ParseInt(state.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid keyword id", err.Error())
		return
	}
	got, err := r.client.GetKeyword(ctx, campaignID, adGroupID, keywordID)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.Append(apiErrorDiagnostic("Unable to read Apple Ads keyword", err)...)
		return
	}
	if got.Deleted {
		resp.State.RemoveResource(ctx)
		return
	}
	newState, diags := keywordModelFromClient(got)
	resp.Diagnostics.Append(diags...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *keywordResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var state, plan keywordModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.AdGroupID.ValueString() != plan.AdGroupID.ValueString() {
		resp.Diagnostics.AddError(
			`Cannot change immutable keyword field "ad_group_id"`,
			fmt.Sprintf(
				"Apple Ads does not allow moving keyword %s between ad groups. "+
					"Automatically replacing this resource would delete historical keyword identity. "+
					"Create a new appleads_keyword explicitly instead.",
				state.ID.ValueString(),
			),
		)
		return
	}
	if state.Text.ValueString() != plan.Text.ValueString() {
		resp.Diagnostics.AddError(
			`Cannot change immutable keyword field "text"`,
			fmt.Sprintf(
				"Apple Ads does not allow changing keyword text in place for keyword %s. "+
					"Create a new appleads_keyword with the desired text instead.",
				state.ID.ValueString(),
			),
		)
		return
	}
	if state.MatchType.ValueString() != plan.MatchType.ValueString() {
		resp.Diagnostics.AddError(
			`Cannot change immutable keyword field "match_type"`,
			fmt.Sprintf(
				"Apple Ads does not allow changing match_type in place for keyword %s. "+
					"Create a new appleads_keyword with the desired match type instead.",
				state.ID.ValueString(),
			),
		)
		return
	}

	campaignID, _ := strconv.ParseInt(state.CampaignID.ValueString(), 10, 64)
	adGroupID, _ := strconv.ParseInt(state.AdGroupID.ValueString(), 10, 64)
	keywordID, _ := strconv.ParseInt(state.ID.ValueString(), 10, 64)

	bid, diags := moneyFromStrings(plan.BidAmount, plan.BidCurrency)
	resp.Diagnostics.Append(diags...)
	if bid != nil {
		if amt, err := decimal.NewFromString(bid.Amount); err == nil && !amt.IsPositive() {
			resp.Diagnostics.AddError("Invalid bid_amount", "must be greater than zero")
		}
	}
	if resp.Diagnostics.HasError() {
		return
	}

	upd := &client.KeywordUpdate{
		ID:        keywordID,
		BidAmount: bid,
	}
	if !plan.Status.IsNull() && !plan.Status.IsUnknown() {
		upd.Status = plan.Status.ValueString()
	}

	updated, err := r.client.UpdateKeyword(ctx, campaignID, adGroupID, upd)
	if err != nil {
		resp.Diagnostics.Append(apiErrorDiagnostic("Unable to update Apple Ads keyword", err)...)
		return
	}
	if updated.CampaignID == 0 {
		updated.CampaignID = campaignID
	}
	if updated.AdGroupID == 0 {
		updated.AdGroupID = adGroupID
	}
	// Preserve immutable fields if bulk update response omits them.
	if updated.Text == "" {
		updated.Text = state.Text.ValueString()
	}
	if updated.MatchType == "" {
		updated.MatchType = state.MatchType.ValueString()
	}
	newState, diags := keywordModelFromClient(updated)
	resp.Diagnostics.Append(diags...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *keywordResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state keywordModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	campaignID, _ := strconv.ParseInt(state.CampaignID.ValueString(), 10, 64)
	adGroupID, _ := strconv.ParseInt(state.AdGroupID.ValueString(), 10, 64)
	keywordID, _ := strconv.ParseInt(state.ID.ValueString(), 10, 64)
	if err := r.client.DeleteKeyword(ctx, campaignID, adGroupID, keywordID); err != nil {
		resp.Diagnostics.Append(apiErrorDiagnostic("Unable to delete Apple Ads keyword", err)...)
		return
	}
}

func (r *keywordResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	campaignID, adGroupID, keywordID, err := client.ParseKeywordImportID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import id", err.Error())
		return
	}
	var got *client.Keyword
	if campaignID == 0 {
		ag, err := r.client.FindAdGroupByID(ctx, adGroupID)
		if err != nil {
			resp.Diagnostics.Append(apiErrorDiagnostic("Unable to resolve ad group for keyword import", err)...)
			return
		}
		campaignID = ag.CampaignID
		got, err = r.client.GetKeyword(ctx, campaignID, adGroupID, keywordID)
	} else {
		got, err = r.client.GetKeyword(ctx, campaignID, adGroupID, keywordID)
	}
	if err != nil {
		resp.Diagnostics.Append(apiErrorDiagnostic("Unable to import Apple Ads keyword", err)...)
		return
	}
	if got.Deleted {
		resp.Diagnostics.AddError("Cannot import deleted keyword", fmt.Sprintf("Keyword %d is deleted.", keywordID))
		return
	}
	if got.CampaignID == 0 {
		got.CampaignID = campaignID
	}
	if got.AdGroupID == 0 {
		got.AdGroupID = adGroupID
	}
	state, diags := keywordModelFromClient(got)
	resp.Diagnostics.Append(diags...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func keywordModelFromClient(k *client.Keyword) (keywordModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	m := keywordModel{
		ID:               types.StringValue(strconv.FormatInt(k.ID, 10)),
		CampaignID:       types.StringValue(strconv.FormatInt(k.CampaignID, 10)),
		AdGroupID:        types.StringValue(strconv.FormatInt(k.AdGroupID, 10)),
		Text:             types.StringValue(k.Text),
		MatchType:        types.StringValue(k.MatchType),
		Status:           types.StringValue(k.Status),
		ModificationTime: types.StringValue(k.ModificationTime),
	}
	if k.BidAmount != nil {
		m.BidAmount = types.StringValue(k.BidAmount.Amount)
		m.BidCurrency = types.StringValue(k.BidAmount.Currency)
	} else {
		m.BidAmount = types.StringNull()
		m.BidCurrency = types.StringNull()
	}
	return m, diags
}
