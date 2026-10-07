data "appleads_product_page" "example" {
  adam_id         = "899247964"
  product_page_id = "45812c9b-c296-43d3-c6a0-c5a02f74bf6e"
}

output "product_page_name" {
  value = data.appleads_product_page.example.name
}
