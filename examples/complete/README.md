# Complete campaign example

End-to-end sample for managing an Apple Ads search campaign with this provider:

1. Resolve an App Store app via `data.apple_ads_app`
2. Create a paused campaign with a daily budget
3. Create an ad group with Search Match enabled
4. Add EXACT and BROAD targeting keywords
5. Add campaign-scoped and ad-group-scoped negative keywords

## Safety

- Provider `allow_campaign_deletion` is `false` in this example.
- The campaign resource also sets `lifecycle.prevent_destroy = true` as a second,
  Terraform-native guard. Apple campaign deletion permanently archives the
  campaign; both layers help prevent accidents.

## Usage

```bash
export APPLEADS_ORG_ID=...
export APPLEADS_CLIENT_ID=...
export APPLEADS_TEAM_ID=...
export APPLEADS_KEY_ID=...
export APPLEADS_PRIVATE_KEY="..."

cd examples/complete
terraform init
terraform plan -var='app_name=Your App Name'
```

All names, bids, and budgets here are fictitious documentation values.
