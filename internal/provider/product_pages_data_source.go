// Copyright (c) 2026 Pi Cubed LLC
// SPDX-License-Identifier: MIT

package provider

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/picubedllc/terraform-provider-appleads/internal/client"
)

var (
	_ datasource.DataSource              = &productPagesDataSource{}
	_ datasource.DataSourceWithConfigure = &productPagesDataSource{}
)

func NewProductPagesDataSource() datasource.DataSource {
	return &productPagesDataSource{}
}

type productPagesDataSource struct {
	client *client.Client
}

type productPagesDataSourceModel struct {
	ID           types.String       `tfsdk:"id"`
	AdamID       types.String       `tfsdk:"adam_id"`
	Name         types.String       `tfsdk:"name"`
	States       types.List         `tfsdk:"states"`
	ProductPages []productPageModel `tfsdk:"product_pages"`
}

type productPageModel struct {
	ID               types.String `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	State            types.String `tfsdk:"state"`
	AdamID           types.String `tfsdk:"adam_id"`
	DeepLink         types.String `tfsdk:"deep_link"`
	CreationTime     types.String `tfsdk:"creation_time"`
	ModificationTime types.String `tfsdk:"modification_time"`
}

func (d *productPagesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_product_pages"
}

func (d *productPagesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists custom product pages for an App Store app via Apple Ads " +
			"`GET /apps/{adamId}/product-pages`.\n\n" +
			"Custom product pages are owned and edited in [App Store Connect](https://appstoreconnect.apple.com); " +
			"the Ads API is **read-only** here. Use returned `id` values when creating creatives/ads.\n\n" +
			"Optional `name` and `states` (`VISIBLE`, `HIDDEN`) filter the list. An empty result is valid " +
			"(no matching pages), not an error.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Composite id of the list parameters (`adam_id` plus optional filters).",
			},
			"adam_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Adam ID (App Store identifier) of the app whose product pages to list.",
			},
			"name": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Filter by custom product page name as configured in App Store Connect.",
			},
			"states": schema.ListAttribute{
				Optional:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Filter by visibility state. Allowed values: `VISIBLE`, `HIDDEN`.",
				Validators: []validator.List{
					listvalidator.ValueStringsAre(stringvalidator.OneOf(
						client.ProductPageStateVisible,
						client.ProductPageStateHidden,
					)),
				},
			},
			"product_pages": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Matching product pages (may be empty).",
				NestedObject: schema.NestedAttributeObject{
					Attributes: productPageSchemaAttributes(),
				},
			},
		},
	}
}

func productPageSchemaAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "Product page UUID from App Store Connect (used when creating creatives).",
		},
		"name": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "Custom product page display name.",
		},
		"state": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "`VISIBLE` or `HIDDEN`.",
		},
		"adam_id": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "Adam ID of the app owning the page.",
		},
		"deep_link": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "Deep link from App Store Connect product page metadata, when set.",
		},
		"creation_time": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "Creation timestamp from Apple Ads.",
		},
		"modification_time": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "Last modification timestamp from Apple Ads.",
		},
	}
}

func (d *productPagesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	data, ok := req.ProviderData.(*ProviderData)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected data source configure type",
			fmt.Sprintf("Expected *ProviderData, got: %T", req.ProviderData),
		)
		return
	}
	d.client = data.Client
}

func (d *productPagesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config productPagesDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if d.client == nil {
		resp.Diagnostics.AddError("Client not configured", "The provider client was not configured before reading appleads_product_pages.")
		return
	}

	adamID, ok := parsePositiveAdamID(config.AdamID.ValueString(), path.Root("adam_id"), resp)
	if !ok {
		return
	}

	params := client.ProductPageListParams{
		Name: config.Name.ValueString(),
	}
	if !config.States.IsNull() && !config.States.IsUnknown() {
		var states []string
		resp.Diagnostics.Append(config.States.ElementsAs(ctx, &states, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		params.States = states
	}

	pages, err := d.client.ListProductPages(ctx, adamID, params)
	if err != nil {
		resp.Diagnostics.Append(apiErrorDiagnostic("Unable to list Apple Ads product pages", err)...)
		return
	}

	state := productPagesDataSourceModel{
		AdamID:       config.AdamID,
		Name:         config.Name,
		States:       config.States,
		ProductPages: make([]productPageModel, 0, len(pages)),
	}
	state.ID = types.StringValue(productPagesCompositeID(config.AdamID.ValueString(), config.Name.ValueString(), params.States))
	for _, page := range pages {
		state.ProductPages = append(state.ProductPages, mapProductPageModel(page))
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func productPagesCompositeID(adamID, name string, states []string) string {
	parts := []string{adamID}
	if name != "" {
		parts = append(parts, "name="+name)
	}
	if len(states) > 0 {
		parts = append(parts, "states="+strings.Join(states, ","))
	}
	return strings.Join(parts, "/")
}

func mapProductPageModel(page client.ProductPage) productPageModel {
	m := productPageModel{
		ID:    types.StringValue(page.ID),
		Name:  stringOrNull(page.Name),
		State: stringOrNull(page.State),
	}
	if page.AdamID != 0 {
		m.AdamID = types.StringValue(strconv.FormatInt(page.AdamID, 10))
	} else {
		m.AdamID = types.StringNull()
	}
	m.DeepLink = stringOrNull(page.DeepLink)
	m.CreationTime = stringOrNull(page.CreationTime)
	m.ModificationTime = stringOrNull(page.ModificationTime)
	return m
}

func stringOrNull(v string) types.String {
	if v == "" {
		return types.StringNull()
	}
	return types.StringValue(v)
}

func parsePositiveAdamID(raw string, attr path.Path, resp *datasource.ReadResponse) (int64, bool) {
	raw = strings.TrimSpace(raw)
	adamID, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || adamID <= 0 {
		resp.Diagnostics.AddAttributeError(
			attr,
			"Invalid adam_id",
			fmt.Sprintf("adam_id must be a positive numeric Adam ID; got %q", raw),
		)
		return 0, false
	}
	return adamID, true
}
