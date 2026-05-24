---
trigger: always_on
glob: *
description: General non-functional requirements, including technology stack locks, monorepo Makefile build system, semantic versioning lifecycle, and HTTP framework conventions.
---

# General NFR and System Constraints

This rule enforces general software constraints, technologies stack lock-in, monorepo build orchestration, semantic versioning lifecycle, and HTTP application conventions as specified in `SPEC-NFR-STACK`, `SPEC-NFR-BUILDING`, `SPEC-NFR-VERSIONING`, and `SPEC-NFR-HTTP`.

## 1. Approved Technology Stack (SPEC-NFR-STACK)

The following software versions, frameworks, and client libraries are strictly locked. Any deviation or addition requires formal approval/constitution amendment:

- **Core Language:** Go 1.26
- **HTTP Routing:** Gin (`github.com/gin-gonic/gin`)
- **Structured Logging:** zerolog (`github.com/rs/zerolog`)
- **Tracing Platform:** OpenTelemetry OTLP gRPC (`go.opentelemetry.io/otel`)
- **Testing Libraries:** testify (`github.com/stretchr/testify`) and testcontainers-go
- **Configuration Engine:** Viper (`github.com/spf13/viper`)
- **LLM Integrations:** OpenAI Go library (`github.com/openai/openai-go`)
- **Short-Term Memory (STM):** Redis (`github.com/redis/go-redis`)
- **Long-Term Memory (LTM):** Qdrant gRPC (`github.com/qdrant/go-client`)
- **Databases & Migrations:** PostgreSQL (`github.com/jackc/pgx`) and goose (`github.com/pressly/goose`)
- **Event Bus & Messaging:** NATS (`github.com/nats-io/nats.go`)
- **Model Context Protocol:** MCP SDK (`github.com/modelcontextprotocol/go-sdk`)
- **Identity & OIDC:** Zitadel OIDC client (`github.com/zitadel/oidc/v3`)
- **Kubernetes Operators:** Kubebuilder (`sigs.k8s.io/controller-runtime`)
- **Frontend Architecture:** React 19 + Compiler
- **Object Storage:** S3-compatible (with MinIO in dev environment)
- **Umbrella Helm Chart:** Located under `deploy/helm/tacito-square/`

## 2. Monorepo Build System (SPEC-NFR-BUILDING)

A single root-level `Makefile` orchestrates all system activities. Never create nested component-specific Makefiles:

- **Target PHONY Declaration:** Every single target declared in the `Makefile` MUST be declared as `.PHONY`.
- **Consistent Go Build Flags:**
  - Build without CGO: `CGO_ENABLED=0`
  - Build stripped static binaries: `-ldflags="-s -w"`
- **Version Sourcing:** Read the target component version from its `VERSION.<component>` file at the repository root.
- **Required Make Targets:**
  - **Compile:** `build`, `build-<component>`
  - **Testing:** `test` (unit tests), `test-integration`, `test-operator`, `test-e2e`, `test-bench`, `test-race`, `test-contract`
  - **Fidelity/Quality:** `lint`, `generate`
  - **Docker:** `docker-build`, `docker-build-<component>`, `docker-push`
  - **Helm Application:** `helm-template`, `helm-install`, `helm-uninstall`
  - **Helm Infrastructure:** `helm-infra-deps`, `helm-infra-lint`, `helm-infra-template`, `helm-infra-install`, `helm-infra-uninstall`
  - **CI/CD pipeline:** `ci` (lint + test + build + docker combined)
- **Automatic Target Help Documentation:** The `make help` command must parse and print descriptions from `## ` comments.

## 3. Component Versioning and Lifecycle (SPEC-NFR-VERSIONING)

Tacito Square components use independent semantic versions:

- **Version File:** Each component must maintain its current SemVer in a flat text file at the root: `VERSION.keeper`, `VERSION.agent`, `VERSION.operator`, `VERSION.bff`.
- **SemVer Format:** Strictly adhere to Semantic Versioning 2.0.0 (`MAJOR.MINOR.PATCH`).
- **Helm System Versioning:** The parent `deploy/helm/tacito-square/Chart.yaml` version field governs the overall **system version**, which operates independently of component releases.
- **Git Tags Structure:**
  - Component Release tag format: `<component>-v<version>` (e.g. `keeper-v0.2.0`, `agent-v0.1.3`).
  - Helm Chart Release tag format: `chart-<chart-name>-v<version>` (e.g. `chart-tacito-square-v0.2.0`).
- **Atomic Bumps:** Perform version bumps as standalone atomic commits (do not mix code adjustments and version bumps in a single commit).

## 4. HTTP Service Framework Conventions (SPEC-NFR-HTTP)

All HTTP APIs must strictly follow these structural routing and testing guidelines:

- **HTTP Router:** Use the approved **Gin** engine context (`c *gin.Context`).
- **Request Parameters Binding:** Declare specific struct fields with Gin binding tags (e.g., `binding:"required"`) and validate requests using Gin's binding context (e.g. `c.ShouldBindJSON`).
- **Unified Error Payload:** Return a standardized JSON structure on any server-side or validation failure:
  ```json
  { "error": "Clear and descriptive error message" }
  ```
- **Route Centralization:** Group routes logically and register them in package-level functions using `RegisterRoutes(*gin.Engine)` per handler group.
- **HTTP Handlers Testing:** Test all HTTP handlers in Gin test mode using `gin.SetMode(gin.TestMode)`. Feed mock requests using `httptest.NewRecorder()` and trigger routing using Gin's `ServeHTTP` stack to verify full middleware execution.

---

## Developer Checklists & Verifications

- [ ] Am I using CGO_ENABLED=0 and -ldflags="-s -w" in my compilation steps?
- [ ] Is the new library I am importing listed under SPEC-NFR-STACK?
- [ ] Are all added Makefile targets marked as `.PHONY`?
- [ ] Do my HTTP error JSON structures use the exact key `"error"`?
- [ ] Are my handler unit tests running through `ServeHTTP(w, req)` in `gin.TestMode`?
