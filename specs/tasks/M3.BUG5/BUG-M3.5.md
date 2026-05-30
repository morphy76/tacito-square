# BUG-M3.5: Missing OpenAPI Contract Tests

| Field         | Value                                                              |
|---------------|--------------------------------------------------------------------|
| ID            | BUG-M3.5                                                           |
| Status        | CLOSED                                                             |
| Severity      | MEDIUM                                                             |
| Milestone     | M3 — Keeper Core                                                   |
| Affects       | Makefile, test/contract/                                          |
| Violates      | SPEC-NFR-OPENAPI §5, §10, SPEC-FR-M9.4                             |
| Discovered    | M3 candidate post-implementation NFR review                         |

## Problem Statement

According to the API-first design specifications defined in `SPEC-NFR-OPENAPI`, the codebase is required to enforce contract testing to ensure the live `/openapi.json` endpoint served by the application exactly matches the committed api contract at `api/openapi/openapi.json`.

While the live endpoint `/openapi.json` is implemented in `bootstrap.go` and serves the embedded specification file, the following testing gaps exist:
1. **Missing test/contract/ directory**: There are no contract test suites implemented in the codebase.
2. **Useless Makefile Target**: The Makefile target `test-contract` is defined as:
   ```make
   test-contract: ## Run contract tests (OpenAPI compatibility)
   	$(GOTEST) ./test/contract/... -count=1 -v
   ```
   but since the `./test/contract/` directory does not exist, running `make test-contract` does not perform any tests and returns an empty package output, leading to a false sense of test coverage.

## Affected Aggregates and Files

| File / Component | Location | Issue |
|------------------|----------|-------|
| `test/contract/` | `test/contract/` | Directory does not exist; no contract assertions are implemented. |
| `Makefile` | `Makefile` | The `test-contract` target executes a non-existent package. |

## Impact

1. **Undetected API Drift**: Developers can modify REST endpoint parameters, request/response models, or paths in the Gin HTTP handlers without updating the static API contract, leading to discrepancies between live implementation and documentation.
2. **Client Code Breakage**: Downstream components (like the BFF configurator UI) generated from the static `openapi.json` file will encounter runtime exceptions if the live keeper server has drifted from the spec contract.
3. **Compliance Failure**: The build pipeline cannot satisfy SPEC-NFR-OPENAPI AC-3 ("CI fails if live spec diverges from committed contract").

## Expected Behaviour

1. A dedicated contract test suite MUST be implemented inside a valid `test/contract/` Go package.
2. The contract test must dynamically bootstrap a mock HTTP server using `NewServer`, fetch `GET /openapi.json`, and compare the returned JSON schema against the static file located in `api/openapi/openapi.json`.
3. If the live endpoints, tags, fields, or parameters deviate from the static contract, the test must fail.
4. The CI pipeline must run `make test-contract` as part of the validation phase.

1. Running `make test-contract` (or running integration tests) successfully executes a Go contract test suite.
2. Bidirectional Path/Method Parity: The contract test detects any divergence in paths or HTTP methods between the live Gin router and `api/openapi/openapi.json` (asserts no more, no less).
3. Bidirectional Schema/Model Parity: The contract test uses Go reflection to assert that every model property defined in the `openapi.json` schema exactly maps to a field in its corresponding Go structure (asserts no more, no less).
4. CI fails if there is any contract mismatch.


