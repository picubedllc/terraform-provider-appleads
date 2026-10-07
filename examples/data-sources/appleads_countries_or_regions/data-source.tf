data "appleads_countries_or_regions" "north_america" {
  countries_or_regions_filter = ["US", "CA"]
}

output "supported_us_languages" {
  value = [
    for lang in data.appleads_countries_or_regions.north_america.countries_or_regions[0].supported_languages : lang.language_code
  ]
}
