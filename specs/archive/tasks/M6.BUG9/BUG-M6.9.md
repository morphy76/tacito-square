# BUG-M6.9: Agent Brain Embeds LLM Binding Instead of Referring LLM Binding Model

| Field         | Value                                                              |
|---------------|--------------------------------------------------------------------|
| ID            | BUG-M6.9                                                           |
| Status        | CLOSED                                                             |
| Severity      | HIGH                                                               |
| Milestone     | M6 — Communities & Messaging                                       |
| Affects       | internal/keeper/domain/model/agent.go, internal/keeper/adapters/inbound/http/agent_handlers.go, internal/keeper/adapters/outbound/crd/crd_coordinator.go, internal/keeper/application/service/agent_service.go |
| Violates      | SPEC-FR-M3.5, SPEC-FR-M3.1                                         |
| Discovered    | Code inspection showing LLM settings (Endpoint and Credentials) are duplicated inside each Agent template's `BrainConfig` instead of referring to the centralized `LLMBinding` model. |

## Problem Statement

Currently, the `Agent` model's `BrainConfig` embeds the entire connection configuration for the Large Language Model (including `endpoint` and `credentials_secret`). This design duplicates connection settings across multiple agents and violates the modular architecture described in `SPEC-FR-M3.1`, which states that the system should manage a catalog of reusable `LLMBinding` models rather than hardcoding LLM configuration parameters in each agent.

Furthermore, this coupling means any changes to an LLM provider's endpoint or credentials require updating all individual agent templates rather than simply updating the referenced `LLMBinding`.

## Affected Components and Files

| Component / File | Location | Issue |
|------------------|----------|-------|
| Agent Domain Model | [agent.go](file:///Users/R.Pasquini/Projects/side/tacito-square/internal/keeper/domain/model/agent.go) | `BrainConfig` contains `Endpoint` and `CredentialsSecret` directly instead of referencing `LLMBinding` ID. |
| Agent API Handlers | [agent_handlers.go](file:///Users/R.Pasquini/Projects/side/tacito-square/internal/keeper/adapters/inbound/http/agent_handlers.go) | Requests for agent creation/updates expect embedded brain connection details instead of an `llm_binding_id`. |
| CRD Coordinator | [crd_coordinator.go](file:///Users/R.Pasquini/Projects/side/tacito-square/internal/keeper/adapters/outbound/crd/crd_coordinator.go) | Populates the Custom Resource spec from the embedded configuration instead of resolving it from the referenced `LLMBinding`. |
| Agent Service | [agent_service.go](file:///Users/R.Pasquini/Projects/side/tacito-square/internal/keeper/application/service/agent_service.go) | Lacks check/validation to ensure the referenced `LLMBinding` exists in the database. |
| DB Migrations | [00001_init.sql](file:///Users/R.Pasquini/Projects/side/tacito-square/deploy/postgres/migrations/00001_init.sql) | The `check_agent_brain` check constraint enforces that `endpoint` and `credentials_secret` keys are in the `brain` JSONB. |

## Impact

1. **Poor Modularity and Redundancy**: LLM endpoints and credential configurations are duplicated, making maintenance difficult.
2. **Security Risk**: Clear-text API secrets or direct secret references must be re-provided on every agent configuration.
3. **Inconsistency**: Reusable `LLMBinding` catalog is bypassed during agent definition.

## Expected Behaviour

1. **Centralized Reference**: The `Agent` model's `BrainConfig` MUST reference the LLM Binding via `llm_binding_id` (UUID).
2. **Parameters Override**: The agent can still optionally override the `model`, `temperature`, and `max_tokens` if specified.
3. **Reference Verification**: During Agent creation or update, the system MUST verify that the referenced `LLMBinding` exists in the database.
4. **CRD Resolution**: During CRD provisioning, the coordinator MUST fetch the referenced `LLMBinding` and use its API configuration (URL, Secret Ref) along with the agent's optional overrides to construct the final `v1alpha1.LLMConfig` spec of the `TacitoAgent`.

## Acceptance Criteria

1. **Domain Isolation**: `BrainConfig` model definition is updated to use `llm_binding_id` instead of `endpoint` and `credentials_secret`.
2. **Referential Integrity**: Creating or updating an agent template with a non-existent `llm_binding_id` is rejected.
3. **Database Constraints**: Database constraint `check_agent_brain` checks for the presence of `llm_binding_id` rather than `endpoint`/`credentials_secret`.
4. **CRD Synthesis**: The provisioned agent pod runs successfully with the resolved connection configuration fetched from the referenced `LLMBinding`.
5. **Passing Tests**: All unit and integration tests are updated to use the new model design and pass successfully.
