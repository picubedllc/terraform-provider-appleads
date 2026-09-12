# Manages an Apple Ads ad group under an existing campaign.
# campaign_id is immutable — changing it returns an error instead of replacing
# the resource (which would destroy ad-group-level performance history).

resource "apple-ads_ad_group" "main" {
  campaign_id               = apple-ads_campaign.example.id
  name                      = "Search — core terms"
  status                    = "PAUSED"
  default_bid_amount        = "1.25"
  default_bid_currency      = "USD"
  automated_keywords_opt_in = true
}
