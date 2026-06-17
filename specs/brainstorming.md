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
| 6 | **Standardize query parameters for GET APIs**<br>Ensure GET APIs (excluding specific lists like agent cards) accept standardized query parameters for model filtering, sorting, paginating, and timezone/timestamp formats. | 🌐 Platform & UX | **New Functional Spec**<br>(Create `SPEC-FR-M8.13`) | **M8: Governance** | 2 | 4 | **16** |
| 7 | **Implement physical NATS bus separation or community gateways**<br>Assess and implement physical segregation of community NATS buses or community-specific gateways to isolate raw internal hub-spoke event traffic from the main Keeper bus, ensuring secure multi-tenant network boundaries. | 🌐 Platform & UX | **New Functional Spec**<br>(Create `SPEC-FR-M6.10`) | **M6: Communities** | 2 | 4 | **16** |
| 8 | **Encrypt agent brain credential secrets at rest**<br>Securely store credentials, API keys, and model secrets using state-of-the-art encryption algorithms at rest. | 🌐 Platform & UX | **New Functional Spec**<br>(Create `SPEC-FR-M8.11`) | **M8: Governance** | 3 | 4 | **12** |
| 9 | **Implement a Summary Agent/tool for context compaction**<br>Develop a specialized summary tool/agent to reduce long conversation history when it crosses specific thresholds, preserving semantic context. | 🧠 Cognitive & Agents | **New Functional Spec**<br>(Create `SPEC-FR-M5.13`) | **M6: Communities** | 3 | 4 | **12** |
| 10 | **Design architectural integration patterns for RAG and web search**<br>Define clean ports and adapters interfaces for Retrieval-Augmented Generation (RAG) and web search within the agent's flow. | 🧠 Cognitive & Agents | **New Functional Spec**<br>(Create `SPEC-FR-M5.14`) | **M6: Communities** | 3 | 4 | **12** |
| 11 | **BFF primary API surface to use GraphQL**<br>Migrate the BFF's primary API surface to GraphQL to support flexible UI frontend client queries, while keeping REST APIs to support CLI clients. | 🌐 Platform & UX | **New Functional Spec**<br>(Create `SPEC-FR-M7.5`) | **M7: BFF & UIs** | 2 | 3 | **12** |
| 12 | **Evaluate moving community end user operations to a Gateway component**<br>Assess moving end-user community operations out of the Keeper into a dedicated new API gateway component. | 🌐 Platform & UX | **New Functional Spec**<br>(Create `SPEC-FR-M10.4`) | **M10: Federation** | 2 | 3 | **12** |
| 13 | **Integrate Unleash feature flag management**<br>Add Unleash SDK integration to support runtime feature toggling and dynamic configurations across components. | 🌐 Platform & UX | **New Functional Spec**<br>(Create `SPEC-FR-M8.12`) | **M8: Governance** | 3 | 3 | **9** |
| 14 | **Implement OpenTelemetry distributed tracing across Kubernetes CRD reconciliation**<br>Propagate trace contexts across the Kubernetes Operator's reconciliation loops (e.g., tracking agent assignment and activities). | ⚙️ Infra & Quality | **Past Spec Bug**<br>(Missing in `SPEC-FR-M4.3`) | **M9: Hardening** | 3 | 3 | **9** |
| 15 | **Implement Redis-based leader election among agent replicas**<br>Ensure only the active leader agent replica runs heartbeat loops by implementing a Redis-based leader election mechanism. | ⚙️ Infra & Quality | **New Functional Spec**<br>(Create `SPEC-FR-M4.10`) | **M9: Hardening** | 3 | 3 | **9** |
| 16 | **Diagnose and optimize slow initial connections on SSE streams**<br>Investigate and resolve high initial latency when initiating Server-Sent Events (SSE) streams in the BFF API layer. | ⚙️ Infra & Quality | **Past Spec Bug**<br>(Gap in `SPEC-FR-M7.1`) | **M7: BFF & UIs** | 3 | 3 | **9** |
| 17 | **Implement a new brain adapter for Google Gemini**<br>Create a brain adapter utilizing the official Google `go-genai` SDK to enable Gemini-based agents. | 🧠 Cognitive & Agents | **New Functional Spec**<br>(Create `SPEC-FR-M5.15`) | **M6: Communities** | 4 ⭐ | 4 | **8** |
| 18 | **Create a Skillset abstraction to group multiple skills**<br>Introduce a Skillset domain model to aggregate, package, and assign multiple related skills to agents. | 🧠 Cognitive & Agents | **New Functional Spec**<br>(Create `SPEC-FR-M3.9`) | **M8: Governance** | 4 ⭐ | 4 | **8** |
| 19 | **Provide configuration flags to opt-out of specific built-in tools**<br>Allow agents to optionally disable specific default tools to enforce strict sandbox capabilities. | 🧠 Cognitive & Agents | **New Functional Spec**<br>(Create `SPEC-FR-M5.16`) | **M8: Governance** | 4 ⭐ | 3 | **6** |
| 20 | **Introduce template engines for brain prompts**<br>Implement dynamic templating for system and user prompts before sending them to LLM brains. | 🧠 Cognitive & Agents | **New Functional Spec**<br>(Create `SPEC-FR-M3.10`) | **M8: Governance** | 4 ⭐ | 3 | **6** |
| 21 | **Refactor large source files and packages with excessive LOC**<br>Clean up technical debt by splitting oversized code files and packaging boundaries into cleaner, smaller structures. | ⚙️ Infra & Quality | **Past Spec Bug**<br>(Violates architecture rules) | **M9: Hardening** | 3 | 2 | **6** |
| 22 | **Include built-in and MCP tool definitions in Agent Cards**<br>Expose active tool catalogs and MCP bindings in A2A Agent Cards to allow agents to dynamically discover available skills. | 🧠 Cognitive & Agents | **Past Spec Bug**<br>(Extension of `SPEC-FR-M6.5`) | **M6: Communities** | 4 ⭐ | 3 | **6** |
| 23 | **Deploy a test MCP server for standalone agent verification**<br>Deploy a public/mock MCP server to validate tool capabilities and standalone agent integration loops. | ⚙️ Infra & Quality | **New Functional Spec**<br>(Create `SPEC-FR-M9.8`) | **M9: Hardening** | 4 ⭐ | 3 | **6** |
| 24 | **Standardize Redis connection pool and driver logs to structured JSON**<br>Ensure Redis driver connection pool logs adhere strictly to structured JSON standard (`SPEC-NFR-LOG`). | ⚙️ Infra & Quality | **Past Spec Bug**<br>(Violates observability rules) | **M9: Hardening** | 4 ⭐ | 2 | **4** |
| 25 | **Refactor S3 storage adapters to use the official AWS SDK**<br>Replace manual HTTP calls to S3 with the official AWS/S3 SDK, including OpenTelemetry instrumentation. | ⚙️ Infra & Quality | **Past Spec Bug**<br>(Violates standard in `SPEC-FR-M5.6`) | **M9: Hardening** | 4 ⭐ | 2 | **4** |
| 26 | **Conduct a comprehensive review of application log consistency**<br>Audit log levels and tracing correlation to align logs with the system's observability rules. | ⚙️ Infra & Quality | **Past Spec Bug**<br>(Violates observability rules) | **M9: Hardening** | 4 ⭐ | 2 | **4** |
| 27 | **Provide REST API endpoints to clear STM per agent and per community**<br>Add administrative API endpoints to manually purge short-term memory states for debugging or user-facing resets. | 🧠 Cognitive & Agents | **New Functional Spec**<br>(Create `SPEC-FR-M4.9`) | **M6: Communities** | 5 ⭐ | 3 | **3** |
| 28 | **Include NATS connection status in readiness probes**<br>Check NATS connection health inside `/readyz` probes for the Agent and Keeper components to prevent routing black holes. | ⚙️ Infra & Quality | **Past Spec Bug**<br>(Gap in `/readyz` specs) | **M9: Hardening** | 5 ⭐ | 3 | **3** |
| 29 | **Standardize component lifecycle versioning with VERSION files**<br>Transition away from hardcoded versions to external `VERSION.<component>` files matching `SPEC-NFR-VERSIONING`. | ⚙️ Infra & Quality | **Past Spec Bug**<br>(Violates standard packaging rules) | **M9: Hardening** | 5 ⭐ | 3 | **3** |
| 30 | **Fix Agent Cards caching expires HTTP header**<br>Correct HTTP header response caching configurations where `Expires` incorrectly equals `Last-Modified`. | ⚙️ Infra & Quality | **Past Spec Bug**<br>(Violates standard in `SPEC-FR-M6.5`) | **M6: Communities** | 5 ⭐ | 2 | **2** |
| 31 | **Remove the tacitocommunities CRD from Operator**<br>Clean up the deprecated and unused `tacitocommunities.tacito.square.io` CustomResourceDefinition from the operator controller. | ⚙️ Infra & Quality | **Past Spec Bug**<br>(Cleanup after rejection of `SPEC-FR-M4.2`) | **M4: Operator Core** | 5 ⭐ | 2 | **2** |
| 32 | **Align codebase terminology: MCP server to MCP client**<br>Replace incorrect usages of "MCP server" with "MCP client" across documentation and code comments. | ⚙️ Infra & Quality | **Past Spec Bug**<br>(Terminology bug in implementation) | **M9: Hardening** | 5 ⭐ | 1 | **1** |
| 33 | **Redesign administrative UI workflows into guided, wizard-like interfaces**<br>Provide intuitive wizard-driven UI setup for agents, skills, and prompts for non-technical users, alongside an advanced mode toggle for power users. | 🌐 Platform & UX | **Already Addressed**<br>(Incorp. in draft `SPEC-FR-M7.2`) | **M7: BFF & UIs** | 2 | 5 | **0** |
| 34 | **Implement a non-blocking, parallel E2E automated test suite**<br>Develop comprehensive, parallelized end-to-end tests validating the UI, API, and Kubernetes operator within CI/CD and local environments. | ⚙️ Infra & Quality | **Already Addressed**<br>(Incorp. in draft `SPEC-FR-M9.3`) | **M9: Hardening** | 2 | 4 | **0** |
| 35 | **Research and design patterns to leverage NATS JetStream**<br>Explore and document NATS JetStream usage patterns for persistent event-driven workflows, message replay, and at-least-once message delivery. | 🌐 Platform & UX | **Already Addressed**<br>(Incorp. in draft `SPEC-FR-M6.2`) | **M6: Communities** | 3 | 4 | **0** |
| 36 | **Enable dynamic loading of remote agent skillsets**<br>Allow loading and executing agent skillsets dynamically from external repositories like GitHub. | 🧠 Cognitive & Agents | **Already Addressed**<br>(Incorp. in draft `SPEC-FR-M8.8`) | **M8: Governance** | 3 | 4 | **0** |
| 37 | **Define roles and functional resource ownership in multi-tenant environments**<br>Establish user roles, resource boundaries, and tenancy rules governed by platform quality-of-service (QoS) quotas. | 🌐 Platform & UX | **Already Addressed**<br>(Incorp. in draft `SPEC-FR-M8.2`) | **M8: Governance** | 3 | 4 | **0** |
| 38 | **Implement Time-to-Live (TTL) and automatic closure for inactive threads**<br>Automatically close idle thread sessions after a configured TTL period to reclaim system memory and storage resources. | 🧠 Cognitive & Agents | **Already Addressed**<br>(Incorp. in draft `SPEC-FR-M6.4`) | **M6: Communities** | 4 ⭐ | 3 | **0** |
| 39 | **Roll back STM updates on brain response failures or fallbacks**<br>Ensure any Short-Term Memory (STM) mutations are reverted if the LLM adapter request fails or falls back to prevent corrupted state. | 🧠 Cognitive & Agents | **Already Addressed**<br>(Incorp. in `SPEC-FR-M6.0` Sec 5.5) | **M6: Communities** | 4 ⭐ | 3 | **0** |
| 40 | **Document standardized test scenarios for ingestion, chat, and search**<br>Define and write comprehensive test scripts and documentation mapping out validation flows for chat, ingestion, and search. | ⚙️ Infra & Quality | **Already Addressed**<br>(Incorp. in draft `SPEC-FR-M9.6`) | **M9: Hardening** | 4 ⭐ | 3 | **0** |
| 41 | **Optimize development workflows using multi-stage Docker builds**<br>Introduce additional multi-stage Docker layers and Makefile steps to speed up iterations and decouple development workspaces. | ⚙️ Infra & Quality | **Already Addressed**<br>(Refines `SPEC-FR-M2.7`) | **M9: Hardening** | 4 ⭐ | 3 | **0** |
| 42 | **Provide a standardized API test suite for the Keeper component**<br>Build a Bruno or Postman collection to verify and regression-test Keeper's HTTP endpoints. | ⚙️ Infra & Quality | **Already Addressed**<br>(Incorp. in draft `SPEC-FR-M9.3`) | **M9: Hardening** | 4 ⭐ | 3 | **0** |
| 43 | **Establish BFF and Frontend development guidelines**<br>Define guidelines for stateless architecture, avoiding HTTP sessions, and storing auth tokens securely in encrypted cookies. | ⚙️ Infra & Quality | **Already Addressed**<br>(Incorp. in draft `SPEC-FR-M7.1`) | **M7: BFF & UIs** | 4 ⭐ | 3 | **0** |
| 44 | **Optimize component startup times and container initialization**<br>Profile and streamline system initialization processes to allow faster scaling and cold starts. | ⚙️ Infra & Quality | **Already Addressed**<br>(Refines `SPEC-FR-M2.7`) | **M9: Hardening** | 3 | 2 | **0** |
| 45 | **Save conversation history for recovery and Long-Term Memory**<br>Persist structured conversation history to support crash recovery and feed the Qdrant Long-Term Memory (LTM) pipeline. | 🧠 Cognitive & Agents | **Already Addressed**<br>(Incorp. in draft `SPEC-FR-M6.4`) | **M6: Communities** | 3 | 4 | **0** |
| 46 | **Enforce community resource attributes and usage quotas**<br>Impose and validate community quotas including rate limits, maximum agent counts, and knowledge base memory bounds. | 🌐 Platform & UX | **Already Addressed**<br>(Incorp. in draft `SPEC-FR-M8.3`/`SPEC-FR-M8.4`) | **M8: Governance** | 3 | 4 | **0** |

## Free

- community topology, sequential
- decouple the polishing assistant (built-in assistant) from the hub. It should be a skill that agents can use, not a separate agent
- kubernetes labels and annotations
- 