provider "apple-ads" {
  org_id      = var.apple_ads_org_id
  client_id   = var.apple_ads_client_id
  team_id     = var.apple_ads_team_id
  key_id      = var.apple_ads_key_id
  private_key = file(var.apple_ads_private_key_path)

  # Default is false. Set true only when intentional campaign archival is required.
  allow_campaign_deletion = false
}

# Credentials may instead be supplied via APPLEADS_* environment variables:
#   APPLEADS_ORG_ID
#   APPLEADS_CLIENT_ID
#   APPLEADS_TEAM_ID
#   APPLEADS_KEY_ID
#   APPLEADS_PRIVATE_KEY
#
# provider "apple-ads" {}
