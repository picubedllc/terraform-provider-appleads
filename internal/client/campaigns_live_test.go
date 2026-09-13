// Copyright (c) 2026 Pi Cubed LLC
// SPDX-License-Identifier: MIT

package client_test

import (
	"testing"

	"github.com/picubedllc/terraform-provider-appleads/internal/acctest"
)

// Read-only live campaign list. Allowlisting specific campaigns is PI-31.
func TestLiveCampaignsList(t *testing.T) {
	c, _ := acctest.LiveClient(t)
	page := acctest.RequireCampaignsPage(t, c)
	t.Logf("GET /campaigns ok (items=%d total=%d)", len(page.Data), page.TotalCount)
}
