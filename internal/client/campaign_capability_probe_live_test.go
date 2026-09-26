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

// Live probes for Phase 1 campaign capability gaps. Findings should be recorded
// on GitHub #41 (no secrets, org IDs, or private identifiers in comments).
//
// Requires APPLEADS_LIVE_TEST=1 and credentials. Prefer APPLEADS_TEST_ADAM_ID.

func probeAdamID(t *testing.T, c *client.Client) int64 {
	t.Helper()
	if raw := os.Getenv("APPLEADS_TEST_ADAM_ID"); raw != "" {
		var adamID int64
		if _, err := fmt.Sscan(raw, &adamID); err != nil {
			t.Fatalf("APPLEADS_TEST_ADAM_ID: %v", err)
		}
		return adamID
	}
	page := acctest.RequireCampaignsPage(t, c)
	if len(page.Data) == 0 {
		t.Skip("no existing campaigns to copy adamId from; set APPLEADS_TEST_ADAM_ID")
	}
	return page.Data[0].AdamID
}

func archiveCampaign(t *testing.T, c *client.Client, id int64) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	if err := c.DeleteCampaign(ctx, id); err != nil {
		t.Errorf("cleanup DeleteCampaign(%d): %v", id, err)
	}
}

// TestLiveCampaignBudgetAmountUpdateProbe checks whether total budgetAmount can
// be changed after create. Apple docs say create-only; this records the live
// outcome for Terraform mutability classification (#46).
func TestLiveCampaignBudgetAmountUpdateProbe(t *testing.T) {
	c, creds := acctest.LiveClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	acls := acctest.RequireUserACLs(t, c)
	acctest.RequireOrgAccess(t, acls, creds.OrgID)
	adamID := probeAdamID(t, c)

	name := fmt.Sprintf("tf-probe-budget-%d", time.Now().UnixNano())
	created, err := c.CreateCampaign(ctx, &client.CampaignCreate{
		Name:               name,
		AdamID:             adamID,
		CountriesOrRegions: []string{"US"},
		AdChannelType:      client.AdChannelTypeSearch,
		SupplySources:      []string{client.SupplySourceSearchResults},
		BillingEvent:       client.BillingEventTaps,
		BiddingStrategy:    client.BiddingStrategyManualCPT,
		Status:             "PAUSED",
		BudgetAmount:       &client.Money{Amount: "100.00", Currency: "USD"},
		DailyBudgetAmount:  &client.Money{Amount: "1.00", Currency: "USD"},
	})
	if err != nil {
		t.Logf("CreateCampaign with budgetAmount failed (note for #41): %v", err)
		t.Skip("could not create campaign with budgetAmount; cannot probe update")
	}
	t.Cleanup(func() { archiveCampaign(t, c, created.ID) })

	t.Logf("created campaign id=%d budgetAmount=%v daily=%v paymentModel=%q",
		created.ID, moneyAmt(created.BudgetAmount), moneyAmt(created.DailyBudgetAmount), created.PaymentModel)

	// Raw probe: BudgetAmount is not on CampaignUpdate (create-only). Use DoJSON.
	var env client.Response[client.Campaign]
	err = c.DoJSON(ctx, "PUT", fmt.Sprintf("campaigns/%d", created.ID), map[string]any{
		"campaign": map[string]any{
			"budgetAmount": map[string]string{"amount": "150.00", "currency": "USD"},
		},
	}, &env)
	if err != nil {
		t.Logf("FINDING budget_amount update: REJECTED — treat as create-only in Terraform. err=%v", err)
		return
	}
	t.Logf("FINDING budget_amount update: ACCEPTED — amount now %v (unexpected vs Apple create-only docs)",
		moneyAmt(env.Data.BudgetAmount))
}

// TestLiveCampaignMaxConversionsProbe creates a Maximize Conversions campaign
// and checks for an Apple auto-created Automated Ad Group (#41 / #45).
func TestLiveCampaignMaxConversionsProbe(t *testing.T) {
	c, creds := acctest.LiveClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	acls := acctest.RequireUserACLs(t, c)
	acctest.RequireOrgAccess(t, acls, creds.OrgID)
	adamID := probeAdamID(t, c)

	name := fmt.Sprintf("tf-probe-maxconv-%d", time.Now().UnixNano())
	created, err := c.CreateCampaign(ctx, &client.CampaignCreate{
		Name:               name,
		AdamID:             adamID,
		CountriesOrRegions: []string{"US"},
		AdChannelType:      client.AdChannelTypeSearch,
		SupplySources:      []string{client.SupplySourceSearchResults},
		BillingEvent:       client.BillingEventTaps,
		BiddingStrategy:    client.BiddingStrategyMaxConversions,
		TargetCpa:          &client.Money{Amount: "10.00", Currency: "USD"},
		Status:             "PAUSED",
		DailyBudgetAmount:  &client.Money{Amount: "5.00", Currency: "USD"},
	})
	if err != nil {
		t.Fatalf("CreateCampaign MAX_CONVERSIONS: %v", err)
	}
	t.Cleanup(func() { archiveCampaign(t, c, created.ID) })

	t.Logf("created Max Conversions campaign id=%d bidding=%q targetCpa=%v serving=%q reasons=%v",
		created.ID, created.BiddingStrategy, moneyAmt(created.TargetCpa), created.ServingStatus, created.ServingStateReasons)

	if created.BiddingStrategy != client.BiddingStrategyMaxConversions {
		t.Fatalf("biddingStrategy = %q, want %s", created.BiddingStrategy, client.BiddingStrategyMaxConversions)
	}
	if created.TargetCpa == nil {
		t.Fatal("expected targetCpa in create response")
	}

	var groups []client.AdGroup
	deadline := time.Now().Add(30 * time.Second)
	for {
		page, listErr := c.ListAdGroups(ctx, created.ID, client.PageParams{Limit: 50, Offset: 0})
		if listErr != nil {
			t.Fatalf("ListAdGroups: %v", listErr)
		}
		groups = page.Data
		if len(groups) > 0 || time.Now().After(deadline) {
			break
		}
		time.Sleep(2 * time.Second)
	}

	t.Logf("FINDING Max Conversions auto ad groups: count=%d", len(groups))
	for _, g := range groups {
		t.Logf("  adGroup id=%d name=%q pricing=%q automatedKeywordsOptIn=%v status=%q deleted=%v",
			g.ID, g.Name, g.PricingModel, g.AutomatedKeywordsOptIn, g.Status, g.Deleted)
	}
	if len(groups) == 0 {
		t.Log("FINDING: no ad group present shortly after Max Conversions create — import docs must cover delayed creation or explicit AG create")
	}

	upd, err := c.UpdateCampaign(ctx, created.ID, &client.CampaignUpdate{
		TargetCpa: &client.Money{Amount: "12.00", Currency: "USD"},
	})
	if err != nil {
		t.Fatalf("UpdateCampaign targetCpa: %v", err)
	}
	t.Logf("FINDING target_cpa update: ok amount=%v", moneyAmt(upd.TargetCpa))
}

// TestLiveCampaignDisplaySupplyProbe creates a Display channel campaign for each
// documented supply source and records which combinations Apple accepts (#41 / #43).
func TestLiveCampaignDisplaySupplyProbe(t *testing.T) {
	c, creds := acctest.LiveClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	acls := acctest.RequireUserACLs(t, c)
	acctest.RequireOrgAccess(t, acls, creds.OrgID)
	adamID := probeAdamID(t, c)

	cases := []struct {
		name   string
		supply string
	}{
		{name: "today-tab", supply: client.SupplySourceTodayTab},
		{name: "search-tab", supply: client.SupplySourceSearchTab},
		{name: "product-pages-browse", supply: client.SupplySourceProductPagesBrowse},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			name := fmt.Sprintf("tf-probe-display-%s-%d", tc.name, time.Now().UnixNano())
			created, err := c.CreateCampaign(ctx, &client.CampaignCreate{
				Name:               name,
				AdamID:             adamID,
				CountriesOrRegions: []string{"US"},
				AdChannelType:      client.AdChannelTypeDisplay,
				SupplySources:      []string{tc.supply},
				BillingEvent:       client.BillingEventTaps,
				BiddingStrategy:    client.BiddingStrategyManualCPT,
				Status:             "PAUSED",
				DailyBudgetAmount:  &client.Money{Amount: "1.00", Currency: "USD"},
			})
			if err != nil {
				t.Logf("FINDING Display+%s: REJECTED — %v", tc.supply, err)
				return
			}
			t.Cleanup(func() { archiveCampaign(t, c, created.ID) })
			t.Logf("FINDING Display+%s: ACCEPTED id=%d channel=%q supply=%v billing=%q",
				tc.supply, created.ID, created.AdChannelType, created.SupplySources, created.BillingEvent)
		})
	}
}

// TestLiveCampaignMaxConversionsRejectsDisplaySupply confirms Max Conversions
// cannot be paired with Display supply sources (validator matrix).
func TestLiveCampaignMaxConversionsRejectsDisplaySupply(t *testing.T) {
	c, creds := acctest.LiveClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	acls := acctest.RequireUserACLs(t, c)
	acctest.RequireOrgAccess(t, acls, creds.OrgID)
	adamID := probeAdamID(t, c)

	_, err := c.CreateCampaign(ctx, &client.CampaignCreate{
		Name:               fmt.Sprintf("tf-probe-bad-max-%d", time.Now().UnixNano()),
		AdamID:             adamID,
		CountriesOrRegions: []string{"US"},
		AdChannelType:      client.AdChannelTypeDisplay,
		SupplySources:      []string{client.SupplySourceSearchTab},
		BillingEvent:       client.BillingEventTaps,
		BiddingStrategy:    client.BiddingStrategyMaxConversions,
		TargetCpa:          &client.Money{Amount: "10.00", Currency: "USD"},
		Status:             "PAUSED",
		DailyBudgetAmount:  &client.Money{Amount: "1.00", Currency: "USD"},
	})
	if err == nil {
		t.Fatal("FINDING: expected Max Conversions + Display supply to be rejected")
	}
	t.Logf("FINDING Max Conversions + Display supply: REJECTED as expected — %v", err)
}

func moneyAmt(m *client.Money) string {
	if m == nil {
		return "<nil>"
	}
	return m.Amount + " " + m.Currency
}
