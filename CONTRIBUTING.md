# Contributing

This is a Terraform Plugin Framework provider for Apple Ads. The goal is
configuration-as-code **without** silently archiving campaign identity.

## Local setup

- [Go](https://go.dev/doc/install) **1.25.8** (see `go.mod`)
- [Terraform](https://developer.hashicorp.com/terraform/install) **>= 1.0** (needed for `make generate` and acceptance tests)
- [golangci-lint](https://golangci-lint.run/) for `make lint`

```shell
git clone https://github.com/picubedllc/terraform-provider-appleads
cd terraform-provider-appleads
make fmt
make build
make test
```

Useful targets:

| Target | What it does |
| --- | --- |
| `make test` | Unit tests with `httptest` mocks. No Apple credentials. This is what PR CI runs. |
| `make testlive` | Read-only live tests (`APPLEADS_LIVE_TEST=1`). |
| `make testacc` | Full acceptance suite (`TF_ACC=1` and `APPLEADS_LIVE_TEST=1`). Creates real resources. |
| `make generate` | `terraform fmt` examples + [tfplugindocs](https://github.com/hashicorp/terraform-plugin-docs). |
| `make lint` | golangci-lint |

To hit a real Apple Ads org locally, export:

```
APPLEADS_ORG_ID
APPLEADS_CLIENT_ID
APPLEADS_TEAM_ID
APPLEADS_KEY_ID
APPLEADS_PRIVATE_KEY
```

The certificate must be allowed to manage campaigns. Read-only certs fail mutating calls. Destructive acceptance tests also need `APPLEADS_TEST_ADAM_ID` (the App Store Adam ID used as `adam_id`).

Do not commit credentials. PR CI never receives `APPLEADS_*` secrets.

Set `APPLEADS_HTTP_DEBUG=1` to log each Apple Ads HTTP attempt (method, redacted URL, status, content-type). Authorization headers are not logged.

## Architecture

Layering is Terraform-agnostic client first, then Plugin Framework resources.

```
internal/client     Apple Ads Campaign Management API v5 HTTP client
                    auth, transport, retries, typed errors, pagination
internal/provider   Terraform resources and data sources
internal/acctest    Shared live-test helpers (PreCheck, credentials, ACL probe)
```

Established in [PI-2](https://linear.app/picubed/issue/PI-2/implement-apple-ads-api-client) through [PI-7](https://linear.app/picubed/issue/PI-7/add-provider-level-destructive-operation-safety-design):

1. **`internal/client`** talks to `https://api.searchads.apple.com/api/v5`. It must not import Plugin Framework types.
2. **Auth** (`auth.go`) mints an OAuth client-assertion JWT and exchanges it for a bearer token (`TokenSource`).
3. **Transport** (`AuthTransport`) injects `Authorization` and `X-AP-Context: orgId=…`. Resources never set those headers.
4. **Retry** (`WithRetry` / `RetryTransport`) retries 429 and transient 5xx with backoff. Provider `Configure` wraps the authenticated client with retries.
5. **`internal/provider`** maps HCL ↔ client structs, classifies mutable vs immutable fields, and enforces `allow_campaign_deletion` (default `false`). See [campaign deletion safety](docs/decisions/campaign-deletion-safety.md).

Register new resources and data sources in `AppleAdsProvider.Resources` / `DataSources` in `internal/provider/provider.go`.

## Adding API endpoints

Put new methods next to the existing domain files:

- `campaigns.go` / `campaigns_api.go` — types vs HTTP
- `adgroups.go`, `keywords.go`, `negative_keywords.go`, `apps.go`

Use `Client.DoJSON` / `DoJSONWithQuery` for JSON endpoints. Failures come back as `*client.APIError` (status, Apple `messageCode`, message, `X-Request-Id`). Provider code should wrap those with `apiErrorDiagnostic` (or equivalent) rather than string-matching response bodies.

Helpers:

- `client.IsNotFound(err)` — HTTP 404
- `FetchPage[T]` / `FetchAllPages[T]` — offset pagination (`limit` / `offset`)
- Find endpoints take a `Selector` with `SelectorPagination`

Unit-test the client against `httptest.NewServer` in `internal/client/*_test.go` (`package client_test`). Inject `client.WithBaseURL(srv.URL)` and a fake `TokenSource`. See `client_test.go`.

## Adding resources

Follow `appleads_campaign` ([PI-9](https://linear.app/picubed/issue/PI-9/implement-appleads-campaign-schema) through [PI-15](https://linear.app/picubed/issue/PI-15/campaign-lifecycle-acceptance-tests)):

1. **Client types** — document mutable / immutable / computed on the Go struct (see `client.Campaign`).
2. **Schema** — Plugin Framework schema with `MarkdownDescription` on every attribute.
3. **Create** — validate plan, call the API, set state from the **API response**, not the plan (captures computed fields and server normalization).
4. **Read** — GET by id. Soft-deleted (`deleted=true`) or 404 removes the resource from state. Do not treat “missing from a list” as deletion.
5. **Update** — apply only mutable fields. Reject immutable drift with diagnostics.
6. **Delete** — campaigns must honor `allow_campaign_deletion`. Other resources may soft-delete via Apple’s API.
7. **Import** — parse a documented import id, GET, reject archived objects.
8. **Acceptance tests** — create → update mutable fields → import → destroy, plus an immutable-field rejection case.

### Never `RequiresReplace` on immutable fields

Apple campaign delete **archives** the campaign. Terraform `RequiresReplace` would destroy-then-create and throw away historical identity.

Canonical example: [PI-12](https://linear.app/picubed/issue/PI-12/implement-safe-campaign-update) / `internal/provider/campaign_update.go`. Detect immutable plan vs state changes and return an error telling the practitioner to create a **new** resource. The same rule applies to ad groups (`campaign_id`), keywords (`ad_group_id`, `text`, `match_type`), and negative keywords (scope, `text`, `match_type`).

Do not add `stringplanmodifier.RequiresReplace()` (or list equivalents) to those attributes.

## Adding schema fields

- Every attribute needs `MarkdownDescription`. tfplugindocs renders those into `docs/`.
- Mark secrets `Sensitive: true` (provider `private_key` is the template).
- Money is a **decimal string**, never `float64`. Validate with the existing money regexp pattern (see `budget_amount` on campaigns).
- Enums use `stringvalidator.OneOf`.
- Computed ids use `UseStateForUnknown()`.
- Put example HCL in `examples/resources/<resource_type>/` (or `examples/data-sources/<type>/`) so generated docs stay accurate.

## Testing

Default `go test ./...` / `make test` must stay offline.

| Layer | Location | Style |
| --- | --- | --- |
| Client | `internal/client/*_test.go` | `httptest.Server`, fake tokens, assert paths/headers/JSON |
| Provider | `internal/provider/*_test.go` | Schema, mapping, immutable diagnostics, delete guardrails |
| Live client | `internal/client/*_live_test.go` | Gated on `APPLEADS_LIVE_TEST=1` via `acctest.PreCheck` |
| Acceptance | `internal/provider/*_acc_test.go` | `TF_ACC=1` **and** `APPLEADS_LIVE_TEST=1` |

If `APPLEADS_LIVE_TEST=1` is set but credentials are missing, tests **Fatal** — they must not Skip. That is how CI would catch a miswired live workflow.

Name acceptance objects with a `tf-acc-` prefix (campaigns use `tf-acc-campaign-*`) so leftover test data is obvious.

## Acceptance tests

`resource.Test` cases live next to the resource. They must:

- Skip unless `TF_ACC=1`
- Call `testAccPreCheck` (which calls `acctest.PreCheck` and, for campaigns, requires `APPLEADS_TEST_ADAM_ID`)
- Set `CheckDestroy` so teardown actually archives/deletes in Apple when `allow_campaign_deletion = true`
- Cover import and immutable-field rejection

PR CI does **not** run these. The opt-in workflow is [PI-23](https://linear.app/picubed/issue/PI-23/add-acceptance-test-workflow):

- Read-only live tests: `.github/workflows/live-integration.yml` (`workflow_dispatch` + weekly cron, GitHub Environment `appleads-integration`)
- Mutating ACC: `make testacc` locally (or a future dedicated workflow with a write-capable cert). Do not put API Account Manager credentials on `appleads-integration`.

## Documentation generation

```shell
make generate
```

This formats `examples/` and runs tfplugindocs (`tools/tools.go`, provider name `appleads`). CI fails if `make generate` produces a diff ([PI-21](https://linear.app/picubed/issue/PI-21/set-up-tfplugindocs), [PI-22](https://linear.app/picubed/issue/PI-22/add-ci)).

Hand-written decision docs stay under `docs/decisions/`. Regenerated resource pages are `docs/resources/` and `docs/data-sources/`.

## Pull requests

- CI must pass: format, vet, lint, unit tests, `make generate` ([PI-22](https://linear.app/picubed/issue/PI-22/add-ci)).
- Run `make generate` and commit the result when schema or examples change.
- New resources need unit tests **and** acceptance tests before merge (ACC may be skip-gated; the test file still has to exist and be correct).
- Do not add Apple credentials, GPG material, or `.p8` keys to the repo or to PR CI.
- Link the Linear issue in the PR body (`Closes [PI-NN](https://linear.app/picubed/issue/…)`).

Releases (semver tags, GPG signing) are documented in [docs/RELEASING.md](docs/RELEASING.md).
