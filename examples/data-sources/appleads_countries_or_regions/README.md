# appleads_countries_or_regions

List supported Apple Ads countries or regions (`GET /countries-or-regions`).

```hcl
data "appleads_countries_or_regions" "north_america" {
  countries_or_regions_filter = ["US", "CA"]
}
```

Omit `countries_or_regions_filter` to return the full supported set.
