# SPEC-FR-M11.4: Thread Management

| Field         | Value                                                         |
|---------------|---------------------------------------------------------------|
| ID            | SPEC-FR-M11.4                                                 |
| Status        | DRAFT                                                         |
| Milestone     | M11                                                           |
| Component     | conversation-hub                                              |
| Depends On    | SPEC-FR-M3.6, SPEC-FR-M6.0                                    |
| Supersedes    | none                                                          |

## Context

Conversations within communities are organized into threads for multi-turn engagement tracking. To keep the Keeper (control plane) separated from runtime data-plane traffic, thread management and message history persistence are isolated into a new dedicated service: `conversation-hub`.

The `conversation-hub` is responsible for persisting thread/message states in a dedicated PostgreSQL schema/database, exposing CRUD APIs, and subscribing to the NATS event bus to keep historical states in sync with real-time domain events.

---

## Specification

### 1. Domain Entities

The `conversation-hub` service MUST define the following domain entities in its domain layer:

#### 1.1 `Thread` Entity
* **Fields**:
  * `ID`: UUID (Primary Key)
  * `TenantID`: String (tenant isolation)
  * `CommunityID`: UUID (reference to the community)
  * `Status`: Enum/String (`active`, `paused`, `completed`)
  * `Participants`: List/Array of Agent IDs (agents actively assigned or involved)
  * `CreatedAt`: Timestamp (UTC)
  * `UpdatedAt`: Timestamp (UTC)
* **Constraints**:
  * Strict validation on `Status`.
  * Composite index on `(tenant_id, community_id)`.

#### 1.2 `Message` Entity
* **Fields**:
  * `ID`: UUID (Primary Key)
  * `TenantID`: String
  * `ThreadID`: UUID (Foreign Key to `Thread`)
  * `SenderID`: String (e.g., `"user"` or `agent_id`)
  * `Role`: Enum/String (`user`, `assistant`)
  * `Content`: Text (sanitized message content)
  * `OccurredAt`: Timestamp (UTC)
* **Constraints**:
  * Foreign key constraint linking `ThreadID` to `Thread` table.
  * Index on `(tenant_id, thread_id)`.
  * **Scope**: Captured messages are strictly limited to complete conversational messages (user input and final agent replies). Intermediate LLM reasoning traces or tool execution details MUST NOT be persisted in this entity.

---

### 2. REST API Endpoints

The `conversation-hub` service MUST expose the following REST endpoints using the Gin framework (per `SPEC-NFR-HTTP`). These endpoints are designed to be internal-facing and proxied by the BFF (`SPEC-FR-M7.1`), which handles client-facing security, model translations, and OIDC token verification.

#### 2.1 GET `/api/v1/communities/{community_id}/threads`
* **Purpose**: List all threads in a community.
* **Query Params**:
  * `status`: Optional filter (e.g. `active`, `completed`).
  * `limit`, `offset`: Standard pagination parameters.
* **Response**: `200 OK` with JSON array of `Thread` models (empty array if none found).

#### 2.2 POST `/api/v1/communities/{community_id}/threads`
* **Purpose**: Create a new conversation thread.
* **Payload**:
  ```json
  {
    "thread_id": "optional-uuid",
    "participants": ["agent-uuid-1"],
    "initial_message": {
      "sender_id": "user",
      "content": "Hello, world!"
    }
  }
  ```
* **Behavior**:
  * Generates a `Thread` entity.
  * If `initial_message` is provided, sanitizes the message content using the shared sanitization utility (Cc/Cs Unicode characters stripping, 4096-character limit truncation) and creates a `Message` entity.
  * Publishes a `start-thread` domain event (and optionally `add-user-message` event) to NATS subject `ts.community.{community_id}.agent.{target_agent}`.
* **Response**: `201 Created` with the created `Thread` entity and a `Location` header pointing to `/api/v1/communities/{community_id}/threads/{thread_id}`.

#### 2.3 GET `/api/v1/communities/{community_id}/threads/{id}`
* **Purpose**: Get a specific thread along with its message history.
* **Response**: `200 OK` with the `Thread` entity nested with its historical list of `Message` entities. Returns `404 Not Found` if missing or tenant mismatch.

#### 2.5 PUT `/api/v1/communities/{community_id}/threads/{id}`
* **Purpose**: Update thread status (e.g. archiving or pausing).
* **Payload**:
  ```json
  {
    "status": "completed"
  }
  ```
* **Behavior**: Updates the thread status and publishes an `end-thread` event to NATS if the status changes to `completed`.
* **Response**: `200 OK` with the updated `Thread` object.

#### 2.6 DELETE `/api/v1/communities/{community_id}/threads/{id}`
* **Purpose**: Hard/soft delete a thread and its messages.
* **Response**: `204 No Content`.

---

### 3. NATS Integration (Data Ingestion Loop)

The `conversation-hub` MUST act as both an event producer and consumer to remain in sync with the asynchronous agent cluster.

#### 3.1 Wildcard Subscription (Consumer)
* **Subject**: Subscribes to the NATS hierarchical wildcard subject `ts.community.>` (capturing both inbound requests and agent replies).
* **Event Ingestion Handler**:
  * **`start-thread` event**: Creates the thread record in PostgreSQL if it doesn't already exist.
  * **`add-user-message` event**: Appends a `user` role message to the thread record. The content is sanitized before insertion.
  * **`agent-response` event**: Appends an `assistant` role message with the agent response to the thread record.
  * **`end-thread` event**: Marks the thread status as `completed` in PostgreSQL.

#### 3.2 Event Publisher
* When threads or messages are created directly via `conversation-hub` REST endpoints, the service MUST construct and publish standard `DomainEvent` envelopes to NATS (complying with `SPEC-FR-M6.0`).

---

### 4. Non-Functional & Architecture Requirements

* **Code Architecture**: Follow Ports & Adapters (`RULE[code-architecture.md]`).
  ```text
  internal/conversation-hub/
  ├── domain/
  │   └── model/             # Thread, Message aggregates
  ├── application/
  │   ├── ports/
  │   │   ├── inbound/       # ThreadUseCase, EventIngressPort
  │   │   └── outbound/      # ThreadRepository, EventPublisher
  │   └── service/           # Thread service logic
  └── adapters/
      ├── inbound/           # Gin handlers, NATS subscribers
      └── outbound/          # pgx database client, NATS publisher client
  ```
* **Shared Message Sanitizer**: A shared helper utility (`pkg/events/sanitizer.go`) is utilized by both `keeper` (during event ingress) and `conversation-hub` (during REST/NATS message processing) to execute message sanitization uniformly.
* **Multitenancy**: Strict validation and injection of `tenant_id` from claims forwarded by the BFF proxy (governed by `RULE[cloud-first.md]`).
* **Database Isolation**: The `conversation-hub` connects to the same PostgreSQL instance as `keeper` but MUST use a separate logical database or dedicated database schema to ensure isolation, utilizing `goose` for migrations.
* **Observability**: Standard metrics (connections, thread count, latency) exposed via `GET /metrics` and spans instrumented via OpenTelemetry.
* **Liveness & Readiness Probes**: `/healthz` and `/readyz` verifying database and NATS connections in parallel.

---

## Acceptance Criteria

1. **Database Schema Separation**: Thread and message tables are managed via independent migrations in `deploy/postgres/migrations/conversation-hub/` and run on startup.
2. **Tenant Separation**: All Thread and Message tables feature a non-nullable `tenant_id` column with corresponding database indexes. All CRUD SQL queries filter strictly by this parameter.
3. **NATS Wildcard Sync**: Emitting a conversational event (`start-thread`, `add-user-message`, or `agent-response`) to NATS results in a corresponding DB insert in the `conversation-hub` PostgreSQL schema.
4. **CRUD REST Endpoints**: The Gin routes for thread list, get, create, status update, and delete are exposed, tested in `gin.TestMode`, and return clean JSON structures.
5. **Location Header**: Creating a thread via `POST` returns `201 Created` with a `Location` header pointing to `/api/v1/communities/{community_id}/threads/{id}`.
6. **OpenAPI Spec**: The service serves a valid OpenAPI 3.x document at `GET /openapi.json`.
7. **Containerization**: The runtime stage uses `gcr.io/distroless/base-nossl-debian13`.

---

## Test Plan

### Automated Tests

1. **Unit Tests**:
   * Model validations (Thread status, Message sender properties).
   * Schema router matching logic.
   * Shared sanitization utility verification.
2. **Integration Tests**:
   * Database repository tests (CRUD, tenant isolation checks, index coverage verification).
   * Event publication and wildcard consumer subscription tests using an in-memory NATS server.
3. **HTTP API tests**:
   * Mocking the `ThreadUseCase` in `gin.TestMode` to verify REST API responses and headers.

### Manual Verification
1. Spin up the `conversation-hub` alongside `keeper`, `agent`, and PostgreSQL.
2. Direct a `POST` request to start a thread; verify record in DB.
3. Stream a user message and ensure NATS wildcard handler appends it.
4. Verify `GET /api/v1/communities/{id}/threads` lists the active thread.
