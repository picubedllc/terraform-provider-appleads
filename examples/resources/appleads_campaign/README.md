# `appleads_campaign`

Manages an Apple Ads campaign.

## Bidding

| Strategy | Supply | Target CPA |
|---|---|---|
| `MANUAL_CPT` (default) | Search Results or Display supplies | Optional |
| `MAX_CONVERSIONS` | `APPSTORE_SEARCH_RESULTS` only | Required (`target_cpa_amount`) |

`bidding_strategy` and `target_cpa_amount` / `target_cpa_currency` are mutable —
you can switch strategies and change target CPA in place when the API allows.

Lifetime lifetime `budget_amount` / `budget_currency` are create-only (immutable after
create). Daily budget (`daily_budget_amount` / `daily_budget_currency`) stays mutable.

## Import

```bash
terraform import appleads_campaign.search_manual 1234567890
```

Immutable / create-only fields (`adam_id`, `countries_or_regions`, `supply_sources`,
`ad_channel_type`, `billing_event`, `budget_amount`, `budget_currency`) cannot change
in place. Create a new campaign resource instead of relying on Terraform replacement.
