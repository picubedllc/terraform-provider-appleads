# apple_ads_app

Resolve an App Store app to its Adam ID for use with Apple Ads campaigns.

```hcl
data "apple_ads_app" "example" {
  name = "OrbitNote"
}
```

Or look up by Adam ID:

```hcl
data "apple_ads_app" "example" {
  id = "1234567890"
}
```

Exactly one of `name` or `id` must be set. When looking up by name, the match must be unique; if multiple apps share the same display name, the data source returns an error listing candidate IDs.
