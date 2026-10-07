# `appleads_ad`

Manages an Apple Ads ad — the assignment of a creative to an ad group.

Display campaigns need creatives and ads to serve. Create `appleads_creative`
first, then attach it here.

## Import

```bash
# Preferred when campaign and ad group ids are known:
terraform import appleads_ad.cpp 111/222/333

# Ad group + ad (provider resolves campaign via find):
terraform import appleads_ad.cpp 222/333
```

`ad_group_id` and `creative_id` cannot be changed in place. Create a new ad
instead of relying on Terraform replacement. `name` and `status` are mutable.
