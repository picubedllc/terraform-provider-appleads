# Decision: Campaign deletion safety

Status: Accepted  
Date: 2026-09-09  
Issue: [PI-7](https://linear.app/picubed/issue/PI-7/add-provider-level-destructive-operation-safety-design)

## Context

Apple Ads treats campaign deletion as permanent archival (soft delete with no practical undelete). Destroying a campaign throws away historical identity that matters for performance reporting and continuity.

Terraform's resource replacement model and casual `terraform destroy` usage make accidental archival a real risk for this provider.

## Decision

**Provider-level configuration is the primary guardrail.**

```hcl
provider "apple_ads" {
  allow_campaign_deletion = false # default
}
```

When `allow_campaign_deletion` is `false` (the default), any attempt to destroy/archive an `apple_ads_campaign` returns a clear error diagnostic and does not call Apple's delete API.

Users who intentionally need deletion must opt in:

```hcl
provider "apple_ads" {
  allow_campaign_deletion = true
}
```

## Alternatives considered

| Option | Outcome |
| --- | --- |
| Resource-level attribute (e.g. `force_delete`) | Rejected as primary control — easy to forget per resource; inconsistent defaults across configs. |
| Rely solely on Terraform `lifecycle { prevent_destroy = true }` | Rejected as primary control — opt-in per resource, not provider-enforced. Still recommended as a *second*, user-opt-in layer in examples. |
| Always allow deletion | Rejected — conflicts with the initiative's historical-identity safety principle. |

## Consequences

- Campaign `Delete` (Project 2) must read `ProviderData.AllowCampaignDeletion` and refuse when false.
- Documentation and README safety sections should describe this model.
- `prevent_destroy` remains a useful additional belt-and-suspenders control in user configs and complete examples.
