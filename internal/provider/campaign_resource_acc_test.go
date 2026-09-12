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

// Acceptance tests require TF_ACC=1 and real Apple Ads credentials.
// They are excluded from default CI (see Makefile test vs testacc).

func TestAccCampaignResource_Lifecycle(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("Acceptance tests skipped unless TF_ACC=1")
	}
	testAccPreCheck(t)

	adamID := os.Getenv("APPLEADS_TEST_ADAM_ID")
	namePrefix := "tf-acc-campaign"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckCampaignDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig(true) + fmt.Sprintf(`
resource "apple-ads_campaign" "test" {
  name                 = "%[1]s-create"
  adam_id              = "%[2]s"
  countries_or_regions = ["US"]
  status               = "PAUSED"
  daily_budget_amount  = "1.00"
  daily_budget_currency = "USD"
}
`, namePrefix, adamID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("apple-ads_campaign.test", "id"),
					resource.TestCheckResourceAttr("apple-ads_campaign.test", "name", namePrefix+"-create"),
					resource.TestCheckResourceAttr("apple-ads_campaign.test", "status", "PAUSED"),
					resource.TestCheckResourceAttr("apple-ads_campaign.test", "daily_budget_amount", "1.00"),
				),
			},
			// Update mutable fields
			{
				Config: testAccProviderConfig(true) + fmt.Sprintf(`
resource "apple-ads_campaign" "test" {
  name                 = "%[1]s-updated"
  adam_id              = "%[2]s"
  countries_or_regions = ["US"]
  status               = "PAUSED"
  daily_budget_amount  = "2.00"
  daily_budget_currency = "USD"
}
`, namePrefix, adamID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("apple-ads_campaign.test", "name", namePrefix+"-updated"),
					resource.TestCheckResourceAttr("apple-ads_campaign.test", "daily_budget_amount", "2.00"),
				),
			},
			// Import
			{
				ResourceName:      "apple-ads_campaign.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccCampaignResource_DeletionProtection(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("Acceptance tests skipped unless TF_ACC=1")
	}
	testAccPreCheck(t)

	adamID := os.Getenv("APPLEADS_TEST_ADAM_ID")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig(false) + fmt.Sprintf(`
resource "apple-ads_campaign" "protected" {
  name                 = "tf-acc-protected"
  adam_id              = "%s"
  countries_or_regions = ["US"]
  status               = "PAUSED"
  daily_budget_amount  = "1.00"
  daily_budget_currency = "USD"
}
`, adamID),
			},
			{
				Config:      testAccProviderConfig(false),
				Destroy:     true,
				ExpectError: regexp.MustCompile(`Campaign deletion is disabled by provider configuration`),
			},
		},
	})
}

func TestAccCampaignResource_ImmutableChangeRejected(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("Acceptance tests skipped unless TF_ACC=1")
	}
	testAccPreCheck(t)

	adamID := os.Getenv("APPLEADS_TEST_ADAM_ID")
	var originalID string

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckCampaignDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig(true) + fmt.Sprintf(`
resource "apple-ads_campaign" "immutable" {
  name                 = "tf-acc-immutable"
  adam_id              = "%s"
  countries_or_regions = ["US"]
  status               = "PAUSED"
  daily_budget_amount  = "1.00"
  daily_budget_currency = "USD"
}
`, adamID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("apple-ads_campaign.immutable", "id"),
					func(s *terraform.State) error {
						rs := s.RootModule().Resources["apple-ads_campaign.immutable"]
						originalID = rs.Primary.ID
						return nil
					},
				),
			},
			{
				Config: testAccProviderConfig(true) + fmt.Sprintf(`
resource "apple-ads_campaign" "immutable" {
  name                 = "tf-acc-immutable"
  adam_id              = "%s"
  countries_or_regions = ["CA"]
  status               = "PAUSED"
  daily_budget_amount  = "1.00"
  daily_budget_currency = "USD"
}
`, adamID),
				ExpectError: regexp.MustCompile(`Cannot change immutable campaign field "countries_or_regions"`),
			},
			{
				// Prove original campaign identity was preserved (not archived/replaced).
				Config: testAccProviderConfig(true) + fmt.Sprintf(`
resource "apple-ads_campaign" "immutable" {
  name                 = "tf-acc-immutable"
  adam_id              = "%s"
  countries_or_regions = ["US"]
  status               = "PAUSED"
  daily_budget_amount  = "1.00"
  daily_budget_currency = "USD"
}
`, adamID),
				Check: resource.ComposeAggregateTestCheckFunc(
					func(s *terraform.State) error {
						rs := s.RootModule().Resources["apple-ads_campaign.immutable"]
						if rs.Primary.ID != originalID {
							return fmt.Errorf("campaign id changed from %s to %s (replacement occurred)", originalID, rs.Primary.ID)
						}
						id, err := strconv.ParseInt(originalID, 10, 64)
						if err != nil {
							return err
						}
						c, err := testAccAPIClient(t)
						if err != nil {
							return err
						}
						got, err := c.GetCampaign(context.Background(), id)
						if err != nil {
							return err
						}
						if got.Deleted {
							return fmt.Errorf("campaign %s was archived after immutable change attempt", originalID)
						}
						if len(got.CountriesOrRegions) != 1 || got.CountriesOrRegions[0] != "US" {
							return fmt.Errorf("countries changed unexpectedly: %#v", got.CountriesOrRegions)
						}
						return nil
					},
				),
			},
		},
	})
}

func testAccCheckCampaignDestroy(s *terraform.State) error {
	c, err := testAccAPIClientFromEnv()
	if err != nil {
		return err
	}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "apple-ads_campaign" {
			continue
		}
		id, err := strconv.ParseInt(rs.Primary.ID, 10, 64)
		if err != nil {
			return err
		}
		got, err := c.GetCampaign(context.Background(), id)
		if err != nil {
			if client.IsNotFound(err) {
				continue
			}
			return err
		}
		if got != nil && !got.Deleted {
			return fmt.Errorf("campaign %s still exists and is not archived", rs.Primary.ID)
		}
	}
	return nil
}

func testAccAPIClient(t *testing.T) (*client.Client, error) {
	t.Helper()
	return testAccAPIClientFromEnv()
}

func testAccAPIClientFromEnv() (*client.Client, error) {
	creds := client.Credentials{
		OrgID:      os.Getenv("APPLEADS_ORG_ID"),
		ClientID:   os.Getenv("APPLEADS_CLIENT_ID"),
		TeamID:     os.Getenv("APPLEADS_TEAM_ID"),
		KeyID:      os.Getenv("APPLEADS_KEY_ID"),
		PrivateKey: os.Getenv("APPLEADS_PRIVATE_KEY"),
	}
	tokens, err := client.NewOAuthTokenSource(client.OAuthConfig{Credentials: creds})
	if err != nil {
		return nil, err
	}
	httpClient := client.NewAuthenticatedHTTPClient(tokens, creds.OrgID, nil)
	httpClient = client.WithRetry(httpClient, client.RetryConfig{})
	return client.New(client.WithHTTPClient(httpClient))
}
