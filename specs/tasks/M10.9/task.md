# Task SPEC-FR-M10.9: HTTP Cache Headers for OpenAPI Specification Endpoints

## Status
- **Status:** IMPLEMENTED
- **Milestone:** M10

## Tasks

- [x] Modify BFF `/openapi.json` and `/ui/openapi.json` to return Cache-Control and ETag headers, and check If-None-Match header.
- [x] Modify Keeper `/openapi.json` to return Cache-Control and ETag headers, and check If-None-Match header.
- [x] Add and update tests in `internal/bff/bootstrap_test.go` and `internal/keeper/bootstrap_test.go` to assert caching behavior.
- [x] Verify test correctness via `make test` and `make test-contract`.
