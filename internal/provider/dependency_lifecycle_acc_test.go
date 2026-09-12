// Copyright (c) 2026 Pi Cubed LLC
// SPDX-License-Identifier: MIT

package provider

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/picubedllc/terraform-provider-appleads/internal/client"
)

// TestAccDependencyLifecycle_FullHierarchy exercises
// app → campaign → ad group → keyword + negative keywords together.
//
// It verifies:
//   - Terraform creates via implicit references (no depends_on)
//   - Removing nested resources does not archive the parent campaign
//   - Final destroy archives the campaign only when allow_campaign_deletion=true
func TestAccDependencyLifecycle_FullHierarchy(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("Acceptance tests skipped unless TF_ACC=1")
	}
	testAccPreCheck(t)

	adamID := os.Getenv("APPLEADS_TEST_ADAM_ID")
	var campaignID, adGroupID string

	fullConfig := testAccProviderConfig(true) + fmt.Sprintf(`
data "apple-ads_app" "app" {
  id = "%[1]s"
}

resource "apple-ads_campaign" "hierarchy" {
  name                  = "tf-acc-hierarchy"
  adam_id               = data.apple-ads_app.app.adam_id
  countries_or_regions  = ["US"]
  status                = "PAUSED"
  daily_budget_amount   = "1.00"
  daily_budget_currency = "USD"
}

resource "apple-ads_ad_group" "hierarchy" {
  campaign_id               = apple-ads_campaign.hierarchy.id
  name                      = "tf-acc-hierarchy-ag"
  status                    = "PAUSED"
  default_bid_amount        = "1.00"
  default_bid_currency      = "USD"
  automated_keywords_opt_in = false
}

resource "apple-ads_keyword" "exact" {
  ad_group_id  = apple-ads_ad_group.hierarchy.id
  text         = "tf acc hierarchy exact"
  match_type   = "EXACT"
  status       = "PAUSED"
  bid_amount   = "1.00"
  bid_currency = "USD"
}

resource "apple-ads_keyword" "broad" {
  ad_group_id  = apple-ads_ad_group.hierarchy.id
  text         = "tf acc hierarchy broad"
  match_type   = "BROAD"
  status       = "PAUSED"
  bid_amount   = "0.75"
  bid_currency = "USD"
}

resource "apple-ads_negative_keyword" "campaign" {
  campaign_id = apple-ads_campaign.hierarchy.id
  text        = "tf acc hierarchy free"
  match_type  = "EXACT"
  status      = "PAUSED"
}

resource "apple-ads_negative_keyword" "adgroup" {
  ad_group_id = apple-ads_ad_group.hierarchy.id
  text        = "tf acc hierarchy cheap"
  match_type  = "BROAD"
  status      = "PAUSED"
}
`, adamID)

	campaignOnlyConfig := testAccProviderConfig(true) + fmt.Sprintf(`
data "apple-ads_app" "app" {
  id = "%[1]s"
}

resource "apple-ads_campaign" "hierarchy" {
  name                  = "tf-acc-hierarchy"
  adam_id               = data.apple-ads_app.app.adam_id
  countries_or_regions  = ["US"]
  status                = "PAUSED"
  daily_budget_amount   = "1.00"
  daily_budget_currency = "USD"
}
`, adamID)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckCampaignDestroy,
		Steps: []resource.TestStep{
			{
				Config: fullConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.apple-ads_app.app", "adam_id"),
					resource.TestCheckResourceAttrSet("apple-ads_campaign.hierarchy", "id"),
					resource.TestCheckResourceAttrSet("apple-ads_ad_group.hierarchy", "id"),
					resource.TestCheckResourceAttrSet("apple-ads_keyword.exact", "id"),
					resource.TestCheckResourceAttrSet("apple-ads_keyword.broad", "id"),
					resource.TestCheckResourceAttrSet("apple-ads_negative_keyword.campaign", "id"),
					resource.TestCheckResourceAttrSet("apple-ads_negative_keyword.adgroup", "id"),
					resource.TestCheckResourceAttrPair("apple-ads_ad_group.hierarchy", "campaign_id", "apple-ads_campaign.hierarchy", "id"),
					resource.TestCheckResourceAttrPair("apple-ads_keyword.exact", "ad_group_id", "apple-ads_ad_group.hierarchy", "id"),
					resource.TestCheckResourceAttrPair("apple-ads_keyword.exact", "campaign_id", "apple-ads_campaign.hierarchy", "id"),
					resource.TestCheckResourceAttrPair("apple-ads_negative_keyword.campaign", "campaign_id", "apple-ads_campaign.hierarchy", "id"),
					resource.TestCheckResourceAttrPair("apple-ads_negative_keyword.adgroup", "ad_group_id", "apple-ads_ad_group.hierarchy", "id"),
					func(s *terraform.State) error {
						campaignID = s.RootModule().Resources["apple-ads_campaign.hierarchy"].Primary.ID
						adGroupID = s.RootModule().Resources["apple-ads_ad_group.hierarchy"].Primary.ID
						return nil
					},
				),
			},
			{
				// Tear down nested resources only. Parent campaign must stay unarchived
				// and keep the same id (no accidental allow_campaign_deletion side effects).
				Config: campaignOnlyConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					func(s *terraform.State) error {
						rs := s.RootModule().Resources["apple-ads_campaign.hierarchy"]
						if rs.Primary.ID != campaignID {
							return fmt.Errorf("campaign id changed from %s to %s after nested destroy", campaignID, rs.Primary.ID)
						}
						c, err := testAccAPIClientFromEnv()
						if err != nil {
							return err
						}
						id, err := strconv.ParseInt(campaignID, 10, 64)
						if err != nil {
							return err
						}
						got, err := c.GetCampaign(context.Background(), id)
						if err != nil {
							return err
						}
						if got.Deleted {
							return fmt.Errorf("campaign %s was archived after removing nested resources", campaignID)
						}
						agID, err := strconv.ParseInt(adGroupID, 10, 64)
						if err != nil {
							return err
						}
						ag, err := c.GetAdGroup(context.Background(), id, agID)
						if err != nil {
							if client.IsNotFound(err) {
								return nil
							}
							return err
						}
						if ag != nil && !ag.Deleted {
							return fmt.Errorf("ad group %s still present after nested teardown", adGroupID)
						}
						return nil
					},
				),
			},
		},
	})
}
