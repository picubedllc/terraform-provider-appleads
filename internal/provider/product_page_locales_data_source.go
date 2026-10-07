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
	_ datasource.DataSource              = &productPageLocalesDataSource{}
	_ datasource.DataSourceWithConfigure = &productPageLocalesDataSource{}
)

func NewProductPageLocalesDataSource() datasource.DataSource {
	return &productPageLocalesDataSource{}
}

type productPageLocalesDataSource struct {
	client *client.Client
}

type productPageLocalesDataSourceModel struct {
	ID            types.String             `tfsdk:"id"`
	AdamID        types.String             `tfsdk:"adam_id"`
	ProductPageID types.String             `tfsdk:"product_page_id"`
	DeviceClasses types.List               `tfsdk:"device_classes"`
	LanguageCodes types.List               `tfsdk:"language_codes"`
	Languages     types.List               `tfsdk:"languages"`
	Expand        types.Bool               `tfsdk:"expand"`
	Locales       []productPageLocaleModel `tfsdk:"locales"`
}

type productPageLocaleModel struct {
	AdamID           types.String `tfsdk:"adam_id"`
	AppName          types.String `tfsdk:"app_name"`
	DeviceClasses    types.String `tfsdk:"device_classes"`
	Language         types.String `tfsdk:"language"`
	LanguageCode     types.String `tfsdk:"language_code"`
	ProductPageID    types.String `tfsdk:"product_page_id"`
	PromotionalText  types.String `tfsdk:"promotional_text"`
	ShortDescription types.String `tfsdk:"short_description"`
	SubTitle         types.String `tfsdk:"sub_title"`
}

func (d *productPageLocalesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_product_page_locales"
}

func (d *productPageLocalesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists locale details for a custom product page via Apple Ads " +
			"`GET /apps/{adamId}/product-pages/{productPageId}/locale-details`.\n\n" +
			"Locale metadata is owned in [App Store Connect](https://appstoreconnect.apple.com); " +
			"the Ads API is **read-only** here.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Composite id of the lookup parameters.",
			},
			"adam_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Adam ID of the app that owns the product page.",
			},
			"product_page_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Product page UUID from App Store Connect.",
			},
			"device_classes": schema.ListAttribute{
				Optional:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Filter by device class. Allowed values: `IPHONE`, `IPAD`.",
				Validators: []validator.List{
					listvalidator.ValueStringsAre(stringvalidator.OneOf(
						client.DeviceClassIPhone,
						client.DeviceClassIPad,
					)),
				},
			},
			"language_codes": schema.ListAttribute{
				Optional:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Filter by ISO language-country codes, e.g. `en-US`.",
			},
			"languages": schema.ListAttribute{
				Optional:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Filter by language display names, e.g. `English`.",
			},
			"expand": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "When true, ask Apple to expand nested asset details (`expand=true`).",
			},
			"locales": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Matching locale details (may be empty).",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"adam_id": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Adam ID on the locale record.",
						},
						"app_name": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Localized app name.",
						},
						"device_classes": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Device class assigned to the locale (`IPHONE` or `IPAD`).",
						},
						"language": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Language display name.",
						},
						"language_code": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Language code such as `en-US`.",
						},
						"product_page_id": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Product page UUID.",
						},
						"promotional_text": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Promotional text on the product page.",
						},
						"short_description": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Short description on the product page.",
						},
						"sub_title": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Subtitle on the product page.",
						},
					},
				},
			},
		},
	}
}

func (d *productPageLocalesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *productPageLocalesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config productPageLocalesDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if d.client == nil {
		resp.Diagnostics.AddError("Client not configured", "The provider client was not configured before reading appleads_product_page_locales.")
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

	params := client.ProductPageLocaleListParams{}
	if !config.DeviceClasses.IsNull() && !config.DeviceClasses.IsUnknown() {
		var v []string
		resp.Diagnostics.Append(config.DeviceClasses.ElementsAs(ctx, &v, false)...)
		params.DeviceClasses = v
	}
	if !config.LanguageCodes.IsNull() && !config.LanguageCodes.IsUnknown() {
		var v []string
		resp.Diagnostics.Append(config.LanguageCodes.ElementsAs(ctx, &v, false)...)
		params.LanguageCodes = v
	}
	if !config.Languages.IsNull() && !config.Languages.IsUnknown() {
		var v []string
		resp.Diagnostics.Append(config.Languages.ElementsAs(ctx, &v, false)...)
		params.Languages = v
	}
	if resp.Diagnostics.HasError() {
		return
	}
	if !config.Expand.IsNull() && !config.Expand.IsUnknown() {
		expand := config.Expand.ValueBool()
		params.Expand = &expand
	}

	locales, err := d.client.ListProductPageLocales(ctx, adamID, pageID, params)
	if err != nil {
		resp.Diagnostics.Append(apiErrorDiagnostic("Unable to list Apple Ads product page locales", err)...)
		return
	}

	state := productPageLocalesDataSourceModel{
		AdamID:        config.AdamID,
		ProductPageID: types.StringValue(pageID),
		DeviceClasses: config.DeviceClasses,
		LanguageCodes: config.LanguageCodes,
		Languages:     config.Languages,
		Expand:        config.Expand,
		Locales:       make([]productPageLocaleModel, 0, len(locales)),
	}
	idParts := []string{config.AdamID.ValueString(), pageID}
	if len(params.DeviceClasses) > 0 {
		idParts = append(idParts, "device="+strings.Join(params.DeviceClasses, ","))
	}
	if len(params.LanguageCodes) > 0 {
		idParts = append(idParts, "codes="+strings.Join(params.LanguageCodes, ","))
	}
	if len(params.Languages) > 0 {
		idParts = append(idParts, "langs="+strings.Join(params.Languages, ","))
	}
	if params.Expand != nil {
		idParts = append(idParts, "expand="+strconv.FormatBool(*params.Expand))
	}
	state.ID = types.StringValue(strings.Join(idParts, "/"))

	for _, loc := range locales {
		m := productPageLocaleModel{
			AppName:          stringOrNull(loc.AppName),
			DeviceClasses:    stringOrNull(loc.DeviceClasses),
			Language:         stringOrNull(loc.Language),
			LanguageCode:     stringOrNull(loc.LanguageCode),
			ProductPageID:    stringOrNull(loc.ProductPageID),
			PromotionalText:  stringOrNull(loc.PromotionalText),
			ShortDescription: stringOrNull(loc.ShortDescription),
			SubTitle:         stringOrNull(loc.SubTitle),
		}
		if loc.AdamID != 0 {
			m.AdamID = types.StringValue(strconv.FormatInt(loc.AdamID, 10))
		} else {
			m.AdamID = types.StringNull()
		}
		state.Locales = append(state.Locales, m)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
