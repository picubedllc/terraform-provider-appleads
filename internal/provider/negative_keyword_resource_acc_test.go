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

func TestAccNegativeKeywordResource_CampaignAndAdGroup(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("Acceptance tests skipped unless TF_ACC=1")
	}
	testAccPreCheck(t)
	adamID := os.Getenv("APPLEADS_TEST_ADAM_ID")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckNegativeKeywordDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig(true) + testAccCampaignPausedConfig("parent", adamID) + testAccAdGroupPausedConfig() + `
resource "appleads_negative_keyword" "campaign" {
  campaign_id = appleads_campaign.parent.id
  text        = "tf acc free"
  match_type  = "EXACT"
  status      = "PAUSED"
}

resource "appleads_negative_keyword" "adgroup" {
  ad_group_id = appleads_ad_group.ag.id
  text        = "tf acc cheap"
  match_type  = "BROAD"
  status      = "PAUSED"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("appleads_negative_keyword.campaign", "id"),
					resource.TestCheckResourceAttr("appleads_negative_keyword.campaign", "text", "tf acc free"),
					resource.TestCheckResourceAttrSet("appleads_negative_keyword.adgroup", "id"),
					resource.TestCheckResourceAttrSet("appleads_negative_keyword.adgroup", "campaign_id"),
					resource.TestCheckResourceAttrPair("appleads_negative_keyword.adgroup", "ad_group_id", "appleads_ad_group.ag", "id"),
				),
			},
			{
				Config: testAccProviderConfig(true) + testAccCampaignPausedConfig("parent", adamID) + testAccAdGroupPausedConfig() + `
resource "appleads_negative_keyword" "campaign" {
  campaign_id = appleads_campaign.parent.id
  text        = "tf acc free"
  match_type  = "EXACT"
  status      = "ACTIVE"
}

resource "appleads_negative_keyword" "adgroup" {
  ad_group_id = appleads_ad_group.ag.id
  text        = "tf acc cheap"
  match_type  = "BROAD"
  status      = "PAUSED"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("appleads_negative_keyword.campaign", "status", "ACTIVE"),
				),
			},
			{
				ResourceName:      "appleads_negative_keyword.campaign",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs := s.RootModule().Resources["appleads_negative_keyword.campaign"]
					return "campaign/" + rs.Primary.Attributes["campaign_id"] + "/" + rs.Primary.ID, nil
				},
			},
			{
				ResourceName:      "appleads_negative_keyword.adgroup",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs := s.RootModule().Resources["appleads_negative_keyword.adgroup"]
					return "adgroup/" + rs.Primary.Attributes["campaign_id"] + "/" +
						rs.Primary.Attributes["ad_group_id"] + "/" + rs.Primary.ID, nil
				},
			},
		},
	})
}

func TestAccNegativeKeywordResource_ImmutableTextRejected(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("Acceptance tests skipped unless TF_ACC=1")
	}
	testAccPreCheck(t)
	adamID := os.Getenv("APPLEADS_TEST_ADAM_ID")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckNegativeKeywordDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig(true) + testAccCampaignPausedConfig("parent", adamID) + `
resource "appleads_negative_keyword" "immutable" {
  campaign_id = appleads_campaign.parent.id
  text        = "tf acc nk immutable"
  match_type  = "EXACT"
  status      = "PAUSED"
}
`,
			},
			{
				Config: testAccProviderConfig(true) + testAccCampaignPausedConfig("parent", adamID) + `
resource "appleads_negative_keyword" "immutable" {
  campaign_id = appleads_campaign.parent.id
  text        = "tf acc nk changed"
  match_type  = "EXACT"
  status      = "PAUSED"
}
`,
				ExpectError: regexp.MustCompile(`Cannot change immutable negative keyword field "text"`),
			},
		},
	})
}

func testAccCheckNegativeKeywordDestroy(s *terraform.State) error {
	c, err := testAccAPIClientFromEnv()
	if err != nil {
		return err
	}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "appleads_negative_keyword" {
			continue
		}
		campaignID, err := strconv.ParseInt(rs.Primary.Attributes["campaign_id"], 10, 64)
		if err != nil {
			return err
		}
		keywordID, err := strconv.ParseInt(rs.Primary.ID, 10, 64)
		if err != nil {
			return err
		}
		var got *client.NegativeKeyword
		var getErr error
		if ag := rs.Primary.Attributes["ad_group_id"]; ag != "" {
			adGroupID, parseErr := strconv.ParseInt(ag, 10, 64)
			if parseErr != nil {
				return parseErr
			}
			got, getErr = c.GetAdGroupNegativeKeyword(context.Background(), campaignID, adGroupID, keywordID)
		} else {
			got, getErr = c.GetCampaignNegativeKeyword(context.Background(), campaignID, keywordID)
		}
		if getErr != nil {
			if client.IsNotFound(getErr) {
				continue
			}
			return getErr
		}
		if got != nil && !got.Deleted {
			return fmt.Errorf("negative keyword %s still exists", rs.Primary.ID)
		}
	}
	return testAccCheckAdGroupDestroy(s)
}
