# SPEC-FR-M11.3: Spawning MCP Servers using CRD from Keeper

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | SPEC-FR-M11.3                               |
| Status        | DRAFT                                       |
| Milestone     | M11                                         |
| Component     | keeper, operator                            |
| Depends On    | SPEC-FR-M3.2, SPEC-FR-M4.1, SPEC-FR-M4.6     |
| Supersedes    | none                                        |

## Context

To perform actions and interact with external development environments, filesystems, databases, or third-party APIs, agents utilize **Model Context Protocol (MCP)** clients. The keeper service manages a registry of **MCP Server Configurations** (represented as `MCPClient` configurations in the database).

Currently, these configurations support two execution profiles at runtime:
1.  **Local Command Exec (stdio)**: The agent container spawns the MCP server as a subprocess. This couples runtime dependencies, duplicates server instances for every agent replica (wasting CPU/memory), and lacks network/resource isolation boundaries.
2.  **Remote Connection (sse)**: The agent connects to a pre-existing HTTP/SSE endpoint hosted externally.

To resolve these limitations, this specification introduces **Standalone MCP Server Workloads** in Kubernetes. MCP servers will be spawned dynamically as dedicated, isolated cluster resources managed by a custom resource definition (CRD) called `TacitoMCPServer`. These standalone workloads serve as a new, specialized image/component acting as a sort of Function-as-a-Service (FaaS) for custom business logic and tool execution.

When an MCP configuration is deployed via Keeper, Keeper constructs and submits a `TacitoMCPServer` CRD to the Kubernetes API server. The Operator reconciles this resource into an independent `Deployment` and `Service` running the designated tool/business logic image. When an agent using this MCP server is deployed, Keeper dynamically overrides its local stdio transport to SSE, pointing directly to the spawned FaaS-like MCP service's internal cluster DNS endpoint.

---

## Specification

### 1. `TacitoMCPServer` Kubernetes Custom Resource Definition (CRD)
A namespace-scoped CustomResourceDefinition `TacitoMCPServer` MUST be registered in the API group `tacito.square.io/v1alpha1`.

*   **Go struct location**: `pkg/kubernetes/apis/tacito/v1alpha1/tacitomcpserver_types.go` (shared between Keeper and Operator).
*   **Resource Naming**: To conform to RFC 1123 DNS subdomain requirements, the generated resource name in Kubernetes MUST use the format:
    ```
    u-<lowercase-mcp-server-UUID>
    ```
*   **CRD Spec Fields**:
    *   `tenantId` (string, required): The UUID of the tenant owning the resource.
    *   `serverName` (string, required): The unique alphanumeric name of the MCP server.
    *   `image` (string, required): The container image containing the MCP server executable.
    *   `command` ([]string, optional): Entrypoint override command.
    *   `args` ([]string, optional): Command line arguments.
    *   `env` (map[string]string, optional): Key-value environment variables.
    *   `authSecretRef` (string, optional): Reference to a K8s secret containing sensitive tokens/keys.
    *   `replicas` (int32, default: 1): Desired number of pod replicas.
    *   `port` (int32, default: 8080): Port the SSE server listens on inside the container.
    *   `resources` (ResourceRequirements, optional): CPU and memory requests/limits.
*   **CRD Status Fields**:
    *   `phase` (string): Lifecycle state (`Pending`, `Running`, `Error`, `Terminated`).
    *   `url` (string): The resolved connection URL (e.g., `http://u-<uuid>.<namespace>.svc.cluster.local:8080/sse`).
    *   `conditions` ([]metav1.Condition): Real-time observations.

### 2. Database Schema Extensions in Keeper
The `mcp_clients` (or `mcp_servers`) database table MUST be updated to support container-specific metadata for standalone deployment:
*   Add `image` (varchar, nullable) to store the container image.
*   Add `mcp_port` (integer, default: 8080) to store the container port.

These fields MUST be added to the domain `model.MCPClient` aggregate struct and validated during domain verification. If a standalone deployment is requested, `image` MUST NOT be empty.

### 3. MCP Server Lifecycle API Endpoints (Keeper Component)
To manage running MCP workloads independently, Keeper MUST expose three lifecycle HTTP REST endpoints:

#### A. Deploy MCP Server (Workload Spawn)
*   **Endpoint**: `POST /api/v1/mcp-servers/:id/deploy`
*   **Behavior**:
    *   The dynamic OIDC/JWT tenant context MUST match the resource owner.
    *   The MCP configuration MUST have `image` specified.
    *   Keeper invokes the outbound `MCPServerCRDCoordinator` to build and submit the `TacitoMCPServer` CRD.
    *   Status in the database transitions to `active`.
*   **Responses**:
    *   `202 Accepted` on successful CRD submission.
    *   `400 Bad Request` if `image` is not configured.
    *   `409 Conflict` if already in a deployed state.

#### B. Undeploy MCP Server (Workload Teardown)
*   **Endpoint**: `POST /api/v1/mcp-servers/:id/undeploy`
*   **Behavior**:
    *   Keeper deletes the corresponding `TacitoMCPServer` CRD from the cluster.
    *   Status in the database transitions to `inactive`.
*   **Responses**:
    *   `200 OK` on successful CRD deletion.

#### C. Get MCP Server Workload Status
*   **Endpoint**: `GET /api/v1/mcp-servers/:id/status`
*   **Behavior**:
    *   Queries real-time phase, replicas, and connection URL from the `TacitoMCPServer` CRD status subresource via the outbound coordinator.
*   **Responses**:
    *   `200 OK` returning:
        ```json
        {
          "mcp_server_id": "uuid",
          "status": "running|pending|inactive|error",
          "url": "http://u-<uuid>.<namespace>.svc.cluster.local:8080/sse",
          "replicas": 1
        }
        ```

### 4. Operator Reconciler Loop (`TacitoMCPServer` Controller)
A new reconciler controller `TacitoMCPServerReconciler` MUST be registered in the Operator:
*   Watches custom resources of type `TacitoMCPServer` in the cluster.
*   For each custom resource:
    1.  **Reconcile Deployment**: Spawns a K8s `Deployment` running the configured `image`, injecting `command`, `args`, and `env` overrides. If `authSecretRef` is provided, the secret is mapped as environment variables or secret volumes.
    2.  **Reconcile Service**: Creates a ClusterIP `Service` targeting the deployment pods on the specified container `port`.
    3.  **Update CRD Status**: Updates `status.phase` to `Running` once deployment replicas are healthy, and sets `status.url` to the internal ClusterIP service URL (e.g., `http://u-<uuid>.<namespace>.svc.cluster.local:8080/sse`).

### 5. Dynamic Connection Injection in Agent Compilation
When Keeper constructs the `TacitoAgent` custom resource during Agent deployment:
1.  Keeper fetches the active `MCPClient` configurations associated with the Agent.
2.  For each associated MCP client:
    *   If the MCP server has an active `TacitoMCPServer` deployed in the cluster, Keeper MUST dynamically translate its configuration:
        *   Override `transport` to `sse`.
        *   Override `url` to the resolved internal ClusterIP service DNS (e.g., `http://u-<uuid>.<namespace>.svc.cluster.local:8080/sse`).
3.  The agent pod receives this SSE URL at boot, establishing a clean out-of-process tool execution connection.

---

## Acceptance Criteria

1.  **Multi-Tenancy Isolation**:
    *   All lifecycle calls MUST validate the dynamic `tenant_id` context. A tenant MUST NOT be allowed to deploy, undeploy, or fetch the status of an MCP server owned by a different tenant.
    *   The `TacitoMCPServer` custom resource MUST have `spec.tenantId` correctly populated.
2.  **Hexagonal Separation**:
    *   The application core in Keeper MUST NOT import standard `client-go` Kubernetes primitives. All Kube-API operations must occur through the outbound `ports/outbound/MCPServerCRDCoordinator` port interface.
3.  **OpenAPI Compliance**:
    *   The new lifecycle endpoints MUST be registered in the centralized routing table and fully documented in `api/openapi/openapi.json`.
4.  **Operator Resilience**:
    *   The operator `TacitoMCPServerReconciler` controller must cleanly handle deployment updates, scale changes, and container failures, updating the `phase` and `conditions` fields in the CRD status appropriately.

---

## Test Plan

### Automated Tests
1.  **Unit Tests**:
    *   Test domain validation of `MCPClient` aggregate with new `Image` and `McpPort` fields.
    *   Verify routing tables map `POST /deploy`, `POST /undeploy`, and `GET /status` to the correct lifecycle handlers.
    *   Verify dynamic URL resolution and transport translation logic in `AgentService` constructor.
2.  **Integration Tests**:
    *   Verify database schema migration updates work correctly using `goose` and `pgx`.
    *   Mock `client-go` tests in Keeper using `fake.Clientset` to assert correct generation of `TacitoMCPServer` resources (including DNS-compliant `u-` name formats and tenant IDs).
    *   Mock Operator controller tests verifying that creating a `TacitoMCPServer` resource correctly triggers the creation of corresponding K8s `Deployment` and `Service` resources.
3.  **OpenAPI Contract Tests**:
    *   Execute `make test-contract` to verify Zero-Drift compliance between routes and `api/openapi/openapi.json`.

---

## API Contract

### Deploy MCP Server
*   Endpoint: `POST /api/v1/mcp-servers/{id}/deploy`
*   Headers: `Content-Type: application/json`, `X-Tenant-ID: <tenant-uuid>`
*   Payload: None
*   Response Status: `202 Accepted`
*   Response Payload:
    ```json
    {
      "mcp_server_id": "uuid",
      "status": "pending"
    }
    ```

### Undeploy MCP Server
*   Endpoint: `POST /api/v1/mcp-servers/{id}/undeploy`
*   Headers: `X-Tenant-ID: <tenant-uuid>`
*   Payload: None
*   Response Status: `200 OK`

### Get MCP Server Status
*   Endpoint: `GET /api/v1/mcp-servers/{id}/status`
*   Headers: `X-Tenant-ID: <tenant-uuid>`
*   Response Status: `200 OK`
*   Response Payload:
    ```json
    {
      "mcp_server_id": "uuid",
      "status": "running",
      "url": "http://u-123e4567-e89b-12d3-a456-426614174000.default.svc.cluster.local:8080/sse",
      "replicas": 1
    }
    ```

---

## Files Affected

*   `[NEW] pkg/kubernetes/apis/tacito/v1alpha1/tacitomcpserver_types.go`
*   `[NEW] internal/keeper/application/ports/outbound/mcp_crd_coordinator.go`
*   `[NEW] internal/keeper/adapters/outbound/crd/mcp_crd_coordinator.go`
*   `[NEW] internal/keeper/adapters/inbound/http/mcp_lifecycle_handlers.go`
*   `[MODIFY] internal/keeper/domain/model/mcp_client.go`
*   `[MODIFY] internal/keeper/adapters/outbound/postgres/mcp_client_repository.go`
*   `[NEW] internal/operator/adapters/inbound/k8s/tacitomcpserver_controller.go`
*   `[MODIFY] internal/operator/bootstrap.go`
*   `[NEW] tools/helm/tacito-square/crds/tacitomcpserver-crd.yaml`
*   `[MODIFY] api/openapi/openapi.json`
