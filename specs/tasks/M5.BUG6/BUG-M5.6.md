# BUG-M5.6: Pod Memory Exhaustion and Message Pressure under Large Payload Processing

| Field         | Value                                                              |
|---------------|--------------------------------------------------------------------|
| ID            | BUG-M5.6                                                           |
| Status        | OPEN                                                               |
| Severity      | HIGH                                                               |
| Milestone     | M5 — Agent Core                                                    |
| Affects       | `internal/agent/adapters/inbound/nats/echo_subscriber.go`, `internal/agent/application/service/cognitive_engine.go` |
| Violates      | SPEC-FR-M5.6, SPEC-NFR-STACK (Go memory constraints)               |
| Discovered    | Performance verification under simulated memory pressure (128MB container limits) |

## Problem Statement

When processing large conversational or tool payloads (e.g., documents, raw data, or images exceeding 256KB), the agent component is vulnerable to memory exhaustion and message pressure defects:

1. **NATS Message Pressure**: Attempting to transmit multi-megabyte payloads directly via NATS message envelopes violates NATS max payload limits (defaulting to 1MB) and degrades network performance due to connection clogging.
2. **Flat Slice Memory Aggression**: The current NATS subscriber ingress and the cognitive tools (`read_large_payload` and `write_large_payload`) lack strict stream-buffered boundaries. They attempt to load the entire object payload into a flat in-memory byte slice (`[]byte` or `string`). Under concurrent high-throughput request cycles, this causes heap allocations to spike rapidly, causing the Kubernetes kernel to terminate the container with an OutOfMemory (`OOMKilled`) signal.
3. **Lack of Read/Write Backpressure**: There are no boundary checks or chunked streaming limits during object reads and writes, meaning extremely large payloads will execute without backpressure, crashing the pod.

## Affected Components and Files

| Component / File | Location | Issue |
|------------------|----------|-------|
| NATS Inbound Adapter | `internal/agent/adapters/inbound/nats/echo_subscriber.go` | Lacks transparent stream-buffered offloading; attempts flat memory reading of payload. |
| Cognitive Engine | `internal/agent/application/service/cognitive_engine.go` | Execution of read/write tools processes payloads as flat string structures rather than streamed `io.Reader`/`io.Writer` blocks. |

## Impact

1. **Service Instability**: High-throughput transfers of documents cause immediate agent pod restarts due to Kubernetes OOM terminations.
2. **NATS Failure**: Large messages are rejected by the NATS server, causing complete request drops and breaking message flows.
3. **Degraded Performance**: Garbage collector pauses spike due to high heap allocation churn during flat byte copy operations.

## Expected Behaviour

1. **Transparent Stream-Buffered Offloading**: Payloads exceeding 256KB MUST be intercepted by the NATS subscriber and uploaded using streamed, chunk-buffered writing (`io.Copy` or similar chunked allocations) directly from the NATS reader to S3/MinIO.
2. **Memory Bounded Reading/Writing**: The cognitive tools `read_large_payload` and `write_large_payload` MUST process data strictly via buffered streams (e.g., using `bufio.Reader`, bounded chunked readers, and streamed `io.Reader` ports) rather than accumulating the entire payload in memory.
3. **Safety Limits**: Strict upper bounds must be placed on read allocations (e.g., `io.LimitReader` or chunk-size ceilings) to protect against memory limit aggression.

## Acceptance Criteria

1. Memory profiling (using Go `pprof` or integration logs) under simulated 10MB file transfers must confirm that heap allocation remains within bounded limits (peak memory remains below 32MB above baseline).
2. Incoming NATS message size checks are performed prior to flat string generation, offloading payloads to S3 using streamed pipelining.
3. /readyz health probes verify S3/MinIO is online before enabling the offloading pipeline.
