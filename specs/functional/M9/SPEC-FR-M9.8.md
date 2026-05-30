# SPEC-FR-M9.8: Comprehensive System Documentation

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | SPEC-FR-M9.8                                |
| Status        | DRAFT                                       |
| Milestone     | M9                                          |
| Component     | docs                                        |
| Depends On    | SPEC-FR-M2.9, SPEC-FR-M3.1, SPEC-FR-M3.5, SPEC-FR-M3.6, SPEC-FR-M4.1, SPEC-FR-M4.3, SPEC-FR-M4.6, SPEC-FR-M4.7, SPEC-FR-M4.8 |
| Supersedes    | SPEC-FR-M2.9                                |

## Context

SPEC-FR-M2.9 established a minimal README-level documentation baseline for the project scaffold. As the system matures through milestones M3–M4 and beyond, four distinct audiences emerge with distinct and unmet documentation needs:

1. **Workspace Contributors** (anyone cloning the repo) need a single authoritative place to learn how to set up their local development environment, which tools are required or optional, and how to verify the setup works.
2. **Helm Deployers** (operators provisioning the platform) need a precise reference for every Helm value exposed by each chart — its type, default, and effect — without having to parse raw `values.yaml` files.
3. **End Users** (developers integrating with Tacito Square APIs) need task-oriented guides, quickstart flows, and an API reference entry point.
4. **System Architects and Extension Developers** (reviewers, contributors) need accurate architectural documentation and concrete code-rooted walkthroughs.

The root `README.md` remains the canonical entry point for workspace setup. The Helm chart `README.md` files are the canonical deployer interface reference for each chart. All broader architectural and developer documentation MUST live under a root-level `docs/` directory.

## Specification

### General

1. All documentation artifacts MUST be authored in Markdown.
2. Documentation MUST NOT duplicate information already present in OpenAPI specs at `GET /openapi.json`; instead it MUST reference the live endpoint and the committed file in `api/openapi/`.
3. All code examples in documentation MUST reference actual files in the repository with relative links. Stale or fabricated snippets are prohibited.
4. Diagrams MUST be authored in Mermaid (embedded in Markdown) so they are renderable by GitHub and remain version-controlled alongside the prose.

### Project README (`README.md`)

5. The root `README.md` MUST be the single authoritative guide for setting up a local development workspace. It MUST cover:
   - **Required tools** with minimum versions: Go 1.26+, Docker, Helm 3, Kind, `golangci-lint`, `kubectl`.
   - **Optional / recommended tools** with their purpose: `k9s`, `stern`, Rancher Desktop vs. Docker Desktop trade-offs.
   - **Repository layout** — one-line description of each top-level directory.
   - **Local workspace bootstrap** — exact ordered steps: clone, install tools, run `make test`, start local infrastructure (`make helm-infra-install`), load images, install app chart (`make helm-install`), verify with a health probe call.
   - **Common Makefile targets** grouped by category (build, test, docker, helm, lint, docs) with one-line descriptions matching `make help` output.
   - **Troubleshooting** — at least three common local setup issues and their resolutions (e.g. image not found in K8s, port-forward commands, goose migration failures).
   - A **Documentation** section linking to `docs/` with one-line descriptions of each guide category.
6. The root `README.md` MUST NOT duplicate API endpoint listings, Helm value references, or architectural prose — those belong in `docs/`.

### Helm Chart READMEs

7. Each Helm chart MUST have a `README.md` co-located with its `Chart.yaml` that serves as the **complete deployer interface reference** for that chart. No deployer should need to read `values.yaml` to understand configuration options.
8. The infrastructure chart `README.md` at `tools/helm/tacito-square-infra/README.md` MUST contain:
   - A summary of what infrastructure services the chart deploys.
   - A **Values Reference** table with columns: `Key`, `Type`, `Default`, `Description` — covering every configurable value in `values.yaml`.
   - Installation prerequisites and the exact `make helm-infra-deps` + `make helm-infra-install` commands.
   - Notes on any values that MUST be overridden in production (e.g. passwords, TLS certs).
9. The application chart `README.md` at `tools/helm/tacito-square/README.md` MUST contain:
   - A summary of which Tacito Square components the chart deploys (Keeper, BFF, Operator).
   - A **Values Reference** table covering every configurable value, including binding interface values that wire components to the infrastructure chart release (e.g. NATS address, Redis address, PostgreSQL DSN).
   - The exact `make helm-install` command and any prerequisite steps (infrastructure chart must be installed first).
   - Per-component sections documenting image tag overrides, replica counts, HPA toggles, resource limits, and environment variable injection.

### Directory Structure

```
docs/
├── user/
│   ├── quickstart.md            # Prerequisites, local Kind cluster, first agent deployment
│   ├── rest-api.md              # Keeper REST API usage guide (authentication, pagination, error format)
│   ├── operator-admin.md        # Deployment, configuration, scaling, version management
│   └── api-reference.md        # Pointer to /openapi.json + swagger UI dev endpoint
├── architecture/
│   ├── overview.md              # High-level architecture + component responsibilities
│   ├── context-propagation.md  # trace_id / tenant_id / OTel span flow across HTTP + NATS
│   ├── hexagonal.md             # Ports & adapters pattern: domain → ports → adapters
│   ├── data-flow.md             # Agent lifecycle, Keeper ↔ Operator ↔ Agent ↔ NATS flows
│   ├── multitenancy.md          # Tenant resolution, isolation guarantees per deployment unit
│   └── observability.md         # OTel → Collector → backends pipeline
└── developer/
    ├── contributing.md          # Spec-driven + TDD workflow, branch/commit conventions
    ├── new-keeper-entity.md     # Step-by-step: domain model → persistence → HTTP API
    ├── new-llm-adapter.md       # Step-by-step: outbound port interface → adapter implementation
    └── new-agent-capability.md  # Step-by-step: inbound NATS subscriber → outbound adapter
```

10. The `docs/user/` directory MUST contain the following documents:
    - **`quickstart.md`**: prerequisites (pointing to README for tool install), Kind cluster bootstrap via `make helm-infra-install` + `make helm-install`, first Keeper API call, verification commands.
    - **`rest-api.md`**: authentication model (`X-Tenant-ID` header and bearer JWT), pagination conventions, standard error payload (`{"error": "..."}`), and worked examples for key resource types (agents, communities, LLM bindings).
    - **`operator-admin.md`**: references the Helm chart READMEs for values, covers image version management via `VERSION.*` files, scaling configuration (HPA parameters), health probe descriptions, and rollback procedure.
    - **`api-reference.md`**: Explains that the authoritative API reference is served live at `GET /openapi.json` on each component and via Swagger UI at `GET /swagger/` in dev mode. MUST include the committed file path `api/openapi/<component>.json`.

11. The `docs/architecture/` directory MUST contain the following documents:
   - **`overview.md`**: ASCII/Mermaid component diagram (matching README), component responsibility table, inter-component communication matrix (REST vs NATS).
   - **`context-propagation.md`**: Annotated sequence diagram (Mermaid) tracing a full request from an HTTP inbound call through the Gin middleware stack, showing how `trace_id`, `tenant_id`, and OTel span context are extracted, stored in `context.Context`, propagated to service/adapter layers, and then injected into NATS message headers on outbound publish. MUST cover both HTTP→NATS (Keeper→Agent) and NATS→HTTP reply paths. MUST describe the zerolog field injection (`trace_id`, `span_id`).
   - **`hexagonal.md`**: Mermaid class or package diagram showing the `domain/model`, `application/ports/inbound`, `application/ports/outbound`, `adapters/inbound`, `adapters/outbound` package layers with import direction arrows. MUST include a concrete example from the Keeper bounded context (e.g., LLM Bindings entity end-to-end).
   - **`data-flow.md`**: Sequence diagrams for (a) Agent spawn lifecycle (Keeper REST → CRD submission → Operator reconcile → Agent pod ready → NATS subscription), (b) Echo/message flow (Keeper REST → NATS publish → Agent NATS subscriber → reply → Keeper response), (c) Community creation and hub assignment.
   - **`multitenancy.md`**: Documents the two deployment unit models (dedicated single-tenant: Agent/AgentCommunity pods; shared multi-tenant: Keeper, BFF, API Gateways), tenant ID resolution per unit (env var vs JWT claim vs `X-Tenant-ID` header vs NATS message header), and the data isolation layer enforcement (`WHERE tenant_id = ?` at the pgx query layer).
   - **`observability.md`**: Documents the three pillars as implemented: (a) Metrics — Prometheus exposition at `/metrics`, custom domain metrics (`active_threads`, agent counts), HPA consumption; (b) Traces — OTel OTLP gRPC export, W3C `traceparent` propagation, sampler configuration; (c) Logs — zerolog JSON to stdout, `trace_id`/`span_id` injection, log level configuration. MUST include a Mermaid diagram of the pipeline from component → OTel Collector → backends.

12. The `docs/developer/` directory MUST contain the following documents:
   - **`contributing.md`**: SDD workflow (DRAFT → ACCEPTED → IN_PROGRESS → IMPLEMENTED → VERIFIED), TDD red/green/refactor loop, spec file creation instructions, task file structure, branch naming, commit conventions, and `make ci` verification gate.
   - **`new-keeper-entity.md`**: A worked guide (using a hypothetical `Widget` entity) covering: (1) domain model in `internal/keeper/domain/model/`, (2) outbound repository port in `application/ports/outbound/`, (3) application service in `application/service/`, (4) pgx repository adapter in `adapters/outbound/`, (5) Gin HTTP handler in `adapters/inbound/`, (6) goose migration file in `migrations/`, (7) route registration, (8) OpenAPI annotation update, (9) unit test patterns for each layer.
   - **`new-llm-adapter.md`**: A guide covering: (1) outbound port interface in `application/ports/outbound/` (e.g. `BrainPort`), (2) adapter struct in `adapters/outbound/` implementing the port, (3) circuit breaker and retry wiring, (4) OTel span instrumentation on outbound calls, (5) dependency injection in `cmd/<component>/main.go`, (6) unit test with mock port.
   - **`new-agent-capability.md`**: A guide covering: (1) inbound port interface in `application/ports/inbound/`, (2) NATS subscriber adapter in `adapters/inbound/` (W3C trace context extraction from NATS headers), (3) application service method, (4) optional outbound adapter if new dependency required, (5) bootstrap wiring, (6) unit test patterns.

### Documentation Quality Standards

13. Every guide MUST include a **Prerequisites** section at the top listing required tools, environment, and relevant specs.
14. Every guide MUST be accurate as of the milestone in which it is shipped. A `> Last updated: M{X}` callout MUST appear at the top of each document.
15. Mermaid diagrams MUST render without syntax errors on GitHub Markdown preview.
16. All internal cross-references MUST use relative Markdown links.

## Acceptance Criteria

1. Root `README.md` covers all items listed in Requirement 5 (tools, bootstrap steps, Makefile reference, troubleshooting) and contains a Documentation section.
2. `tools/helm/tacito-square-infra/README.md` contains a complete Values Reference table with no missing rows relative to the chart's `values.yaml`.
3. `tools/helm/tacito-square/README.md` contains a complete Values Reference table covering all binding interface values and per-component configuration sections.
4. `docs/` directory exists at repository root with the exact sub-directory structure defined in Requirement 10.
5. All fourteen documentation files under `docs/` are present and non-empty (stubs are not acceptable).
6. `docs/architecture/context-propagation.md` contains a working Mermaid sequence diagram covering HTTP→NATS context propagation including `trace_id`, `tenant_id`, and OTel `traceparent`.
7. `docs/architecture/hexagonal.md` contains a Mermaid diagram with correct import direction arrows (domain ← application ← adapters) and a concrete Keeper example.
8. `docs/architecture/data-flow.md` contains sequence diagrams for the three flows defined in Requirement 11 (`data-flow.md`).
9. `docs/developer/new-keeper-entity.md` covers all nine steps defined in Requirement 12 (`new-keeper-entity.md`) and references real existing files in `internal/keeper/`.
10. `docs/developer/new-llm-adapter.md` and `docs/developer/new-agent-capability.md` reference real existing port interfaces and adapters in the repository.
11. `docs/user/api-reference.md` correctly identifies the `GET /openapi.json` and `GET /swagger/` (dev only) endpoints and links to `api/openapi/`.
12. `make ci` passes (no broken Markdown link checker if enabled, all tests pass).
13. Every Mermaid diagram in `docs/architecture/` renders correctly in a GitHub Markdown preview (verified manually).

## Test Plan

### Automated Tests
1. **Link Validation**: Add a `docs-lint` Makefile target that runs a Markdown link checker (e.g. `lychee` or `mlc`) over `docs/` to verify no broken relative or absolute links exist.
2. **Mermaid Syntax Check**: The `docs-lint` target SHOULD also invoke a Mermaid CLI dry-run (`mmdc --input <file> --output /dev/null`) on all `*.md` files under `docs/architecture/` to catch syntax errors in CI.

### Manual Verification
1. Follow `docs/user/quickstart.md` from scratch on a clean machine with only the listed prerequisites. The cluster MUST come up and the first Keeper API call MUST succeed.
2. Open each `docs/architecture/` file on GitHub (or a local Markdown renderer with Mermaid support) and verify all diagrams render correctly.
3. Follow `docs/developer/new-keeper-entity.md` in a feature branch. Verify that following it step-by-step produces a compilable, test-passing implementation.
4. Review `docs/architecture/context-propagation.md` against the actual `pkg/middleware/` and NATS subscriber adapter code to confirm accuracy.

## Files Affected

- `[MODIFY] README.md` — rewritten to cover workspace setup, required tools, bootstrap, Makefile reference, troubleshooting, and documentation links
- `[MODIFY] tools/helm/tacito-square-infra/README.md` — complete Values Reference table for the infrastructure chart
- `[MODIFY] tools/helm/tacito-square/README.md` — complete Values Reference table for the application chart
- `[NEW] docs/user/quickstart.md`
- `[NEW] docs/user/rest-api.md`
- `[NEW] docs/user/operator-admin.md`
- `[NEW] docs/user/api-reference.md`
- `[NEW] docs/architecture/overview.md`
- `[NEW] docs/architecture/context-propagation.md`
- `[NEW] docs/architecture/hexagonal.md`
- `[NEW] docs/architecture/data-flow.md`
- `[NEW] docs/architecture/multitenancy.md`
- `[NEW] docs/architecture/observability.md`
- `[NEW] docs/developer/contributing.md`
- `[NEW] docs/developer/new-keeper-entity.md`
- `[NEW] docs/developer/new-llm-adapter.md`
- `[NEW] docs/developer/new-agent-capability.md`
- `[MODIFY] Makefile` — add `docs-lint` target
