# appleads_keyword_bid_recommendations

Read Apple's suggested CPT for targeting keywords. Observational only — do not copy `suggested_bid_amount` into `appleads_keyword.bid_amount`.

```hcl
data "appleads_keyword_bid_recommendations" "generic" {
  campaign_id = appleads_campaign.search.id
  ad_group_id = appleads_ad_group.generic.id
  start_time  = "2026-09-01"
  end_time    = "2026-09-14"
}
```
