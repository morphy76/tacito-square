# SPEC-FR-M5.6: Object Storage (S3/MinIO)

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | SPEC-FR-M5.6                                |
| Status        | ACCEPTED                                    |
| Milestone     | M5                                          |
| Component     | agent                                       |
| Depends On    | SPEC-FR-M5.1                                |
| Supersedes    | none                                        |

## Context

Large payloads (files, images, documents) generated or consumed during agent reasoning are stored in S3-compatible object storage rather than in-memory or database, keeping message sizes manageable.

This is a built-in tool/capability available to all agents. It is shared across the agents of the same community, with each tenant isolated within its own dedicated bucket.

---

## Specification

### 1. Hexagonal Ports & Domain Interfaces
The system MUST reuse the shared outbound port `BlobStore` interface defined in `internal/shared/ports/outbound/blobstore.go`. This port interface is imported directly into the agent's application layer, allowing consistency and code reusability across components while satisfying hexagonal architecture layering constraints:

```go
package outbound

import (
	"context"
	"io"
)

// BlobStore is the outbound port for S3-compatible object storage.
type BlobStore interface {
	// Put stores data and returns the object URL.
	Put(ctx context.Context, key string, data io.Reader, contentType string) (string, error)
	// Get retrieves data by key.
	Get(ctx context.Context, key string) (io.ReadCloser, error)
	// Delete removes an object by key.
	Delete(ctx context.Context, key string) error
	// Exists checks if an object exists.
	Exists(ctx context.Context, key string) (bool, error)
}
```

### 2. S3/MinIO Infrastructure Adapter
* **Layering**: The system MUST reuse the shared S3/MinIO adapter `BlobStoreAdapter` located in `internal/shared/adapters/outbound/s3/s3_adapter.go` as the concrete infrastructure adapter, wired to the `BlobStore` port in the agent's startup bootstrap context.
* **Key Namespace Isolation**: Object keys MUST strictly follow the pattern:
  `{community_id}/{functional_root}[/{agent_id}/{thread_id}]/{object_id}`
  The structure after the functional root may vary dynamically by function/use-case. Functional root come from an array of configurable strings rpovided to the agent from the environment (e.g. `TS_AGENT_FUNCTIONAL_ROOTS=ingress,summary,embeddings`) according to CRD agent configuration propagation standard.
* **Bucket Naming Resolution & Normalization**:
  The bucket name is inferred dynamically from the tenant's name (resolving `TENANT_NAME` or `TENANT_ID` from the environment).
  To ensure S3/MinIO compatibility, the inferred name must be normalized using these rules:
  1. Convert all characters to lowercase.
  2. Replace any character that is not a lowercase alphanumeric character (`a-z`, `0-9`) or a hyphen (`-`) with a hyphen.
  3. Collapse multiple consecutive hyphens into a single hyphen.
  4. Trim leading and trailing hyphens.
  5. Truncate to a maximum of 63 characters.

### 3. Resiliency & Graceful Degradation (Cloud-First)
* **Circuit Breakers**: Wrap every outbound call to S3/MinIO in a circuit breaker configured via agent parameters (`TS_AGENT_S3_CB_THRESHOLD`, `TS_AGENT_S3_CB_TIMEOUT`).
* **Retries & Backoff**: Standardize on exponential backoff retries with random jitter for transient errors (e.g., DNS timeouts, HTTP 500/503).
* **Timeout Deadlines**: Enforce explicit timeouts on S3 calls using Go's `context.Context` (defaulting to 10 seconds).
* **Graceful Fallback**: If S3/MinIO is completely unreachable or the circuit breaker is open:
  * Log the failure as a structured JSON warning using `zerolog`.
  * Return a standardized error JSON block to the reasoning loop: `{"error": "Object storage temporarily unavailable."}`.
  * The agent must degrade gracefully, allowing the LLM reasoning loop to continue using its internal knowledge.

### 4. Inbound Message Pipeline & NATS Ingress (Option A)
To prevent large payloads from clogging NATS streams:
* **Ingress Detection**: The driving NATS subscriber adapter (`internal/agent/adapters/inbound/nats/echo_subscriber.go`) transparently intercepts incoming payloads.
* **Offloading Trigger**: If an incoming payload exceeds `TS_AGENT_S3_OFFLOAD_THRESHOLD` (defaulting to `256KB`), the subscriber:
  1. Generates a unique `object_id` (UUID).
  2. Uploads the payload to S3 under the pattern `{community_id}/{functional root}/{agent_id}/{thread_id}/{object_id}` (where `{functional root}` is set to `ingress` or a similar designated context).
  3. Replaces the message content sent to `MessageProcessor` with a structured S3 reference:
     ```json
     {
       "_type": "s3_reference",
       "bucket": "acme-corp",
       "key": "community-1/ingress/agent-1/thread-1/obj-12345",
       "size_bytes": 314572,
       "content_type": "text/plain"
     }
     ```
* **Tool for Agents**: Introduce a built-in cognitive tool called `read_large_payload` to the LLM's active tool list. If the LLM identifies an `s3_reference` block in the conversation history, it invokes `read_large_payload(key: string)` to dynamically download and read the content.

### 5. Observability Correlation
* **OTel Instrumentation**: All operations in the S3 adapter must propagate OpenTelemetry (OTel) request trace context. Spans generated by S3 actions must be children of the parent message processing span.
* **Structured JSON Telemetry**: Log S3 actions via `zerolog` with structured parameters (`tenant_id`, `agent_id`, `thread_id`, `trace_id`, `span_id`, `s3_key`, `s3_bucket`).

---

## Acceptance Criteria

1. **Hexagonal Architecture Compliance**:
   - The outbound port `BlobStore` is imported from the shared codebase directly into the application layer.
   - The concrete shared S3 adapter is wired inside the bootstrap step, keeping domain and application layers entirely pure.
2. **Transparent Ingress Offloading**:
   - Sending a payload larger than 256KB via NATS results in an automatic upload to S3.
   - The NATS subscriber processes the reference block without crashing.
3. **Strict Normalization & Isolation**:
   - An environment tenant name of `"Acme_Corp & Co."` resolves safely to the bucket name `"acme-corp-co"`.
   - Tenant bucket isolation is strictly enforced. An agent under tenant `A` cannot read objects from tenant `B`.
4. **Resiliency & Circuit Breaking**:
   - When S3 is artificially shut down, the agent does not crash. It catches the error, logs a structured Zerolog warning, and yields a fallback JSON block.
5. **Observability Integration**:
   - Outbound S3 operations generate distinct OTel spans correlated with the parent reasoning trace.

---

## Test Plan

### Automated Tests
1. **Unit Tests**:
   - Assert the bucket name normalization rules against various inputs (spaces, special characters, uppercase, length limits).
   - Assert that payloads <= 256KB bypass the offload logic, while payloads > 256KB trigger it.
2. **Integration Tests**:
   - Setup a MinIO service container using `testcontainers-go`.
   - Assert standard `Put`, `Get`, `Delete`, and `Exists` operations on the adapted client.
   - Inject connection failures to verify that circuit breakers open and trigger the graceful fallback.
3. **Contract Probes**:
   - Add MinIO ping checks to the Agent's parallel `/readyz` probe pipeline.

---

## Files Affected

- `[MODIFY] internal/agent/adapters/inbound/nats/echo_subscriber.go` — Inbound offloading interception (reusing shared `BlobStore`).
- `[MODIFY] internal/agent/bootstrap.go` — Wire up `/readyz` check and bootstrap configurations.
- `[MODIFY] cmd/agent/main.go` — Load S3/MinIO configurations, initialize shared `BlobStoreAdapter`, and inject it.
- `[MODIFY] specs/INDEX.md` — Transition SPEC-FR-M5.6 to ACCEPTED.
