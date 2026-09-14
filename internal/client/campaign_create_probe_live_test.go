// Copyright (c) 2026 Pi Cubed LLC
// SPDX-License-Identifier: MIT

package client_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/picubedllc/terraform-provider-appleads/internal/acctest"
	"github.com/picubedllc/terraform-provider-appleads/internal/client"
)

// TestLiveCampaignCreateProbe creates one paused campaign over REST (via the
// provider client) and archives it. This is the live check that campaign
// create works.
func TestLiveCampaignCreateProbe(t *testing.T) {
	c, creds := acctest.LiveClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	acls := acctest.RequireUserACLs(t, c)
	acctest.RequireOrgAccess(t, acls, creds.OrgID)

	adamID := int64(0)
	if raw := os.Getenv("APPLEADS_TEST_ADAM_ID"); raw != "" {
		if _, err := fmt.Sscan(raw, &adamID); err != nil {
			t.Fatalf("APPLEADS_TEST_ADAM_ID: %v", err)
		}
	} else {
		page := acctest.RequireCampaignsPage(t, c)
		if len(page.Data) == 0 {
			t.Skip("no existing campaigns to copy adamId from; set APPLEADS_TEST_ADAM_ID")
		}
		adamID = page.Data[0].AdamID
	}

	name := fmt.Sprintf("tf-acc-probe-%d", time.Now().UnixNano())
	created, err := c.CreateCampaign(ctx, &client.CampaignCreate{
		Name:               name,
		AdamID:             adamID,
		CountriesOrRegions: []string{"US"},
		AdChannelType:      "SEARCH",
		SupplySources:      []string{"APPSTORE_SEARCH_RESULTS"},
		BillingEvent:       "TAPS",
		BiddingStrategy:    "MANUAL_CPT",
		Status:             "PAUSED",
		DailyBudgetAmount:  &client.Money{Amount: "1.00", Currency: "USD"},
	})
	if err != nil {
		t.Fatalf("CreateCampaign: %v", err)
	}
	t.Logf("created campaign id=%d name=%s", created.ID, created.Name)
	if err := c.DeleteCampaign(ctx, created.ID); err != nil {
		t.Fatalf("DeleteCampaign(%d): %v", created.ID, err)
	}
}
