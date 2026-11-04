# Migrating to provider 1.0.0 (Platform API v1)

Issue: #84 · Parent: #70

Apple sunsets Campaign Management API v5 on **January 26, 2027**. Provider 1.0.0 targets Platform API v1 on `https://api.ads.apple.com/v1/`.

## Provider configuration

### Before (0.x)

```hcl
provider "appleads" {
  org_id      = "123456789"
  client_id   = "..."
  team_id     = "..."
  key_id      = "..."
  private_key = file("AuthKey.pem")
}
```

### After (1.0)

```hcl
provider "appleads" {
  ad_account_id = "987654321" # NOT the organization id
  org_id        = "123456789" # transitional during develop; see release notes
  client_id     = "..."
  team_id       = "..."
  key_id        = "..."
  private_key   = file("AuthKey.pem")
}
```

Environment equivalents: `APPLEADS_AD_ACCOUNT_ID`, `APPLEADS_ORG_ID`, and existing OAuth vars.

### Resolving `org_id` → `ad_account_id`

1. Authenticate with the same OAuth client used today.
2. List ad accounts via Platform API ad-account / ACL discovery (`ProviderData.V1` helpers).
3. Pick the ad account that owns the campaigns you manage.
4. **Never** copy `org_id` into `ad_account_id` unless Apple confirms they match for your account (they generally do not).

## State compatibility

| Resource IDs | Preserved when the Apple entity id is unchanged |
|--------------|--------------------------------------------------|
| Import formats | Keep existing `id` / `campaign_id/ad_group_id` forms where documented |
| Money attrs | Remain decimal strings |
| Lifetime `budget_amount` | Create-only; Platform API deprecates lifetime budgets — prefer daily budget |

Expect a one-time configure change for `ad_account_id`. Resource HCL attribute names are preserved where semantics match; wire-format renames are handled by adapters.

## Semantic version

**1.0.0** is recommended because:

- New required provider argument (`ad_account_id`)
- API host / context header change
- Possible computed attribute semantics shifts (serving/status reason fields)

## Changelog / release notes

See repository `CHANGELOG.md` Unreleased → 1.0.0 section (filled when cutting the release).
