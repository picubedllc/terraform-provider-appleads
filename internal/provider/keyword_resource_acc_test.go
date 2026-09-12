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

func TestAccKeywordResource_Lifecycle(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("Acceptance tests skipped unless TF_ACC=1")
	}
	testAccPreCheck(t)

	adamID := os.Getenv("APPLEADS_TEST_ADAM_ID")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckKeywordDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig(true) + testAccCampaignPausedConfig("parent", adamID) + testAccAdGroupPausedConfig() + `
resource "apple-ads_keyword" "test" {
  ad_group_id  = apple-ads_ad_group.ag.id
  text         = "tf acc keyword create"
  match_type   = "EXACT"
  status       = "PAUSED"
  bid_amount   = "1.00"
  bid_currency = "USD"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("apple-ads_keyword.test", "id"),
					resource.TestCheckResourceAttrSet("apple-ads_keyword.test", "campaign_id"),
					resource.TestCheckResourceAttr("apple-ads_keyword.test", "text", "tf acc keyword create"),
					resource.TestCheckResourceAttr("apple-ads_keyword.test", "match_type", "EXACT"),
					resource.TestCheckResourceAttr("apple-ads_keyword.test", "status", "PAUSED"),
					resource.TestCheckResourceAttr("apple-ads_keyword.test", "bid_amount", "1.00"),
					resource.TestCheckResourceAttrPair("apple-ads_keyword.test", "ad_group_id", "apple-ads_ad_group.ag", "id"),
				),
			},
			{
				Config: testAccProviderConfig(true) + testAccCampaignPausedConfig("parent", adamID) + testAccAdGroupPausedConfig() + `
resource "apple-ads_keyword" "test" {
  ad_group_id  = apple-ads_ad_group.ag.id
  text         = "tf acc keyword create"
  match_type   = "EXACT"
  status       = "PAUSED"
  bid_amount   = "1.50"
  bid_currency = "USD"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("apple-ads_keyword.test", "bid_amount", "1.50"),
				),
			},
			{
				ResourceName:      "apple-ads_keyword.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs := s.RootModule().Resources["apple-ads_keyword.test"]
					return rs.Primary.Attributes["campaign_id"] + "/" +
						rs.Primary.Attributes["ad_group_id"] + "/" +
						rs.Primary.ID, nil
				},
			},
		},
	})
}

func TestAccKeywordResource_ImmutableTextRejected(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("Acceptance tests skipped unless TF_ACC=1")
	}
	testAccPreCheck(t)

	adamID := os.Getenv("APPLEADS_TEST_ADAM_ID")
	var originalID string

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckKeywordDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig(true) + testAccCampaignPausedConfig("parent", adamID) + testAccAdGroupPausedConfig() + `
resource "apple-ads_keyword" "immutable" {
  ad_group_id  = apple-ads_ad_group.ag.id
  text         = "tf acc keyword immutable"
  match_type   = "BROAD"
  status       = "PAUSED"
  bid_amount   = "1.00"
  bid_currency = "USD"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("apple-ads_keyword.immutable", "id"),
					func(s *terraform.State) error {
						originalID = s.RootModule().Resources["apple-ads_keyword.immutable"].Primary.ID
						return nil
					},
				),
			},
			{
				Config: testAccProviderConfig(true) + testAccCampaignPausedConfig("parent", adamID) + testAccAdGroupPausedConfig() + `
resource "apple-ads_keyword" "immutable" {
  ad_group_id  = apple-ads_ad_group.ag.id
  text         = "tf acc keyword changed"
  match_type   = "BROAD"
  status       = "PAUSED"
  bid_amount   = "1.00"
  bid_currency = "USD"
}
`,
				ExpectError: regexp.MustCompile(`Cannot change immutable keyword field "text"`),
			},
			{
				Config: testAccProviderConfig(true) + testAccCampaignPausedConfig("parent", adamID) + testAccAdGroupPausedConfig() + `
resource "apple-ads_keyword" "immutable" {
  ad_group_id  = apple-ads_ad_group.ag.id
  text         = "tf acc keyword immutable"
  match_type   = "BROAD"
  status       = "PAUSED"
  bid_amount   = "1.00"
  bid_currency = "USD"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					func(s *terraform.State) error {
						rs := s.RootModule().Resources["apple-ads_keyword.immutable"]
						if rs.Primary.ID != originalID {
							return fmt.Errorf("keyword id changed from %s to %s", originalID, rs.Primary.ID)
						}
						return nil
					},
				),
			},
		},
	})
}

func testAccAdGroupPausedConfig() string {
	return `
resource "apple-ads_ad_group" "ag" {
  campaign_id               = apple-ads_campaign.parent.id
  name                      = "tf-acc-ag-ag"
  status                    = "PAUSED"
  default_bid_amount        = "1.00"
  default_bid_currency      = "USD"
  automated_keywords_opt_in = false
}
`
}

func testAccCheckKeywordDestroy(s *terraform.State) error {
	c, err := testAccAPIClientFromEnv()
	if err != nil {
		return err
	}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "apple-ads_keyword" {
			continue
		}
		campaignID, err := strconv.ParseInt(rs.Primary.Attributes["campaign_id"], 10, 64)
		if err != nil {
			return err
		}
		adGroupID, err := strconv.ParseInt(rs.Primary.Attributes["ad_group_id"], 10, 64)
		if err != nil {
			return err
		}
		keywordID, err := strconv.ParseInt(rs.Primary.ID, 10, 64)
		if err != nil {
			return err
		}
		got, err := c.GetKeyword(context.Background(), campaignID, adGroupID, keywordID)
		if err != nil {
			if client.IsNotFound(err) {
				continue
			}
			return err
		}
		if got != nil && !got.Deleted {
			return fmt.Errorf("keyword %s still exists and is not deleted", rs.Primary.ID)
		}
	}
	return testAccCheckAdGroupDestroy(s)
}
