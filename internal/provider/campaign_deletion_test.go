// Copyright (c) 2026 Pi Cubed LLC
// SPDX-License-Identifier: MIT

package provider

import (
	"strings"
	"testing"
)

func TestEnsureCampaignDeletionAllowed(t *testing.T) {
	t.Parallel()

	t.Run("blocked by default", func(t *testing.T) {
		t.Parallel()
		diags := EnsureCampaignDeletionAllowed(nil)
		if !diags.HasError() {
			t.Fatal("expected error")
		}
		if diags[0].Summary() != campaignDeletionBlockedSummary {
			t.Fatalf("summary = %q", diags[0].Summary())
		}
		if !strings.Contains(diags[0].Detail(), "allow_campaign_deletion = true") {
			t.Fatalf("detail = %q", diags[0].Detail())
		}
	})

	t.Run("blocked when flag false", func(t *testing.T) {
		t.Parallel()
		diags := EnsureCampaignDeletionAllowed(&ProviderData{AllowCampaignDeletion: false})
		if !diags.HasError() {
			t.Fatal("expected error")
		}
	})

	t.Run("allowed when flag true", func(t *testing.T) {
		t.Parallel()
		diags := EnsureCampaignDeletionAllowed(&ProviderData{AllowCampaignDeletion: true})
		if diags.HasError() {
			t.Fatalf("unexpected diagnostics: %#v", diags)
		}
	})
}
