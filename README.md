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

## Keeper API Endpoints

The Keeper component exposes a REST API for configuration management, lifecycle actions, and system observability.

### Observability & Specification
*   `GET /openapi.json` — Returns the live OpenAPI 3.0.3 contract specification.
*   `GET /metrics` — Exposes technical performance and request metrics in standard Prometheus exposition format.
*   `GET /healthz` — Liveness probe (returns `200 OK` if the process is running).
*   `GET /readyz` — Readiness probe (performs parallel liveness/reachability checks on downstream dependencies like PostgreSQL).

### LLM Provider Bindings (`/api/v1/llm-bindings`)
*   `POST /api/v1/llm-bindings` — Create a new LLM provider binding.
*   `GET /api/v1/llm-bindings` — List all configured LLM provider bindings.
*   `GET /api/v1/llm-bindings/:id` — Retrieve details of a specific LLM provider binding by UUID.
*   `PUT /api/v1/llm-bindings/:id` — Update configuration properties of an existing LLM provider binding.
*   `DELETE /api/v1/llm-bindings/:id` — Remove an LLM provider binding.

### Model Context Protocol (MCP) Servers (`/api/v1/mcp-servers`)
*   `POST /api/v1/mcp-servers` — Register a new MCP server profile (supporting `stdio` or `sse` transport).
*   `GET /api/v1/mcp-servers` — List all registered MCP server profiles.
*   `GET /api/v1/mcp-servers/:id` — Retrieve details of a specific MCP server profile by UUID.
*   `PUT /api/v1/mcp-servers/:id` — Update configuration properties of an existing MCP server profile.
*   `DELETE /api/v1/mcp-servers/:id` — Unregister an MCP server profile.

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

The platform is deployed using a decoupled two-chart model:
1. **Infrastructure Chart** (`tacito-square-infra`): Deploys external resources (PostgreSQL, Redis, Qdrant, NATS, OTel, Keycloak, etc.).
2. **Application Chart** (`tacito-square`): Deploys Tacito components (Keeper, BFF, Operator) configured to bind to the infrastructure release.

```bash
# Create local Kind cluster
kind create cluster --config deploy/dev/kind-config.yaml

# Step 1: Install external infrastructure dependencies
make helm-infra-deps
make helm-infra-install

# Step 2: Install Tacito Square application components
make helm-install

# Uninstall both
make helm-uninstall
make helm-infra-uninstall
```

See [Infrastructure Chart docs](file:///Users/R.Pasquini/Projects/side/tacito-square/tools/helm/tacito-square-infra/README.md) and [Application Chart docs](file:///Users/R.Pasquini/Projects/side/tacito-square/tools/helm/tacito-square/README.md) for detailed configuration.

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
├── tools/helm/             # Decoupled Helm charts (infra & app)
├── docs/                   # Architecture docs, feature mapping
├── migrations/             # Keeper DB migrations
└── test/                   # Integration, contract, E2E tests
```

## License

TBD
