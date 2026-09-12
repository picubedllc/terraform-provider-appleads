# `apple-ads_negative_keyword`

Manages an Apple Ads negative keyword at **campaign** or **ad-group** scope.

Exactly one of `campaign_id` or `ad_group_id` must be set. `text` and `match_type`
are immutable — changing them returns an error instead of replacing the resource.

## Import

```bash
terraform import apple-ads_negative_keyword.campaign_free campaign/111/555
terraform import apple-ads_negative_keyword.adgroup_cheap adgroup/111/222/666
# Or without campaign id (resolved via ad group find):
terraform import apple-ads_negative_keyword.adgroup_cheap adgroup/222/666
```
