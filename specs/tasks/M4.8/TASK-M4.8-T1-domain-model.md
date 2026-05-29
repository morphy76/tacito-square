# TASK-M4.8-T1: Echo Domain Model

| Field       | Value                                         |
|-------------|-----------------------------------------------|
| Task ID     | TASK-M4.8-T1                                  |
| Spec        | SPEC-FR-M4.8                                  |
| Boundary    | Domain Layer — `internal/keeper/domain/model` |
| Status      | IMPLEMENTED                                   |
| Depends On  | —                                             |

## Objective

Define the pure domain value types and stateless functions for the echo feature: input sanitization, message decoration, and the data structures shared across all layers.

## Files

| File | Action |
|------|--------|
| `internal/keeper/domain/model/echo.go` | NEW |
| `internal/keeper/domain/model/echo_test.go` | NEW |

## RED Phase

Create `internal/keeper/domain/model/echo_test.go` with the following test cases:

- `TestSanitizeMessage_StripNonPrintable`: Input `"hello\x00world\x01"` → `"helloworld"`.
- `TestSanitizeMessage_Truncates`: Input of 1100 characters → sanitized output is exactly 1000 characters.
- `TestSanitizeMessage_EmptyAfterStrip`: Input of only control characters → returns `""`.
- `TestSanitizeMessage_CleanInput`: Input `"hello world"` → unchanged `"hello world"`.
- `TestDecorateMessage`: `DecorateMessage("agent-alpha", "hello")` returns a string matching the pattern `[agent:agent-alpha at <RFC3339>] hello`. Assert prefix and suffix, accept any valid RFC3339 timestamp.
- `TestCommunityEchoResponse_Fields`: Verify `CommunityEchoResponse`, `AgentEchoResult`, `EchoRequest`, `EchoReply` structs can be marshalled/unmarshalled to/from JSON with expected field names.

Run `make test` — tests must fail (RED).

## GREEN Phase

Create `internal/keeper/domain/model/echo.go`:

```go
package model

import (
    "fmt"
    "time"
    "unicode"
)

// EchoRequest is the payload sent from the Keeper to an agent via NATS request-reply.
type EchoRequest struct {
    Message     string `json:"message"`
    CommunityID string `json:"community_id"`
    TenantID    string `json:"tenant_id"`
}

// EchoReply is the decorated response an agent sends back to the Keeper.
type EchoReply struct {
    AgentName string `json:"agent_name"`
    Decorated string `json:"decorated"`
    Timestamp string `json:"timestamp"` // RFC3339
}

// AgentEchoResult holds the outcome of a single per-agent echo request.
type AgentEchoResult struct {
    AgentName string `json:"agent_name"`
    Decorated string `json:"decorated"`
    Error     string `json:"error"`
}

// CommunityEchoResponse is the HTTP response payload of the echo endpoint.
type CommunityEchoResponse struct {
    CommunityID   string            `json:"community_id"`
    WokeCommunity bool              `json:"woke_community"`
    Results       []AgentEchoResult `json:"results"`
}

// SanitizeMessage strips non-printable Unicode characters from s and truncates to 1000 characters.
// Returns the sanitized string; the caller must check for an empty result.
func SanitizeMessage(s string) string {
    runes := make([]rune, 0, len(s))
    for _, r := range s {
        if unicode.IsPrint(r) {
            runes = append(runes, r)
        }
    }
    if len(runes) > 1000 {
        runes = runes[:1000]
    }
    return string(runes)
}

// DecorateMessage wraps a sanitized message with the agent's identity and a timestamp.
func DecorateMessage(agentName, message string) string {
    return fmt.Sprintf("[agent:%s at %s] %s", agentName, time.Now().UTC().Format(time.RFC3339), message)
}
```

Run `make test` — tests must pass (GREEN).

## REFACTOR Phase

- Confirm `SanitizeMessage` uses `unicode.IsPrint` consistently (covers Cc, Cs, Co categories).
- Confirm `DecorateMessage` is a pure function with no side effects beyond calling `time.Now()`.
- Ensure JSON field names match exactly those in the API Contract section of SPEC-FR-M4.8.
