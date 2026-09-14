## Unreleased

- Send `pricingModel` and `startTime` on ad group create when omitted. Terraform `pricing_model` accepts `CPC` or `CPM` and defaults to `CPC`.
- Join Campaign Management API request paths under `/api/v5`.
- Treat an empty `budgetOrders` list as unset `budget_orders`.
- Include HTTP status in Apple Ads API error diagnostics.
- Add `APPLEADS_HTTP_DEBUG=1` request logging (method, redacted URL, status, content-type).
- Treat an empty `budgetOrders` list as unset `budget_orders`.
- Include HTTP status in Apple Ads API error diagnostics.
- Add `APPLEADS_HTTP_DEBUG=1` request logging (method, redacted URL, status, content-type).
