# BUG-M6.12: Unsafe Distributed Lock Implementation

| Field         | Value                                                              |
|---------------|--------------------------------------------------------------------|
| ID            | BUG-M6.12                                                          |
| Status        | OPEN                                                               |
| Severity      | HIGH                                                               |
| Milestone     | M6 — Communities & Messaging                                       |
| Affects       | [state_lock_adapter.go](file:///Users/R.Pasquini/Projects/side/tacito-square/internal/agent/adapters/outbound/redis/state_lock_adapter.go) |
| Violates      | `cloud-first` / `SPEC-NFR-CLOUD`                                  |
| Discovered    | Concurrency audit of the Orchestration ThreadLock mechanism.       |

## Problem Statement

The distributed lock implementation in [state_lock_adapter.go](file:///Users/R.Pasquini/Projects/side/tacito-square/internal/agent/adapters/outbound/redis/state_lock_adapter.go) is unsafe and prone to race conditions:

1. **Static Lock Value:** The `Lock` method (line 151) uses a static string `"locked"` as the Redis key value:
   ```go
   success, err := a.client.SetNX(ctx, key, "locked", ttl).Result()
   ```
2. **Unconditional Deletion:** The `Unlock` method (line 205) releases the lock using a plain Redis `DEL` command without verifying lock ownership:
   ```go
   err := a.client.Del(ctx, key).Err()
   ```

If a slow orchestration turn or external call (like a slow LLM inference request) exceeds the lock's hardcoded TTL of 30 seconds, the lock will expire, allowing another process (e.g., Process B) to acquire it. When the original process (Process A) completes its work, its deferred `Unlock()` call will perform a plain `DEL`, releasing the lock currently held by Process B. This allows a third process (Process C) to acquire the lock while Process B is still running, violating mutual exclusion.

## Affected Components and Files

| Component / File | Location | Issue |
|------------------|----------|-------|
| Redis Memory Adapter (State/Lock) | [state_lock_adapter.go](file:///Users/R.Pasquini/Projects/side/tacito-square/internal/agent/adapters/outbound/redis/state_lock_adapter.go) | Static value `"locked"` used in `SetNX` and unconditional `DEL` in `Unlock` without ownership checking. |

## Impact

1. **Lost Isolation / Concurrency Violations:** Multiple concurrent processes can execute orchestration steps on the same conversation thread simultaneously, leading to state corruption, duplicate NATS delegation publishing, and out-of-order message processing.
2. **Fragile Error Handling:** Under load spikes or transient network slowdowns, lock safety degrades entirely, violating `SPEC-NFR-CLOUD`.

## Expected Behaviour

1. **UUID/Owner Token:** Each lock acquisition should generate a unique identifier (e.g., UUID) to serve as the lock value.
2. **Safe Unlock with Lua Script:** The `Unlock` method must only delete the Redis key if the value currently stored in Redis matches the unique identifier generated during `Lock`. This check-and-delete operation must be executed atomically using a Lua script.
3. **Configurable TTL/Retries:** The lock TTL and acquisition timeouts should be configurable rather than hardcoded.

## Acceptance Criteria

1. `Lock` generates a unique UUID per lock request and stores it in the Redis key.
2. `Unlock` verifies lock ownership using a Lua script that compares the key's value to the generated UUID before deleting it.
3. Unit tests verify that a lock cannot be released by a caller who does not own it.
4. Concurrency integration tests verify mutual exclusion behavior under high-load scenarios.
