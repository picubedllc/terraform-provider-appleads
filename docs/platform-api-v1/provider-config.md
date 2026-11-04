# Provider config: `ad_account_id`

Issue: #73 · Parent: #70 · Depends on: #72

## Behavior

| Attribute | Role |
|-----------|------|
| `ad_account_id` | Required. Platform API v1 `X-AP-Context: adAccountId=…` |
| `org_id` | Required during develop transition for remaining v5 client calls |
| OAuth fields | Unchanged (`client_id`, `team_id`, `key_id`, `private_key`) |

Environment: `APPLEADS_AD_ACCOUNT_ID` (and existing `APPLEADS_*`).

## Compatibility

- **Breaking for 1.0.0:** configs without `ad_account_id` fail configure.
- No silent mapping from `org_id` → `ad_account_id`.
- Identical values emit a warning (likely copy-paste mistake).

## Docs

- https://developer.apple.com/documentation/apple-ads-platform-api
- Preview notes: request headers, ad account endpoints
