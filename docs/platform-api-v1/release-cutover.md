# Phase 6 — Cut 1.0.0 from develop to main

Issue: #85 · Parent: #70

## Preconditions

- [ ] #71–#84 merged to `develop`
- [ ] Phase 5 checklist green on a non-production ad account
- [ ] Migration guide reviewed
- [ ] Generated docs (`make generate`) committed
- [ ] CHANGELOG 1.0.0 section complete

## Cutover steps

1. Open PR: `develop` → `main` summarizing the Platform API v1 migration.
2. Merge after review (no force-push).
3. Tag `v1.0.0` per `docs/RELEASING.md`.
4. Confirm GitHub Release / registry artifacts.
5. Close tracking issue #70.

## Guardrails

- Keep `main` on 0.x until this checklist is complete.
- Do not publish 1.0.0 from a partial `develop`.
- Call out Apple’s January 26, 2027 Campaign Management API v5 sunset in release notes.
