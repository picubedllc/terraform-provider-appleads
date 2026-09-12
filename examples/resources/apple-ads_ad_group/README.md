# `apple-ads_ad_group`

Manages an Apple Ads ad group nested under a campaign.

## Import

```bash
# Preferred when campaign id is known:
terraform import apple-ads_ad_group.main 1234567890/9876543210

# Bare ad group id (provider resolves campaign via find):
terraform import apple-ads_ad_group.main 9876543210
```

`campaign_id` cannot be changed in place. Create a new ad group under the
target campaign instead of relying on Terraform replacement.
