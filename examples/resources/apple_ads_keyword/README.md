# `apple_ads_keyword`

Manages an Apple Ads targeting keyword nested under an ad group.

## Import

```bash
# Preferred when campaign and ad group ids are known:
terraform import apple_ads_keyword.exact 111/222/333

# Ad group + keyword (provider resolves campaign via find):
terraform import apple_ads_keyword.exact 222/333
```

`ad_group_id`, `text`, and `match_type` cannot be changed in place. Create a new
keyword instead of relying on Terraform replacement.
