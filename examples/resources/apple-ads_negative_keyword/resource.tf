# Campaign-scoped and ad-group-scoped negatives share one resource type.
# Exactly one of campaign_id or ad_group_id must be set.

resource "apple-ads_negative_keyword" "campaign_free" {
  campaign_id = apple-ads_campaign.example.id
  text        = "free"
  match_type  = "EXACT"
  status      = "ACTIVE"
}

resource "apple-ads_negative_keyword" "adgroup_cheap" {
  ad_group_id = apple-ads_ad_group.main.id
  text        = "cheap"
  match_type  = "BROAD"
  status      = "ACTIVE"
}
