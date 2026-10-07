# Manages an Apple Ads campaign.
#
# Immutable / create-only fields (adam_id, countries_or_regions, supply_sources,
# ad_channel_type, billing_event, budget_amount, budget_currency) never trigger
# automatic replacement — changing them returns an error so historical campaign
# identity is preserved. daily_budget_amount remains mutable in place.
#
# Manual CPT (default when bidding_strategy is omitted):

resource "appleads_campaign" "search_manual" {
  name                  = "Search — Manual CPT"
  adam_id               = data.appleads_app.app.adam_id
  countries_or_regions  = ["US"]
  status                = "PAUSED"
  daily_budget_amount   = "25.00"
  daily_budget_currency = "USD"
  ad_channel_type       = "SEARCH"
  supply_sources        = ["APPSTORE_SEARCH_RESULTS"]
  billing_event         = "TAPS"
  bidding_strategy      = "MANUAL_CPT"

  lifecycle {
    prevent_destroy = true
  }
}

# Maximize Conversions requires Search Results supply and target CPA:

resource "appleads_campaign" "search_max_conversions" {
  name                  = "Search — Max Conversions"
  adam_id               = data.appleads_app.app.adam_id
  countries_or_regions  = ["US"]
  status                = "PAUSED"
  daily_budget_amount   = "25.00"
  daily_budget_currency = "USD"
  ad_channel_type       = "SEARCH"
  supply_sources        = ["APPSTORE_SEARCH_RESULTS"]
  billing_event         = "TAPS"
  bidding_strategy      = "MAX_CONVERSIONS"
  target_cpa_amount     = "10.00"
  target_cpa_currency   = "USD"

  lifecycle {
    prevent_destroy = true
  }
}

# Display campaigns support MANUAL_CPT only (creatives/ads still required to serve):

resource "appleads_campaign" "display" {
  name                  = "Display — Today Tab"
  adam_id               = data.appleads_app.app.adam_id
  countries_or_regions  = ["US"]
  status                = "PAUSED"
  daily_budget_amount   = "25.00"
  daily_budget_currency = "USD"
  ad_channel_type       = "DISPLAY"
  supply_sources        = ["APPSTORE_TODAY_TAB"]
  billing_event         = "TAPS"
  bidding_strategy      = "MANUAL_CPT"

  lifecycle {
    prevent_destroy = true
  }
}
