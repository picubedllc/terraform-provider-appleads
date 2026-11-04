# Platform API v1 client foundation

Issue: #72 · Parent: #70

## Package

`internal/client/v1` implements Apple Ads Platform API v1 transport alongside the existing Campaign Management API v5 client in `internal/client`.

| Concern | Behavior |
|---------|----------|
| Base URL | `https://api.ads.apple.com/v1/` |
| Context | `X-AP-Context: adAccountId=<id>` |
| Unscoped | `/acls`, `/me`, `/adaccounts` omit account context |
| Envelope | `result` (+ `pagination`) |
| Query | Shared `QueryRequest` / `QueryPage` / `QueryAll` |
| Money | Reuses decimal-string `client.Money` |
| Auth | Same OAuth client-assertion `TokenSource` as v5 |
| Retry | Compose with `client.WithRetry` on the HTTP client |

## Docs

- https://developer.apple.com/documentation/apple-ads-platform-api
- Platform API Preview Release Notes (July 2026): base URL, headers, query operations, ad accounts

## Compatibility

No Terraform schema changes in this PR. Provider wiring lands in #73.
