# BUG-M5.3: Keeper-Agent Interface Lacks Support for Dynamic Prompt Sets and Procedural Skill Contents

| Field         | Value                                                              |
|---------------|--------------------------------------------------------------------|
| ID            | BUG-M5.3                                                           |
| Status        | OPEN                                                               |
| Severity      | HIGH                                                               |
| Milestone     | M5 — Agent Core                                                    |
| Affects       | `internal/keeper/adapters/outbound/crd/crd_coordinator.go`         |
| Violates      | SPEC-FR-M3.3, SPEC-FR-M3.4, SPEC-FR-M5.1                           |
| Discovered    | Code investigation of prompt and skill propagation pathways         |

## Problem Statement

The current keeper-agent interface, managed by `K8sCRDCoordinator.ResolveAndSynthesizeSystemPrompt`, flattens all prompt templates and assigned skill listings out-of-band at deploy-time. The resulting single concatenated string is injected into `Spec.SystemPrompt` in the `TacitoAgent` CRD.

This creates severe architectural gaps:
1. **No support for prompt sets**: The agent receives a single static prompt string. It cannot dynamically select, retrieve, or swap prompts from a "prompt set" based on the ongoing conversation context.
2. **Missing procedural skill content**: The coordinator only appends a list of skill *names* and *descriptions* (e.g. `- math: Description`) to the system prompt. The actual procedural guidelines/content (`Content` field of the skill) are completely omitted. Since the agent pod runs in stateless isolation without direct access to keeper's database, the agent has no way to dynamically fetch or load the procedural contents of those skills when requested by the brain.

## Affected Components and Files

| Component / File | Location | Issue |
|------------------|----------|-------|
| K8s CRD Coordinator | `internal/keeper/adapters/outbound/crd/crd_coordinator.go` | Compiles a single flat system prompt string at deploy-time instead of exposing rich prompt/skill references or sets. |
| TacitoAgent CRD | `pkg/kubernetes/apis/tacito/v1alpha1/tacitoagent_types.go` | `TacitoAgentSpec` only has a single `SystemPrompt` field and lacks representation for prompt sets or rich skill contents. |
| Agent Main/Processor | `internal/agent/application/service/cognitive_engine.go` | Cognitive engine only receives a static system prompt and lacks a backend communication channel to pull procedural skill contents dynamically. |

## Impact

1. **Failure to execute dynamic skills**: Even if the brain correctly identifies a needed skill from the descriptions and attempts to load it, the agent has no access to the skill's procedural instructions, rendering dynamic loading non-functional in production.
2. **Loss of prompt sets flexibility**: The agent cannot dynamically adapt its system prompt or utilize prompt sets managed by the keeper, violating the core requirement of "prompt and prompt sets for agent behavioral knowledge".

## Expected Behaviour

1. The keeper-agent interface MUST support conveying distinct prompts/prompt sets and authorized skills/skill-set definitions.
2. The agent pod MUST have a secure, stateless, and authenticated channel (e.g., Zitadel/OIDC authenticated REST API or NATS client request-reply topic) to dynamically pull prompt templates and procedural skill contents from keeper on-demand.
3. When the brain calls `enable_skill` or requests a prompt template update, the agent should dynamically query the keeper to retrieve the authorized procedural instructions, keeping the local agent stateless and context-lean.

## Acceptance Criteria

1. **Rich Interface Support**:
   - `TacitoAgentSpec` or secure runtime APIs allow the agent to look up prompts by ID/name in a prompt set.
   - The agent is equipped to pull the raw `Content` of any authorized skill from keeper dynamically during reasoning steps.
2. **Stateless Operations**:
   - Agent pods remain stateless, retrieving necessary knowledge extensions on-demand rather than relying on heavy pre-loaded configurations.
