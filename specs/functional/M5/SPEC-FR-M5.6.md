# SPEC-FR-M5.6: Object Storage (S3/MinIO)

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | SPEC-FR-M5.6                                |
| Status        | DRAFT                                       |
| Milestone     | M5                                          |
| Component     | agent                                       |
| Depends On    | SPEC-FR-M5.1                                |
| Supersedes    | none                                        |

## Context

Large payloads (files, images, documents) generated or consumed during agent reasoning are stored in S3-compatible object storage rather than in-memory or database, keeping message sizes manageable.

## Specification

1. The system MUST define a `BlobStore` outbound port in the agent domain layer.
2. The system MUST implement an S3/MinIO adapter.
3. Payloads exceeding a configurable size threshold (default: 256KB) MUST be offloaded to object storage.
4. The adapter MUST support: upload, download, delete, presigned URL generation.
5. Object keys MUST follow the pattern `{community_id}/{agent_id}/{thread_id}/{object_id}`.
6. The adapter MUST use configurable endpoint, bucket, and region from agent configuration.

## Acceptance Criteria

To be defined during spec review.

## Test Plan

To be defined during spec review.

## Files Affected

To be defined during spec review.
