# SPEC-FR-16: Object Storage (S3)

| Field         | Value                              |
|---------------|------------------------------------|
| ID            | SPEC-FR-16                         |
| Status        | IMPLEMENTED                        |
| Milestone     | M2                                 |
| Component     | agent, keeper, shared              |
| Depends On    | SPEC-FR-04.1                       |

## Context

Large payloads (tool outputs, file artifacts, conversation exports, long-form LLM responses) should be offloaded to S3-compatible storage instead of flowing through NATS/Redis.

## Specification

### FR-16.1: S3 Outbound Port
1. A `BlobStore` outbound port MUST be defined:
   - `Put(ctx, key string, data io.Reader, contentType string) (string, error)` — returns URL
   - `Get(ctx, key string) (io.ReadCloser, error)`
   - `Delete(ctx, key string) error`
   - `Exists(ctx, key string) (bool, error)`
2. Key naming: `{community}/{thread}/{agent}/{uuid}.{ext}`

### FR-16.2: S3 Adapter
1. Adapter MUST use the AWS SDK v2 for Go (`github.com/aws/aws-sdk-go-v2`).
2. Adapter MUST support MinIO (S3-compatible) for local/dev deployments.
3. Bucket name and endpoint MUST be configurable via environment variables.
4. Adapter MUST support pre-signed URLs for direct client downloads.

### FR-16.3: Payload Offloading
1. Agent artifacts exceeding a configurable size threshold (default: 64KB) MUST be stored in S3 with only a reference (URL) in NATS/Redis.
2. HITL callback payloads MAY be offloaded to S3.
3. Conversation export (`GET /api/v1/threads/{id}/export`) MUST stream from S3.

## Acceptance Criteria

1. `BlobStore.Put` stores data and returns accessible URL
2. `BlobStore.Get` retrieves stored data
3. Payloads above threshold are offloaded transparently
4. MinIO works as S3-compatible backend
5. Pre-signed URLs generated for client access

## Test Plan

- Unit: BlobStore port + mock tests
- Integration: testcontainers with MinIO image

## Files Affected

- `internal/shared/ports/outbound/blobstore.go` (NEW — shared port)
- `internal/shared/adapters/outbound/s3/s3_adapter.go` (NEW)
- `internal/shared/adapters/outbound/s3/s3_adapter_test.go` (NEW)
