# Complete example: app lookup → campaign → ad group → keywords + negatives.
#
# Values are fictitious samples for documentation. Do not paste production
# org IDs, app IDs, or credentials into this example.

terraform {
  required_providers {
    apple_ads = {
      source = "picubedllc/appleads"
    }
  }
}

provider "apple_ads" {
  # Prefer APPLEADS_* env vars locally. keep allow_campaign_deletion=false
  # so provider-level archival stays blocked even if prevent_destroy is removed.
  allow_campaign_deletion = false
}

variable "app_name" {
  type        = string
  description = "App Store name used to resolve the Adam ID (fictitious in docs)."
  default     = "Example Screenshot Organizer"
}

data "apple_ads_app" "app" {
  name = var.app_name
}

resource "apple_ads_campaign" "search" {
  name                  = "Search — Screenshot Organizer (example)"
  adam_id               = data.apple_ads_app.app.adam_id
  countries_or_regions  = ["US"]
  status                = "PAUSED"
  daily_budget_amount   = "25.00"
  daily_budget_currency = "USD"

  # Apple campaign deletion permanently archives the campaign. Even with the
  # provider's allow_campaign_deletion=false guardrail, prevent_destroy adds a
  # second Terraform-native layer against accidental destroy or config removal.
  lifecycle {
    prevent_destroy = true
  }
}

resource "apple_ads_ad_group" "core" {
  campaign_id               = apple_ads_campaign.search.id
  name                      = "Core terms"
  status                    = "PAUSED"
  default_bid_amount        = "1.25"
  default_bid_currency      = "USD"
  automated_keywords_opt_in = true
}

resource "apple_ads_keyword" "exact_primary" {
  ad_group_id  = apple_ads_ad_group.core.id
  text         = "screenshot organizer"
  match_type   = "EXACT"
  status       = "PAUSED"
  bid_amount   = "1.50"
  bid_currency = "USD"
}

resource "apple_ads_keyword" "exact_secondary" {
  ad_group_id  = apple_ads_ad_group.core.id
  text         = "screenshot manager"
  match_type   = "EXACT"
  status       = "PAUSED"
  bid_amount   = "1.25"
  bid_currency = "USD"
}

resource "apple_ads_keyword" "broad_discovery" {
  ad_group_id  = apple_ads_ad_group.core.id
  text         = "organize screenshots"
  match_type   = "BROAD"
  status       = "PAUSED"
  bid_amount   = "0.80"
  bid_currency = "USD"
}

resource "apple_ads_negative_keyword" "campaign_free" {
  campaign_id = apple_ads_campaign.search.id
  text        = "free"
  match_type  = "EXACT"
  status      = "ACTIVE"
}

resource "apple_ads_negative_keyword" "adgroup_cheap" {
  ad_group_id = apple_ads_ad_group.core.id
  text        = "cheap"
  match_type  = "BROAD"
  status      = "ACTIVE"
}

output "campaign_id" {
  value = apple_ads_campaign.search.id
}

output "ad_group_id" {
  value = apple_ads_ad_group.core.id
}

output "adam_id" {
  value = data.apple_ads_app.app.adam_id
}
