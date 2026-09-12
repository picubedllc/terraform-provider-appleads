// Copyright (c) 2026 Pi Cubed LLC
// SPDX-License-Identifier: MIT

package provider

import (
	"context"
	"fmt"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/picubedllc/terraform-provider-appleads/internal/client"
)

const (
	envOrgID      = "APPLEADS_ORG_ID"
	envClientID   = "APPLEADS_CLIENT_ID"
	envTeamID     = "APPLEADS_TEAM_ID"
	envKeyID      = "APPLEADS_KEY_ID"
	envPrivateKey = "APPLEADS_PRIVATE_KEY"
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

// appleAdsProviderModel describes the provider configuration schema.
type appleAdsProviderModel struct {
	OrgID                 types.String `tfsdk:"org_id"`
	ClientID              types.String `tfsdk:"client_id"`
	TeamID                types.String `tfsdk:"team_id"`
	KeyID                 types.String `tfsdk:"key_id"`
	PrivateKey            types.String `tfsdk:"private_key"`
	AllowCampaignDeletion types.Bool   `tfsdk:"allow_campaign_deletion"`
}

// ProviderData is passed to resources and data sources via Configure.
type ProviderData struct {
	Client                *client.Client
	AllowCampaignDeletion bool
}

func (p *AppleAdsProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "apple_ads"
	resp.Version = p.version
}

func (p *AppleAdsProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manage Apple Ads configuration as code.",
		Attributes: map[string]schema.Attribute{
			"org_id": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Apple Ads organization ID. May also be set via `APPLEADS_ORG_ID`.",
			},
			"client_id": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Apple Ads API client ID. May also be set via `APPLEADS_CLIENT_ID`.",
			},
			"team_id": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Apple Ads API team ID. May also be set via `APPLEADS_TEAM_ID`.",
			},
			"key_id": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Apple Ads API key ID. May also be set via `APPLEADS_KEY_ID`.",
			},
			"private_key": schema.StringAttribute{
				Optional:            true,
				Sensitive:           true,
				MarkdownDescription: "PEM-encoded EC private key used to sign the OAuth client assertion. May also be set via `APPLEADS_PRIVATE_KEY`.",
			},
			"allow_campaign_deletion": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "When false (default), campaign destroy/archive operations are rejected. See the campaign deletion safety decision doc.",
			},
		},
	}
}

func (p *AppleAdsProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config appleAdsProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := stringValueOrEnv(config.OrgID, envOrgID)
	clientID := stringValueOrEnv(config.ClientID, envClientID)
	teamID := stringValueOrEnv(config.TeamID, envTeamID)
	keyID := stringValueOrEnv(config.KeyID, envKeyID)
	privateKey := stringValueOrEnv(config.PrivateKey, envPrivateKey)

	resp.Diagnostics.Append(requireCredential("org_id", orgID, envOrgID)...)
	resp.Diagnostics.Append(requireCredential("client_id", clientID, envClientID)...)
	resp.Diagnostics.Append(requireCredential("team_id", teamID, envTeamID)...)
	resp.Diagnostics.Append(requireCredential("key_id", keyID, envKeyID)...)
	resp.Diagnostics.Append(requireCredential("private_key", privateKey, envPrivateKey)...)
	if resp.Diagnostics.HasError() {
		return
	}

	allowDeletion := false
	if !config.AllowCampaignDeletion.IsNull() && !config.AllowCampaignDeletion.IsUnknown() {
		allowDeletion = config.AllowCampaignDeletion.ValueBool()
	}

	data, diags := buildProviderData(client.Credentials{
		OrgID:      orgID,
		ClientID:   clientID,
		TeamID:     teamID,
		KeyID:      keyID,
		PrivateKey: privateKey,
	}, allowDeletion)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.DataSourceData = data
	resp.ResourceData = data
}

func buildProviderData(creds client.Credentials, allowDeletion bool) (*ProviderData, diag.Diagnostics) {
	var diags diag.Diagnostics

	tokens, err := client.NewOAuthTokenSource(client.OAuthConfig{Credentials: creds})
	if err != nil {
		diags.AddError("Unable to configure Apple Ads authentication", err.Error())
		return nil, diags
	}

	httpClient := client.NewAuthenticatedHTTPClient(tokens, creds.OrgID, nil)
	httpClient = client.WithRetry(httpClient, client.RetryConfig{})

	apiClient, err := client.New(
		client.WithHTTPClient(httpClient),
		client.WithUserAgent(fmt.Sprintf("terraform-provider-appleads/%s", "dev")),
	)
	if err != nil {
		diags.AddError("Unable to configure Apple Ads client", err.Error())
		return nil, diags
	}

	return &ProviderData{
		Client:                apiClient,
		AllowCampaignDeletion: allowDeletion,
	}, diags
}

func stringValueOrEnv(attr types.String, envName string) string {
	if !attr.IsNull() && !attr.IsUnknown() && attr.ValueString() != "" {
		return attr.ValueString()
	}
	return os.Getenv(envName)
}

func requireCredential(attrName, value, envName string) diag.Diagnostics {
	var diags diag.Diagnostics
	if value == "" {
		diags.AddAttributeError(
			path.Root(attrName),
			"Missing Apple Ads credential",
			fmt.Sprintf("The provider attribute %q must be set in the provider configuration or via the %s environment variable.", attrName, envName),
		)
	}
	return diags
}

func (p *AppleAdsProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewCampaignResource,
		NewAdGroupResource,
		NewKeywordResource,
		NewNegativeKeywordResource,
	}
}

func (p *AppleAdsProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewAppDataSource,
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &AppleAdsProvider{
			version: version,
		}
	}
}
