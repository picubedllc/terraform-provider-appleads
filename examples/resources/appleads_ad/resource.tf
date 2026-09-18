# Manages an Apple Ads ad (creative assigned to an ad group).
# ad_group_id and creative_id are immutable — changing them returns an error
# instead of replacing the resource (which would discard ad identity/history).

resource "appleads_ad" "cpp" {
  ad_group_id = appleads_ad_group.today_tab.id
  name        = "CPP ad"
  creative_id = appleads_creative.cpp.id
  status      = "PAUSED"
}
