## Unreleased

- Fix API path joining so requests go to `/api/v5/...` instead of `/api/...`. The previous `url.ResolveReference` join dropped the `v5` segment, and Apple's CDN answers the wrong path with an HTML 503.
- Map an empty Apple `budgetOrders` list to Terraform `null` so omitted `budget_orders` does not fail apply with an inconsistent result.
- Include HTTP status in Terraform diagnostics for Apple Ads API errors.
- Add `APPLEADS_HTTP_DEBUG=1` request logging (method, redacted URL, status, content-type; no tokens).
