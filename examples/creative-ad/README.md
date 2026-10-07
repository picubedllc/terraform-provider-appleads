# Display / Today Tab creative + ad wiring

Shows `appleads_campaign` (Display / `APPSTORE_TODAY_TAB`) → `appleads_ad_group`
→ `appleads_creative` (custom product page) → `appleads_ad`.

**Today Tab localization:** the custom product page app name and subtitle must
be localized for the default language of each campaign country or region.
Use `data.appleads_product_page_locales` (and
`data.appleads_countries_or_regions`) to verify before enabling the ad.

Creatives cannot be updated or deleted via API v5 — destroy only drops Terraform
state. Ads support name/status updates and delete; `creative_id` is immutable.
