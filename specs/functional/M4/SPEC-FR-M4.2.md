# SPEC-FR-M4.2: AgentCommunity CRD Definition

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | SPEC-FR-M4.2                                |
| Status        | REJECTED                                    |
| Milestone     | M4                                          |
| Component     | operator                                    |
| Depends On    | SPEC-FR-M4.1                                |
| Supersedes    | none                                        |

## Rejection Rationale

This specification has been **rejected** on architectural grounds. A `TacitoCommunity` Kubernetes CRD must **not** be created.

### Why a TacitoCommunity CRD is Wrong

A **Community** is a purely logical, database-layer grouping concept owned by the **Keeper** component. It is:

- Persisted as a row in the PostgreSQL `communities` table.
- Managed exclusively via Keeper's REST API (`POST /api/v1/communities`, `GET`, `PUT`, `DELETE`).
- Referenced by `TacitoAgent` records via a `community_id` UUID foreign key.
- Used as a namespacing boundary for NATS subjects and agent isolation.

A Community has **no independent Kubernetes workload**, **no pods**, **no containers**, and **no runtime lifecycle** of its own in the cluster. The Kubernetes Operator's sole responsibility is to watch `TacitoAgent` CRDs and reconcile them into Deployments and headless Services. There is no Operator controller, no reconciler loop, and no K8s resource kind for communities.

### What "Community Lifecycle" Actually Means

The lifecycle endpoints `POST /api/v1/communities/:id/deploy` and `POST /api/v1/communities/:id/undeploy` (defined in `SPEC-FR-M4.7`) are **Keeper REST API calls** that iterate over all agent templates assigned to a community and bulk-submit or bulk-delete their corresponding `TacitoAgent` CRDs. Community state (`created`, `active`, `inactive`, `suspended`, `terminated`) is a database field, not a Kubernetes resource status.

### Correct Data Model

```
Keeper DB (PostgreSQL)
└── communities (id, name, status, ...)
    └── agents (id, community_id FK, status, ...)

Kubernetes API Server
└── TacitoAgent CRD (spec.communityRef = community UUID string)
    └── Deployment  ← reconciled by Operator
    └── headless Service ← reconciled by Operator
```

`TacitoCommunity` does not appear in the Kubernetes API Server at any point.

## Superseded By

None. This design decision is final. Any future requirement to represent community-level cluster state should open a new superseding spec with explicit justification.
