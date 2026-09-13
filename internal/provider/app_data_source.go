// Copyright (c) 2026 Pi Cubed LLC
// SPDX-License-Identifier: MIT

package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework-validators/datasourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/picubedllc/terraform-provider-appleads/internal/client"
)

var (
	_ datasource.DataSource                     = &appDataSource{}
	_ datasource.DataSourceWithConfigure        = &appDataSource{}
	_ datasource.DataSourceWithConfigValidators = &appDataSource{}
)

func NewAppDataSource() datasource.DataSource {
	return &appDataSource{}
}

type appDataSource struct {
	client *client.Client
}

type appDataSourceModel struct {
	ID              types.String `tfsdk:"id"`
	Name            types.String `tfsdk:"name"`
	AdamID          types.String `tfsdk:"adam_id"`
	DeveloperName   types.String `tfsdk:"developer_name"`
	CountryOrRegion types.String `tfsdk:"country_or_region"`
}

func (d *appDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_app"
}

func (d *appDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Resolves an App Store app to the Adam ID and related identifiers required when creating Apple Ads campaigns.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "App Store display name to look up. Exactly one of `name` or `id` must be set.",
			},
			"id": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Adam ID (App Store identifier) of the app. Exactly one of `name` or `id` must be set as input; always populated in the result.",
			},
			"adam_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Adam ID of the resolved app (same value as `id`).",
			},
			"developer_name": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Developer name returned by Apple Ads for the app.",
			},
			"country_or_region": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Country or region associated with the app listing when returned by Apple Ads.",
			},
		},
	}
}

func (d *appDataSource) ConfigValidators(ctx context.Context) []datasource.ConfigValidator {
	return []datasource.ConfigValidator{
		datasourcevalidator.ExactlyOneOf(
			path.MatchRoot("name"),
			path.MatchRoot("id"),
		),
	}
}

func (d *appDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *appDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config appDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if d.client == nil {
		resp.Diagnostics.AddError("Client not configured", "The provider client was not configured before reading appleads_app.")
		return
	}

	var (
		app *client.App
		err error
	)

	switch {
	case !config.ID.IsNull() && !config.ID.IsUnknown() && config.ID.ValueString() != "":
		adamID, parseErr := strconv.ParseInt(config.ID.ValueString(), 10, 64)
		if parseErr != nil {
			resp.Diagnostics.AddAttributeError(
				path.Root("id"),
				"Invalid app id",
				fmt.Sprintf("id must be a numeric Adam ID; got %q: %s", config.ID.ValueString(), parseErr),
			)
			return
		}
		app, err = d.client.GetApp(ctx, adamID)
	default:
		app, err = d.client.FindAppByName(ctx, config.Name.ValueString())
	}

	if err != nil {
		resp.Diagnostics.AddError("Unable to resolve Apple Ads app", err.Error())
		return
	}

	adamID := strconv.FormatInt(app.AdamID, 10)
	state := appDataSourceModel{
		ID:            types.StringValue(adamID),
		AdamID:        types.StringValue(adamID),
		Name:          types.StringValue(app.AppName),
		DeveloperName: types.StringValue(app.DeveloperName),
	}
	if app.CountryOrRegion != "" {
		state.CountryOrRegion = types.StringValue(app.CountryOrRegion)
	} else {
		state.CountryOrRegion = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
