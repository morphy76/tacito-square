---
trigger: always_on
glob: **/*.{go,ts}
description: Systems observability standards, including Prometheus metrics, OpenTelemetry distributed tracing, and zerolog/winston structured JSON logging.
---

# Observability Guidelines

This rule enforces metrics collection, distributed context tracing, and structured JSON logging requirements as specified in `SPEC-NFR-OBSERVABILITY` and `SPEC-NFR-LOG`.

## 1. Integrated Pillars of Observability (SPEC-NFR-OBSERVABILITY)

Every deployable component (agent, keeper, operator, bff) must support integrated Pillars of Observability:

### A. Prometheus Metrics
- **Exposition Endpoint:** Expose a `GET /metrics` endpoint in the standard Prometheus exposition text format.
- **Required Metrics:**
  - **HTTP:** Request count, latency histograms, error rates (partitioned by route, method, and HTTP status).
  - **Runtime:** Goroutines count, memory allocations, garbage collection (GC) pauses.
  - **Domain-Specific:** Active agent counts (by status), active processing threads, and pending HITL callbacks.
  - **Quota Limits:** Resource utilization gauges per community and active agent.
  - **Dependencies:** Outbound request latency histograms to external dependencies (LLM, Redis, Qdrant, NATS, PostgreSQL, S3).

### B. Distributed Tracing (OpenTelemetry)
- **OTel Instrumentation:** Instrument all HTTP, NATS, and database client interactions with **OpenTelemetry (OTel)**.
- **Context Propagation:** Ensure W3C traceparent context headers are correctly extracted on ingress (HTTP request handlers, NATS subscribers) and injected on egress (outbound HTTP clients, RPCs, NATS publishers).
- **Traced Attributes:** Include service metadata (component name, version, environment), operational details (SQL query state, NATS topic), and detailed errors on span exceptions.

### C. Observability Correlation
- **Linkage:** Correlate tracing, logging, and metrics to ensure traceability.
- **Context Injection:** When an active span is present in the `context.Context`, inject the `trace_id` and `span_id` directly into the logs.

## 2. Structured JSON Logging (SPEC-NFR-LOG)

Ensure all application logs are highly structured and machine-readable:

- **Framework Lock:** 
  - **Golang components:** MUST use **zerolog** (`github.com/rs/zerolog`).
  - **TypeScript components:** MUST use **winston**.
- **Output Channel:** Direct logs to stdout in raw JSON format. NEVER use formatted text or console-friendly coloring in production.
- **Metadata Fields:**
  - Inject OTel `trace_id` and `span_id` automatically from the active request context.
  - Inject a configurable set of JWT claims (e.g., `sub`, `email`) configured at build time in `LogClaimsKeys`.
- **Log Levels:** Adhere to standardized levels: `trace`, `debug`, `info`, `warn`, `error` (default to `info`). Suppress logging below the configured severity level.
- **Startup:** Log the component build/version at application startup.

---

## Developer Checklists & Verifications

- [ ] Am I using `zerolog` (Go) or `winston` (TS) to log events?
- [ ] Do my logs output JSON to stdout?
- [ ] Are `trace_id` and `span_id` present in logs during active requests?
- [ ] Is `GET /metrics` returning valid Prometheus format data?
- [ ] Does my outbound HTTP, NATS, or DB client propagate OTel span context?
