# Tacito Square

> A silent plaza where agents gather, deliberate, and act without noise.

**Tacito Square** is a Kubernetes-native multi-agent system built in Go. It provides a platform for deploying, orchestrating, and observing communities of autonomous AI agents that collaborate through structured protocols.

## Principles

- **Hexagonal Architecture** — Agents are built with ports & adapters, keeping domain logic decoupled from infrastructure. Every external dependency (LLM, memory, messaging, tools) is behind a port interface.
- **Stateless Agents** — Each agent is a K8s Deployment configured at runtime via ConfigMaps/Secrets. Go's sub-second bootstrap eliminates warm pool needs.
- **Spec-Driven** — Development follows a milestone-based roadmap with numbered functional requirements. Every feature is test-driven (Red → Green → Refactor).
- **API-First** — All functionality is available through authenticated REST APIs (Bearer JWT/OIDC). UIs are optional consumers, not gatekeepers.
- **Contract-Based** — Each component (agent, keeper, operator, bff) has an independent version lifecycle. Inter-component contracts are defined via OpenAPI specs.
- **Community-Centric** — Agents form communities with configurable topologies (Hub-Spoke, Mesh, Pipeline). Communities support multi-threaded concurrent interactions and human-in-the-loop via callback hooks.
- **Accountable** — RBAC-driven access control (Keycloak OIDC) with role-based policies enforced at the API layer. All state transitions and interactions are auditable.

## Architecture

```
┌──────────────────────────────────────────────────────┐
│                    End Users                         │
│         Keeper UI (React 19)  User UI (React 19)     │
│                      │              │                │
│              ┌───────┴──────────────┴──────┐         │
│              │     BFF Layer (Go/Gin)      │         │
│              │  Bearer JWT auth, API-first │         │
│              └───────┬──────────────┬──────┘         │
│     ┌────────────────┴──────┐       │                │
│     │    Agent Keeper       │       │                │
│     │  (Control Plane)      │  ┌────┴──────┐         │
│     │  - Spawn/terminate    │  │ K8s       │         │
│     │  - Prompts & skills   │  │ Operator  │         │
│     │  - Audit log          │  │ (CRDs)    │         │
│     │  - HITL callbacks     │  └───────────┘         │
│     │  - A2A gateway        │                        │
│     └────────┬──────────────┘                        │
│              │ NATS                                  │
│     ┌────────┴──────────────────────────┐            │
│     │      Agent Communities            │            │
│     │  ┌─────┐  ┌─────┐  ┌─────┐        │            │
│     │  │Hub  │──│Agent│──│Agent│        │            │
│     │  └─────┘  └─────┘  └─────┘        │            │
│     └───────────────────────────────────┘            │
│                                                      │
│  Infrastructure: NATS │ Redis │ Qdrant │ PostgreSQL  │
│                  Keycloak │ OpenTelemetry Collector  │
└──────────────────────────────────────────────────────┘
```

## Components

| Component | Description | Image |
|-----------|-------------|-------|
| **Agent** | Hexagonal stateless worker — LLM reasoning, memory, MCP tools | `tacito-square/agent` |
| **Keeper** | Control plane — lifecycle, prompts, skills, audit, A2A gateway | `tacito-square/keeper` |
| **Operator** | Kubebuilder CRD controller — Agent & AgentCommunity resources | `tacito-square/operator` |
| **BFF** | Backend-for-frontend — auth, aggregation, API translation | `tacito-square/bff` |

## Technology Stack

| Layer | Technology |
|-------|-----------|
| Language | Go 1.26 |
| LLM | OpenAI-compatible REST API |
| Short-Term Memory | Redis |
| Long-Term Memory | Qdrant (vector) |
| Keeper Persistence | PostgreSQL |
| Messaging | NATS |
| Auth | Keycloak (OIDC) |
| HTTP Framework | Gin |
| Logging | zerolog (with trace_id + token claims) |
| Observability | OpenTelemetry (OTLP gRPC) |
| Frontend | React 19 with Compiler |
| Deployment | Helm umbrella chart |

## Build Lifecycle

### Prerequisites

- Go 1.26+
- Docker
- Helm 3
- Kind (for local K8s)
- `golangci-lint`

### Local Development

```bash
# Run unit tests (TDD — always start here)
make test

# Run integration tests (requires Docker for testcontainers)
make test-integration

# Run benchmarks
make test-bench

# Run with race detector
make test-race

# Lint
make lint

# Build all binaries
make build

# Full CI pipeline
make ci
```

### Docker Images

```bash
# Build all images
make docker-build

# Build a specific component
make docker-build-agent
make docker-build-keeper
```

### Version Management

Each component has an **independent semantic version** tracked in `VERSION.*` files:

```
VERSION.agent      # Agent image tag
VERSION.keeper     # Keeper image tag
VERSION.operator   # Operator image tag
VERSION.bff        # BFF image tag
```

To release a component:
1. Bump the version in the corresponding `VERSION.*` file
2. Run `make docker-build-<component>` to build the tagged image
3. Run `make docker-push` to push to the registry
4. Update the Helm chart `values.yaml` with the new tag

### Deployment

```bash
# Create local Kind cluster
kind create cluster --config deploy/dev/kind-config.yaml

# Install the full platform
make helm-install

# Uninstall
make helm-uninstall
```

See [`deploy/helm/tacito-square/README.md`](deploy/helm/tacito-square/README.md) for detailed Helm configuration.

## Project Structure

```
tacito-square/
├── cmd/                    # Binary entry points (agent, keeper, operator, bff)
├── internal/
│   ├── agent/              # Agent hexagonal architecture
│   │   ├── domain/         # Core domain models
│   │   ├── ports/          # Inbound & outbound interfaces
│   │   ├── service/        # Application services
│   │   └── adapters/       # Driven adapters (openai, redis, qdrant, mcp, nats)
│   ├── keeper/             # Keeper control plane
│   │   ├── domain/         # AgentInstance, SpawnRequest, state machine
│   │   ├── ports/          # Inbound & outbound interfaces
│   │   ├── service/        # KeeperService, spawner, observer
│   │   └── adapters/       # HTTP handlers, PostgreSQL, K8s client
│   ├── bff/                # BFF layer (keeper & user routes)
│   └── shared/             # Config, observability, auth, errors
├── operator/               # Kubebuilder CRD operator
├── ui/                     # React 19 frontends
├── api/openapi/            # OpenAPI contracts
├── deploy/helm/            # Umbrella Helm chart
├── docs/                   # Architecture docs, feature mapping
├── migrations/             # Keeper DB migrations
└── test/                   # Integration, contract, E2E tests
```

## License

TBD
