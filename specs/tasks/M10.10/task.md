# Task SPEC-FR-M10.10: HTTP Cache Headers for BFF Static Resources

## Status
- **Status:** VERIFIED
- **Milestone:** M10

## Tasks

- [x] Modify BFF welcome HTML endpoints to return Cache-Control and ETag headers, and check If-None-Match header.
- [x] Modify BFF secure HTML endpoints to return Cache-Control and ETag headers, and check If-None-Match header.
- [x] Modify BFF `/favicon.ico` endpoint to return Cache-Control and ETag headers, and check If-None-Match header.
- [x] Add and update tests in `internal/bff/bootstrap_test.go` to assert caching behavior.
- [x] Verify test correctness via `make test`.
