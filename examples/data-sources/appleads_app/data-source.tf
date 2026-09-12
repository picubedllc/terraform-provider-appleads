data "appleads_app" "example" {
  name = "OrbitNote"
}

output "adam_id" {
  value = data.appleads_app.example.adam_id
}
