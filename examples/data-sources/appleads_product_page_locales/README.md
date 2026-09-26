# appleads_product_page_locales

List locale details for a custom product page. Locale metadata is owned in App
Store Connect; the Ads API is read-only.

```hcl
data "appleads_product_page_locales" "example" {
  adam_id         = "899247964"
  product_page_id = "45812c9b-c296-43d3-c6a0-c5a02f74bf6e"
  device_classes  = ["IPHONE"]
}
```
