# SPEC-FR-M3.6: Community Domain Model & CRUD API

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | SPEC-FR-M3.6                                |
| Status        | IMPLEMENTED                                 |
| Milestone     | M3                                          |
| Component     | keeper                                      |
| Depends On    | SPEC-FR-M2.3, SPEC-FR-M3.5                  |
| Supersedes    | none                                        |

## Context

### Definition of a Community
A **Community** represents a logical boundary of interconnected agents. Under the hood, it defines a secure messaging domain where agents within the same community can interact asynchronously. 
- **Logical Boundary**: Agents belonging to different communities **cannot** communicate with each other directly. Cross-community communication is blocked at the network/subject level. Explicit gateways and cross-community routing are deferred to future specifications (e.g., `SPEC-FR-M9.1`).
- **Tenant-Awareness**: Communities are fully tenant-aware. A community belongs to exactly one tenant. All configurations, definitions, and boundaries are strictly segregated by tenant ID.
- **Deployability**: Communities are logical structures and **are not** deployable units themselves (i.e., they do not run as standalone pods or containers). 
- **Runtime Existence**: In the runtime environment (the cluster), a community is instantiated and exists only once the first agent belonging to it is deployed. Until that first agent deployment, it remains a purely logical definition in `keeper`.
- **Agent Relationship**: An agent can be part of **just one** community at any given time. This ensures clean isolation boundaries and strict hierarchical mapping.

---

## Architectural & Functional Design

### 1. Messaging Boundary Realization
To enforce the logical boundary, the community maps to namespaced **NATS subjects**. For example:
- All agent-to-agent communication in a community uses subjects prefixed with:
  `tacito.community.<community_id>.*`
- An agent's mailbox subject is structured as:
  `tacito.community.<community_id>.agent.<agent_id>.mailbox`
- Because NATS credentials and authorization rules (provisioned during agent startup) restrict access strictly to `tacito.community.<community_id>.*` for agents of that community, **cross-community communication is impossible** without a dedicated gateway.

### 2. Topology Configuration
Initially, communities support a `hub-spoke` topology:
- **Topology Definition**: Deferred to future specifications (e.g. `SPEC-FR-M6.1`), where we will drill down into topologies, gateways, and external connectivity.
- *Constraint Avoidance*: To maintain simplicity and flexibility, `keeper` **does not** enforce topology-specific validation rules (such as requiring a Hub agent) to consider a community "active". The community model simply maintains the metadata representing the topology choice.

### 3. Cascading & Status Rules
- **Suspended/Terminated status**: If a community's status is set to `suspended` or `terminated`, all constituent agents must transition to a non-functional state (e.g., stopped or suspended).
- **Deletion Check**: A community cannot be deleted if there are any agents associated with it, ensuring referential integrity.

---

## Specification

### 1. Domain Layer (`internal/keeper/domain`)

The domain layer MUST define the `Community` aggregate root and helper enums:

```go
package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

type CommunityStatus string

const (
	CommunityStatusCreated    CommunityStatus = "created"
	CommunityStatusActive     CommunityStatus = "active"
	CommunityStatusSuspended  CommunityStatus = "suspended"
	CommunityStatusTerminated CommunityStatus = "terminated"
)

type CommunityTopology string

const (
	CommunityTopologyHubSpoke CommunityTopology = "hub-spoke"
)

type Community struct {
	ID            uuid.UUID              `json:"id"`
	TenantID      string                 `json:"tenant_id"`
	Name          string                 `json:"name"` // Unique per tenant
	Description   string                 `json:"description"`
	Topology      CommunityTopology      `json:"topology"`
	Configuration map[string]interface{} `json:"configuration"`
	Status        CommunityStatus        `json:"status"`
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
}

func (c Community) Validate() error {
	if c.ID == uuid.Nil {
		return errors.New("id is required")
	}
	if c.TenantID == "" {
		return errors.New("tenant id is required")
	}
	if c.Name == "" {
		return errors.New("name is required")
	}
	if c.Topology != CommunityTopologyHubSpoke {
		return errors.New("invalid or unsupported topology")
	}
	if c.Status != CommunityStatusCreated &&
		c.Status != CommunityStatusActive &&
		c.Status != CommunityStatusSuspended &&
		c.Status != CommunityStatusTerminated {
		return errors.New("invalid community status")
	}
	return nil
}
```

> [!NOTE]
> Per `SPEC-NFR-HEXAGONAL`, the domain files MUST NOT import any application or adapter packages.

### 2. Modifying the `Agent` Domain Model
To support the requirement that **an agent can be part of just one community**, we modify the `Agent` struct in `internal/keeper/domain/agent.go` to include:
```go
// Add to internal/keeper/domain/agent.go
type Agent struct {
	// Existing fields ...
	CommunityID *uuid.UUID `json:"community_id"` // Reference to the assigned community (optional/nullable)
}
```

---

## Database Schema (PostgreSQL & Goose Migrations)

### 1. `communities` Table
```sql
CREATE TABLE communities (
    id UUID PRIMARY KEY,
    tenant_id VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    topology VARCHAR(50) NOT NULL DEFAULT 'hub-spoke',
    configuration JSONB NOT NULL DEFAULT '{}',
    status VARCHAR(50) NOT NULL DEFAULT 'created',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL,
    
    CONSTRAINT uq_tenant_community_name UNIQUE (tenant_id, name)
);

CREATE INDEX idx_communities_tenant ON communities(tenant_id);
```

### 2. `agents` Table Migration
```sql
ALTER TABLE agents ADD COLUMN community_id UUID;
ALTER TABLE agents ADD CONSTRAINT fk_agents_community FOREIGN KEY (community_id) REFERENCES communities(id) ON DELETE RESTRICT;
CREATE INDEX idx_agents_community ON agents(community_id);
```

---

## HTTP REST API Endpoints

All endpoints are prefixed with `/api/v1` and must run the `TenantResolutionMiddleware` to extract `TenantID` from request context.

### 1. Endpoints List

- `POST /api/v1/communities`: Create a community.
- `GET /api/v1/communities`: List communities for the tenant.
- `GET /api/v1/communities/:id`: Get a specific community by ID.
- `PUT /api/v1/communities/:id`: Update a community.
- `DELETE /api/v1/communities/:id`: Delete a community.

### 2. Request/Response Payloads Documented

#### POST `/api/v1/communities`
- **Request Body**:
  ```json
  {
    "name": "sales-agents-net",
    "description": "Internal network for sales automation",
    "topology": "hub-spoke",
    "configuration": {
      "max_messages_per_sec": 100
    }
  }
  ```
- **Response (201 Created)**:
  ```json
  {
    "id": "e2a149cb-40fc-4cbb-a30f-b4de9756b1ad",
    "tenant_id": "morphy76.tacito-square.io",
    "name": "sales-agents-net",
    "description": "Internal network for sales automation",
    "topology": "hub-spoke",
    "configuration": {
      "max_messages_per_sec": 100
    },
    "status": "created",
    "created_at": "2026-05-23T14:00:00Z",
    "updated_at": "2026-05-23T14:00:00Z"
  }
  ```

#### GET `/api/v1/communities/:id`
- **Response (200 OK)**:
  ```json
  {
    "id": "e2a149cb-40fc-4cbb-a30f-b4de9756b1ad",
    "tenant_id": "morphy76.tacito-square.io",
    "name": "sales-agents-net",
    "description": "Internal network for sales automation",
    "topology": "hub-spoke",
    "configuration": {
      "max_messages_per_sec": 100
    },
    "status": "active",
    "created_at": "2026-05-23T14:00:00Z",
    "updated_at": "2026-05-23T14:15:00Z"
  }
  ```

#### Error Response Format (Standard JSON Error):
- **Response (400 Bad Request / 409 Conflict)**:
  ```json
  {
    "error": "community name 'sales-agents-net' already exists for this tenant"
  }
  ```

---

## Acceptance Criteria

1. **Domain Isolation**:
   - `Community` and `Agent` models support tenant scoping.
   - `Agent` model includes a nullable `community_id` to guarantee that an agent template belongs to exactly one (or zero) community at any time.

2. **Tenant Segregation**:
   - A tenant cannot access, retrieve, modify, or delete a community belonging to a different tenant.
   - Unique name constraint `uq_tenant_community_name` prevents duplicate community names inside the *same* tenant, but allows identical community names across *different* tenants.

3. **Hexagonal Boundaries**:
   - Outbound ports (repository interfaces) declared in `internal/keeper/ports` use domain entities.
   - Domain layer holds zero external frameworks or adapter library imports.

4. **Robust REST Controller**:
   - Handlers correctly return standard HTTP status codes: 201 for success creation, 400 for structural invalidity, 409 for name conflicts, 404 for non-existent entities, and 401/403 for missing or unauthorized tenant headers.

---

## Test Plan

### 1. Repository Integration Tests (`internal/keeper/adapters/postgres/community_repository_test.go`)
- **Lifecycle Integration**:
  - Test saving, updating, fetching, listing, and deleting `Community` records.
  - Verify GORM database serialization of `configuration` JSONB map field.
- **Tenant Scope Enforcement**:
  - Insert community under `Tenant A`.
  - Verify that query with `Tenant B` context cannot locate the community by ID, Name, or List.
- **Unique Per-Tenant Name Constraint**:
  - Attempt to insert two communities with identical names under `Tenant A` (must fail/conflict).
  - Insert community with the same name under `Tenant B` (must succeed).
- **Referential Integrity & Cascading Constraint**:
  - Associate an agent with a community by setting `community_id`.
  - Attempt to delete the community and verify that `ON DELETE RESTRICT` raises a foreign key violation, preventing community deletion with active agents.

### 2. HTTP Handler Unit & Integration Tests
- **Payload Validation**:
  - Test that POST payloads missing `name` or with unsupported `topology` fail structurally with 400 Bad Request.
- **Tenant Middleware Resolution**:
  - Test that omitting the tenant headers returns 401 Unauthorized.
- **Mocked repository behavior**:
  - Validate GET, PUT, and DELETE routes return appropriate success and error responses based on repo returns.

---

## Files Affected

- `internal/keeper/domain/community.go` [NEW] — Defines the `Community` aggregate root and validators.
- `internal/keeper/domain/agent.go` [MODIFY] — Adds `CommunityID` reference.
- `internal/keeper/ports/repositories.go` [MODIFY] — Declares outbound ports for `CommunityRepository`.
- `internal/keeper/adapters/postgres/community_repository.go` [NEW] — Implements DB persistence using PostgreSQL.
- `internal/keeper/adapters/postgres/community_repository_test.go` [NEW] — Complete integration testing suite covering tenant isolation.
- `internal/keeper/adapters/http/community_handlers.go` [NEW] — Gin-based handlers for communities CRUD.
- `internal/keeper/adapters/http/community_handlers_test.go` [NEW] — HTTP handler test suites.
- `internal/keeper/bootstrap.go` [MODIFY] — Hooks routes and binds dependencies under `/api/v1/communities`.
