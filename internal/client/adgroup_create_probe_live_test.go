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

func TestLiveAdGroupCreateProbe(t *testing.T) {
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

	campaign, err := c.CreateCampaign(ctx, &client.CampaignCreate{
		Name:               fmt.Sprintf("tf-acc-adgroup-probe-%d", time.Now().UnixNano()),
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
	t.Cleanup(func() {
		_ = c.DeleteCampaign(context.Background(), campaign.ID)
	})

	group, err := c.CreateAdGroup(ctx, campaign.ID, &client.AdGroupCreate{
		Name:             "tf-acc-adgroup-probe",
		Status:           "PAUSED",
		DefaultBidAmount: &client.Money{Amount: "1.00", Currency: "USD"},
	})
	if err != nil {
		t.Fatalf("CreateAdGroup: %v", err)
	}
	t.Logf("created ad group id=%d campaign=%d pricingModel=%s", group.ID, campaign.ID, group.PricingModel)
	if err := c.DeleteAdGroup(ctx, campaign.ID, group.ID); err != nil {
		t.Fatalf("DeleteAdGroup(%d): %v", group.ID, err)
	}
}
