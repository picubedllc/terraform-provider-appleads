// Copyright (c) 2026 Pi Cubed LLC
// SPDX-License-Identifier: MIT

package provider

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/datasourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/picubedllc/terraform-provider-appleads/internal/client"
)

var (
	_ datasource.DataSource                     = &geolocationsDataSource{}
	_ datasource.DataSourceWithConfigure        = &geolocationsDataSource{}
	_ datasource.DataSourceWithConfigValidators = &geolocationsDataSource{}
)

func NewGeolocationsDataSource() datasource.DataSource {
	return &geolocationsDataSource{}
}

type geolocationsDataSource struct {
	client *client.Client
}

type geolocationsDataSourceModel struct {
	ID          types.String       `tfsdk:"id"`
	Query       types.String       `tfsdk:"query"`
	Entity      types.String       `tfsdk:"entity"`
	CountryCode types.String       `tfsdk:"country_code"`
	GeoID       types.String       `tfsdk:"geo_id"`
	Limit       types.Int64        `tfsdk:"limit"`
	Locations   []geolocationModel `tfsdk:"locations"`
}

type geolocationModel struct {
	ID          types.String `tfsdk:"id"`
	Entity      types.String `tfsdk:"entity"`
	DisplayName types.String `tfsdk:"display_name"`
}

func (d *geolocationsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_geolocations"
}

func (d *geolocationsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Looks up Apple Ads targetable locations via Search for Geolocations (`GET /search/geo`).\n\n" +
			"Use the returned `id` values in `appleads_ad_group.targeting_dimensions` " +
			"(`country`, `admin_area`, `locality`). Locality IDs look like `US|NY|New York`; " +
			"admin areas like `US|NY`; countries are ISO alpha-2.\n\n" +
			"At least one of `query` or `geo_id` is required. `query` is prefix-matched and must be at least three characters. " +
			"Geo targeting only works on single-country campaigns.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Composite id of the search parameters.",
			},
			"query": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Prefix search against location display names (minimum three characters). Example: `New York`.",
				Validators:          []validator.String{stringvalidator.LengthAtLeast(3)},
			},
			"entity": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Restrict results to `Country`, `AdminArea`, or `Locality`.",
				Validators:          []validator.String{stringvalidator.OneOf(client.GeoEntityCountry, client.GeoEntityAdminArea, client.GeoEntityLocality)},
			},
			"country_code": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "ISO alpha-2 country filter (Apple `countrycode` query parameter), e.g. `US`.",
				Validators:          []validator.String{stringvalidator.LengthBetween(2, 2)},
			},
			"geo_id": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Look up a specific location id such as `US|NY|New York`.",
			},
			"limit": schema.Int64Attribute{
				Optional:            true,
				MarkdownDescription: "Maximum number of locations to return. Defaults to 20.",
				Validators:          []validator.Int64{int64validator.Between(1, 1000)},
			},
			"locations": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Matching locations. Use `id` as a targeting criteria string.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Criteria id (`US`, `US|NY`, or `US|NY|New York`).",
						},
						"entity": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "`Country`, `AdminArea`, or `Locality`.",
						},
						"display_name": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Localized display name, e.g. `New York, New York, United States`.",
						},
					},
				},
			},
		},
	}
}

func (d *geolocationsDataSource) ConfigValidators(ctx context.Context) []datasource.ConfigValidator {
	return []datasource.ConfigValidator{
		datasourcevalidator.AtLeastOneOf(
			path.MatchRoot("query"),
			path.MatchRoot("geo_id"),
		),
	}
}

func (d *geolocationsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *geolocationsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config geolocationsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if d.client == nil {
		resp.Diagnostics.AddError("Client not configured", "The provider client was not configured before reading appleads_geolocations.")
		return
	}

	params := client.GeoSearchParams{
		Query:       config.Query.ValueString(),
		Entity:      config.Entity.ValueString(),
		CountryCode: config.CountryCode.ValueString(),
		ID:          config.GeoID.ValueString(),
	}
	if !config.Limit.IsNull() && !config.Limit.IsUnknown() {
		params.Limit = int(config.Limit.ValueInt64())
	}

	found, err := d.client.SearchGeoLocations(ctx, params)
	if err != nil {
		resp.Diagnostics.Append(apiErrorDiagnostic("Unable to search Apple Ads geolocations", err)...)
		return
	}

	limit := params.Limit
	if limit <= 0 {
		limit = 20
	}
	state := geolocationsDataSourceModel{
		ID: types.StringValue(strings.Join([]string{
			config.Query.ValueString(),
			config.Entity.ValueString(),
			config.CountryCode.ValueString(),
			config.GeoID.ValueString(),
			strconv.Itoa(limit),
		}, "/")),
		Query:       config.Query,
		Entity:      config.Entity,
		CountryCode: config.CountryCode,
		GeoID:       config.GeoID,
		Limit:       config.Limit,
		Locations:   make([]geolocationModel, 0, len(found)),
	}
	for _, loc := range found {
		state.Locations = append(state.Locations, geolocationModel{
			ID:          types.StringValue(loc.ID),
			Entity:      types.StringValue(loc.Entity),
			DisplayName: types.StringValue(loc.DisplayName),
		})
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
