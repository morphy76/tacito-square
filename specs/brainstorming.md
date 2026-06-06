# Brainstorming Directions and Evaluation

This document outlines future development directions, feature ideas, and architectural refactoring targets for Tacito Square.

## Evaluation Methodology

Each direction is evaluated on two scales (from 1 to 5):
- **Easy Win (Ease)**: `1` (very hard / high effort) to `5` (very easy / low effort). Easy Wins (scores of `4` or `5`) are highlighted with a star (⭐).
- **Business Value (Value)**: `1` (low business value / internal utility only) to `5` (critical / high user/system value).

**Magnitude** is calculated as:
$$\text{Magnitude} = \text{Difficulty (Easy Win Inverted)} \times \text{Business Value} = (6 - \text{Easy Win}) \times \text{Business Value}$$
Where a higher Magnitude score indicates a larger/more complex project with significant business value.

---

## Prioritized Directions (Sorted by Magnitude)

| # | Direction / Intent | Easy Win (1-5) | Business Value (1-5) | Magnitude |
|---|--------------------|:---:|:---:|:---:|
| 1 | **Deploy local open-source LLMs/embeddings on GPU using vLLM**<br>Deploy small open-source models/embeddings on GPU with vLLM for specialized, private offline tasks and Long-Term Memory (LTM) support. | 1 | 4 | **20** |
| 2 | **Redesign administrative UI workflows into guided, wizard-like interfaces**<br>Provide intuitive wizard-driven UI setup for agents, skills, and prompts for non-technical users, alongside an advanced mode toggle for power users. | 2 | 5 | **20** |
| 3 | **Establish an end-to-end community ingress pipeline**<br>Coordinate the event/trigger path (Ingress &rarr; Keeper &rarr; CRD &rarr; Community Agent) to automate triggers for conversations, ingestion, and background tasks. | 2 | 5 | **20** |
| 4 | **Build a Gateway Agent for streaming interactions**<br>Create a central hub to handle real-time streaming of user requests and outbound agent reasoning, improving perceived latency and UX responsiveness. | 2 | 5 | **20** |
| 5 | **Implement advanced cognitive capabilities for agents**<br>Add agent support for context compaction, language identification, dynamic planning, task delegation, and parallel task execution. | 2 | 5 | **20** |
| 6 | **Implement a non-blocking, parallel E2E automated test suite**<br>Develop comprehensive, parallelized end-to-end tests validating the UI, API, and Kubernetes operator within CI/CD and local environments. | 2 | 4 | **16** |
| 7 | **Short-Term Memory (STM) Lifecycle and Retention Strategy**<br>Define mechanisms to identify contextually relevant conversation data for STM preservation, and establish clean trigger-based cleanup intervals. | 2 | 4 | **16** |
| 8 | **Encrypt agent brain credential secrets at rest**<br>Securely store credentials, API keys, and model secrets using state-of-the-art encryption algorithms at rest. | 3 | 4 | **12** |
| 9 | **Research and design patterns to leverage NATS JetStream**<br>Explore and document NATS JetStream usage patterns for persistent event-driven workflows, message replay, and at-least-once message delivery. | 3 | 4 | **12** |
| 10 | **Implement a Summary Agent/tool for context compaction**<br>Develop a specialized summary tool/agent to reduce long conversation history when it crosses specific thresholds, preserving semantic context. | 3 | 4 | **12** |
| 11 | **Enable dynamic loading of remote agent skillsets**<br>Allow loading and executing agent skillsets dynamically from external repositories like GitHub. | 3 | 4 | **12** |
| 12 | **Design architectural integration patterns for RAG and web search**<br>Define clean ports and adapters interfaces for Retrieval-Augmented Generation (RAG) and web search within the agent's flow. | 3 | 4 | **12** |
| 13 | **Define roles and functional resource ownership in multi-tenant environments**<br>Establish user roles, resource boundaries, and tenancy rules governed by platform quality-of-service (QoS) quotas. | 3 | 4 | **12** |
| 14 | **Integrate Unleash feature flag management**<br>Add Unleash SDK integration to support runtime feature toggling and dynamic configurations across components. | 3 | 3 | **9** |
| 15 | **Implement OpenTelemetry distributed tracing across Kubernetes CRD reconciliation**<br>Propagate trace contexts across the Kubernetes Operator's reconciliation loops (e.g., tracking agent assignment and activities). | 3 | 3 | **9** |
| 16 | **Implement a new brain adapter for Google Gemini**<br>Create a brain adapter utilizing the official Google `go-genai` SDK to enable Gemini-based agents. | 4 ⭐ | 4 | **8** |
| 17 | **Create a Skillset abstraction to group multiple skills**<br>Introduce a Skillset domain model to aggregate, package, and assign multiple related skills to agents. | 4 ⭐ | 4 | **8** |
| 18 | **Implement Time-to-Live (TTL) and automatic closure for inactive threads**<br>Automatically close idle thread sessions after a configured TTL period to reclaim system memory and storage resources. | 4 ⭐ | 3 | **6** |
| 19 | **Roll back STM updates on brain response failures or fallbacks**<br>Ensure any Short-Term Memory (STM) mutations are reverted if the LLM adapter request fails or falls back to prevent corrupted state. | 4 ⭐ | 3 | **6** |
| 20 | **Provide configuration flags to opt-out of specific built-in tools**<br>Allow agents to optionally disable specific default tools to enforce strict sandbox capabilities. | 4 ⭐ | 3 | **6** |
| 21 | **Introduce template engines for brain prompts**<br>Implement dynamic templating for system and user prompts before sending them to LLM brains. | 4 ⭐ | 3 | **6** |
| 22 | **Document standardized test scenarios for ingestion, chat, and search**<br>Define and write comprehensive test scripts and documentation mapping out validation flows for chat, ingestion, and search. | 4 ⭐ | 3 | **6** |
| 23 | **Optimize development workflows using multi-stage Docker builds**<br>Introduce additional multi-stage Docker layers and Makefile steps to speed up iterations and decouple development workspaces. | 4 ⭐ | 3 | **6** |
| 24 | **Provide a standardized API test suite for the Keeper component**<br>Build a Bruno or Postman collection to verify and regression-test Keeper's HTTP endpoints. | 4 ⭐ | 3 | **6** |
| 25 | **Establish BFF and Frontend development guidelines**<br>Define guidelines for stateless architecture, avoiding HTTP sessions, and storing auth tokens securely in encrypted cookies. | 4 ⭐ | 3 | **6** |
| 26 | **Refactor large source files and packages with excessive LOC**<br>Clean up technical debt by splitting oversized code files and packaging boundaries into cleaner, smaller structures. | 3 | 2 | **6** |
| 27 | **Optimize component startup times and container initialization**<br>Profile and streamline system initialization processes to allow faster scaling and cold starts. | 3 | 2 | **6** |
| 28 | **Standardize Redis connection pool and driver logs to structured JSON**<br>Ensure Redis driver connection pool logs adhere strictly to structured JSON standard (`SPEC-NFR-LOG`). | 4 ⭐ | 2 | **4** |
| 29 | **Refactor S3 storage adapters to use the official AWS SDK**<br>Replace manual HTTP calls to S3 with the official AWS/S3 SDK, including OpenTelemetry instrumentation. | 4 ⭐ | 2 | **4** |
| 30 | **Conduct a comprehensive review of application log consistency**<br>Audit log levels and tracing correlation to align logs with the system's observability rules. | 4 ⭐ | 2 | **4** |
| 31 | **Provide REST API endpoints to clear STM per agent and per community**<br>Add administrative API endpoints to manually purge short-term memory states for debugging or user-facing resets. | 5 ⭐ | 3 | **3** |
| 32 | **Include NATS connection status in readiness probes**<br>Check NATS connection health inside `/readyz` probes for the Agent and Keeper components to prevent routing black holes. | 5 ⭐ | 3 | **3** |
| 33 | **Standardize component lifecycle versioning with VERSION files**<br>Transition away from hardcoded versions to external `VERSION.<component>` files matching `SPEC-NFR-VERSIONING`. | 5 ⭐ | 3 | **3** |
| 34 | **Align codebase terminology: MCP server to MCP client**<br>Replace incorrect usages of "MCP server" with "MCP client" across documentation and code comments. | 5 ⭐ | 1 | **1** |
