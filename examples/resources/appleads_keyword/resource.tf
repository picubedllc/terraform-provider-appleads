# Manages an Apple Ads targeting keyword under an ad group.
# text and match_type are immutable — changing them returns an error instead of
# replacing the resource (which would destroy keyword-level performance history).

resource "appleads_keyword" "exact" {
  ad_group_id  = appleads_ad_group.main.id
  text         = "screenshot organizer"
  match_type   = "EXACT"
  status       = "PAUSED"
  bid_amount   = "1.50"
  bid_currency = "USD"
}

resource "appleads_keyword" "broad" {
  ad_group_id  = appleads_ad_group.main.id
  text         = "screenshot app"
  match_type   = "BROAD"
  status       = "PAUSED"
  bid_amount   = "0.75"
  bid_currency = "USD"
}
