// Copyright (c) 2026 Pi Cubed LLC
// SPDX-License-Identifier: MIT

package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework-validators/resourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/picubedllc/terraform-provider-appleads/internal/client"
)

var (
	_ resource.Resource                     = &negativeKeywordResource{}
	_ resource.ResourceWithConfigure        = &negativeKeywordResource{}
	_ resource.ResourceWithImportState      = &negativeKeywordResource{}
	_ resource.ResourceWithConfigValidators = &negativeKeywordResource{}
)

func NewNegativeKeywordResource() resource.Resource { return &negativeKeywordResource{} }

type negativeKeywordResource struct {
	client *client.Client
}

// negativeKeywordModel maps appleads_negative_keyword.
//
// Exactly one of campaign_id or ad_group_id must be set in configuration
// (Ivan/PI-18 decision: single resource type for both scopes).
//
// Immutable: campaign_id, ad_group_id, text, match_type (Update errors; no RequiresReplace).
// Mutable: status
// Computed: id, modification_time; campaign_id is also computed when resolved from an ad group.
type negativeKeywordModel struct {
	ID               types.String `tfsdk:"id"`
	CampaignID       types.String `tfsdk:"campaign_id"`
	AdGroupID        types.String `tfsdk:"ad_group_id"`
	Text             types.String `tfsdk:"text"`
	MatchType        types.String `tfsdk:"match_type"`
	Status           types.String `tfsdk:"status"`
	ModificationTime types.String `tfsdk:"modification_time"`
}

func (r *negativeKeywordResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_negative_keyword"
}

func (r *negativeKeywordResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an Apple Ads negative keyword at either campaign or ad-group scope.\n\n" +
			"Exactly one of `campaign_id` or `ad_group_id` must be set. Campaign-scoped and ad-group-scoped " +
			"negatives use different Apple Ads endpoints under the hood; this resource branches internally.\n\n" +
			"**Immutable:** `campaign_id`, `ad_group_id`, `text`, `match_type` — changing these returns an error (no `RequiresReplace`).\n\n" +
			"**Mutable:** `status` (`ACTIVE` / `PAUSED`).\n\n" +
			"**Computed:** `id`, `modification_time`. When only `ad_group_id` is set, `campaign_id` is resolved and stored as computed.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Negative keyword identifier.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"campaign_id": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Campaign id for campaign-scoped negatives (immutable). Also computed for ad-group-scoped negatives.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"ad_group_id": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Ad group id for ad-group-scoped negatives (immutable). Mutually exclusive with configuring `campaign_id`.",
			},
			"text": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Negative keyword text (immutable).",
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
			"modification_time": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Last modification timestamp.",
			},
		},
	}
}

func (r *negativeKeywordResource) ConfigValidators(ctx context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		resourcevalidator.ExactlyOneOf(
			path.MatchRoot("campaign_id"),
			path.MatchRoot("ad_group_id"),
		),
	}
}

func (r *negativeKeywordResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *negativeKeywordResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan negativeKeywordModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	in := &client.NegativeKeywordCreate{
		Text:      plan.Text.ValueString(),
		MatchType: plan.MatchType.ValueString(),
	}
	if !plan.Status.IsNull() && !plan.Status.IsUnknown() {
		in.Status = plan.Status.ValueString()
	}

	var created *client.NegativeKeyword
	var err error

	if !plan.AdGroupID.IsNull() && !plan.AdGroupID.IsUnknown() && plan.AdGroupID.ValueString() != "" {
		adGroupID, parseErr := strconv.ParseInt(plan.AdGroupID.ValueString(), 10, 64)
		if parseErr != nil {
			resp.Diagnostics.AddAttributeError(path.Root("ad_group_id"), "Invalid ad_group_id", parseErr.Error())
			return
		}
		ag, findErr := r.client.FindAdGroupByID(ctx, adGroupID)
		if findErr != nil {
			resp.Diagnostics.Append(apiErrorDiagnostic("Unable to resolve parent ad group for negative keyword", findErr)...)
			return
		}
		created, err = r.client.CreateAdGroupNegativeKeyword(ctx, ag.CampaignID, adGroupID, in)
		if err == nil {
			if created.CampaignID == 0 {
				created.CampaignID = ag.CampaignID
			}
			if created.AdGroupID == 0 {
				created.AdGroupID = adGroupID
			}
		}
	} else {
		campaignID, parseErr := strconv.ParseInt(plan.CampaignID.ValueString(), 10, 64)
		if parseErr != nil {
			resp.Diagnostics.AddAttributeError(path.Root("campaign_id"), "Invalid campaign_id", parseErr.Error())
			return
		}
		created, err = r.client.CreateCampaignNegativeKeyword(ctx, campaignID, in)
		if err == nil && created.CampaignID == 0 {
			created.CampaignID = campaignID
		}
	}
	if err != nil {
		resp.Diagnostics.Append(apiErrorDiagnostic("Unable to create Apple Ads negative keyword", err)...)
		return
	}
	state, diags := negativeKeywordModelFromClient(created)
	resp.Diagnostics.Append(diags...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *negativeKeywordResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state negativeKeywordModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	got, err := r.fetch(ctx, state)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.Append(apiErrorDiagnostic("Unable to read Apple Ads negative keyword", err)...)
		return
	}
	if got.Deleted {
		resp.State.RemoveResource(ctx)
		return
	}
	newState, diags := negativeKeywordModelFromClient(got)
	resp.Diagnostics.Append(diags...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *negativeKeywordResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var state, plan negativeKeywordModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.CampaignID.ValueString() != plan.CampaignID.ValueString() &&
		!(state.AdGroupID.ValueString() != "" && plan.CampaignID.IsUnknown()) {
		// Allow computed campaign_id unknown in plan for ad-group-scoped resources.
		if !plan.CampaignID.IsUnknown() {
			resp.Diagnostics.AddError(
				`Cannot change immutable negative keyword field "campaign_id"`,
				fmt.Sprintf("Negative keyword %s cannot change campaign scope in place. Create a new appleads_negative_keyword instead.", state.ID.ValueString()),
			)
			return
		}
	}
	if nullish(state.AdGroupID) != nullish(plan.AdGroupID) || state.AdGroupID.ValueString() != plan.AdGroupID.ValueString() {
		if !plan.AdGroupID.IsUnknown() {
			resp.Diagnostics.AddError(
				`Cannot change immutable negative keyword field "ad_group_id"`,
				fmt.Sprintf("Negative keyword %s cannot change ad group scope in place. Create a new appleads_negative_keyword instead.", state.ID.ValueString()),
			)
			return
		}
	}
	if state.Text.ValueString() != plan.Text.ValueString() {
		resp.Diagnostics.AddError(
			`Cannot change immutable negative keyword field "text"`,
			fmt.Sprintf("Apple Ads does not allow changing negative keyword text in place for %s. Create a new appleads_negative_keyword instead.", state.ID.ValueString()),
		)
		return
	}
	if state.MatchType.ValueString() != plan.MatchType.ValueString() {
		resp.Diagnostics.AddError(
			`Cannot change immutable negative keyword field "match_type"`,
			fmt.Sprintf("Apple Ads does not allow changing match_type in place for negative keyword %s. Create a new appleads_negative_keyword instead.", state.ID.ValueString()),
		)
		return
	}

	keywordID, _ := strconv.ParseInt(state.ID.ValueString(), 10, 64)
	upd := &client.NegativeKeywordUpdate{ID: keywordID}
	if !plan.Status.IsNull() && !plan.Status.IsUnknown() {
		upd.Status = plan.Status.ValueString()
	}

	var updated *client.NegativeKeyword
	var err error
	campaignID, _ := strconv.ParseInt(state.CampaignID.ValueString(), 10, 64)
	if !nullish(state.AdGroupID) {
		adGroupID, _ := strconv.ParseInt(state.AdGroupID.ValueString(), 10, 64)
		updated, err = r.client.UpdateAdGroupNegativeKeyword(ctx, campaignID, adGroupID, upd)
		if err == nil {
			if updated.CampaignID == 0 {
				updated.CampaignID = campaignID
			}
			if updated.AdGroupID == 0 {
				updated.AdGroupID = adGroupID
			}
		}
	} else {
		updated, err = r.client.UpdateCampaignNegativeKeyword(ctx, campaignID, upd)
		if err == nil && updated.CampaignID == 0 {
			updated.CampaignID = campaignID
		}
	}
	if err != nil {
		resp.Diagnostics.Append(apiErrorDiagnostic("Unable to update Apple Ads negative keyword", err)...)
		return
	}
	if updated.Text == "" {
		updated.Text = state.Text.ValueString()
	}
	if updated.MatchType == "" {
		updated.MatchType = state.MatchType.ValueString()
	}
	newState, diags := negativeKeywordModelFromClient(updated)
	resp.Diagnostics.Append(diags...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *negativeKeywordResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state negativeKeywordModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	campaignID, _ := strconv.ParseInt(state.CampaignID.ValueString(), 10, 64)
	keywordID, _ := strconv.ParseInt(state.ID.ValueString(), 10, 64)
	var err error
	if !nullish(state.AdGroupID) {
		adGroupID, _ := strconv.ParseInt(state.AdGroupID.ValueString(), 10, 64)
		err = r.client.DeleteAdGroupNegativeKeyword(ctx, campaignID, adGroupID, keywordID)
	} else {
		err = r.client.DeleteCampaignNegativeKeyword(ctx, campaignID, keywordID)
	}
	if err != nil {
		resp.Diagnostics.Append(apiErrorDiagnostic("Unable to delete Apple Ads negative keyword", err)...)
	}
}

func (r *negativeKeywordResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	scope, campaignID, adGroupID, keywordID, err := client.ParseNegativeKeywordImportID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import id", err.Error())
		return
	}
	var got *client.NegativeKeyword
	switch scope {
	case "campaign":
		got, err = r.client.GetCampaignNegativeKeyword(ctx, campaignID, keywordID)
	case "adgroup":
		if campaignID == 0 {
			ag, findErr := r.client.FindAdGroupByID(ctx, adGroupID)
			if findErr != nil {
				resp.Diagnostics.Append(apiErrorDiagnostic("Unable to resolve ad group for negative keyword import", findErr)...)
				return
			}
			campaignID = ag.CampaignID
		}
		got, err = r.client.GetAdGroupNegativeKeyword(ctx, campaignID, adGroupID, keywordID)
	default:
		resp.Diagnostics.AddError("Invalid import id", "unknown scope")
		return
	}
	if err != nil {
		resp.Diagnostics.Append(apiErrorDiagnostic("Unable to import Apple Ads negative keyword", err)...)
		return
	}
	if got.Deleted {
		resp.Diagnostics.AddError("Cannot import deleted negative keyword", fmt.Sprintf("Negative keyword %d is deleted.", keywordID))
		return
	}
	if got.CampaignID == 0 {
		got.CampaignID = campaignID
	}
	if scope == "adgroup" && got.AdGroupID == 0 {
		got.AdGroupID = adGroupID
	}
	state, diags := negativeKeywordModelFromClient(got)
	resp.Diagnostics.Append(diags...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *negativeKeywordResource) fetch(ctx context.Context, state negativeKeywordModel) (*client.NegativeKeyword, error) {
	campaignID, _ := strconv.ParseInt(state.CampaignID.ValueString(), 10, 64)
	keywordID, err := strconv.ParseInt(state.ID.ValueString(), 10, 64)
	if err != nil {
		return nil, err
	}
	if !nullish(state.AdGroupID) {
		adGroupID, _ := strconv.ParseInt(state.AdGroupID.ValueString(), 10, 64)
		return r.client.GetAdGroupNegativeKeyword(ctx, campaignID, adGroupID, keywordID)
	}
	return r.client.GetCampaignNegativeKeyword(ctx, campaignID, keywordID)
}

func negativeKeywordModelFromClient(k *client.NegativeKeyword) (negativeKeywordModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	m := negativeKeywordModel{
		ID:               types.StringValue(strconv.FormatInt(k.ID, 10)),
		CampaignID:       types.StringValue(strconv.FormatInt(k.CampaignID, 10)),
		Text:             types.StringValue(k.Text),
		MatchType:        types.StringValue(k.MatchType),
		Status:           types.StringValue(k.Status),
		ModificationTime: types.StringValue(k.ModificationTime),
	}
	if k.AdGroupID != 0 {
		m.AdGroupID = types.StringValue(strconv.FormatInt(k.AdGroupID, 10))
	} else {
		m.AdGroupID = types.StringNull()
	}
	return m, diags
}

func nullish(v types.String) bool {
	return v.IsNull() || v.IsUnknown() || v.ValueString() == ""
}
