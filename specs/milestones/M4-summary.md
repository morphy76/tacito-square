# Milestone M4: Operator Core Summary

This document provides a consolidated summary of the completed Milestone 4, serving as a high-level reference of architectural decisions, completed features, and resolved issues in the Operator Core.

---

- **Status**: ✔️ IMPLEMENTED
- **Goal**: The Operator watches `TacitoAgent` CRDs and instantiates or destroys agent Deployments and headless Services accordingly. Supports operator-managed zero-scaling of idle agents.
- **Deliverable**: `kubectl apply -f agent.yaml` creates agent Deployment + headless Service, pod starts and becomes healthy. `kubectl delete` cleans resources. Idle agents scale to zero.

### Completed Specifications

| Spec ID | Title | Component | Description |
|---------|-------|-----------|-------------|
| **SPEC-FR-M4.1** | Agent CRD Definition & Registration | operator | Defined CRD properties, schema, and validation rules. |
| **SPEC-FR-M4.2** | AgentCommunity CRD Definition | operator | *REJECTED — communities are Keeper DB entities, not K8s CRDs.* |
| **SPEC-FR-M4.3** | Reconciliation Controller | operator | Operator controller coordinating CRD status with actual replica status. |
| **SPEC-FR-M4.6** | Agent CRD Submission | keeper | Client library to POST new agent specs into the K8s API group. |
| **SPEC-FR-M4.7** | Agent & Community Lifecycle API | keeper, operator | Keeper endpoints `/deploy` and `/undeploy` translating actions to CRDs. |
| **SPEC-FR-M4.8** | Community Echo Endpoint | keeper, agent | Event loop validating end-to-end NATS connectivity on deployment. |

### Resolved Bugs

| Bug ID | Title | Status | Severity | Description |
|--------|-------|--------|----------|-------------|
| **BUG-M4.1** | Assigned Agent Pods Fail to Deploy | CLOSED | HIGH | Removed stubbed mock implementations in the Operator controller. |
| **BUG-M4.2** | Health and Metrics Endpoints Leak Tracing Spans | CLOSED | MEDIUM | Omit generating tracing spans for internal/readiness polling. |
| **BUG-M4.3** | Overlapping Observability Frameworks | CLOSED | HIGH | Fixed metrics clashes between custom Prometheus registry and OTel. |
| **BUG-M4.4** | Redundant Infrastructure Monitoring Services | CLOSED | MEDIUM | Removed Keycloak Zipkin tracing links and extra Prometheus server. |
| **BUG-M4.5** | Echo Endpoint Fails with 503 due to DB Status Check | CLOSED | HIGH | Prevented static DB connectivity checks from blocking echo health. |
| **BUG-M4.6** | Echo Message Fails to Dispatch due to NATS Subject Mismatch | CLOSED | HIGH | Aligned NATS target routing subjects to agent UUID strings. |
| **BUG-M4.7** | Agent Pod Never Subscribes to NATS Echo Subject | CLOSED | HIGH | Wired the NATS echo subscriber block in Agent's main bootstrap. |
| **BUG-M4.8** | OTel Trace Context Not Propagated Across NATS Boundary | CLOSED | HIGH | Correctly injected/extracted span context headers in NATS message. |
