# Brainstorming Directions and Evaluation

This document outlines future development directions, feature ideas, and architectural refactoring targets for Tacito Square.

## Evaluation Methodology

Each direction is evaluated on two scales (from 1 to 5):
- **Easy Win (Ease)**: `1` (very hard / high effort) to `5` (very easy / low effort). Easy Wins (scores of `4` or `5`) are highlighted with a star (⭐).
- **Business Value (Value)**: `1` (low business value / internal utility only) to `5` (critical / high user/system value).

**Magnitude** is calculated as:
$$\text{Magnitude} = \text{Difficulty (Easy Win Inverted)} \times \text{Business Value} = (6 - \text{Easy Win}) \times \text{Business Value}$$
*(Note: Magnitude is forced to `0` for directions that are **Already Addressed** by existing/draft specs. This prioritizes net-new gaps and bugs at the top.)*

### Categorization (Max 3 Categories)
Each direction is labeled with one of the following categories:
* 🧠 **Cognitive & Agents**: AI models, memory structures, skillset definitions, and agent behaviors.
* 🌐 **Platform & UX**: Frontend/UI workflows, gateways, authorization, multitenancy, feature flags, encryption, and ingress.
* ⚙️ **Infra & Quality**: Observability, testing, containerization, build workflows, logging, refactoring, and lifecycle management.

### Functional Spec Mapping Types
- **New Functional Spec**: Gaps in current scope that require a new atomic specification.
- **Past Spec Bug**: Gaps or rule violations in already implemented features that require opening a bug/fix task.
- **Already Addressed**: Capabilities that are covered by existing or draft specs and should be implemented as part of their lifecycle (Magnitude forced to `0`).

---

## Prioritized Directions (Sorted by Magnitude)

| # | Direction / Intent | Category | Spec Mapping & Status | Target Milestone | Easy Win (1-5) | Business Value (1-5) | Magnitude |
|---|--------------------|----------|-----------------------|------------------|:---:|:---:|:---:|
| 1 | **Deploy local open-source LLMs/embeddings on GPU using vLLM**<br>Deploy small open-source models/embeddings on GPU with vLLM for specialized, private offline tasks and Long-Term Memory (LTM) support. | 🧠 Cognitive & Agents | **New Functional Spec**<br>(Create `SPEC-FR-M5.11`) | **M11: Local AI Serving**<br>(New Milestone) | 1 | 4 | **20** |
| 2 | **Establish an end-to-end community ingress pipeline**<br>Coordinate the event/trigger path (Ingress &rarr; Keeper &rarr; CRD &rarr; Community Agent) to automate triggers for conversations, ingestion, and background tasks. | 🌐 Platform & UX | **New Functional Spec**<br>(Create `SPEC-FR-M6.8`) | **M6: Communities** | 2 | 5 | **20** |
| 3 | **Build a Gateway Agent for streaming interactions**<br>Create a central hub to handle real-time streaming of user requests and outbound agent reasoning, improving perceived latency and UX responsiveness. | 🌐 Platform & UX | **New Functional Spec**<br>(Create `SPEC-FR-M6.9`) | **M7: BFF & UIs** | 2 | 5 | **20** |
| 4 | **Implement advanced cognitive capabilities for agents**<br>Add agent support for context compaction, language identification, dynamic planning, task delegation, and parallel task execution. | 🧠 Cognitive & Agents | **New Functional Spec**<br>(Create `SPEC-FR-M5.12`) | **M6: Communities** | 2 | 5 | **20** |
| 5 | **Short-Term Memory (STM) Lifecycle and Retention Strategy**<br>Define mechanisms to identify contextually relevant conversation data for STM preservation, and establish clean trigger-based cleanup intervals. | 🧠 Cognitive & Agents | **New Functional Spec**<br>(Extension of `SPEC-FR-M5.3`) | **M6: Communities** | 2 | 4 | **16** |
| 6 | **Encrypt agent brain credential secrets at rest**<br>Securely store credentials, API keys, and model secrets using state-of-the-art encryption algorithms at rest. | 🌐 Platform & UX | **New Functional Spec**<br>(Create `SPEC-FR-M8.11`) | **M8: Governance** | 3 | 4 | **12** |
| 7 | **Implement a Summary Agent/tool for context compaction**<br>Develop a specialized summary tool/agent to reduce long conversation history when it crosses specific thresholds, preserving semantic context. | 🧠 Cognitive & Agents | **New Functional Spec**<br>(Create `SPEC-FR-M5.13`) | **M6: Communities** | 3 | 4 | **12** |
| 8 | **Design architectural integration patterns for RAG and web search**<br>Define clean ports and adapters interfaces for Retrieval-Augmented Generation (RAG) and web search within the agent's flow. | 🧠 Cognitive & Agents | **New Functional Spec**<br>(Create `SPEC-FR-M5.14`) | **M6: Communities** | 3 | 4 | **12** |
| 9 | **Integrate Unleash feature flag management**<br>Add Unleash SDK integration to support runtime feature toggling and dynamic configurations across components. | 🌐 Platform & UX | **New Functional Spec**<br>(Create `SPEC-FR-M8.12`) | **M8: Governance** | 3 | 3 | **9** |
| 10 | **Implement OpenTelemetry distributed tracing across Kubernetes CRD reconciliation**<br>Propagate trace contexts across the Kubernetes Operator's reconciliation loops (e.g., tracking agent assignment and activities). | ⚙️ Infra & Quality | **Past Spec Bug**<br>(Missing in `SPEC-FR-M4.3`) | **M9: Hardening** | 3 | 3 | **9** |
| 11 | **Implement a new brain adapter for Google Gemini**<br>Create a brain adapter utilizing the official Google `go-genai` SDK to enable Gemini-based agents. | 🧠 Cognitive & Agents | **New Functional Spec**<br>(Create `SPEC-FR-M5.15`) | **M6: Communities** | 4 ⭐ | 4 | **8** |
| 12 | **Create a Skillset abstraction to group multiple skills**<br>Introduce a Skillset domain model to aggregate, package, and assign multiple related skills to agents. | 🧠 Cognitive & Agents | **New Functional Spec**<br>(Create `SPEC-FR-M3.9`) | **M8: Governance** | 4 ⭐ | 4 | **8** |
| 13 | **Provide configuration flags to opt-out of specific built-in tools**<br>Allow agents to optionally disable specific default tools to enforce strict sandbox capabilities. | 🧠 Cognitive & Agents | **New Functional Spec**<br>(Create `SPEC-FR-M5.16`) | **M8: Governance** | 4 ⭐ | 3 | **6** |
| 14 | **Introduce template engines for brain prompts**<br>Implement dynamic templating for system and user prompts before sending them to LLM brains. | 🧠 Cognitive & Agents | **New Functional Spec**<br>(Create `SPEC-FR-M3.10`) | **M8: Governance** | 4 ⭐ | 3 | **6** |
| 15 | **Refactor large source files and packages with excessive LOC**<br>Clean up technical debt by splitting oversized code files and packaging boundaries into cleaner, smaller structures. | ⚙️ Infra & Quality | **Past Spec Bug**<br>(Violates architecture rules) | **M9: Hardening** | 3 | 2 | **6** |
| 16 | **Standardize Redis connection pool and driver logs to structured JSON**<br>Ensure Redis driver connection pool logs adhere strictly to structured JSON standard (`SPEC-NFR-LOG`). | ⚙️ Infra & Quality | **Past Spec Bug**<br>(Violates observability rules) | **M9: Hardening** | 4 ⭐ | 2 | **4** |
| 17 | **Refactor S3 storage adapters to use the official AWS SDK**<br>Replace manual HTTP calls to S3 with the official AWS/S3 SDK, including OpenTelemetry instrumentation. | ⚙️ Infra & Quality | **Past Spec Bug**<br>(Violates standard in `SPEC-FR-M5.6`) | **M9: Hardening** | 4 ⭐ | 2 | **4** |
| 18 | **Conduct a comprehensive review of application log consistency**<br>Audit log levels and tracing correlation to align logs with the system's observability rules. | ⚙️ Infra & Quality | **Past Spec Bug**<br>(Violates observability rules) | **M9: Hardening** | 4 ⭐ | 2 | **4** |
| 19 | **Provide REST API endpoints to clear STM per agent and per community**<br>Add administrative API endpoints to manually purge short-term memory states for debugging or user-facing resets. | 🧠 Cognitive & Agents | **New Functional Spec**<br>(Create `SPEC-FR-M4.9`) | **M6: Communities** | 5 ⭐ | 3 | **3** |
| 20 | **Include NATS connection status in readiness probes**<br>Check NATS connection health inside `/readyz` probes for the Agent and Keeper components to prevent routing black holes. | ⚙️ Infra & Quality | **Past Spec Bug**<br>(Gap in `/readyz` specs) | **M9: Hardening** | 5 ⭐ | 3 | **3** |
| 21 | **Standardize component lifecycle versioning with VERSION files**<br>Transition away from hardcoded versions to external `VERSION.<component>` files matching `SPEC-NFR-VERSIONING`. | ⚙️ Infra & Quality | **Past Spec Bug**<br>(Violates standard packaging rules) | **M9: Hardening** | 5 ⭐ | 3 | **3** |
| 22 | **Align codebase terminology: MCP server to MCP client**<br>Replace incorrect usages of "MCP server" with "MCP client" across documentation and code comments. | ⚙️ Infra & Quality | **Past Spec Bug**<br>(Terminology bug in implementation) | **M9: Hardening** | 5 ⭐ | 1 | **1** |
| 23 | **Redesign administrative UI workflows into guided, wizard-like interfaces**<br>Provide intuitive wizard-driven UI setup for agents, skills, and prompts for non-technical users, alongside an advanced mode toggle for power users. | 🌐 Platform & UX | **Already Addressed**<br>(Incorp. in draft `SPEC-FR-M7.2`) | **M7: BFF & UIs** | 2 | 5 | **0** |
| 24 | **Implement a non-blocking, parallel E2E automated test suite**<br>Develop comprehensive, parallelized end-to-end tests validating the UI, API, and Kubernetes operator within CI/CD and local environments. | ⚙️ Infra & Quality | **Already Addressed**<br>(Incorp. in draft `SPEC-FR-M9.3`) | **M9: Hardening** | 2 | 4 | **0** |
| 25 | **Research and design patterns to leverage NATS JetStream**<br>Explore and document NATS JetStream usage patterns for persistent event-driven workflows, message replay, and at-least-once message delivery. | 🌐 Platform & UX | **Already Addressed**<br>(Incorp. in draft `SPEC-FR-M6.2`) | **M6: Communities** | 3 | 4 | **0** |
| 26 | **Enable dynamic loading of remote agent skillsets**<br>Allow loading and executing agent skillsets dynamically from external repositories like GitHub. | 🧠 Cognitive & Agents | **Already Addressed**<br>(Incorp. in draft `SPEC-FR-M8.8`) | **M8: Governance** | 3 | 4 | **0** |
| 27 | **Define roles and functional resource ownership in multi-tenant environments**<br>Establish user roles, resource boundaries, and tenancy rules governed by platform quality-of-service (QoS) quotas. | 🌐 Platform & UX | **Already Addressed**<br>(Incorp. in draft `SPEC-FR-M8.2`) | **M8: Governance** | 3 | 4 | **0** |
| 28 | **Implement Time-to-Live (TTL) and automatic closure for inactive threads**<br>Automatically close idle thread sessions after a configured TTL period to reclaim system memory and storage resources. | 🧠 Cognitive & Agents | **Already Addressed**<br>(Incorp. in draft `SPEC-FR-M6.4`) | **M6: Communities** | 4 ⭐ | 3 | **0** |
| 29 | **Roll back STM updates on brain response failures or fallbacks**<br>Ensure any Short-Term Memory (STM) mutations are reverted if the LLM adapter request fails or falls back to prevent corrupted state. | 🧠 Cognitive & Agents | **Already Addressed**<br>(Incorp. in `SPEC-FR-M6.0` Sec 5.5) | **M6: Communities** | 4 ⭐ | 3 | **0** |
| 30 | **Document standardized test scenarios for ingestion, chat, and search**<br>Define and write comprehensive test scripts and documentation mapping out validation flows for chat, ingestion, and search. | ⚙️ Infra & Quality | **Already Addressed**<br>(Incorp. in draft `SPEC-FR-M9.6`) | **M9: Hardening** | 4 ⭐ | 3 | **0** |
| 31 | **Optimize development workflows using multi-stage Docker builds**<br>Introduce additional multi-stage Docker layers and Makefile steps to speed up iterations and decouple development workspaces. | ⚙️ Infra & Quality | **Already Addressed**<br>(Refines `SPEC-FR-M2.7`) | **M9: Hardening** | 4 ⭐ | 3 | **0** |
| 32 | **Provide a standardized API test suite for the Keeper component**<br>Build a Bruno or Postman collection to verify and regression-test Keeper's HTTP endpoints. | ⚙️ Infra & Quality | **Already Addressed**<br>(Incorp. in draft `SPEC-FR-M9.3`) | **M9: Hardening** | 4 ⭐ | 3 | **0** |
| 33 | **Establish BFF and Frontend development guidelines**<br>Define guidelines for stateless architecture, avoiding HTTP sessions, and storing auth tokens securely in encrypted cookies. | ⚙️ Infra & Quality | **Already Addressed**<br>(Incorp. in draft `SPEC-FR-M7.1`) | **M7: BFF & UIs** | 4 ⭐ | 3 | **0** |
| 34 | **Optimize component startup times and container initialization**<br>Profile and streamline system initialization processes to allow faster scaling and cold starts. | ⚙️ Infra & Quality | **Already Addressed**<br>(Refines `SPEC-FR-M2.7`) | **M9: Hardening** | 3 | 2 | **0** |

## Fee entries (before prioritization)

- all get apis, except well-known like agent cards, should accept query string arguments (managed as shared module, as they will be used across the board) to: limit the reposonse model, by excluding/including fields, sort by field values ascending/descending, simple filtering by field values and values patterns, page and offset arguments for pagination, timestamp formats (RFC3339 with/without fractional seconds, epoch), timezone offset for timestamp conversions,
- GraphQL APIs can be the primary API surface for the BFF, with REST APIs still operating to serve CLI clients.
- why empty events {"event_id":"","schema_ref":"","source":"","tenant_id":"","occurred_at":"","payload":null} in SSE?
- save conversation history for recovery and for long term memory
- built-in tools and mcp tools in agent cards
- bug? agent cards con http header expires == last modified
- evaluate to move community end user operations from the keeper to a new gateway component
- community attributes and usage enforcing, e.g. rate limits, quotas, max resources (e.g. number of agents, knowledge base size, etc.)
