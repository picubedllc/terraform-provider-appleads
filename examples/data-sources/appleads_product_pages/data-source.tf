data "appleads_product_pages" "visible" {
  adam_id = "899247964"
  states  = ["VISIBLE"]
}

output "product_page_ids" {
  value = [
    for page in data.appleads_product_pages.visible.product_pages : page.id
  ]
}
