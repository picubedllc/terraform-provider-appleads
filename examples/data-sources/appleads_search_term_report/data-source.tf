# Campaign-scoped search-term report (Apple requires ORTZ; that is the default).
data "appleads_search_term_report" "campaign" {
  campaign_id = "1234567890"
  start_time  = "2026-09-01"
  end_time    = "2026-09-14"
}

# Ad-group-scoped search-term report.
data "appleads_search_term_report" "ad_group" {
  campaign_id = "1234567890"
  ad_group_id = "9876543210"
  start_time  = "2026-09-01"
  end_time    = "2026-09-14"
}
