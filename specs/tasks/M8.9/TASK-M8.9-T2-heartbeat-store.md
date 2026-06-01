# TASK-M8.9-T2: HeartbeatStore Port & In-Memory Adapter

| Field       | Value                                      |
|-------------|--------------------------------------------|
| Task ID     | TASK-M8.9-T2                               |
| Spec        | SPEC-FR-M8.9                               |
| Boundary    | Application Outbound Port + Memory Adapter |
| Status      | TODO                                       |
| Depends On  | TASK-M8.9-T1                               |

## Objective

Define the `HeartbeatStore` outbound driven port interface and implement it as a goroutine-safe, `sync.Map`-backed in-memory adapter.

## Files

| File | Action |
|------|--------|
| `internal/operator/application/ports/outbound/heartbeat_store.go` | NEW |
| `internal/operator/adapters/outbound/memory/heartbeat_store.go` | NEW |
| `internal/operator/adapters/outbound/memory/heartbeat_store_test.go` | NEW |

## RED Phase

Create `internal/operator/adapters/outbound/memory/heartbeat_store_test.go` with the following test cases:

- `TestRecordAndGet`: Call `RecordHeartbeat("ns/agent-a")`, then `LastHeartbeat("ns/agent-a")` returns a time within the last second.
- `TestLastHeartbeatUnknownKey`: `LastHeartbeat("ns/unknown")` returns `time.Time{}` (zero value).
- `TestDelete`: After `RecordHeartbeat("ns/agent-a")` and `Delete("ns/agent-a")`, `LastHeartbeat("ns/agent-a")` returns zero value.
- `TestConcurrentAccess`: Spawn 50 goroutines each calling `RecordHeartbeat` on distinct keys simultaneously; assert no panic and all keys retrievable. Run with `-race`.

Run `make test` — tests must fail (RED, files don't exist yet).

## GREEN Phase

**Port interface** — `internal/operator/application/ports/outbound/heartbeat_store.go`:

```go
package outbound

import "time"

// HeartbeatStore is the driven outbound port for tracking per-agent last-active timestamps.
// The in-process implementation is in-memory only and not persisted to Kubernetes.
type HeartbeatStore interface {
    // RecordHeartbeat records the current wall clock time for the given agent key (format: "namespace/name").
    RecordHeartbeat(key string)
    // LastHeartbeat returns the last recorded time for the given key, or the zero time.Time if never seen.
    LastHeartbeat(key string) time.Time
    // Delete removes the heartbeat entry for the given key.
    Delete(key string)
}
```

**In-memory adapter** — `internal/operator/adapters/outbound/memory/heartbeat_store.go`:

```go
package memory

import (
    "sync"
    "time"

    "github.com/morphy76/tacito-square/internal/operator/application/ports/outbound"
)

// MemoryHeartbeatStore implements outbound.HeartbeatStore using a sync.Map.
type MemoryHeartbeatStore struct {
    m sync.Map
}

var _ outbound.HeartbeatStore = (*MemoryHeartbeatStore)(nil)

func NewMemoryHeartbeatStore() *MemoryHeartbeatStore {
    return &MemoryHeartbeatStore{}
}

func (s *MemoryHeartbeatStore) RecordHeartbeat(key string) {
    s.m.Store(key, time.Now())
}

func (s *MemoryHeartbeatStore) LastHeartbeat(key string) time.Time {
    v, ok := s.m.Load(key)
    if !ok {
        return time.Time{}
    }
    return v.(time.Time)
}

func (s *MemoryHeartbeatStore) Delete(key string) {
    s.m.Delete(key)
}
```

Run `make test` — tests must pass (GREEN).

## REFACTOR Phase

- Confirm `var _ outbound.HeartbeatStore = (*MemoryHeartbeatStore)(nil)` compile-time guard is present.
- Ensure no `time.Now()` calls leak outside of `RecordHeartbeat` (the `LastHeartbeat` method must be pure read).
