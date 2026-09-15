# appleads_geolocations

Look up Apple Ads targetable locations (Search for Geolocations) so ad groups
can set `targeting_dimensions.locality` / `admin_area` / `country` without
guessing IDs.

```hcl
data "appleads_geolocations" "nyc" {
  query        = "New York"
  entity       = "Locality"
  country_code = "US"
}
```

Locality IDs look like `US|NY|New York`. Use them on `appleads_ad_group`:

```hcl
resource "appleads_ad_group" "nyc_brand" {
  # ...
  targeting_dimensions = {
    locality = {
      included = ["US|NY|New York"]
    }
  }
}
```

Geo targeting only works for single-country campaigns. Query must be at least
three characters. At least one of `query` or `geo_id` is required.
