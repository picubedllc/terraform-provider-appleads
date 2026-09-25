## Unreleased

## 0.3.4

BUG FIXES:

* Send Apple Ads ad group updates as a bare partial body. PUT `/campaigns/{id}/adgroups/{id}` does not use an `{"adGroup":...}` envelope (unlike campaigns); the wrapper caused `UNRECOGNIZED_PROPERTY` on field `[adGroup]` when setting `targeting_dimensions` (including locality).

## 0.3.3

FEATURES:

* Add `appleads_creative` and `appleads_ad` resources for Apple Ads `/creatives` and campaign ad-group `/ads` endpoints. Creatives wire to product page IDs from the product-page data sources; Display / Today Tab docs note localization requirements. Creative destroy is state-only (API v5 has no creative delete). Immutable fields error instead of `RequiresReplace`.
* Add read-only `appleads_product_pages`, `appleads_product_page`, `appleads_product_page_locales`, and `appleads_countries_or_regions` data sources for App Store Connect product-page and country/region lookups via Apple Ads API v5. Product pages are owned in App Store Connect; the Ads API does not create or mutate them.
* Add optional `targeting_dimensions` on `appleads_ad_group` mapped to Apple Ads `targetingDimensions`, including geo (`locality`, `admin_area`, `country`) plus age, gender, `device_class`, `daypart`, and `app_downloaders`. City targeting uses locality IDs such as `US|NY|New York` (single-country campaigns only).
* Add `appleads_geolocations` data source for Search for Geolocations (`GET /search/geo`) so locality IDs can be resolved instead of hardcoding blindly.
* Add `appleads_keyword_bid_recommendations` data source to read Apple's suggested CPT (`insights.bidRecommendation`) from keyword reports. Observational only: do not copy `suggested_bid_amount` into `appleads_keyword.bid_amount`.
* Add read-only `appleads_campaign_report`, `appleads_ad_group_report`, and `appleads_keyword_report` data sources for impressions, taps, TTR, spend, average CPT, installs, conversion rate, and CPA.
* Add `appleads_search_term_report` data source for campaign- or ad-group-scoped search-term performance. Defaults `time_zone` to `ORTZ` (required by Apple). Observational only.

BUG FIXES:

* Include `selector.orderBy` on reporting requests (`localSpend` descending). Apple requires it on campaign, ad group, keyword, and search-term reports (`REQUIRED_INPUT_ORDER_BY_MISSING`).
* Keep configured `appleads_campaign.countries_or_regions` and `supply_sources` order when Apple returns the same set in a different order. Treat membership, not list index, as the immutable identity so multi-country creates do not fail Terraform's after-apply consistency check.
* Keep configured ad group `default_bid_amount` and keyword `bid_amount` decimal strings when Apple only changes scale (`1.00` → `1`). Omit keyword bids stay null in state even if Apple returns the ad group default.

## 0.2.2

BREAKING CHANGES:

* `appleads_ad_group` requires `pricing_model` (`CPC` or `CPM`) and `start_time`. Apple Ads create requires `pricingModel` (`REQUIRED_VALUE`) and `startTime` (`START_TIME_IS_REQUIRED`); the provider does not default omitted values ([#31](https://github.com/picubedllc/terraform-provider-appleads/pull/31)).

FEATURES:

* `pricing_model` accepts Apple's values: `CPC` (cost per tap; SEARCH / `TAPS`) and `CPM` (cost per thousand impressions). Changing `pricing_model` after create returns an error (no `RequiresReplace`).

## 0.2.1

BUG FIXES:

* Join Campaign Management API request paths under `/api/v5` ([#30](https://github.com/picubedllc/terraform-provider-appleads/pull/30)).
* Treat an empty Apple `budgetOrders` list as unset `budget_orders` so campaign create does not fail Terraform's after-apply consistency check.

ENHANCEMENTS:

* Include HTTP status in Apple Ads API error diagnostics.
* Add `APPLEADS_HTTP_DEBUG=1` request logging (method, redacted URL, status, content-type). Authorization headers are not logged.

## 0.2.0

Campaign create now matches Apple Ads Campaign Management API v5.

### Changed

- `POST /campaigns` includes organization context and SEARCH defaults when omitted: `adChannelType=SEARCH`, `supplySources=[APPSTORE_SEARCH_RESULTS]`, `billingEvent=TAPS`, `biddingStrategy=MANUAL_CPT`.
- `X-AP-Context: orgId=…` is sent on org-scoped calls and omitted on `GET /acls` and `GET /me`.

### Fixed

- A wrong or truncated `org_id` is reported against `GET /acls` instead of Apple's generic FORBIDDEN ("feature disabled").
- Configured money decimal strings stay in Terraform state when Apple only changes scale (for example `5.00` vs `5`).
- HTML CDN 5xx bodies are sanitized in diagnostics.

### Added

- Live campaign create probe gated by `APPLEADS_LIVE_TEST` (not PR CI).
