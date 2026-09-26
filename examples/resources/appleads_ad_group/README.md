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
single-country campaigns.

## Maximize Conversions automated ad group

When you create an `appleads_campaign` with `bidding_strategy = "MAX_CONVERSIONS"`,
Apple may auto-create an Automated Ad Group (Search Match on, broad audience,
default product page ad) **outside Terraform state**. Import it with the formats
above — there is no separate automated-ad-group resource type.

Recommended workflow:

1. Create the Max Conversions campaign (`target_cpa_amount` required; Search Results only).
2. List ad groups for the new campaign id (API or UI). Creation can lag a few seconds;
   the campaign may report an ad-group-missing serving state until the group appears.
3. Import with `campaign_id/ad_group_id` (preferred) or bare `ad_group_id`.
4. Keep managing `target_cpa_amount` on the campaign. Do not declare a second
   Terraform ad group that duplicates Apple's automated one.

### Known limitations under Max Conversions

- **Keywords:** Semantics differ from Manual CPT. Apple's auto-bidder drives
  delivery; keywords on the automated group are closer to guides/pauses than
  classic CPT bid control.
- **Default bid:** Ad group `default_bid_amount` may be ignored or constrained
  by the campaign-level target CPA auto-bidder. Prefer adjusting
  `target_cpa_amount` on the campaign.
- **Display:** Max Conversions does not apply to Display channel / Display supplies.
