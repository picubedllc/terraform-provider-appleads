# Display (Today Tab) → ad group → creative → ad.
#
# Values are fictitious samples for documentation. Do not paste production
# org IDs, app IDs, product page UUIDs, or credentials into this example.
#
# Notes:
# - Display campaigns need creatives and ads to serve.
# - Today Tab (`APPSTORE_TODAY_TAB`) requires a CUSTOM_PRODUCT_PAGE creative.
# - Product page name/subtitle must be localized for each campaign country's
#   default language (verify with appleads_product_page_locales).
# - billing_event / full Display validators may land in separate PRs; this
#   example still shows creative ↔ ad wiring on a Display-shaped campaign.

terraform {
  required_providers {
    appleads = {
      source = "picubedllc/appleads"
    }
  }
}

provider "appleads" {
  allow_campaign_deletion = false
}

variable "adam_id" {
  type        = string
  description = "App Store Adam ID (fictitious in docs)."
  default     = "899247964"
}

data "appleads_product_pages" "visible" {
  adam_id = var.adam_id
  states  = ["VISIBLE"]
}

data "appleads_product_page_locales" "cpp" {
  adam_id         = var.adam_id
  product_page_id = data.appleads_product_pages.visible.product_pages[0].id
  device_classes  = ["IPHONE"]
}

resource "appleads_campaign" "today_tab" {
  name                  = "Today Tab — Screenshot Organizer (example)"
  adam_id               = var.adam_id
  countries_or_regions  = ["US"]
  ad_channel_type       = "DISPLAY"
  supply_sources        = ["APPSTORE_TODAY_TAB"]
  status                = "PAUSED"
  daily_budget_amount   = "50.00"
  daily_budget_currency = "USD"

  lifecycle {
    prevent_destroy = true
  }
}

resource "appleads_ad_group" "today_tab" {
  campaign_id          = appleads_campaign.today_tab.id
  name                 = "Today Tab"
  status               = "PAUSED"
  default_bid_amount   = "5.00"
  default_bid_currency = "USD"
  pricing_model        = "CPC"
  start_time           = "2026-01-01T00:00:00.000"

  targeting_dimensions = {
    device_class = {
      included = ["IPHONE"]
    }
  }
}

resource "appleads_creative" "cpp" {
  adam_id         = var.adam_id
  name            = "Today Tab CPP"
  type            = "CUSTOM_PRODUCT_PAGE"
  product_page_id = data.appleads_product_pages.visible.product_pages[0].id
}

resource "appleads_ad" "today_tab" {
  ad_group_id = appleads_ad_group.today_tab.id
  name        = "Today Tab ad"
  creative_id = appleads_creative.cpp.id
  status      = "PAUSED"
}
