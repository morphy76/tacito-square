# SPEC-FR-M6.5.10: Resource Tier Assignment on Community Assignment

| Field       | Value |
|-------------|-------|
| ID          | SPEC-FR-M6.5.10 |
| Status      | DRAFT |
| Milestone   | M6.5 |
| Component   | keeper, operator |
| Depends On  | SPEC-FR-M6.5.1, SPEC-FR-M5.9 |
| Supersedes  | none |

## Context

SPEC-FR-M5.9 defined flexible runtime tiers (CPU/memory profiles: `nano`, `micro`, `small`, `medium`, `large`). The `Agent.Tier string` field exists in the keeper model. When an agent is assigned to a community, the role it receives implies different resource needs: hubs orchestrate multiple spokes and maintain delegated context in Redis, requiring more headroom than spokes. This spec defines role-driven tier recommendations at assignment time.

## Specification

### 1. Role-to-Tier Recommendation Mapping

| Role | Default Recommended Tier |
|------|--------------------------|
| `standalone` | `small` |
| `hub` | `medium` |
| `spoke` | `small` |

### 2. Tier Assignment Logic (in Keeper assignment service)

At assignment time, the service:
1. If `Agent.Tier` is empty or `nano`, set `Agent.Tier` to the role-recommended tier.
2. If the API request body includes an explicit `tier` field, use that value instead (caller override).
3. Validate that the effective tier is a known value (`nano`, `micro`, `small`, `medium`, `large`).
4. If a `hub` role is assigned with a tier below `medium` (i.e., `nano` or `micro`), emit a **warning** in the response body — not an error. The assignment still succeeds.

### 3. Operator Application

The Operator reconciler reads `Agent.Tier` from the `TacitoAgent` CRD `spec.tier` field and applies the corresponding CPU/memory `resources.requests` and `resources.limits` from the tier profile ConfigMap (already defined in M5.9 infrastructure).

## Acceptance Criteria

1. Assigning a `hub` agent without specifying a tier results in `Agent.Tier = medium`.
2. Assigning a `spoke` agent without specifying a tier results in `Agent.Tier = small`.
3. Assigning a `hub` with `tier = nano` succeeds; the response includes `warnings: ["hub agent assigned with tier 'nano', recommended minimum is 'medium'"]`.
4. Specifying an explicit `tier` in the request body overrides the recommendation.
5. The Operator applies CPU/memory resource limits matching the effective tier.
6. An invalid `tier` value returns `422 Unprocessable Entity` with `{"error": "..."}`.

## Test Plan

- **Unit**: Tier recommendation logic in the assignment domain service for all three roles.
- **Unit**: Warning emission for hub assigned below `medium`.
- **Integration**: Keeper API assignment endpoint returns `effective_tier` and optional `warnings` in the response body.
- **Manual**: Verify pod resource limits via `kubectl describe pod` match the tier profile.

## API Contract

```
POST /api/v1/communities/{id}/agents
     Body: { "agent_id": "uuid", "role": "hub|spoke|standalone", "tier"?: "nano|micro|small|medium|large" }
     Response 200: {
       "agent_id": "uuid",
       "role": "hub",
       "effective_tier": "medium",
       "assigned_at": "2026-...",
       "warnings": []   // or ["hub agent assigned with tier 'nano'..."]
     }
```

## Files Affected

- `internal/keeper/application/service/community_service.go` [MODIFY] — tier recommendation logic
- `internal/keeper/adapters/inbound/http/community_handler.go` [MODIFY] — `tier` in request/response
- `internal/operator/application/service/reconcile_service.go` [MODIFY] — apply tier profile to pod spec
