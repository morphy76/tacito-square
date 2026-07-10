# Standalone Tacito Agent Helm Chart

This Helm chart deploys a single statically configured **Tacito Agent** workload in Kubernetes, completely independent of the dynamic Keeper APIs and the Kubernetes Operator controller.

It is designed for local development, debugging, testing, or static single-agent pipelines.

## Architecture

1. **Tacito Agent Deployment**: Runs the standard `agent` container image. It mounts all environment variables from a flat ConfigMap using `envFrom`.
2. **Kubernetes ConfigMap**: Dynamically generates `TS_AGENT_*` environment variables based on structured chart values.
3. **Secure Credentials Injection**: Maps any external LLM api-key secret securely to the `TS_AGENT_LLM_API_KEY` environment variable.

## Installation

To deploy the standalone agent:

```bash
helm upgrade --install my-agent tools/helm/tacito-agent \
  --namespace tacito \
  --create-namespace \
  --set agent.name="alpha-agent"
```

## Configuration

The following table lists the configurable parameters of the chart and their default values:

| Parameter | Description | Default |
|-----------|-------------|---------|
| `replicaCount` | Number of agent replicas | `1` |
| `image.name` | Agent container image name | `tacito-square/agent` |
| `image.tag` | Agent container image tag | `0.1.0` |
| `port` | Container port for HTTP probes | `8081` |
| `agent.id` | Statically configured Agent UUID | `00000000-0000-0000-0000-000000000000` |
| `agent.name` | Name of the static Agent | `standalone-agent` |
| `agent.tenantId` | Tenant isolation ID | `default-tenant` |
| `agent.communityRef` | Parent community ID | `standalone-community` |
| `agent.llm.model` | LLM model name | `gpt-4o` |
| `agent.llm.endpoint` | API endpoint for the LLM | `https://api.openai.com/v1` |
| `agent.llm.temperature` | Sampling temperature | `0.7` |
| `agent.llm.maxTokens` | Maximum completion tokens | `2048` |
| `agent.llm.credentialsSecret` | Secret reference containing the LLM API key | `tacito-agent-llm-secret` |
| `agent.systemPrompt` | Static multi-line system prompt | *(standard helper prompt)* |
| `redis.url` | Short-term memory Redis URL | `rediss://ts-infra-redis:6379` |
| `qdrant.url` | Long-term memory Qdrant URL | `http://ts-infra-qdrant:6334` |
| `nats.url` | NATS message bus URL | `nats://ts-infra-nats:4222` |
| `s3.url` | Object storage URL | `https://ts-infra-minio:9000` |

---
## NATS Stream & Subject Catalog

The standalone agent communicates entirely via NATS. Below is the catalog of messaging subjects utilized by the system:

| Subject Structure | Type | Description | Payload Format |
|---|---|---|---|
| `ts.community.<community_id>.agent.<agent_name>` | Request/Reply | Inbound mailbox for agent interactions. | **Request**: `EchoRequest` (JSON)<br>**Reply**: `EchoReply` (JSON) |
| `ts.community.>` | Wildcard Monitor | Listens to all community messaging streams and agent traffic. | All request/reply frames. |

### Payload Schema Definitions

#### EchoRequest (Request)
```json
{
  "message": "Hello Agent",
  "tenant_id": "default-tenant"
}
```

#### EchoReply (Response)
```json
{
  "agent_name": "standalone-agent",
  "decorated": "[standalone-agent] -> Hello Agent",
  "timestamp": "2026-05-31T10:15:00Z"
}
```

---

## NATS CLI Verification Procedures

Since the standalone agent communicates exclusively through the NATS message bus, verification is performed using the official `nats` CLI.

### Pattern 1: In-Cluster Verification (Recommended)

To thoroughly verify agent execution, combine interactive request dispatching with live agent log tailing:

1. **Shell 1 (Monitor Agent Logs)**: Tail the logs of the running agent container to observe incoming message handlers:
   ```bash
   kubectl -n tacito logs -f deployment/ts-agent-tacito-agent-unit
   ```

2. **Shell 2 (Interactive NATS Box)**: Launch a temporary interactive NATS CLI container:
   ```bash
   kubectl -n tacito run -i --rm --tty nats-box-pub --image=natsio/nats-box --restart=Never
   ```

3. **Publish Request (Inside NATS Box)**: Send a request payload directly to the standalone agent's mailbox:
   ```bash
   nats pub -J -s "nats://ts-infra-nats:4222" ts.community.standalone-community.agent.standalone-agent -H X-Tacito-Schema:"urn:tacito:schema:conversational:add-user-message:v1" -H X-Tacito-Tenant:"default-tenant" '{"tenant_id": "default-tenant", "schema_ref": "urn:tacito:schema:conversational:add-user-message:v1", "payload": {"message": "Hello, my name is Riccardo, how are you? I need help.", "thread_id": "test-28", "community_id": "8f559a1d-2de5-496f-bac7-bdc6ebff3daa"}}'
   ```

4. **Shell 3 (Subscribe to all agent messages)**:
   ```bash
   nats sub -s "nats://ts-infra-nats:4222" "ts.community.>"
   ```

   **Agent Logs Output (Shell 1)**:
   The agent will process the request and log a structured entry:
   ```json
   {"level":"info","trace_id":"b8296482c537acd90fcb1c3aefe20875","span_id":"188b7a54c0c209a9","agent_name":"standalone-agent","community_id":"standalone-community","tenant_id":"default-tenant","message":"Hello Agent","time":"2026-05-31T08:28:20Z","message":"echo request received"}
   ```

   **NATS Box Reply (Shell 2)**:
   The NATS CLI will print the response returned by the agent:
   ```json
   {
     "agent_name": "standalone-agent",
     "decorated": "[standalone-agent] -> Hello Agent",
     "timestamp": "2026-05-31T08:28:20Z"
   }
   ```

### Pattern 2: Local Host Verification via Port-Forwarding

1. Port-forward the NATS infrastructure service to your local machine:
   ```bash
   kubectl port-forward svc/ts-infra-nats 4222:4222 -n tacito
   ```
2. In your local terminal, subscribe to monitor all communications:
   ```bash
   nats sub -s "nats://localhost:4222" "ts.community.>"
   ```
3. Publish queries directly to the agent's subject:
   ```bash
   nats request -s "nats://localhost:4222" ts.community.standalone-community.agent.standalone-agent '{"message": "Tell me a joke", "tenant_id": "default-tenant"}'
   ```
