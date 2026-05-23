# SPEC-NFR-MULTITENANCY: Multitenancy Architecture

| Field         | Value                              |
|---------------|------------------------------------|
| ID            | SPEC-NFR-MULTITENANCY              |
| Status        | ACCEPTED                           |
| Component     | all                                |

## Specification

To support multiple tenants securely and efficiently, the system implements multitenancy in two distinct fashions depending on the deployment unit:

### 1. Dedicated Single-Tenant Units (Dedicated Deployments)
*   **Definition**: Units such as individual **Agents** or **Agent Communities** are deployed as dedicated instances for a specific tenant.
*   **Shared Infrastructure**: These dedicated units leverage shared backing services and states (e.g., shared PostgreSQL database, shared Redis cache, shared Qdrant collection) but all persisted entities and state elements are qualified/scoped within the tenant.
*   **Resolution Mechanism**: The tenant resolution for these dedicated units is driven strictly by the **environment configuration** (e.g., K8s environment variables injected during provisioning). This configuration must match and satisfy any runtime requests handled by the unit.
*   **Isolation**: Ensures that tenant-specific logic and processing are physically isolated at the compute layer (pod level), while keeping data isolated at the logical schema/filtering layer.

### 2. Shared Multi-Tenant Units (Shared Services)
*   **Definition**: Components that are shared across multiple tenants (e.g., **Keeper**, **BFF**, shared API gateways, shared event subscribers).
*   **Resolution Mechanism**: These shared units resolve the tenant dynamically at runtime from incoming requests:
    *   **REST Calls (Unauthenticated)**: Extracted from custom HTTP request headers (e.g., `X-Tenant-ID`).
    *   **REST Calls (Authenticated)**: Resolved dynamically by parsing the Bearer JWT token and extracting the tenant identifier from the JWT claims (e.g., OIDC/OAuth2 claims mapping).
    *   **Messaging (NATS/Pub-Sub)**: Extracted from message metadata or NATS message headers (e.g., `tenant_id` header or subject namespace matching `/tenants/{tenant_id}/...`).
*   **Isolation**: Logic inside the shared components must dynamically propagate the resolved tenant context (e.g., using `context.Context` in Go) to all downstream services, repositories, and caches, ensuring logical partition and strict access controls.

## Acceptance Criteria

1.  **Dedicated Units**:
    *   Any dedicated Agent or Community deployment MUST read its tenant identity from environment variables (e.g., `TENANT_ID`).
    *   All outbound port calls (database, cache, memory) executed by these units must qualify requests using this statically injected tenant identity.
    *   A dedicated unit MUST reject or fail to handle any request that mismatches its configured tenant identity.
2.  **Shared Units**:
    *   Shared services MUST implement middleware (e.g., HTTP request interceptors, NATS header decoders) to dynamically resolve the tenant ID on every request.
    *   If tenant resolution fails or is missing, requests must be rejected with appropriate authorization or validation errors (e.g., `401 Unauthorized` or standard JSON error format).
    *   The resolved tenant context must be propagated safely across package boundaries via Go's `context.Context` and enforced at the persistence layer (e.g., pgx queries using tenant-qualified filtering).
