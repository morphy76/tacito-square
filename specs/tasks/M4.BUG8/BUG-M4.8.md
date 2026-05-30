# BUG-M4.8: OTel Trace Context Not Propagated Across NATS Boundary in Echo Flow

| Field         | Value                                                                          |
|---------------|--------------------------------------------------------------------------------|
| ID            | BUG-M4.8                                                                          |
| Status        | VERIFIED                                                                          |
| Severity      | HIGH                                                                              |
| Milestone     | M4 — Operator Core                                                                |
| Affects       | internal/shared/observability/nats_tracing.go (new), internal/keeper/adapters/outbound/nats/community_broadcaster.go, internal/agent/adapters/inbound/nats/echo_subscriber.go, internal/keeper/adapters/inbound/http/echo_handlers.go |
| Violates      | SPEC-NFR-OBSERVABILITY (observability.md rule), SPEC-NFR-LOG                      |
| Discovered    | Zipkin shows no agent-side spans linked to the `keeper.echo_community` HTTP span  |

## Problem Statement

The community echo endpoint (`POST /api/v1/communities/:community_id/echo`) creates an OTel
span on the HTTP ingress side, but the distributed trace is **silently truncated at the NATS
boundary** in both directions. As a result, Zipkin shows only the keeper-side HTTP span with
no downstream agent activity, making the decoupled asynchronous operation invisible to
distributed tracing.

### Specific Gaps

**1. Keeper NATS egress — no `traceparent` injection (`community_broadcaster.go`)**

`NATSCommunityBroadcaster.RequestEcho` calls `nc.RequestMsgWithContext(ctx, &nats.Msg{...})`
with the active OTel span in `ctx`, but never initializes `msg.Header` and never calls
`otel.GetTextMapPropagator().Inject(ctx, ...)`. The W3C `traceparent` header is therefore
absent from every outbound NATS message. Additionally, no `SpanKindClient` span is created
around the NATS request, and no `outbound_dependency_duration_seconds` metric is recorded for
NATS latency.

**2. Agent NATS ingress — no `traceparent` extraction (`echo_subscriber.go`)**

`EchoSubscriber.handleEcho` receives the `*nats.Msg` but never reads `msg.Header`. It
proceeds with an implicit `context.Background()`, making it impossible for the agent-side
processing to be associated with the originating trace. No `SpanKindConsumer` span is created,
and logger calls carry no `trace_id` or `span_id` fields.

**3. Keeper HTTP handler — span name deviates from spec (`echo_handlers.go`)**

The keeper's HTTP handler starts a span named `"http.echo_community"` (line 43), but
SPEC-FR-M4.8 §8 requires the span to be named `"keeper.echo_community"`. This minor deviation
makes the span harder to find by operation name in Zipkin/Jaeger queries.

### Observability Rule Violated

The active `RULE[observability.md]` (SPEC-NFR-OBSERVABILITY) explicitly requires:

> **OTel Instrumentation:** Instrument all HTTP, NATS, and database client interactions with
> OpenTelemetry (OTel).
>
> **Context Propagation:** Ensure W3C traceparent context headers are correctly extracted on
> ingress (HTTP request handlers, **NATS subscribers**) and injected on egress (outbound HTTP
> clients, RPCs, **NATS publishers**).

## Resolution Approach

Rather than patching the echo path in isolation, the fix introduces a **reusable NATS OTel
framework** in `internal/shared/observability/nats_tracing.go`, following the same pattern as
the existing `db_tracing.go`. This ensures every future NATS publisher and subscriber benefits
from context propagation, spans, trace-correlated logging, and latency metrics without any
additional effort.

| Framework primitive | Responsibility |
|---|---|
| `NATSHeaderCarrier` | `propagation.TextMapCarrier` over `nats.Header` — enables `Inject`/`Extract` |
| `InjectNATSContext(ctx, msg)` | Initializes `msg.Header` and injects `traceparent` — for any NATS egress |
| `ExtractNATSContext(ctx, msg)` | Extracts trace context from `msg.Header` — for any NATS ingress |
| `RequestMsgWithTrace(ctx, nc, subject, msg)` | Drop-in for `nc.RequestMsgWithContext`: adds client span + inject + NATS latency metric |
| `WrapNATSHandler(spanName, logger, inner)` | Wraps any handler with consumer span + extract + trace-correlated logger |

Components `community_broadcaster.go` and `echo_subscriber.go` become thin one-line consumers
of these primitives. No OTel logic lives in the adapter layer.

## Affected Components and Files

| Component / File | Location | Issue |
|------------------|----------|-------|
| shared / observability | [nats_tracing.go](file:///Users/R.Pasquini/Projects/side/tacito-square/internal/shared/observability/nats_tracing.go) (**new**) | Missing reusable NATS OTel framework (`NATSHeaderCarrier`, `InjectNATSContext`, `ExtractNATSContext`, `RequestMsgWithTrace`, `WrapNATSHandler`) |
| keeper / outbound NATS | [community_broadcaster.go](file:///Users/R.Pasquini/Projects/side/tacito-square/internal/keeper/adapters/outbound/nats/community_broadcaster.go) | No `traceparent` injection into `nats.Msg.Header`; no `SpanKindClient` span; no NATS latency metric |
| agent / inbound NATS | [echo_subscriber.go](file:///Users/R.Pasquini/Projects/side/tacito-square/internal/agent/adapters/inbound/nats/echo_subscriber.go) | No `traceparent` extraction from `msg.Header`; no `SpanKindConsumer` span; logger not enriched with trace context |
| keeper / HTTP handler | [echo_handlers.go](file:///Users/R.Pasquini/Projects/side/tacito-square/internal/keeper/adapters/inbound/http/echo_handlers.go) | Span name `"http.echo_community"` deviates from SPEC-FR-M4.8 §8 (`"keeper.echo_community"`) |

## Impact

1. **Invisible async operations in Zipkin/Jaeger:** The agent's echo processing is entirely
   absent from distributed traces. Operators cannot determine whether a slow echo response is
   caused by Keeper logic, NATS latency, or agent processing.
2. **Missing NATS latency observability:** No histogram tracks NATS request-reply round-trip
   duration, making SLA compliance for the echo path unmeasurable.
3. **No trace-correlated logs on the agent side:** Agent logs for echo request handling carry
   no `trace_id` or `span_id`, breaking the log-trace correlation required by SPEC-NFR-LOG
   and making incident investigation significantly harder.
4. **Spec deviation:** The wrong span name (`"http.echo_community"`) makes Zipkin operation
   name searches and alerting rules based on SPEC-FR-M4.8 §8 ineffective.

## Expected Behaviour

1. A new `internal/shared/observability/nats_tracing.go` file MUST provide `NATSHeaderCarrier`,
   `InjectNATSContext`, `ExtractNATSContext`, `RequestMsgWithTrace`, and `WrapNATSHandler` as
   reusable primitives tested in isolation. Future NATS publishers and subscribers use these
   primitives without any additional OTel wiring.
2. `NATSCommunityBroadcaster.RequestEcho` MUST delegate to `observability.RequestMsgWithTrace`,
   which handles: initializing `nats.Msg.Header`, injecting the W3C `traceparent`, creating a
   `SpanKindClient` span named `"nats.request"` with `messaging.system="nats"` and
   `messaging.destination=<subject>`, and recording NATS round-trip duration in
   `OutboundDependencyDuration{dependency="nats"}`.
3. `EchoSubscriber` MUST delegate to `observability.WrapNATSHandler("nats.echo_handler", ...)`,
   which handles: extracting OTel context from `msg.Header`, starting a `SpanKindConsumer` span,
   and enriching the logger with `trace_id` / `span_id` before calling the inner handler.
4. Agent echo handler logs MUST contain `trace_id` and `span_id` fields, correlated with the
   originating keeper request trace.
5. The keeper HTTP handler span name MUST be `"keeper.echo_community"` (matching SPEC-FR-M4.8 §8).

## Acceptance Criteria

1. `TestNATSHeaderCarrier_GetSetKeys`, `TestInjectExtractNATSContext_RoundTrip`,
   `TestRequestMsgWithTrace_SpanAndMetric`, and `TestWrapNATSHandler_ConsumerSpanAndLog` all
   pass GREEN in `internal/shared/observability/`.
2. Sending a `POST /api/v1/communities/:community_id/echo` request and inspecting Zipkin shows
   a complete trace tree: `keeper.echo_community` (server) → `nats.request` (client) →
   `nats.echo_handler` (consumer on the agent).
3. Agent pod logs for an echo request contain `"trace_id"` and `"span_id"` fields matching the
   originating HTTP trace.
4. The `outbound_dependency_duration_seconds` Prometheus histogram includes samples with
   `dependency="nats"`.
5. The keeper HTTP span name in Zipkin is `"keeper.echo_community"`.
6. All existing unit tests for `community_broadcaster.go` and `echo_subscriber.go` remain GREEN.
