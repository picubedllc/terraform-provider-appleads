// Copyright (c) 2026 Pi Cubed LLC
// SPDX-License-Identifier: MIT

package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/diag"
)

const campaignDeletionBlockedSummary = "Campaign deletion is disabled by provider configuration"

const campaignDeletionBlockedDetail = `Apple Ads uses soft deletion: destroying a campaign permanently archives it.
Archived campaigns lose practical identity for reporting continuity and historical performance.

This provider refuses campaign destroy/archive operations unless you explicitly opt in:

  provider "apple_ads" {
    allow_campaign_deletion = true
  }

Set allow_campaign_deletion = true only when intentional archival is required.`

// CampaignDeletionBlockedDiagnostics returns the provider-level guardrail diagnostic
// used when Delete is attempted while allow_campaign_deletion is false.
// Enforcement in the apple_ads_campaign Delete lifecycle lands in Project 2.
func CampaignDeletionBlockedDiagnostics() diag.Diagnostics {
	var diags diag.Diagnostics
	diags.AddError(campaignDeletionBlockedSummary, campaignDeletionBlockedDetail)
	return diags
}

// EnsureCampaignDeletionAllowed returns diagnostics when deletion is not permitted.
func EnsureCampaignDeletionAllowed(data *ProviderData) diag.Diagnostics {
	if data != nil && data.AllowCampaignDeletion {
		return nil
	}
	return CampaignDeletionBlockedDiagnostics()
}
