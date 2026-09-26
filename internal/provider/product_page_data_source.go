// Copyright (c) 2026 Pi Cubed LLC
// SPDX-License-Identifier: MIT

package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/picubedllc/terraform-provider-appleads/internal/client"
)

var (
	_ datasource.DataSource              = &productPageDataSource{}
	_ datasource.DataSourceWithConfigure = &productPageDataSource{}
)

func NewProductPageDataSource() datasource.DataSource {
	return &productPageDataSource{}
}

type productPageDataSource struct {
	client *client.Client
}

type productPageDataSourceModel struct {
	ID               types.String `tfsdk:"id"`
	AdamID           types.String `tfsdk:"adam_id"`
	ProductPageID    types.String `tfsdk:"product_page_id"`
	Name             types.String `tfsdk:"name"`
	State            types.String `tfsdk:"state"`
	DeepLink         types.String `tfsdk:"deep_link"`
	CreationTime     types.String `tfsdk:"creation_time"`
	ModificationTime types.String `tfsdk:"modification_time"`
}

func (d *productPageDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_product_page"
}

func (d *productPageDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetches a single custom product page via Apple Ads " +
			"`GET /apps/{adamId}/product-pages/{productPageId}`.\n\n" +
			"Custom product pages are owned and edited in [App Store Connect](https://appstoreconnect.apple.com); " +
			"the Ads API is **read-only** here. Use `id` / `product_page_id` when creating creatives/ads.",
		Attributes: map[string]schema.Attribute{
			"adam_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Adam ID (App Store identifier) of the app that owns the product page.",
			},
			"product_page_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Product page UUID from App Store Connect (path segment / query `?ppid=` on the App Store URL).",
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Product page UUID (same value as `product_page_id`).",
			},
			"name": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Custom product page display name.",
			},
			"state": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "`VISIBLE` or `HIDDEN`.",
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
		},
	}
}

func (d *productPageDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *productPageDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config productPageDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if d.client == nil {
		resp.Diagnostics.AddError("Client not configured", "The provider client was not configured before reading appleads_product_page.")
		return
	}

	adamID, ok := parsePositiveAdamID(config.AdamID.ValueString(), path.Root("adam_id"), resp)
	if !ok {
		return
	}
	pageID := strings.TrimSpace(config.ProductPageID.ValueString())
	if pageID == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("product_page_id"),
			"Invalid product_page_id",
			"product_page_id is required",
		)
		return
	}

	page, err := d.client.GetProductPage(ctx, adamID, pageID)
	if err != nil {
		resp.Diagnostics.Append(apiErrorDiagnostic("Unable to read Apple Ads product page", err)...)
		return
	}

	mapped := mapProductPageModel(*page)
	state := productPageDataSourceModel{
		AdamID:           config.AdamID,
		ProductPageID:    types.StringValue(pageID),
		ID:               mapped.ID,
		Name:             mapped.Name,
		State:            mapped.State,
		DeepLink:         mapped.DeepLink,
		CreationTime:     mapped.CreationTime,
		ModificationTime: mapped.ModificationTime,
	}
	if state.ID.IsNull() || state.ID.ValueString() == "" {
		state.ID = types.StringValue(pageID)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
