# `appleads_budget_order`

Manages an Apple Ads budget order used for LOC / agency billing workflows.
Campaigns attach managed orders through `budget_orders = [appleads_budget_order.example.id]`.

Money amounts are decimal strings (for example `"5000.00"`), never floating point.

## Import

```bash
terraform import appleads_budget_order.q1 542370539
```

## Destroy

Apple Ads API v5 has no delete endpoint for budget orders. Destroy removes the
resource from Terraform state only; the order remains in Apple Ads.
