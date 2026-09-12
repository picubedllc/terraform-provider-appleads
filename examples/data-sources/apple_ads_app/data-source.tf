data "apple_ads_app" "example" {
  name = "OrbitNote"
}

output "adam_id" {
  value = data.apple_ads_app.example.adam_id
}
