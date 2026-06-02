# TASK-M5.6.1: NATS Ingress Interception & Bucket Normalization

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M5.6.1                                 |
| Status        | IMPLEMENTED                                 |
| Spec          | SPEC-FR-M5.6                                |
| Depends On    | none                                        |

## Description

Implement bucket name dynamic normalization logic and integrate incoming message offloading within the NATS driving ingress adapter (`echo_subscriber.go`). If an incoming payload exceeds the offload threshold (defaulting to 256KB), it must be uploaded to the inferred tenant bucket via the shared `BlobStore` port interface, and the message content replaced with a structured `s3_reference` JSON block before passing it to the message processor. To prevent pod memory limit aggression, the upload operation MUST be stream-buffered directly from the NATS message body reader instead of loading the entire byte array into memory.

## Work Items

1. **RED Phase**:
   - Write a unit test asserting the bucket name normalization rules: converts uppercase to lowercase, swaps non-alphanumeric/non-hyphen characters for a hyphen, collapses duplicate hyphens, and trims end-hyphens.
   - Write a unit test in `internal/agent/adapters/inbound/nats/echo_subscriber_test.go` asserting that incoming payloads exceeding the 256KB threshold trigger S3 `Put` operations and are substituted with the `s3_reference` JSON schema, while payloads below 256KB are forwarded inline.
   - Run tests and verify failure (RED).

2. **GREEN Phase**:
   - Implement the bucket normalization function in the agent code (or a shared utility).
   - Update `EchoSubscriber` in `internal/agent/adapters/inbound/nats/echo_subscriber.go` to accept the shared `BlobStore` interface (`github.com/morphy76/tacito-square/internal/shared/ports/outbound.BlobStore`).
   - Implement interception logic: check the incoming payload length. If greater than threshold, run the normalization on tenant name to get the bucket name, construct the target key (`{community_id}/ingress/{agent_id}/{thread_id}/{object_id}`), and invoke `Put` using streamed buffers.
   - Substitute payload with the `s3_reference` JSON envelope.
   - Run tests and verify successful completion (GREEN).

3. **REFACTOR Phase**:
   - Ensure trace IDs and span IDs are correctly propagated and logged via `zerolog` during the S3 upload operation.
   - Ensure that streamed chunked readers and memory allocations are optimal to avoid pod memory limits aggression.

## Acceptance Criteria

1. Verification of the normalization function against complex tenant name strings (e.g. `Acme_Corp & Co.` -> `acme-corp-co`).
2. Payload offloading acts transparently: any message exceeding the threshold gets offloaded to S3, while others are processed normally.
3. The offloading mechanism utilizes streamed readers with zero flat in-memory slice buffer expansion.
