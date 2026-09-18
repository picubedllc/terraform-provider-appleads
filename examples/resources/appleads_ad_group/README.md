# `appleads_ad_group`

Manages an Apple Ads ad group nested under a campaign.

## Import

```bash
# Preferred when campaign id is known:
terraform import appleads_ad_group.main 1234567890/9876543210

# Bare ad group id (provider resolves campaign via find):
terraform import appleads_ad_group.main 9876543210
```

`campaign_id` cannot be changed in place. Create a new ad group under the
target campaign instead of relying on Terraform replacement.

City targeting is `targeting_dimensions.locality` (for example
`US|NY|New York`), not campaign `countries_or_regions`. Resolve IDs with
the `appleads_geolocations` data source. Geo targeting only works on
single-country campaigns. Changing campaign `countries_or_regions`
clears ad-group geo targeting.
