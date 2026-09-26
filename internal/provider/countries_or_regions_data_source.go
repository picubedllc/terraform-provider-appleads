// Copyright (c) 2026 Pi Cubed LLC
// SPDX-License-Identifier: MIT

package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/picubedllc/terraform-provider-appleads/internal/client"
)

var (
	_ datasource.DataSource              = &countriesOrRegionsDataSource{}
	_ datasource.DataSourceWithConfigure = &countriesOrRegionsDataSource{}
)

func NewCountriesOrRegionsDataSource() datasource.DataSource {
	return &countriesOrRegionsDataSource{}
}

type countriesOrRegionsDataSource struct {
	client *client.Client
}

type countriesOrRegionsDataSourceModel struct {
	ID                 types.String           `tfsdk:"id"`
	CountriesOrRegions types.List             `tfsdk:"countries_or_regions_filter"`
	Results            []countryOrRegionModel `tfsdk:"countries_or_regions"`
}

type countryOrRegionModel struct {
	CountryOrRegion    types.String      `tfsdk:"country_or_region"`
	DefaultLanguages   []localeInfoModel `tfsdk:"default_languages"`
	SupportedLanguages []localeInfoModel `tfsdk:"supported_languages"`
}

type localeInfoModel struct {
	Language     types.String `tfsdk:"language"`
	LanguageCode types.String `tfsdk:"language_code"`
}

func (d *countriesOrRegionsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_countries_or_regions"
}

func (d *countriesOrRegionsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	localeAttrs := map[string]schema.Attribute{
		"language": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "Language display name.",
		},
		"language_code": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "Language code such as `en-US`.",
		},
	}

	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists supported Apple Ads countries or regions via " +
			"`GET /countries-or-regions`.\n\n" +
			"Useful when validating campaign `countries_or_regions` or product-page locale coverage. " +
			"This endpoint is **read-only**.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Composite id of the filter parameters (or `all`).",
			},
			"countries_or_regions_filter": schema.ListAttribute{
				Optional:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Optional ISO country/region codes to filter (Apple `countriesOrRegions` query parameter).",
			},
			"countries_or_regions": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Matching countries or regions (may be empty).",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"country_or_region": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "ISO country or region code, e.g. `US`.",
						},
						"default_languages": schema.ListNestedAttribute{
							Computed:            true,
							MarkdownDescription: "Default languages for the country or region.",
							NestedObject: schema.NestedAttributeObject{
								Attributes: localeAttrs,
							},
						},
						"supported_languages": schema.ListNestedAttribute{
							Computed:            true,
							MarkdownDescription: "Supported languages for the country or region.",
							NestedObject: schema.NestedAttributeObject{
								Attributes: localeAttrs,
							},
						},
					},
				},
			},
		},
	}
}

func (d *countriesOrRegionsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *countriesOrRegionsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config countriesOrRegionsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if d.client == nil {
		resp.Diagnostics.AddError("Client not configured", "The provider client was not configured before reading appleads_countries_or_regions.")
		return
	}

	var filter []string
	if !config.CountriesOrRegions.IsNull() && !config.CountriesOrRegions.IsUnknown() {
		resp.Diagnostics.Append(config.CountriesOrRegions.ElementsAs(ctx, &filter, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	found, err := d.client.ListCountriesOrRegions(ctx, filter)
	if err != nil {
		resp.Diagnostics.Append(apiErrorDiagnostic("Unable to list Apple Ads countries or regions", err)...)
		return
	}

	state := countriesOrRegionsDataSourceModel{
		CountriesOrRegions: config.CountriesOrRegions,
		Results:            make([]countryOrRegionModel, 0, len(found)),
	}
	if len(filter) == 0 {
		state.ID = types.StringValue("all")
	} else {
		state.ID = types.StringValue(strings.Join(filter, ","))
	}

	for _, row := range found {
		state.Results = append(state.Results, countryOrRegionModel{
			CountryOrRegion:    types.StringValue(row.CountryOrRegion),
			DefaultLanguages:   mapLocaleInfos(row.DefaultLanguages),
			SupportedLanguages: mapLocaleInfos(row.SupportedLanguages),
		})
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func mapLocaleInfos(in []client.LocaleInfo) []localeInfoModel {
	out := make([]localeInfoModel, 0, len(in))
	for _, loc := range in {
		out = append(out, localeInfoModel{
			Language:     stringOrNull(loc.Language),
			LanguageCode: stringOrNull(loc.LanguageCode),
		})
	}
	return out
}
