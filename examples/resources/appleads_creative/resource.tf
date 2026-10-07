# Manages an Apple Ads creative (organization-scoped).
# Wire product_page_id from product page data sources for CUSTOM_PRODUCT_PAGE.
# adam_id, name, type, and product_page_id are immutable — changing them returns
# an error instead of replacing the resource. Destroy removes Terraform state
# only (API v5 has no creative delete).

data "appleads_product_pages" "visible" {
  adam_id = "899247964"
  states  = ["VISIBLE"]
}

resource "appleads_creative" "cpp" {
  adam_id         = "899247964"
  name            = "Summer CPP creative"
  type            = "CUSTOM_PRODUCT_PAGE"
  product_page_id = data.appleads_product_pages.visible.product_pages[0].id
}

resource "appleads_creative" "default_pp" {
  adam_id = "899247964"
  name    = "Default product page creative"
  type    = "DEFAULT_PRODUCT_PAGE"
}
