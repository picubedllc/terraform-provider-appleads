# Platform API v1 — Compatibility inventory (Phase 0)

Tracking: #70 · This issue: #71

**Status:** Baseline for review. Do not merge broad Terraform schema changes until this inventory is accepted.

**Sources:**

- Current provider client: Campaign Management API **v5** (`https://api.searchads.apple.com/api/v5/`)
- Target: Apple Ads Platform API **v1** (`https://api.ads.apple.com/v1/`)
- [Apple Ads Platform API documentation](https://developer.apple.com/documentation/apple-ads-platform-api)
- Apple Ads Platform API Preview Release Notes (July 2026)
- Sunset: Campaign Management API v5 stops working **January 26, 2027**

**Branching:** Migration work targets `develop` (provider **1.0.0**). `main` remains **0.x.x** on v5 until the cutover release.

---

## 1. Structural differences (global)

| Concern | v5 (today) | v1 (target) |
|---------|------------|-------------|
| Base URL | `https://api.searchads.apple.com/api/v5/` | `https://api.ads.apple.com/v1/` |
| Account header | `X-AP-Context: orgId=<org_id>` | `X-AP-Context: adAccountId=<ad_account_id>` |
| Unscoped paths | `/acls`, `/me` omit context | Confirm Me/ACL/ad-account rules against docs |
| Success envelope | `data` | `result` |
| Collection lookup | nested paths + `/find` | flattened resources + `/query` |
| Auth | OAuth client assertion (JWT) | Same (preview: no auth change) |
| Money | `{ amount, currency }` decimal strings | Same pattern; keep `shopspring/decimal` / string attrs |

**Critical:** `orgId` and `adAccountId` are different identifiers. Never silently substitute one for the other.

---

## 2. HTTP client method and endpoint inventory (v5)

Default base: `internal/client.DefaultBaseURL`. Paths below are relative to that base.

### Transport / shared

| Method | Path / behavior |
|--------|-----------------|
| `DoJSON` / `DoJSONWithQuery` | JSON request/response; unwraps `data` |
| `FetchPage` / `FetchAllPages` | Offset pagination |
| `AuthTransport` | Bearer + `orgId` context (skipped for `/acls`, `/me`) |
| OAuth (`auth.go`) | Token URL + client assertion |
| Retry (`retry.go`) | Backoff / rate-limit handling |
| Errors (`errors.go`) | Typed `APIError`, request ID enrichment via ACL |

### ACLs

| Client method | HTTP | v5 path |
|---------------|------|---------|
| `ListUserACLs` | GET | `acls` |

### Campaigns

| Client method | HTTP | v5 path |
|---------------|------|---------|
| `CreateCampaign` | POST | `campaigns` |
| `GetCampaign` | GET | `campaigns/{id}` |
| `UpdateCampaign` | PUT | `campaigns/{id}` (body: `{"campaign":...}`) |
| `DeleteCampaign` | DELETE | `campaigns/{id}` (soft delete) |

### Ad groups

| Client method | HTTP | v5 path |
|---------------|------|---------|
| `CreateAdGroup` | POST | `campaigns/{campaignId}/adgroups` |
| `GetAdGroup` | GET | `campaigns/{campaignId}/adgroups/{id}` |
| `ListAdGroups` | GET | `campaigns/{campaignId}/adgroups` |
| `UpdateAdGroup` | PUT | `campaigns/{campaignId}/adgroups/{id}` (bare body) |
| `DeleteAdGroup` | DELETE | `campaigns/{campaignId}/adgroups/{id}` |
| `FindAdGroupByID` | POST | `adgroups/find` |

### Keywords (targeting)

| Client method | HTTP | v5 path |
|---------------|------|---------|
| `CreateKeyword` | POST | `campaigns/{c}/adgroups/{g}/targetingkeywords/bulk` |
| `GetKeyword` | GET | `campaigns/{c}/adgroups/{g}/targetingkeywords/{id}` |
| `UpdateKeyword` | PUT | `campaigns/{c}/adgroups/{g}/targetingkeywords/bulk` |
| `DeleteKeyword` | DELETE | `campaigns/{c}/adgroups/{g}/targetingkeywords/{id}` |
| `FindKeywordInCampaign` | POST | `campaigns/{c}/adgroups/targetingkeywords/find` |

### Negative keywords

| Client method | HTTP | v5 path |
|---------------|------|---------|
| `CreateCampaignNegativeKeyword` | POST | `campaigns/{c}/negativekeywords/bulk` |
| `CreateAdGroupNegativeKeyword` | POST | `campaigns/{c}/adgroups/{g}/negativekeywords/bulk` |
| `GetCampaignNegativeKeyword` | GET | `campaigns/{c}/negativekeywords/{id}` |
| `GetAdGroupNegativeKeyword` | GET | `campaigns/{c}/adgroups/{g}/negativekeywords/{id}` |
| `UpdateCampaignNegativeKeyword` | PUT | `campaigns/{c}/negativekeywords/bulk` |
| `UpdateAdGroupNegativeKeyword` | PUT | `campaigns/{c}/adgroups/{g}/negativekeywords/bulk` |
| `DeleteCampaignNegativeKeyword` | POST | `campaigns/{c}/negativekeywords/delete/bulk` |
| `DeleteAdGroupNegativeKeyword` | POST | `campaigns/{c}/adgroups/{g}/negativekeywords/delete/bulk` |
| `FindCampaignNegativeKeyword` | POST | `campaigns/{c}/negativekeywords/find` |

### Creatives

| Client method | HTTP | v5 path |
|---------------|------|---------|
| `CreateCreative` | POST | `creatives` |
| `GetCreative` | GET | `creatives/{id}` |
| `ListCreatives` | GET | `creatives` (paged) |
| `FindCreatives` / `FindCreativeByID` | POST | `creatives/find` |
| *(no update/delete in v5 client)* | — | — |

### Ads

| Client method | HTTP | v5 path |
|---------------|------|---------|
| `CreateAd` | POST | `campaigns/{c}/adgroups/{g}/ads` |
| `GetAd` | GET | `campaigns/{c}/adgroups/{g}/ads/{id}` |
| `ListAds` | GET | `campaigns/{c}/adgroups/{g}/ads` |
| `UpdateAd` | PUT | `campaigns/{c}/adgroups/{g}/ads/{id}` |
| `DeleteAd` | DELETE | `campaigns/{c}/adgroups/{g}/ads/{id}` |
| `FindAdsInCampaign` | POST | `campaigns/{c}/ads/find` |
| `FindAds` | POST | `ads/find` |

### Apps / product pages / geo / countries

| Client method | HTTP | v5 path |
|---------------|------|---------|
| `SearchApps` | GET | `search/apps` |
| `GetApp` / `FindAppByName` | (via search) | `search/apps` |
| `ListProductPages` | GET | `apps/{adamId}/product-pages` |
| `GetProductPage` | GET | `apps/{adamId}/product-pages/{productPageId}` |
| `ListProductPageLocales` | GET | `apps/{adamId}/product-pages/{productPageId}/locale-details` |
| `ListCountriesOrRegions` | GET | `countries-or-regions` |
| `SearchGeoLocations` | GET | `search/geo` |

### Reporting

| Client method | HTTP | v5 path |
|---------------|------|---------|
| `GetCampaignReport` | POST | `reports/campaigns` |
| `GetAdGroupReport` | POST | `reports/campaigns/{c}/adgroups` |
| `GetAdGroupKeywordReport` | POST | `reports/campaigns/{c}/adgroups/{g}/keywords` |
| `GetCampaignKeywordReport` | POST | `reports/campaigns/{c}/keywords` |
| `GetAdReport` | POST | `reports/campaigns/{c}/ads` |
| `GetCampaignSearchTermReport` | POST | `reports/campaigns/{c}/searchterms` |
| `GetAdGroupSearchTermReport` | POST | `reports/campaigns/{c}/adgroups/{g}/searchterms` |

Bid recommendations today are derived from keyword report `insights.bidRecommendation`, not a dedicated recommendations client.

---

## 3. v5 → v1 endpoint map (preview)

Paths relative to each API’s base. Confirm against live docs during Phase 1.

| Area | v5 | v1 |
|------|----|----|
| Campaigns CRUD | `/campaigns`, `/campaigns/{id}` | `/campaigns`, `/campaigns/{id}` |
| Campaigns find | `POST /campaigns/find` (if used) | `POST /campaigns/query` |
| Ad groups | nested under `/campaigns/{id}/adgroups` | `/adgroups`, `POST /adgroups/query` |
| Keywords | nested `targetingkeywords` (+ bulk) | `/keywords`, `/keywords/query`, `/keywords/bulk-*` |
| Negative keywords | nested `negativekeywords` (+ bulk) | `/negative-keywords`, `/negative-keywords/query`, `/negative-keywords/bulk-*` |
| Ads | nested under campaign/ad group | `/ads`, `POST /ads/query` |
| Creatives | `/creatives`, `/creatives/find` | `/creatives`, `POST /creatives/query` |
| Product pages | `/apps/{adamId}/product-pages...` | `/product-pages/query`, `/product-pages/{id}`, locale-details query |
| App search | `GET /search/apps` | `GET /search/apps` |
| App details | `GET /apps/{id}` | `GET /apps/{id}` |
| App eligibility | `POST /apps/{id}/eligibilities/find` | `POST /eligibilities/apps/query` |
| Budget orders | `/budgetorders` *(not yet a resource)* | `/shared-budgets` |
| Reports | nested `/reports/campaigns/...` | Apps reporting endpoints (new request/response shape) |
| Recommendations | report insights | Recommendations / suggestions APIs |

---

## 4. Request / response schema comparison (high level)

| Topic | v5 | v1 | Adapter notes |
|-------|----|----|---------------|
| Envelope | `data` | `result` | Client unwrap change |
| Query | `Selector` / find bodies | Common `/query` filters, sort, pagination | New query DTO |
| Campaign app ID | `adamId` | `promotedObjectId` + `promotedObjectType` | Map TF `adam_id` ↔ promoted object |
| Campaign bid | `biddingStrategy`, `targetCpa` | nested `bidStrategy` | Nested DTO |
| Campaign placement | `supplySources` | `targeting.supplyPlacement` (preview) | Likely nested targeting |
| Campaign countries | top-level `countriesOrRegions` | `targeting.countryOrRegion` (preview) | Nested targeting |
| Campaign daily budget | `dailyBudgetAmount` | `dailyBudget.value` (preview) | Money nest |
| Campaign org | `orgId` | `adAccountId` | Provider config |
| Serving reasons map | `countryOrRegionServingStateReasons` | `systemStatusLimitingReasons` (+ later storefront endpoint) | Computed attrs may change |
| Ad group bid | `defaultBidAmount` | `bidStrategy.bid` | Attribute keep vs wire rename |
| Ad group targeting | `targetingDimensions` | `targeting` | Geo may become integer IDs |
| Ad group `cpaCap` | present historically | sunset — do not send | Drop any usage |
| Keyword status | includes `ACTIVE` in some enums | `ENABLED` | Status normalize |
| Keyword parents | path IDs | body `adGroupId` / `campaignId` | Client only |
| Ad `creativeType` | on ad | removed — fetch creative | Computed attr risk |
| Creative shape | flat `adamId` / `productPageId` / `type` | `creativeType` + `destination` / parameters | Major adapter |
| Reports | flat times + `selector`; rows with `granularity` | `timeRange` + filters; `totalMetrics` / `granularMetrics` | Data source rewrite |
| Lifetime budget | supported | **deprecated / removed** | Align with create-only `budget_amount` behavior |

---

## 5. Terraform surface inventory

### Provider

| Attribute | Notes |
|-----------|--------|
| `org_id` | Today → `X-AP-Context: orgId=` |
| `client_id`, `team_id`, `key_id`, `private_key` | OAuth |
| `allow_campaign_deletion` | Safety gate |

**v1 gap:** need `ad_account_id` (+ env). Explicit migration for existing `org_id`-only configs.

### Resources

| Resource | Public attributes (summary) |
|----------|-----------------------------|
| `appleads_campaign` | `id`, `name`, `status`, `daily_budget_*`, `budget_*` (create-only), `budget_orders`, `adam_id`, `countries_or_regions`, `supply_sources`, `ad_channel_type`, `billing_event`, `bidding_strategy`, `target_cpa_*`, `start_time`, `end_time`, computed serving/payment/display/modification |
| `appleads_ad_group` | `id`, `campaign_id`, `name`, `status`, `default_bid_*`, `cpa_goal_*`, `automated_keywords_opt_in`, `pricing_model`, `start_time`, `end_time`, `targeting_dimensions`, computed serving/display/modification |
| `appleads_keyword` | `id`, `campaign_id`, `ad_group_id`, `text`, `match_type`, `status`, `bid_*`, `modification_time` |
| `appleads_negative_keyword` | `id`, `campaign_id`, optional `ad_group_id`, `text`, `match_type`, `status`, `modification_time` |
| `appleads_creative` | `id`, `org_id`, `adam_id`, `name`, `type`, `product_page_id`, `state`, `state_reasons`, times |
| `appleads_ad` | `id`, `campaign_id`, `ad_group_id`, `name`, `creative_id`, `creative_type`, `status`, serving fields, times |

### Data sources

| Data source | Role |
|-------------|------|
| `appleads_app` | App lookup |
| `appleads_product_pages` / `_page` / `_page_locales` | ASC product pages |
| `appleads_countries_or_regions` | Country metadata |
| `appleads_geolocations` | Locality ID lookup |
| `appleads_campaign_report` | Campaign metrics |
| `appleads_ad_group_report` | Ad group metrics |
| `appleads_keyword_report` | Ad-group keyword metrics |
| `appleads_campaign_keyword_report` | Campaign keyword metrics |
| `appleads_ad_report` | Ad metrics |
| `appleads_search_term_report` | Search terms |
| `appleads_keyword_bid_recommendations` | Suggested CPT (observational) |

---

## 6. Field classification (Terraform ↔ wire)

Legend: **U** unchanged semantics · **R** renamed on wire · **T** transformed shape · **X** removed/sunset · **N** newly required on v1

### Campaign

| Terraform / concept | Class | Notes |
|---------------------|-------|-------|
| Resource `id` (campaign id) | U | Preserve |
| `name`, `status`, times | U | |
| `adam_id` | T | ↔ `promotedObjectId` + type `APPSTORE_APP` |
| `bidding_strategy` / `target_cpa_*` | T | ↔ `bidStrategy` object |
| `supply_sources` | R/T | ↔ `supplyPlacement` / nested targeting |
| `countries_or_regions` | T | Nested under `targeting` in preview examples |
| `daily_budget_*` | T | Nested `dailyBudget` |
| `budget_*` (lifetime) | X risk | Lifetime budget deprecated on Platform API |
| `budget_orders` | R | Shared budgets / `sharedBudgets` / `budgetId` |
| `ad_channel_type` | ? | Confirm v1 field presence |
| `billing_event` | U | |
| `org` context | R | → `ad_account_id` |
| Serving reason maps | T | `systemStatusLimitingReasons` |

### Ad group

| Concept | Class | Notes |
|---------|-------|-------|
| `id`, `campaign_id` | U | `campaignId` moves to body on create |
| `default_bid_*` | T | ↔ `bidStrategy.bid` |
| `targeting_dimensions` | R/T | ↔ `targeting`; geo ID type may change |
| `cpa_goal_*` / caps | X risk | `cpaCap` sunset |
| `pricing_model`, automated keywords flags | U (confirm) | |

### Keyword / negative keyword

| Concept | Class | Notes |
|---------|-------|-------|
| IDs | U | Flattened GET by id |
| `status` ACTIVE→ENABLED | T | Normalize enums |
| `bid_*` | T | Nested `bid` object |
| Parents in path | T | Body/filters only |

### Creative / ad

| Concept | Class | Notes |
|---------|-------|-------|
| Creative `type` / product page / adam | T | `destination` + `creativeSpec`/`creativeType` |
| Creative `org_id` attr | R | Likely `ad_account_id` computed |
| Ad `creative_type` | X | Removed from ad; optional computed via creative GET |
| Soft delete | U | `deleted` flag |

### Reporting / recommendations

| Concept | Class | Notes |
|---------|-------|-------|
| Metric outputs | T | New metrics containers; project to existing TF attrs where equivalent |
| Time zone / granularity | T | New request body |
| Bid recommendations | T | New suggestions APIs |

---

## 7. State compatibility risks

| Risk | Severity | Mitigation |
|------|----------|------------|
| `org_id` ≠ `ad_account_id` | **High** | New required config; migration guide; no silent mapping |
| Nested campaign targeting changes plan for `countries_or_regions` / `supply_sources` | **High** | Adapters must round-trip equal membership; order-stable like today |
| Bid strategy nesting causes spurious diffs | **High** | DTO adapters; decimal scale normalization |
| Lifetime `budget_amount` unsupported | **High** | Diagnostics; document; avoid RequiresReplace if already create-only |
| Geo targeting string → int IDs | **High** | Inventory geolocation DS; plan for state rewrite or dual-read |
| Ad `creative_type` disappears from API | **Medium** | Keep computed via creative fetch or deprecate carefully |
| Creative destroy was state-only on v5 | **Medium** | Re-evaluate v1 delete; document |
| Keyword/negkw status enum ACTIVE vs ENABLED | **Medium** | Normalize on read/write |
| Report attribute renames / required `campaignId` filters | **Medium** | Preserve TF outputs; update required args only with changelog |
| Soft-deleted entities (`deleted: true`) | **Medium** | Keep existing remove-from-state / diagnostic behavior |
| Empty vs omitted targeting arrays clearing audiences | **High** | Partial update tests (Phase 5 item 7) |
| Import ID formats (`campaign_id/ad_group_id`, bare ids) | **Medium** | Keep parsers; verify GET-by-id still works without path parents |
| Forced replace to “simplify” migration | **Critical** | Forbidden unless true immutability |

---

## 8. Tests and fixtures

### Mocked HTTP (deterministic)

No checked-in `testdata/` JSON fixtures today. Coverage uses inline `httptest.Server` handlers in `*_test.go` under `internal/client` and `internal/provider`.

Notable client test files: `campaigns_api_test.go`, `adgroups_test.go`, `keywords_test.go`, `negative_keywords_test.go`, `creatives_test.go`, `ads_test.go`, `reports_test.go`, `product_pages_test.go`, `apps_test.go`, `geolocations_test.go`, `acls_test.go`, `auth_test.go`, `retry_test.go`, `transport_test.go`.

Provider unit tests mirror resources/data sources (create/read/update/delete/import/schema).

### Live / acceptance (`APPLEADS_LIVE_TEST` / `TF_ACC`)

**Client live probes** (`make testlive`):

- `auth_live_test.go`
- `campaigns_live_test.go`
- `campaign_create_probe_live_test.go`
- `campaign_capability_probe_live_test.go`
- `adgroup_create_probe_live_test.go`
- `adgroup_locality_live_test.go`
- `apps_live_test.go`
- `geolocations_live_test.go`

**Provider acceptance:**

- `provider_acc_test.go`
- `campaign_resource_acc_test.go`
- `ad_group_resource_acc_test.go`
- `keyword_resource_acc_test.go`
- `negative_keyword_resource_acc_test.go`
- `max_conversions_auto_adgroup_acc_test.go`
- `dependency_lifecycle_acc_test.go`

CI: unit/lint on PRs; live workflow is environment-gated and **not** PR-triggered. Do not run destructive live tests against production campaigns.

---

## 9. Explicit gaps / do-not-implement-yet

1. Broad Terraform schema renames (wait for review of this inventory).
2. Apple Maps / Brands product surface (out of scope for 1.0.0 parity unless forced).
3. New `appleads_budget_order` / shared-budget resource (tracked separately historically; only needed if campaign `budget_orders` cannot map).
4. Assuming v5 creative “no delete” without re-checking v1.
5. Silent `org_id` → `ad_account_id` conversion.
6. Using `float64` for money.
7. Leaking API DTOs into public Terraform models.

---

## 10. Recommended next steps

1. Review/approve this inventory on `develop` (#71).
2. Implement #72 v1 client foundation alongside v5.
3. Implement #73 provider `ad_account_id`.
4. Proceed resource-by-resource per merge order #74–#85.
