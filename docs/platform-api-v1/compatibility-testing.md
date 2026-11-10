# Phase 5 — Compatibility and acceptance testing

Issue: #83 · Parent: #70

## Required coverage checklist

1. Existing state refresh produces no unintended changes
2. Existing resource import preserves identity
3. A no-op plan remains a no-op
4. Mutable updates do not recreate campaigns
5. Immutable field changes produce correct diagnostics or replacement plans
6. Soft-deleted entities are handled correctly
7. Empty and omitted arrays do not clear targeting unexpectedly
8. All data sources return valid typed results
9. OAuth, ACL, and ad-account validation work
10. Live acceptance tests pass against Platform API v1

## How to run

```shell
make test                 # mocked / unit
make testlive             # APPLEADS_* + APPLEADS_LIVE_TEST=1
make testacc              # TF_ACC=1 + live
```

## Guardrails

- Mocked HTTP for deterministic coverage
- Live tests only against non-production ad accounts
- Do not enable destructive campaign archival in production
- Live workflow remains environment-gated (not PR-triggered)

## Mapping to client packages

| Area | Package |
|------|---------|
| Platform transport | `internal/client/v1` |
| Transitional v5 | `internal/client` |
| Acc helpers | `internal/acctest` (`APPLEADS_AD_ACCOUNT_ID` required) |
