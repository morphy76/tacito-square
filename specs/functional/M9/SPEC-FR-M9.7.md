# SPEC-FR-M9.7: Benchmark Suite & Integration Coverage Verification

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | SPEC-FR-M9.7                                |
| Status        | ACCEPTED                                    |
| Milestone     | M9                                          |
| Component     | test                                        |
| Depends On    | none                                        |
| Supersedes    | none                                        |

## Context

To maintain performance stability and high code reliability across the Tacito Square platform (Keeper, Operator, and Agent), a robust, native testing suite is required. 

Standardizing Go-native CPU and memory benchmarks (`testing.B`) allows developers to detect computational regressions in core application logic (such as prompt rendering, CRD parsing, state machine transitions, and serialization) without the jitter of external network APIs or third-party downstream platforms. 

Furthermore, rigorous integration test coverage (running against actual containerized backing dependencies via `testcontainers-go`) ensures that critical business happy paths are fully validated and stable.

## Specification

### 1. Inner Code Benchmarks (No External Dependency Delay)
* All benchmark suites MUST measure only inner application logic, excluding external network delays or downstream services.
* All external network boundaries, third-party LLM providers, and remote storage layers MUST be mocked or stubbed out in benchmark suites to ensure reproducible, microsecond-level execution times.
* **Target Areas:**
  * **Keeper:** Benchmark prompt compilation, JSON serialization/deserialization, and core REST request validation/mapping routines.
  * **Operator:** Benchmark CRD state reconciliation logic, memory allocation during heartbeat processing, and scale-to-zero decision paths.
  * **Agent:** Benchmark brain adapter prompt structuring, short-term memory key hashing, and MCP tool serialization/deserialization logic.

### 2. Go Native Integration
* Benchmarks MUST use the standard Go `testing.B` library.
* Benchmarks MUST be executed using standard flags (`go test -bench=. -benchmem ./...`).
* The root-level Makefile MUST orchestrate execution via the existing `make test-bench` target.

### 3. Happy Path Integration Tests
* Critical platform happy paths MUST be covered by integration tests using standard Go `testing.T` and the `testify` library.
* **Core Business Flows Covered:**
  * **Keeper Flow:** Successful Agent creation, registration, and assignment to a Community.
  * **Operator Flow:** Successful reconciliation of an Agent CRD, generating the corresponding Pod specifications, and scaling transitions.
  * **Agent Flow:** Successful execution of a multi-turn conversation step utilizing local mocked brain endpoints and local Redis storage.

### 4. Dependency Verification
* All integration tests validating database or messaging adapters MUST verify backing infrastructure using actual containers via the approved `testcontainers-go` library for PostgreSQL, Redis, and Qdrant.
* Thin mocks MUST NOT be used for integration verification; all CRUD use-cases must run against real database instances spun up in ephemeral containers.

## Acceptance Criteria

1. **Represoducible Benchmarks:** Benchmarks are fully native and run with `go test -bench=.` via `make test-bench`.
2. **Sub-millisecond Local Execution:** External components (OpenAI APIs, remote database connections) are fully mocked during benchmarks, ensuring sub-millisecond execution times.
3. **Containerized Integration Tests:** Integration tests verify critical happy paths under `make test-integration` utilizing `testcontainers-go` to spin up ephemeral PostgreSQL, Redis, and Qdrant containers.
4. **Clean Execution:** All test suites compile and pass successfully under the monorepo root-level Makefile without failing.

## Test Plan

### Automated Tests
1. **Unit & Benchmark Tests:**
   * Run benchmark targets:
     ```bash
     make test-bench
     ```
   * Verify that memory allocation metrics (`allocs/op` and `B/op`) are reported for prompt parsing, reconciliation loops, and session hashing.
2. **Integration Tests:**
   * Run integration targets:
     ```bash
     make test-integration
     ```
   * Verify that `testcontainers-go` successfully spawns PostgreSQL, Redis, and Qdrant container dependencies, runs migrations via goose, executes the business flows, and tears down the containers.

## Files Affected

* `[NEW] test/benchmark/keeper_bench_test.go`
* `[NEW] test/benchmark/operator_bench_test.go`
* `[NEW] test/benchmark/agent_bench_test.go`
* `[NEW] test/integration/keeper_integration_test.go`
* `[NEW] test/integration/operator_integration_test.go`
* `[NEW] test/integration/agent_integration_test.go`
* `[MODIFY] Makefile`
