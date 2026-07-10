# SPEC-FR-M6.5.13: A2A Agent Cards Consolidation for Hub Delegation

| Field       | Value                                                                          |
|-------------|--------------------------------------------------------------------------------|
| ID          | SPEC-FR-M6.5.13                                                                |
| Status      | DRAFT                                                                          |
| Milestone   | M6.5                                                                           |
| Component   | agent, keeper                                                                  |
| Depends On  | SPEC-FR-M6.5 (M6 — existing A2A Agent Cards, VERIFIED), SPEC-FR-M6.5.1, SPEC-FR-M6.5.4 |
| Supersedes  | none                                                                           |

## Context

SPEC-FR-M6.5 (M6, VERIFIED) defined the foundational A2A agent card structure published by agents
on NATS. In that baseline, cards carried identity fields (`agent_id`, `name`, `description`,
`status`) but no capability metadata. A hub agent attempting intelligent delegation had no way to
distinguish spokes by their skills without out-of-band knowledge.

This spec enriches the agent card schema with the spoke's active skill capabilities and formalises
the hub's delegation cognitive loop. The hub's own procedural routing skills (loaded via
SPEC-FR-M6.5.4) provide the domain logic for selecting the right spoke; the enriched cards
provide the spoke-side capability signal.

The spec also adds a Keeper REST endpoint that serves the latest cached card state per community,
enabling the BFF and UI to surface spoke capabilities without subscribing to NATS directly.

## Specification

### 1. Enriched Agent Card Schema

Extend the existing agent card with a `capabilities` array and a `role` field. The full schema
(serialised as JSON on NATS and in the Keeper Redis cache) is:

```json
{
  "agent_id":    "uuid",
  "name":        "string",
  "description": "string",
  "role":        "spoke | standalone | hub",
  "community_id":"uuid",
  "status":      "running | idle | stale",
  "capabilities": [
    {
      "name":        "string",
      "description": "string"
    }
  ],
  "published_at": "RFC3339 timestamp"
}
```

**Field rules:**
- `role` reflects the agent's effective role in its current community assignment.
- `capabilities` contains the names and **one-line descriptions** of the agent's resolved active
  skills, sourced from `PropagatedAgentConfig.Skills` loaded at startup (per SPEC-FR-M6.5.6).
- `SkillConfig.Content` is **never** included in the published card. Skill content is sensitive
  procedural knowledge and must remain within the agent's runtime boundary.
- `status: stale` is set by the hub card registry when `published_at` is older than 90 seconds,
  not by the publishing agent itself. Publishing agents emit only `running` or `idle`.

### 2. Card Publication by Agents

**Publication events:**
- On agent startup (after successfully loading `PropagatedAgentConfig`).
- On any status transition (`running` ↔ `idle`).
- Every **60 seconds** as a heartbeat — the agent publishes its current card to maintain
  freshness in the hub's registry.

**NATS publication:**
- The subject is the same as defined in SPEC-FR-M6.3 (existing community card subject).
- The payload is the enriched JSON schema above.
- Publication uses the existing NATS adapter in
  `internal/agent/adapters/inbound/nats/event_subscriber.go` (extended to cover startup and
  heartbeat publication).

**Content sourcing:**
- `capabilities` is populated from `PropagatedAgentConfig.Skills` at startup. Each
  `SkillConfig.{Name, Description}` pair maps to one capability entry.
- If the agent has no active skills, `capabilities` is an empty array `[]` — never `null`.

### 3. Hub Card Registry and Staleness

The hub agent maintains an **in-memory card registry** keyed by `agent_id`:

```
cardRegistry: map[string]EnrichedAgentCard  // agent_id -> card
```

**Update:** when a card arrives on NATS (via the existing subscription), the hub updates the
registry entry with the latest card.

**Staleness rule:**
- At read time (when `list_available_agents` is called), a card is considered `stale` if
  `time.Since(card.PublishedAt) > 90 * time.Second`.
- Stale cards are excluded from the `list_available_agents` response. They are NOT removed from
  the registry immediately; they may recover if a fresh heartbeat arrives.

**Hub-as-agent:** hub agents do not publish their own card to the spoke registry. Only
`spoke` and `standalone` agents publish cards that are useful for delegation.

### 4. `list_available_agents` Built-in Tool

The hub cognitive engine exposes `list_available_agents` as a function-call tool (registered in
`buildToolDefinitions()` for hub-role agents only):

**Tool definition:**
```json
{
  "name": "list_available_agents",
  "description": "Returns the current non-stale spoke and standalone agents in the community, including their capabilities. Use this to identify which agent to delegate a task to.",
  "parameters": {
    "type": "object",
    "properties": {},
    "required": []
  }
}
```

**Execution:**
1. Iterate the in-memory card registry.
2. Filter to cards where `time.Since(PublishedAt) <= 90s`.
3. Return a JSON array of filtered cards (excluding skill `content`, which is never in the card).
4. If the registry is empty or all cards are stale, return an empty array `[]` with a note in
   the observation: `"no available agents at this time"`.

### 5. Hub Delegation Cognitive Loop

The standard hub turn proceeds as follows (the LLM may deviate from this if its routing skill
instructs otherwise):

1. User request arrives as a new thread message.
2. Layer 1 (hub structural template) instructs the hub not to execute actions directly.
3. Hub calls `enable_skill("routing_policy")` (or whichever routing skill is available) to load
   domain-specific routing guidelines from Layer 3.
4. Hub calls `list_available_agents()` to retrieve the current spoke capability manifest.
5. Hub reasons about which spoke's `capabilities` best match the request, informed by the
   routing skill content.
6. Hub calls `delegate_to_agent(spoke_id, task_description)` (existing mechanism, SPEC-FR-M6.6).
7. Hub receives the spoke's response as a tool observation.
8. Hub synthesises the final answer and responds to the user.

This loop is described here as a normative expectation, not as hardcoded control flow. The LLM
drives the loop; the tools and the system prompt make this the natural behaviour.

### 6. Keeper API — Community Agent Cards Endpoint

A new Keeper REST endpoint allows the BFF and UI to retrieve the cached card state without NATS:

```
GET /api/v1/communities/{id}/agent-cards
```

**Behaviour:**
- Returns the latest known card for each agent in the community, sourced from a Redis cache
  updated in real-time as cards arrive on NATS.
- If no cards are known, returns an empty array `[]`.
- Does NOT filter stale cards on the server side — the caller decides how to interpret
  `published_at` freshness. Stale filtering is a client/hub concern.

**Response schema:**
```json
[
  {
    "agent_id":    "uuid",
    "name":        "string",
    "description": "string",
    "role":        "string",
    "community_id":"uuid",
    "status":      "string",
    "capabilities": [
      { "name": "string", "description": "string" }
    ],
    "published_at": "RFC3339"
  }
]
```

**Cache update:** the Keeper's NATS subscriber (or a dedicated card listener) must upsert the
Redis cache entry `community:{community_id}:agent_cards:{agent_id}` on every card publication.
The Redis TTL for each entry MUST be set to 120 seconds (longer than the 90-second staleness
threshold, ensuring at least one missed heartbeat before expiry).

## API Contract

### GET /api/v1/communities/{id}/agent-cards

| Field        | Value                                      |
|--------------|--------------------------------------------|
| Method       | GET                                        |
| Auth         | Required (JWT bearer, standard middleware) |
| Tag          | `community/agent-cards`                    |
| Path Param   | `id` — community UUID (required)           |
| Success      | 200 OK — JSON array of enriched agent cards |
| Not Found    | 404 `{"error": "community not found"}`    |
| Auth Failure | 401 `{"error": "unauthorized"}`           |

**OpenAPI tag:** `community/agent-cards` (follows `domain/subdomain` boundary tagging standard).

## Acceptance Criteria

1. A spoke agent's published NATS card includes a `capabilities` array with one entry per active
   skill. The `content` field is absent from each capability.
2. A hub agent's in-memory card registry correctly marks a card as stale when
   `time.Since(PublishedAt) > 90s`.
3. The `list_available_agents` tool returns only non-stale spoke/standalone cards; it excludes
   hub cards and stale cards.
4. A hub agent with a `routing_policy` skill attached can call `enable_skill("routing_policy")`
   and receives the skill content in the tool observation.
5. `GET /api/v1/communities/{id}/agent-cards` returns cached card data from Redis (including
   `capabilities`), with HTTP 200 and a valid JSON array.
6. The Keeper Redis cache entry for each card has a TTL of 120 seconds.
7. `capabilities` is an empty array `[]` (never null) for agents with no active skills.

## Test Plan

### Unit — Staleness detection
- Build a card with `PublishedAt = time.Now().Add(-91 * time.Second)` and assert the registry
  marks it as stale.
- Build a card with `PublishedAt = time.Now().Add(-89 * time.Second)` and assert it is not stale.

### Unit — `list_available_agents` filtering
- Registry with 3 cards (1 stale, 1 hub, 1 fresh spoke); assert the tool returns exactly 1 card.
- Empty registry; assert the tool returns `[]` and the observation note indicates no agents.

### Unit — Card schema serialization
- Create an `EnrichedAgentCard` with 2 skills; marshal to JSON; assert `capabilities` has 2
  entries each with `name` and `description`; assert no `content` field is present.

### Integration — Card publication and hub registry update
- Start a mock spoke agent that publishes a card with capabilities on NATS.
- Start a mock hub that subscribes to the NATS subject.
- Assert the hub's card registry is updated within 1 second of publication.

### Integration — Keeper REST endpoint
- Publish two cards to the Keeper's Redis cache.
- Call `GET /api/v1/communities/{id}/agent-cards`.
- Assert the response contains both cards with correct structure.

## Files Affected

| File | Change |
|------|--------|
| `internal/agent/domain/model/community_card.go` | **MODIFY** — add `Capabilities []CapabilityEntry` and `Role` fields to enriched card schema |
| `internal/agent/application/service/cognitive_engine.go` | **MODIFY** — add `list_available_agents` built-in tool; hub-only card registry staleness filtering |
| `internal/agent/adapters/inbound/nats/event_subscriber.go` | **MODIFY** — card publication on startup, status change, and 60s heartbeat |
| `internal/agent/adapters/outbound/nats/card_publisher.go` | **NEW** — NATS card publisher (outbound driven adapter) |
| `internal/keeper/adapters/inbound/http/community_handler.go` | **MODIFY** — add `GET /api/v1/communities/{id}/agent-cards` endpoint |
| `internal/keeper/adapters/outbound/redis/card_cache.go` | **MODIFY** — upsert enriched cards with 120s TTL; support by-community list query |
| `internal/keeper/application/ports/outbound/card_cache_port.go` | **NEW or MODIFY** — outbound port interface for card cache operations |
