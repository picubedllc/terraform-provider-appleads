data "appleads_product_page_locales" "example" {
  adam_id         = "899247964"
  product_page_id = "45812c9b-c296-43d3-c6a0-c5a02f74bf6e"
  device_classes  = ["IPHONE"]
}

output "locale_codes" {
  value = [
    for loc in data.appleads_product_page_locales.example.locales : loc.language_code
  ]
}
