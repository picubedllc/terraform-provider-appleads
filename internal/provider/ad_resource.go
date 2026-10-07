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

	"github.com/picubedllc/terraform-provider-appleads/internal/client"
)

var (
	_ resource.Resource                = &adResource{}
	_ resource.ResourceWithConfigure   = &adResource{}
	_ resource.ResourceWithImportState = &adResource{}
)

func NewAdResource() resource.Resource { return &adResource{} }

type adResource struct {
	client *client.Client
}

// adModel maps appleads_ad.
//
// Immutable: campaign_id (computed from ad group), ad_group_id, creative_id
// (Apple AdUpdate treats creativeId as read-only — create a new ad to change).
// Mutable: name, status.
// Computed: id, creative_type, serving_status, serving_state_reasons,
// creation_time, modification_time.
type adModel struct {
	ID                  types.String `tfsdk:"id"`
	CampaignID          types.String `tfsdk:"campaign_id"`
	AdGroupID           types.String `tfsdk:"ad_group_id"`
	Name                types.String `tfsdk:"name"`
	CreativeID          types.String `tfsdk:"creative_id"`
	CreativeType        types.String `tfsdk:"creative_type"`
	Status              types.String `tfsdk:"status"`
	ServingStatus       types.String `tfsdk:"serving_status"`
	ServingStateReasons types.List   `tfsdk:"serving_state_reasons"`
	CreationTime        types.String `tfsdk:"creation_time"`
	ModificationTime    types.String `tfsdk:"modification_time"`
}

func (r *adResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ad"
}

func (r *adResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an Apple Ads ad — the assignment of a creative to an ad group.\n\n" +
			"Display campaigns (for example Today Tab via `APPSTORE_TODAY_TAB`) need both a creative and an " +
			"ad to serve. Create `appleads_creative` first (typically `CUSTOM_PRODUCT_PAGE` wired to a product " +
			"page id), then attach it here. Apple allows one active custom product page ad per ad group.\n\n" +
			"**Today Tab localization:** Custom product page app name and subtitle must be localized for the " +
			"default language of each campaign country or region. Verify with `data.appleads_product_page_locales`.\n\n" +
			"**Immutable:** `ad_group_id`, `creative_id` — changing either returns an error (no `RequiresReplace`). " +
			"Create a new `appleads_ad` to rotate creatives.\n\n" +
			"**Mutable:** `name`, `status` (`ENABLED` or `PAUSED`).\n\n" +
			"**Computed:** `id`, `campaign_id`, `creative_type`, `serving_status`, `serving_state_reasons`, " +
			"`creation_time`, `modification_time`.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Ad identifier (assignment of creative to ad group).",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"campaign_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Parent campaign id (resolved from the ad group).",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"ad_group_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Parent ad group id (immutable). Changing this after create returns an error; create a new appleads_ad instead.",
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Ad name unique within the ad group (mutable).",
				Validators:          []validator.String{stringvalidator.LengthBetween(1, 255)},
			},
			"creative_id": schema.StringAttribute{
				Required: true,
				MarkdownDescription: "Creative id from `appleads_creative` (immutable). Changing this after create " +
					"returns an error; create a new appleads_ad instead.",
			},
			"creative_type": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Creative type reported by Apple (for example CUSTOM_PRODUCT_PAGE).",
			},
			"status": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "User-controlled ad status: `ENABLED` or `PAUSED` (mutable).",
				Validators:          []validator.String{stringvalidator.OneOf(client.AdStatusEnabled, client.AdStatusPaused)},
			},
			"serving_status": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Serving status indicator from Apple Ads.",
			},
			"serving_state_reasons": schema.ListAttribute{
				ElementType:         types.StringType,
				Computed:            true,
				MarkdownDescription: "Reasons the ad is not serving, when applicable.",
			},
			"creation_time": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Creation timestamp.",
			},
			"modification_time": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Last modification timestamp.",
			},
		},
	}
}

func (r *adResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *adResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan adModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	adGroupID, err := strconv.ParseInt(plan.AdGroupID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("ad_group_id"), "Invalid ad_group_id", err.Error())
		return
	}
	creativeID, err := strconv.ParseInt(plan.CreativeID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("creative_id"), "Invalid creative_id", err.Error())
		return
	}

	ag, err := r.client.FindAdGroupByID(ctx, adGroupID)
	if err != nil {
		resp.Diagnostics.Append(apiErrorDiagnostic("Unable to resolve parent ad group for ad", err)...)
		return
	}
	campaignID := ag.CampaignID

	created, err := r.client.CreateAd(ctx, campaignID, adGroupID, &client.AdCreate{
		Name:       plan.Name.ValueString(),
		CreativeID: creativeID,
		Status:     plan.Status.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.Append(apiErrorDiagnostic("Unable to create Apple Ads ad", err)...)
		return
	}
	if created.CampaignID == 0 {
		created.CampaignID = campaignID
	}
	if created.AdGroupID == 0 {
		created.AdGroupID = adGroupID
	}
	if created.CreativeID == 0 {
		created.CreativeID = creativeID
	}
	state, diags := adModelFromClient(ctx, created)
	resp.Diagnostics.Append(diags...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *adResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state adModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	campaignID, _ := strconv.ParseInt(state.CampaignID.ValueString(), 10, 64)
	adGroupID, _ := strconv.ParseInt(state.AdGroupID.ValueString(), 10, 64)
	adID, err := strconv.ParseInt(state.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid ad id", err.Error())
		return
	}

	got, err := r.client.GetAd(ctx, campaignID, adGroupID, adID)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.Append(apiErrorDiagnostic("Unable to read Apple Ads ad", err)...)
		return
	}
	if got.Deleted {
		resp.State.RemoveResource(ctx)
		return
	}
	if got.CampaignID == 0 {
		got.CampaignID = campaignID
	}
	if got.AdGroupID == 0 {
		got.AdGroupID = adGroupID
	}
	newState, diags := adModelFromClient(ctx, got)
	resp.Diagnostics.Append(diags...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *adResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var state, plan adModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.AdGroupID.ValueString() != plan.AdGroupID.ValueString() {
		resp.Diagnostics.AddError(
			`Cannot change immutable ad field "ad_group_id"`,
			fmt.Sprintf(
				"Apple Ads does not allow moving ad %s between ad groups. "+
					"Automatically replacing this resource would delete historical ad identity. "+
					"Create a new appleads_ad explicitly instead.",
				state.ID.ValueString(),
			),
		)
		return
	}
	if state.CreativeID.ValueString() != plan.CreativeID.ValueString() {
		resp.Diagnostics.AddError(
			`Cannot change immutable ad field "creative_id"`,
			fmt.Sprintf(
				"Apple Ads does not allow changing creative_id in place for ad %s (AdUpdate treats creativeId as read-only). "+
					"Create a new appleads_ad with the desired creative instead.",
				state.ID.ValueString(),
			),
		)
		return
	}

	campaignID, _ := strconv.ParseInt(state.CampaignID.ValueString(), 10, 64)
	adGroupID, _ := strconv.ParseInt(state.AdGroupID.ValueString(), 10, 64)
	adID, _ := strconv.ParseInt(state.ID.ValueString(), 10, 64)

	updated, err := r.client.UpdateAd(ctx, campaignID, adGroupID, adID, &client.AdUpdate{
		Name:   plan.Name.ValueString(),
		Status: plan.Status.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.Append(apiErrorDiagnostic("Unable to update Apple Ads ad", err)...)
		return
	}
	if updated.CampaignID == 0 {
		updated.CampaignID = campaignID
	}
	if updated.AdGroupID == 0 {
		updated.AdGroupID = adGroupID
	}
	if updated.CreativeID == 0 {
		cid, _ := strconv.ParseInt(state.CreativeID.ValueString(), 10, 64)
		updated.CreativeID = cid
	}
	if updated.CreativeType == "" {
		updated.CreativeType = state.CreativeType.ValueString()
	}
	newState, diags := adModelFromClient(ctx, updated)
	resp.Diagnostics.Append(diags...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *adResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state adModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	campaignID, _ := strconv.ParseInt(state.CampaignID.ValueString(), 10, 64)
	adGroupID, _ := strconv.ParseInt(state.AdGroupID.ValueString(), 10, 64)
	adID, _ := strconv.ParseInt(state.ID.ValueString(), 10, 64)
	if err := r.client.DeleteAd(ctx, campaignID, adGroupID, adID); err != nil {
		resp.Diagnostics.Append(apiErrorDiagnostic("Unable to delete Apple Ads ad", err)...)
		return
	}
}

func (r *adResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	campaignID, adGroupID, adID, err := client.ParseAdImportID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import id", err.Error())
		return
	}
	var got *client.Ad
	if campaignID == 0 {
		ag, findErr := r.client.FindAdGroupByID(ctx, adGroupID)
		if findErr != nil {
			resp.Diagnostics.Append(apiErrorDiagnostic("Unable to resolve ad group for ad import", findErr)...)
			return
		}
		campaignID = ag.CampaignID
		got, err = r.client.GetAd(ctx, campaignID, adGroupID, adID)
	} else {
		got, err = r.client.GetAd(ctx, campaignID, adGroupID, adID)
	}
	if err != nil {
		resp.Diagnostics.Append(apiErrorDiagnostic("Unable to import Apple Ads ad", err)...)
		return
	}
	if got.Deleted {
		resp.Diagnostics.AddError("Cannot import deleted ad", fmt.Sprintf("Ad %d is deleted.", adID))
		return
	}
	if got.CampaignID == 0 {
		got.CampaignID = campaignID
	}
	if got.AdGroupID == 0 {
		got.AdGroupID = adGroupID
	}
	state, diags := adModelFromClient(ctx, got)
	resp.Diagnostics.Append(diags...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func adModelFromClient(ctx context.Context, a *client.Ad) (adModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	m := adModel{
		ID:               types.StringValue(strconv.FormatInt(a.ID, 10)),
		CampaignID:       types.StringValue(strconv.FormatInt(a.CampaignID, 10)),
		AdGroupID:        types.StringValue(strconv.FormatInt(a.AdGroupID, 10)),
		Name:             types.StringValue(a.Name),
		CreativeID:       types.StringValue(strconv.FormatInt(a.CreativeID, 10)),
		CreativeType:     types.StringValue(a.CreativeType),
		Status:           types.StringValue(a.Status),
		ServingStatus:    types.StringValue(a.ServingStatus),
		CreationTime:     types.StringValue(a.CreationTime),
		ModificationTime: types.StringValue(a.ModificationTime),
	}
	reasons, d := types.ListValueFrom(ctx, types.StringType, a.ServingStateReasons)
	diags.Append(d...)
	if reasons.IsNull() {
		m.ServingStateReasons = types.ListValueMust(types.StringType, nil)
	} else {
		m.ServingStateReasons = reasons
	}
	return m, diags
}
