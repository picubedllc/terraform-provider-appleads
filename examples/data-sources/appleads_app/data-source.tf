data "appleads_app" "example" {
  name = "Screenbase"
}

output "adam_id" {
  value = data.appleads_app.example.adam_id
}
