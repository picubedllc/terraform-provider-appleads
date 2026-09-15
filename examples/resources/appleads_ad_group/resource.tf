# Manages an Apple Ads ad group under an existing campaign.
# campaign_id is immutable — changing it returns an error instead of replacing
# the resource (which would destroy ad-group-level performance history).

resource "appleads_ad_group" "main" {
  campaign_id               = appleads_campaign.example.id
  name                      = "Search — core terms"
  status                    = "PAUSED"
  default_bid_amount        = "1.25"
  default_bid_currency      = "USD"
  pricing_model             = "CPC"
  start_time                = "2026-01-01T00:00:00.000"
  automated_keywords_opt_in = true
}

resource "appleads_ad_group" "today_tab" {
  campaign_id          = appleads_campaign.display.id
  name                 = "Today Tab"
  status               = "PAUSED"
  default_bid_amount   = "5.00"
  default_bid_currency = "USD"
  pricing_model        = "CPM"
  start_time           = "2026-01-01T00:00:00.000"
}

# City targeting is ad-group targeting_dimensions.locality (Apple Ads
# targetingDimensions). Campaign countries_or_regions stays US (immutable).
# Look up IDs with data.appleads_geolocations; NYC locality is US|NY|New York.

resource "appleads_ad_group" "nyc_brand" {
  campaign_id          = appleads_campaign.example.id
  name                 = "NYC — brand"
  status               = "PAUSED"
  default_bid_amount   = "1.25"
  default_bid_currency = "USD"
  pricing_model        = "CPC"
  start_time           = "2026-01-01T00:00:00.000"

  targeting_dimensions = {
    locality = {
      included = ["US|NY|New York"]
    }
  }
}

