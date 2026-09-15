data "appleads_keyword_bid_recommendations" "generic" {
  campaign_id = "1234567890"
  ad_group_id = "9876543210"
  start_time  = "2026-09-01"
  end_time    = "2026-09-14"
}

output "suggested_bids" {
  value = {
    for row in data.appleads_keyword_bid_recommendations.generic.keywords :
    row.text => {
      bid       = row.bid_amount
      suggested = row.suggested_bid_amount
      bid_min   = row.bid_min_amount
      bid_max   = row.bid_max_amount
    }
  }
}
