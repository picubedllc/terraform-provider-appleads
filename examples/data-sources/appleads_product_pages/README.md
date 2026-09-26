# appleads_product_pages

List custom product pages for an Adam ID. Pages are owned in App Store Connect;
the Ads API is read-only.

```hcl
data "appleads_product_pages" "visible" {
  adam_id = "899247964"
  states  = ["VISIBLE"]
}
```

Optional `name` and `states` (`VISIBLE`, `HIDDEN`) filter results. An empty
`product_pages` list means no matches, not an error. Use returned `id` values
when creating creatives/ads.
