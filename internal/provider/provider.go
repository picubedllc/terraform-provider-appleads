// Copyright (c) 2026 Pi Cubed LLC
// SPDX-License-Identifier: MIT

package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// Ensure AppleAdsProvider satisfies various provider interfaces.
var _ provider.Provider = &AppleAdsProvider{}

// AppleAdsProvider defines the provider implementation.
type AppleAdsProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

func (p *AppleAdsProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "appleads"
	resp.Version = p.version
}

func (p *AppleAdsProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	// Provider configuration attributes are added in a later ticket.
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manage Apple Ads configuration as code.",
	}
}

func (p *AppleAdsProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	// Client wiring is added in a later ticket.
}

func (p *AppleAdsProvider) Resources(ctx context.Context) []func() resource.Resource {
	return nil
}

func (p *AppleAdsProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return nil
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &AppleAdsProvider{
			version: version,
		}
	}
}
