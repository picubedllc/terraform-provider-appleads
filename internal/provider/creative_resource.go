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
	_ resource.Resource                = &creativeResource{}
	_ resource.ResourceWithConfigure   = &creativeResource{}
	_ resource.ResourceWithImportState = &creativeResource{}
)

func NewCreativeResource() resource.Resource { return &creativeResource{} }

type creativeResource struct {
	client *client.Client
}

// creativeModel maps appleads_creative.
//
// Immutable / create-only (API v5 has no creative update): adam_id, name, type,
// product_page_id. Changing any returns an error (no RequiresReplace).
//
// Computed: id, org_id, state, state_reasons, creation_time, modification_time.
//
// Destroy removes Terraform state only — Apple Ads API v5 has no creative delete.
type creativeModel struct {
	ID               types.String `tfsdk:"id"`
	OrgID            types.String `tfsdk:"org_id"`
	AdamID           types.String `tfsdk:"adam_id"`
	Name             types.String `tfsdk:"name"`
	Type             types.String `tfsdk:"type"`
	ProductPageID    types.String `tfsdk:"product_page_id"`
	State            types.String `tfsdk:"state"`
	StateReasons     types.List   `tfsdk:"state_reasons"`
	CreationTime     types.String `tfsdk:"creation_time"`
	ModificationTime types.String `tfsdk:"modification_time"`
}

func (r *creativeResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_creative"
}

func (r *creativeResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an Apple Ads creative (organization-scoped).\n\n" +
			"Creatives wrap a Custom Product Page or Default Product Page reference. " +
			"Wire `product_page_id` from `data.appleads_product_page` / `data.appleads_product_pages` " +
			"when `type` is `CUSTOM_PRODUCT_PAGE`. Ads reference the creative via `appleads_ad.creative_id`.\n\n" +
			"**Display / Today Tab:** Campaigns with `supply_sources = [\"APPSTORE_TODAY_TAB\"]` require an " +
			"approved custom product page creative. Product page name and subtitle must be localized in the " +
			"`defaultLanguage` of every country or region where the ad serves (use " +
			"`data.appleads_product_page_locales` and `data.appleads_countries_or_regions` to verify). " +
			"Today Tab is iPhone-only.\n\n" +
			"**Immutable (create-only):** `adam_id`, `name`, `type`, `product_page_id` — Apple Ads API v5 " +
			"has no creative update endpoint. Changing any of these returns an error (no `RequiresReplace`).\n\n" +
			"**Computed:** `id`, `org_id`, `state`, `state_reasons`, `creation_time`, `modification_time`.\n\n" +
			"**Destroy:** API v5 has no creative delete. Destroy removes the resource from Terraform state only; " +
			"the creative remains in Apple Ads.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Creative identifier assigned by Apple Ads.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"org_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Organization that owns the creative.",
			},
			"adam_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Adam ID of the promoted app (immutable).",
				Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Creative name (immutable; API v5 has no creative update).",
				Validators:          []validator.String{stringvalidator.LengthBetween(1, 200)},
			},
			"type": schema.StringAttribute{
				Required: true,
				MarkdownDescription: "Creative type (immutable): `CUSTOM_PRODUCT_PAGE` or `DEFAULT_PRODUCT_PAGE`. " +
					"`CUSTOM_PRODUCT_PAGE` requires `product_page_id`.",
				Validators: []validator.String{stringvalidator.OneOf(
					client.CreativeTypeCustomProductPage,
					client.CreativeTypeDefaultProductPage,
				)},
			},
			"product_page_id": schema.StringAttribute{
				Optional: true,
				MarkdownDescription: "Custom product page UUID from App Store Connect (immutable). Required when " +
					"`type` is `CUSTOM_PRODUCT_PAGE`. Obtain via `data.appleads_product_pages` / `data.appleads_product_page`.",
			},
			"state": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "System state of the creative (for example VALID, INVALID).",
			},
			"state_reasons": schema.ListAttribute{
				ElementType:         types.StringType,
				Computed:            true,
				MarkdownDescription: "Detailed explanation of the system state.",
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

func (r *creativeResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *creativeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan creativeModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	in, diags := creativeCreateFromPlan(plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.client.CreateCreative(ctx, in)
	if err != nil {
		resp.Diagnostics.Append(apiErrorDiagnostic("Unable to create Apple Ads creative", err)...)
		return
	}
	state, diags := creativeModelFromClient(ctx, created)
	resp.Diagnostics.Append(diags...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *creativeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state creativeModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := client.ParseCreativeID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid creative id", err.Error())
		return
	}
	got, err := r.client.GetCreative(ctx, id)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.Append(apiErrorDiagnostic("Unable to read Apple Ads creative", err)...)
		return
	}
	newState, diags := creativeModelFromClient(ctx, got)
	resp.Diagnostics.Append(diags...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *creativeResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var state, plan creativeModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(rejectImmutableCreativeChanges(state, plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// No mutable fields and no creative update API — keep current state.
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *creativeResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Apple Ads API v5 has no creative delete endpoint. Removing from state only.
}

func (r *creativeResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id, err := client.ParseCreativeID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import id", err.Error())
		return
	}
	got, err := r.client.GetCreative(ctx, id)
	if err != nil {
		resp.Diagnostics.Append(apiErrorDiagnostic("Unable to import Apple Ads creative", err)...)
		return
	}
	state, diags := creativeModelFromClient(ctx, got)
	resp.Diagnostics.Append(diags...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func creativeCreateFromPlan(plan creativeModel) (*client.CreativeCreate, diag.Diagnostics) {
	var diags diag.Diagnostics

	adamID, err := strconv.ParseInt(plan.AdamID.ValueString(), 10, 64)
	if err != nil {
		diags.AddAttributeError(path.Root("adam_id"), "Invalid adam_id", err.Error())
		return nil, diags
	}

	in := &client.CreativeCreate{
		AdamID: adamID,
		Name:   plan.Name.ValueString(),
		Type:   plan.Type.ValueString(),
	}
	if !plan.ProductPageID.IsNull() && !plan.ProductPageID.IsUnknown() {
		in.ProductPageID = plan.ProductPageID.ValueString()
	}
	if in.Type == client.CreativeTypeCustomProductPage && in.ProductPageID == "" {
		diags.AddAttributeError(
			path.Root("product_page_id"),
			"Missing product_page_id",
			"product_page_id is required when type is CUSTOM_PRODUCT_PAGE.",
		)
	}
	return in, diags
}

func rejectImmutableCreativeChanges(state, plan creativeModel) diag.Diagnostics {
	var diags diag.Diagnostics
	check := func(field, stateVal, planVal string) {
		if stateVal != planVal {
			diags.AddError(
				fmt.Sprintf(`Cannot change immutable creative field %q`, field),
				fmt.Sprintf(
					"Apple Ads API v5 does not allow updating creative %s in place (no creative update endpoint). "+
						"Automatically replacing this resource would discard creative identity. "+
						"Create a new appleads_creative explicitly instead.",
					state.ID.ValueString(),
				),
			)
		}
	}
	check("adam_id", state.AdamID.ValueString(), plan.AdamID.ValueString())
	check("name", state.Name.ValueString(), plan.Name.ValueString())
	check("type", state.Type.ValueString(), plan.Type.ValueString())
	check("product_page_id", nullString(state.ProductPageID), nullString(plan.ProductPageID))
	return diags
}

func nullString(v types.String) string {
	if v.IsNull() || v.IsUnknown() {
		return ""
	}
	return v.ValueString()
}

func creativeModelFromClient(ctx context.Context, c *client.Creative) (creativeModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	m := creativeModel{
		ID:               types.StringValue(strconv.FormatInt(c.ID, 10)),
		Name:             types.StringValue(c.Name),
		Type:             types.StringValue(c.Type),
		State:            types.StringValue(c.State),
		CreationTime:     types.StringValue(c.CreationTime),
		ModificationTime: types.StringValue(c.ModificationTime),
	}
	if c.OrgID != 0 {
		m.OrgID = types.StringValue(strconv.FormatInt(c.OrgID, 10))
	} else {
		m.OrgID = types.StringNull()
	}
	if c.AdamID != 0 {
		m.AdamID = types.StringValue(strconv.FormatInt(c.AdamID, 10))
	} else {
		m.AdamID = types.StringNull()
	}
	if c.ProductPageID != "" {
		m.ProductPageID = types.StringValue(c.ProductPageID)
	} else {
		m.ProductPageID = types.StringNull()
	}
	reasons, d := types.ListValueFrom(ctx, types.StringType, c.StateReasons)
	diags.Append(d...)
	if reasons.IsNull() {
		m.StateReasons = types.ListValueMust(types.StringType, nil)
	} else {
		m.StateReasons = reasons
	}
	return m, diags
}
