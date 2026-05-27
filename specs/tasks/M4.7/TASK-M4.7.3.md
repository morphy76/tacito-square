# TASK-M4.7.3: Lifecycle Application Inbound Port & Service

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M4.7.3                                 |
| Status        | DRAFT                                       |
| Spec          | SPEC-FR-M4.7                                |
| Depends On    | TASK-M4.7.2                                 |

## Description

Design and implement the application core layer for managing Agent and Community logical lifecycle operations. This involves defining the driving inbound port interface `LifecycleUseCase` and implementing the orchestration service `LifecycleService` which coordinates repository transactions, K8s CRD updates via the coordinator, concurrency control via `errgroup`, and OTel logging/tracing.

## Boundary & Target Functions

- **Packages**: `internal/keeper/application/ports/inbound`, `internal/keeper/application/service`
- **Files**:
  - `[NEW] internal/keeper/application/ports/inbound/lifecycle_ports.go`
  - `[NEW] internal/keeper/application/service/lifecycle_service.go`
- **Target Interfaces, Structs & Functions**:
  - `LifecycleUseCase` (interface)
  - `LifecycleService` (struct)
  - `NewLifecycleService(agentRepo outbound.AgentRepository, commRepo outbound.CommunityRepository, crdCoord outbound.CRDCoordinator, nc *nats.Conn) *LifecycleService`
  - `(s *LifecycleService) DeployAgent(ctx context.Context, agentID uuid.UUID) error`
  - `(s *LifecycleService) UndeployAgent(ctx context.Context, agentID uuid.UUID) error`
  - `(s *LifecycleService) GetAgentStatus(ctx context.Context, agentID uuid.UUID) (*model.AgentStatusDetails, error)`
  - `(s *LifecycleService) DeployCommunity(ctx context.Context, communityID uuid.UUID) (*model.CommunityDeploymentDetails, error)`
  - `(s *LifecycleService) UndeployCommunity(ctx context.Context, communityID uuid.UUID) (*model.CommunityDeploymentDetails, error)`
  - `(s *LifecycleService) GetCommunityStatus(ctx context.Context, communityID uuid.UUID) (*model.CommunityStatusDetails, error)`

## Work Items

1. **RED Phase**:
   * Add unit tests in `internal/keeper/application/service/lifecycle_service_test.go` verifying:
     * `DeployAgent` throws validation errors if the agent is not assigned to a community, transitions database status to `pending`, and publishes NATS `agent.provisioning.started` event.
     * `UndeployAgent` transitions database status to `stopped`, deletes the CRD, and publishes NATS events.
     * `DeployCommunity` and `UndeployCommunity` perform logical operations on all assigned agents concurrently in parallel using `errgroup`, reporting back individual successes/failures.
     * `GetCommunityStatus` correctly aggregates DB and K8s CRD statuses for all community agents.

2. **GREEN Phase**:
   * Create `internal/keeper/application/ports/inbound/lifecycle_ports.go` defining the `LifecycleUseCase` interface.
   * Create `internal/keeper/application/service/lifecycle_service.go` implementing `LifecycleService`.
   * For Agent lifecycle commands:
     * Check tenant context match to enforce multitenancy bounds.
     * Perform database repository actions.
     * Publish structured NATS JSON events under target event bus streams.
   * For Community logical actions:
     * Fetch community agents.
     * Leverage `golang.org/x/sync/errgroup` to trigger parallel deployments/terminations.
     * In the event of a subset failure, aggregate individual status results (such as `failed` with specific message details) and return a custom aggregate details struct.

3. **REFACTOR Phase**:
   * Verify all asynchronous goroutines inherit properly canceled parent contexts to prevent background routine leakage.
   * Validate that structured logs are written via OpenTelemetry correlated contexts.

## Acceptance Criteria

1. Verification tests pass successfully with mocked repositories and coordinators.
2. Inbound port boundaries adhere strictly to Hexagonal layering guidelines.
3. Concurrency patterns operate in non-blocking fashion using standard sync primitives.
