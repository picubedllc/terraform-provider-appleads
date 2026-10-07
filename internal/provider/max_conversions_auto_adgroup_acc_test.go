// Copyright (c) 2026 Pi Cubed LLC
// SPDX-License-Identifier: MIT

package provider

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/picubedllc/terraform-provider-appleads/internal/client"
)

// TestAccAdGroupResource_MaxConversionsAutoImport creates a Maximize
// Conversions campaign via the API, discovers Apple's auto-created Automated
// Ad Group, imports both into Terraform using existing import formats, then
// updates target_cpa on the campaign.
func TestAccAdGroupResource_MaxConversionsAutoImport(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("Acceptance tests skipped unless TF_ACC=1")
	}
	testAccPreCheck(t)

	adamID := os.Getenv("APPLEADS_TEST_ADAM_ID")
	adamIDInt, err := strconv.ParseInt(adamID, 10, 64)
	if err != nil {
		t.Fatalf("APPLEADS_TEST_ADAM_ID: %v", err)
	}

	c, err := testAccAPIClientFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	name := fmt.Sprintf("tf-acc-maxconv-autoag-%d", time.Now().UnixNano())
	created, err := c.CreateCampaign(ctx, &client.CampaignCreate{
		Name:               name,
		AdamID:             adamIDInt,
		CountriesOrRegions: []string{"US"},
		AdChannelType:      client.AdChannelTypeSearch,
		SupplySources:      []string{client.SupplySourceSearchResults},
		BillingEvent:       client.BillingEventTaps,
		BiddingStrategy:    client.BiddingStrategyMaxConversions,
		TargetCpa:          &client.Money{Amount: "10.00", Currency: "USD"},
		Status:             "PAUSED",
		DailyBudgetAmount:  &client.Money{Amount: "1.00", Currency: "USD"},
	})
	if err != nil {
		t.Fatalf("CreateCampaign MAX_CONVERSIONS: %v", err)
	}
	t.Cleanup(func() {
		_ = c.DeleteCampaign(context.Background(), created.ID)
	})

	ag, err := waitForMaxConversionsAutoAdGroup(ctx, c, created.ID, 45*time.Second)
	if err != nil {
		t.Fatal(err)
	}

	campaignImportID := strconv.FormatInt(created.ID, 10)
	compoundImportID := fmt.Sprintf("%d/%d", created.ID, ag.ID)
	bareImportID := strconv.FormatInt(ag.ID, 10)

	status := ag.Status
	if status == "" {
		status = "ENABLED"
	}
	pricingModel := ag.PricingModel
	if pricingModel == "" {
		pricingModel = client.PricingModelCPC
	}
	startTime := ag.StartTime
	if startTime == "" {
		startTime = "2026-01-01T00:00:00.000"
	}
	bidAmount, bidCurrency := "0.01", "USD"
	if ag.DefaultBidAmount != nil {
		bidAmount = ag.DefaultBidAmount.Amount
		bidCurrency = ag.DefaultBidAmount.Currency
	}

	campaignOnlyConfig := testAccProviderConfig(true) +
		testAccMaxConversionsCampaignConfig(name, adamID, "10.00")

	baseConfig := campaignOnlyConfig +
		testAccImportedAutoAdGroupConfig(ag.Name, status, bidAmount, bidCurrency, pricingModel, startTime, ag.AutomatedKeywordsOptIn)

	updatedConfig := testAccProviderConfig(true) +
		testAccMaxConversionsCampaignConfig(name+"-upd", adamID, "12.00") +
		testAccImportedAutoAdGroupConfig(ag.Name, status, bidAmount, bidCurrency, pricingModel, startTime, ag.AutomatedKeywordsOptIn)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckAdGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config:                  campaignOnlyConfig,
				ResourceName:            "appleads_campaign.max",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStatePersist:      true,
				ImportStateId:           campaignImportID,
				ImportStateVerifyIgnore: []string{"modification_time", "serving_status", "display_status"},
			},
			{
				Config:                  baseConfig,
				ResourceName:            "appleads_ad_group.auto",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStatePersist:      true,
				ImportStateId:           compoundImportID,
				ImportStateVerifyIgnore: []string{"modification_time", "serving_status", "display_status"},
			},
			{
				ResourceName:            "appleads_ad_group.auto",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateId:           bareImportID,
				ImportStateVerifyIgnore: []string{"modification_time", "serving_status", "display_status"},
			},
			{
				Config: updatedConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("appleads_campaign.max", "id", campaignImportID),
					resource.TestCheckResourceAttr("appleads_campaign.max", "name", name+"-upd"),
					resource.TestCheckResourceAttr("appleads_campaign.max", "bidding_strategy", "MAX_CONVERSIONS"),
					resource.TestCheckResourceAttr("appleads_campaign.max", "target_cpa_amount", "12.00"),
					resource.TestCheckResourceAttr("appleads_ad_group.auto", "id", bareImportID),
					resource.TestCheckResourceAttrPair("appleads_ad_group.auto", "campaign_id", "appleads_campaign.max", "id"),
					func(s *terraform.State) error {
						rs := s.RootModule().Resources["appleads_ad_group.auto"]
						if rs.Primary.Attributes["automated_keywords_opt_in"] != strconv.FormatBool(ag.AutomatedKeywordsOptIn) {
							return fmt.Errorf("automated_keywords_opt_in = %q", rs.Primary.Attributes["automated_keywords_opt_in"])
						}
						return nil
					},
				),
			},
		},
	})
}

func testAccMaxConversionsCampaignConfig(name, adamID, targetCPA string) string {
	return fmt.Sprintf(`
resource "appleads_campaign" "max" {
  name                  = "%[1]s"
  adam_id               = "%[2]s"
  countries_or_regions  = ["US"]
  status                = "PAUSED"
  daily_budget_amount   = "1.00"
  daily_budget_currency = "USD"
  bidding_strategy      = "MAX_CONVERSIONS"
  target_cpa_amount     = "%[3]s"
  target_cpa_currency   = "USD"
}
`, name, adamID, targetCPA)
}

func testAccImportedAutoAdGroupConfig(name, status, bidAmount, bidCurrency, pricingModel, startTime string, autoKeywords bool) string {
	return fmt.Sprintf(`
resource "appleads_ad_group" "auto" {
  campaign_id               = appleads_campaign.max.id
  name                      = %q
  status                    = %q
  default_bid_amount        = %q
  default_bid_currency      = %q
  pricing_model             = %q
  start_time                = %q
  automated_keywords_opt_in = %t
}
`, name, status, bidAmount, bidCurrency, pricingModel, startTime, autoKeywords)
}

func waitForMaxConversionsAutoAdGroup(ctx context.Context, c *client.Client, campaignID int64, timeout time.Duration) (*client.AdGroup, error) {
	deadline := time.Now().Add(timeout)
	var lastErr error
	for {
		page, err := c.ListAdGroups(ctx, campaignID, client.PageParams{Limit: 50, Offset: 0})
		if err != nil {
			lastErr = err
		} else {
			for i := range page.Data {
				ag := &page.Data[i]
				if ag.Deleted {
					continue
				}
				if ag.AutomatedKeywordsOptIn || len(page.Data) == 1 {
					return ag, nil
				}
			}
			if len(page.Data) > 0 {
				return &page.Data[0], nil
			}
		}
		if time.Now().After(deadline) {
			if lastErr != nil {
				return nil, fmt.Errorf("no auto ad group for campaign %d within %s: %w", campaignID, timeout, lastErr)
			}
			return nil, fmt.Errorf("no auto ad group for Max Conversions campaign %d within %s — Apple may create it asynchronously; retry or create/import manually", campaignID, timeout)
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
}
