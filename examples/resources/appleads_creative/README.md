# `appleads_creative`

Manages an Apple Ads creative. Prefer `CUSTOM_PRODUCT_PAGE` with a product page
id from `data.appleads_product_pages` / `data.appleads_product_page`.

Display / Today Tab campaigns require an approved custom product page creative
with locales matching each campaign country's default language.

## Import

```bash
terraform import appleads_creative.cpp 573408745
```

`adam_id`, `name`, `type`, and `product_page_id` cannot be changed in place.
Create a new creative instead of relying on Terraform replacement. Destroy
removes Terraform state only (no Apple delete endpoint).
