// Copyright (c) 2026 Pi Cubed LLC
// SPDX-License-Identifier: MIT

package provider

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/picubedllc/terraform-provider-appleads/internal/client"
)

func TestAccAdGroupResource_Lifecycle(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("Acceptance tests skipped unless TF_ACC=1")
	}
	testAccPreCheck(t)

	adamID := os.Getenv("APPLEADS_TEST_ADAM_ID")
	namePrefix := "tf-acc-adgroup"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckAdGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig(true) + testAccCampaignPausedConfig("parent", adamID) + fmt.Sprintf(`
resource "appleads_ad_group" "test" {
  campaign_id               = appleads_campaign.parent.id
  name                      = "%[1]s-create"
  status                    = "PAUSED"
  default_bid_amount        = "1.00"
  default_bid_currency      = "USD"
  pricing_model             = "CPC"
  start_time                = "2026-01-01T00:00:00.000"
  automated_keywords_opt_in = false
}
`, namePrefix),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("appleads_ad_group.test", "id"),
					resource.TestCheckResourceAttr("appleads_ad_group.test", "name", namePrefix+"-create"),
					resource.TestCheckResourceAttr("appleads_ad_group.test", "status", "PAUSED"),
					resource.TestCheckResourceAttr("appleads_ad_group.test", "default_bid_amount", "1.00"),
					resource.TestCheckResourceAttr("appleads_ad_group.test", "pricing_model", "CPC"),
					resource.TestCheckResourceAttrPair("appleads_ad_group.test", "campaign_id", "appleads_campaign.parent", "id"),
				),
			},
			{
				Config: testAccProviderConfig(true) + testAccCampaignPausedConfig("parent", adamID) + fmt.Sprintf(`
resource "appleads_ad_group" "test" {
  campaign_id               = appleads_campaign.parent.id
  name                      = "%[1]s-updated"
  status                    = "PAUSED"
  default_bid_amount        = "1.50"
  default_bid_currency      = "USD"
  pricing_model             = "CPC"
  start_time                = "2026-01-01T00:00:00.000"
  automated_keywords_opt_in = true
}
`, namePrefix),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("appleads_ad_group.test", "name", namePrefix+"-updated"),
					resource.TestCheckResourceAttr("appleads_ad_group.test", "default_bid_amount", "1.50"),
					resource.TestCheckResourceAttr("appleads_ad_group.test", "automated_keywords_opt_in", "true"),
				),
			},
			{
				ResourceName:      "appleads_ad_group.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs := s.RootModule().Resources["appleads_ad_group.test"]
					return rs.Primary.Attributes["campaign_id"] + "/" + rs.Primary.ID, nil
				},
			},
		},
	})
}

func TestAccAdGroupResource_ImmutableCampaignIDRejected(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("Acceptance tests skipped unless TF_ACC=1")
	}
	testAccPreCheck(t)

	adamID := os.Getenv("APPLEADS_TEST_ADAM_ID")
	var originalID string

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckAdGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig(true) + testAccCampaignPausedConfig("a", adamID) + testAccCampaignPausedConfig("b", adamID) + `
resource "appleads_ad_group" "immutable" {
  campaign_id          = appleads_campaign.a.id
  name                 = "tf-acc-adgroup-immutable"
  status               = "PAUSED"
  default_bid_amount   = "1.00"
  default_bid_currency = "USD"
  pricing_model        = "CPC"
  start_time           = "2026-01-01T00:00:00.000"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("appleads_ad_group.immutable", "id"),
					func(s *terraform.State) error {
						rs := s.RootModule().Resources["appleads_ad_group.immutable"]
						originalID = rs.Primary.ID
						return nil
					},
				),
			},
			{
				Config: testAccProviderConfig(true) + testAccCampaignPausedConfig("a", adamID) + testAccCampaignPausedConfig("b", adamID) + `
resource "appleads_ad_group" "immutable" {
  campaign_id          = appleads_campaign.b.id
  name                 = "tf-acc-adgroup-immutable"
  status               = "PAUSED"
  default_bid_amount   = "1.00"
  default_bid_currency = "USD"
  pricing_model        = "CPC"
  start_time           = "2026-01-01T00:00:00.000"
}
`,
				ExpectError: regexp.MustCompile(`Cannot change immutable ad group field "campaign_id"`),
			},
			{
				Config: testAccProviderConfig(true) + testAccCampaignPausedConfig("a", adamID) + testAccCampaignPausedConfig("b", adamID) + `
resource "appleads_ad_group" "immutable" {
  campaign_id          = appleads_campaign.a.id
  name                 = "tf-acc-adgroup-immutable"
  status               = "PAUSED"
  default_bid_amount   = "1.00"
  default_bid_currency = "USD"
  pricing_model        = "CPC"
  start_time           = "2026-01-01T00:00:00.000"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					func(s *terraform.State) error {
						rs := s.RootModule().Resources["appleads_ad_group.immutable"]
						if rs.Primary.ID != originalID {
							return fmt.Errorf("ad group id changed from %s to %s (replacement occurred)", originalID, rs.Primary.ID)
						}
						campaignID, err := strconv.ParseInt(rs.Primary.Attributes["campaign_id"], 10, 64)
						if err != nil {
							return err
						}
						adGroupID, err := strconv.ParseInt(originalID, 10, 64)
						if err != nil {
							return err
						}
						c, err := testAccAPIClient(t)
						if err != nil {
							return err
						}
						got, err := c.GetAdGroup(context.Background(), campaignID, adGroupID)
						if err != nil {
							return err
						}
						if got.Deleted {
							return fmt.Errorf("ad group %s was deleted after immutable change attempt", originalID)
						}
						return nil
					},
				),
			},
		},
	})
}

func testAccCampaignPausedConfig(name, adamID string) string {
	return fmt.Sprintf(`
resource "appleads_campaign" "%[1]s" {
  name                  = "tf-acc-parent-%[1]s"
  adam_id               = "%[2]s"
  countries_or_regions  = ["US"]
  status                = "PAUSED"
  daily_budget_amount   = "1.00"
  daily_budget_currency = "USD"
}
`, name, adamID)
}

func testAccCheckAdGroupDestroy(s *terraform.State) error {
	c, err := testAccAPIClientFromEnv()
	if err != nil {
		return err
	}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "appleads_ad_group" {
			continue
		}
		campaignID, err := strconv.ParseInt(rs.Primary.Attributes["campaign_id"], 10, 64)
		if err != nil {
			return err
		}
		adGroupID, err := strconv.ParseInt(rs.Primary.ID, 10, 64)
		if err != nil {
			return err
		}
		got, err := c.GetAdGroup(context.Background(), campaignID, adGroupID)
		if err != nil {
			if client.IsNotFound(err) {
				continue
			}
			return err
		}
		if got != nil && !got.Deleted {
			return fmt.Errorf("ad group %s still exists and is not deleted", rs.Primary.ID)
		}
	}
	// Also ensure parent campaigns are archived when allow_campaign_deletion was true.
	return testAccCheckCampaignDestroy(s)
}

func TestAccAdGroupResource_LocalityTargeting(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("Acceptance tests skipped unless TF_ACC=1")
	}
	testAccPreCheck(t)

	adamID := os.Getenv("APPLEADS_TEST_ADAM_ID")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckAdGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig(true) + testAccCampaignPausedConfig("geo", adamID) + `
resource "appleads_ad_group" "nyc" {
  campaign_id          = appleads_campaign.geo.id
  name                 = "tf-acc-adgroup-nyc"
  status               = "PAUSED"
  default_bid_amount   = "1.00"
  default_bid_currency = "USD"
  pricing_model        = "CPC"
  start_time           = "2026-01-01T00:00:00.000"

  targeting_dimensions = {
    locality = {
      included = ["US|NY|New York"]
    }
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("appleads_ad_group.nyc", "id"),
					resource.TestCheckResourceAttr("appleads_ad_group.nyc", "targeting_dimensions.locality.included.#", "1"),
					resource.TestCheckResourceAttr("appleads_ad_group.nyc", "targeting_dimensions.locality.included.0", "US|NY|New York"),
				),
			},
			{
				Config: testAccProviderConfig(true) + testAccCampaignPausedConfig("geo", adamID) + `
resource "appleads_ad_group" "nyc" {
  campaign_id          = appleads_campaign.geo.id
  name                 = "tf-acc-adgroup-nyc"
  status               = "PAUSED"
  default_bid_amount   = "1.00"
  default_bid_currency = "USD"
  pricing_model        = "CPC"
  start_time           = "2026-01-01T00:00:00.000"

  targeting_dimensions = {
    admin_area = {
      included = ["US|NY"]
    }
    locality = {
      included = ["US|NY|New York"]
    }
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("appleads_ad_group.nyc", "targeting_dimensions.admin_area.included.0", "US|NY"),
					resource.TestCheckResourceAttr("appleads_ad_group.nyc", "targeting_dimensions.locality.included.0", "US|NY|New York"),
				),
			},
		},
	})
}
