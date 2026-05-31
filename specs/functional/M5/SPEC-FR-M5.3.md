# SPEC-FR-M5.3: Short-Term Memory (Redis)

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | SPEC-FR-M5.3                                |
| Status        | VERIFIED                                    |
| Milestone     | M5                                          |
| Component     | agent                                       |
| Depends On    | SPEC-FR-M5.1, SPEC-FR-M5.2                  |
| Supersedes    | none                                        |

## Context

Agents scale horizontally to handle dynamic request loads. To maintain statelessness between reasoning cycles while retaining the context of active conversation threads or task execution logs, short-term memory (STM) is decoupled into Redis. This specification defines the outbound `ShortTermMemory` port, the Redis adapter implementation, strict multi-tenant isolation, memory turn limit configurations, and the integration of memory inside the inbound message processing pipeline.

---

## Specification

### 1. Domain Models
Define the following rich domain models inside `internal/agent/domain/model/memory.go` to represent short-term memory entries:

```go
package model

import "time"

// MemoryEntry represents a single conversational or execution step in a thread.
type MemoryEntry struct {
	Role      string            `json:"role"`                 // e.g., "system", "user", "assistant", "tool"
	Content   string            `json:"content"`
	Timestamp time.Time         `json:"timestamp"`
	Metadata  map[string]string `json:"metadata,omitempty"`   // Type descriptors, tool IDs, execution steps
}
```

Extend `BrainRequest` in `internal/agent/domain/model/brain.go` to support conversation history:
```diff
 type BrainRequest struct {
 	Prompt            string             `json:"prompt"`
 	SystemPrompt      string             `json:"system_prompt,omitempty"`
+	History           []MemoryEntry      `json:"history,omitempty"`
 	Temperature       float64            `json:"temperature,omitempty"`
 	MaxTokens         int                `json:"max_tokens,omitempty"`
 	ProviderOptions   map[string]any     `json:"provider_options,omitempty"`
 }
```

### 2. Hexagonal Ports & Domain Interfaces
Define the `ShortTermMemory` outbound port inside `internal/agent/application/ports/outbound/memory.go`:

```go
package outbound

import (
	"context"
	"github.com/morphy76/tacito-square/internal/agent/domain/model"
)

type ShortTermMemory interface {
	// Append appends a new memory entry to the specified thread.
	Append(ctx context.Context, tenantID, agentID, threadID string, entry model.MemoryEntry) error

	// Get retrieves the sliding window of conversational history up to a limit.
	Get(ctx context.Context, tenantID, agentID, threadID string, limit int) ([]model.MemoryEntry, error)

	// Clear deletes all entries associated with the specified thread.
	Clear(ctx context.Context, tenantID, agentID, threadID string) error
}
```

### 3. Redis Infrastructure Adapter
* **Key Namespace Isolation**: To satisfy `RULE[cloud-first.md]` multitenancy requirements, every key stored in Redis must be strictly prefixed with the tenant ID:
  `ts:tenant:{tenant_id}:agent:{agent_id}:stm:{thread_id}`
* **Storage Data Structure**: Entries must be stored in a Redis `LIST` (ordered by append sequence) and serialized as JSON string arrays.
* **TTL-Based Expiry**: All STM keys must carry a configurable TTL (Time-To-Live) to avoid infinite memory growth.
* **Adapter Package**: Implement `RedisMemoryAdapter` inside `internal/agent/adapters/outbound/redis/memory_adapter.go`.

### 4. Resiliency & Graceful Degradation
* **Failure Boundary**: If the Redis cluster is offline or experiences transient timeouts, the agent MUST NOT crash or fail the request.
* **Stateless Fallback**: The application service must log the failure as a structured JSON warning (`zerolog`) and continue processing the reasoning pipeline using only the incoming request payload (acting statelessly as a fallback behavior).
* **OTel Integration**: The Redis adapter must propagate tracing context via standard OpenTelemetry Go instrumentation (`go.opentelemetry.io/otel`) so that Redis operations are visible under request span traces.

### 5. Inbound Message Pipeline Integration
Integrate STM into the active reasoning workflow by updating the inbound message processing interfaces:

* **Inbound Port Extension**: Update `MessageProcessor` in `internal/agent/application/ports/inbound/message.go`:
  ```go
  type MessageProcessor interface {
      ProcessIncomingMessage(ctx context.Context, tenantID, agentID, threadID string, payload string) (string, error)
  }
  ```

* **Pipeline Execution Sequence**: Inside `internal/agent/application/service/message_processor.go`:
  1. **Append User Turn**: Append the incoming user prompt to Redis:
     `model.MemoryEntry{Role: "user", Content: payload, Timestamp: time.Now().UTC()}`
  2. **Fetch History**: Retrieve the sliding window from Redis up to `limit` (configurable via configuration parameter `TS_AGENT_STM_LIMIT` or parsed from the agent runtime configuration, defaulting to `10` turns).
     - *If Redis fails*, catch the error, log a warning, and construct a fallback slice containing only the current user message turn.
  3. **Trigger reasoning engine**: Call `s.brain.Generate(ctx, BrainRequest{Prompt: payload, History: history, ...})`
  4. **Append Assistant Turn**: Append the generated response:
     `model.MemoryEntry{Role: "assistant", Content: response, Timestamp: time.Now().UTC()}`
  5. **Return output**.

* **NATS Subscriber Ingress**: Update the driving adapter `internal/agent/adapters/inbound/nats/echo_subscriber.go` to capture the `thread_id` from the incoming NATS message envelope (or falling back to a default value/generating one if missing) and pass it to the updated `ProcessIncomingMessage` driving interface.

---

## Acceptance Criteria

1. **Strict Multi-Tenant Separation**:
   - Redis memory keys created under tenant `T1` must never be readable or mutable under tenant `T2`.
   - Redis keys must strictly start with `ts:tenant:{tenant_id}:agent:{agent_id}:stm:{thread_id}`.
2. **Graceful Fallback on Redis Outages**:
   - Simulating Redis connection failure must result in the agent successfully completing the `EchoRequest` turn (returning a valid LLM response based solely on the current prompt) and logging a Zerolog `WARN` message.
3. **Configurable Sliding Window**:
   - Setting `TS_AGENT_STM_LIMIT=3` must restrict the history retrieved and sent to the LLM to at most 3 entries (user + assistant turns).
4. **TTL Expiry**:
   - Setting `TS_AGENT_STM_TTL=1h` must cause inactive threads to expire and be pruned from Redis after exactly 1 hour.
5. **Observability Integration**:
   - OpenTelemetry spans must be generated for all Redis adapter actions and properly correlated with incoming NATS message traces.

---

## Test Plan

### Automated Tests
1. **Unit Tests**:
   - Verify serialization and deserialization of `MemoryEntry` JSON payloads.
   - Verify sliding window truncation limits and index offset calculations.
2. **Integration Tests**:
   - Launch local Redis using `testcontainers-go` and run integration test suites verifying:
     - Multi-tenant query isolation.
     - TTL expiration triggering.
     - Standard `Append`, `Get`, and `Clear` cycles.
   - Verify mock memory processing execution pipeline with connection failure injection to assert graceful stateless fallback behavior.
3. **Contract/E2E Probes**:
   - Assert Redis ping verification is part of the parallel `/readyz` probe logic for the Agent component.

---

## Files Affected

- `[NEW] internal/agent/domain/model/memory.go` — STM domain entities.
- `[MODIFY] internal/agent/domain/model/brain.go` — extend request model to support history.
- `[NEW] internal/agent/application/ports/outbound/memory.go` — STM outbound port interface.
- `[MODIFY] internal/agent/application/ports/inbound/message.go` — expand pipeline interface parameters.
- `[NEW] internal/agent/adapters/outbound/redis/memory_adapter.go` — Redis adapter with OTel.
- `[MODIFY] internal/agent/application/service/message_processor.go` — orchestrate STM write, retrieve, and LLM reasoning.
- `[MODIFY] internal/agent/adapters/inbound/nats/echo_subscriber.go` — parse thread ID and propagate parameters.
- `[MODIFY] specs/INDEX.md` — transition status.
