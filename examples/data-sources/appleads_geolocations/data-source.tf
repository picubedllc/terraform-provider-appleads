data "appleads_geolocations" "nyc" {
  query        = "New York"
  entity       = "Locality"
  country_code = "US"
}

output "nyc_locality_ids" {
  value = [
    for loc in data.appleads_geolocations.nyc.locations : loc.id
  ]
}
