# SPEC-FR-M5.2: LLM Reasoning (Brain Adapter)

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | SPEC-FR-M5.2                                |
| Status        | ACCEPTED                                    |
| Milestone     | M5                                          |
| Component     | agent                                       |
| Depends On    | SPEC-FR-M5.1                                |
| Supersedes    | none                                        |

## Context

Agents reason by sending prompts to an LLM and processing responses. Decoupling reasoning from concrete providers is achieved using the Hexagonal architecture, which abstracts provider-specific clients behind a clean port interface. Keeping the reasoning interface stateless avoids placing premature constraints on future pipeline extensions (such as short-term memory, long-term memory, or parallel fan-out/fan-in patterns).

---

## Specification

### 1. Hexagonal Ports & Domain Models
- **Outbound Port**: Define the `Brain` outbound port interface inside `internal/agent/application/ports/outbound/brain.go`.
  ```go
  package outbound

  import (
      "context"
      "github.com/morphy76/tacito-square/internal/agent/domain/model"
  )

  type Brain interface {
      Generate(ctx context.Context, request model.BrainRequest) (*model.BrainResponse, error)
      GenerateStream(ctx context.Context, request model.BrainRequest) (<-chan model.BrainStreamChunk, <-chan error, error)
  }
  ```
- **Domain Models**: Keep core business models pure and stateless inside `internal/agent/domain/model/brain.go`:
  - `BrainRequest`: Structured model containing prompt, system prompt, temperature, max tokens, and provider options.
  - `BrainResponse`: Contains the resulting generated text, usage metadata (input/output token counts), and completion reasons.
  - `BrainStreamChunk`: Contains partial text fragments and termination flags for streaming responses.

### 2. LLM Provider Adapters & Selection
- **OpenAI Adapter**: Implement an adapter inside `internal/agent/adapters/outbound/openai/openai_adapter.go` using the `openai-go` library.
- **Ollama Adapter**: Implement an adapter inside `internal/agent/adapters/outbound/ollama/ollama_adapter.go` communicating via a standard HTTP client with the local Ollama API.
- **Runtime Selection**: 
  - Parse the active provider from the environment variable `TS_AGENT_BRAIN_PROVIDER`.
  - Supported values: `openai` and `ollama`.
  - Default to `openai` if the variable is missing or empty.
- **Configuration Bindings**:
  - `TS_AGENT_BRAIN_PROVIDER`: Active provider (`openai` or `ollama`, defaults to `openai`).
  - `TS_AGENT_OPENAI_ENDPOINT`: Custom target endpoint for OpenAI requests.
  - `TS_AGENT_OPENAI_API_KEY`: Authentication credentials for OpenAI API.
  - `TS_AGENT_OLLAMA_ENDPOINT`: Target API endpoint for Ollama (e.g. `http://ollama-service:11434`).
  - `TS_AGENT_BRAIN_MODEL`: Model descriptor string (e.g. `gpt-4o` or `llama3`).
  - `TS_AGENT_BRAIN_TEMPERATURE`: Temperature parameter (defaults to `0.7`).
  - `TS_AGENT_BRAIN_MAX_TOKENS`: Maximum output tokens (defaults to `1000`).
  - `TS_AGENT_BRAIN_TIMEOUT_SECONDS`: Request deadline duration (defaults to `30`).

### 3. Outbound Port Resiliency
- **Circuit Breakers**: Every outbound request made by an LLM adapter must be executed within a tested circuit breaker. 
  - **Implementation Choice**: The agent uses a Go-native, lightweight, state-machine-based circuit breaker implemented in `internal/agent/adapters/outbound/resiliency/circuit_breaker.go` rather than an external library.
  - **Rationale**: A Go-native implementation allows custom integration of environment variable configuration mappings and direct injection of a fallback operation block during execution without pulling in third-party library dependencies.
  - **Configuration Keys**:
    - `TS_AGENT_BRAIN_CIRCUIT_FAILURE_THRESHOLD`: The number of consecutive failures before the circuit trips open (defaults to `5`).
    - `TS_AGENT_BRAIN_CIRCUIT_RECOVERY_TIMEOUT_SECONDS`: The duration in seconds to wait in the open state before transitioning to half-open (defaults to `15`).
  - **Fallback Support**: The circuit breaker directly supports a configurable fallback operation function. When the circuit is `Open` or a request fails, the fallback operation is executed automatically.
- **Configurable Timeouts**: LLM requests must carry a configurable deadline propagated via `context.Context`, utilizing the value of `TS_AGENT_BRAIN_TIMEOUT_SECONDS` (defaulting to 30 seconds if unspecified).
- **Retries & Backoff**: Standardize transient network retries with exponential backoff and randomized jitter to prevent service starvation. Utilizes the approved `github.com/sethvargo/go-retry` package already defined in the project.

### 4. Simple Incoming Message Processing Pipeline
- To trigger the stateless `Brain` port, define a simple use case pipeline:
  - **Inbound Port**: Define the `MessageProcessor` interface inside `internal/agent/application/ports/inbound/message.go`.
    ```go
    package inbound

    import "context"

    type MessageProcessor interface {
        ProcessIncomingMessage(ctx context.Context, payload string) (string, error)
    }
    ```
  - **Application Service**: Implement this use case inside `internal/agent/application/service/message_processor.go`. The service accepts the incoming message payload, constructs a stateless `BrainRequest`, triggers the configured outbound `Brain` port, and returns the generated reasoning response.
  - **Driving Adapter Hook**: Wire the simple use case into the NATS subscriber (`internal/agent/adapters/inbound/nats/echo_subscriber.go`) so that incoming event payloads execute the reasoning pipeline and publish responses.

### 5. Local Dev Mock Servers (Helm Chart)
- The standalone agent-only Helm chart (`tools/helm/tacito-agent`) must deploy local API mock servers by default.
- Use free and open-source API mocking platforms (e.g., standard multi-endpoint MockServer or lightweight API simulators using pre-configured mock configurations) packaged as sidecar or separate deployment pods.
- Pre-seed mock configurations to return standard, valid JSON payloads matching OpenAI and Ollama APIs to enable offline development and testing.

---

## Acceptance Criteria

1. **Hexagonal Compile Compliance**:
   - Code residing in `internal/agent/domain/` contains zero imports referencing `application/` or `adapters/`.
   - The application service in `internal/agent/application/service/` interacts with adapters exclusively via outbound port interfaces.
2. **Provider Selection and Fallback**:
   - With `TS_AGENT_BRAIN_PROVIDER=""`, the agent defaults to loading the OpenAI adapter.
   - With `TS_AGENT_BRAIN_PROVIDER="ollama"`, the agent loads the Ollama adapter and points to the configured endpoint.
3. **Resiliency Validation**:
   - Simulating LLM timeout triggers the configured circuit breaker after the specified limit (e.g. 30 seconds default).
   - Transient failures trigger backoff retries before bubbling up.
4. **Mock Server API Parity**:
   - The local Helm deployment deploys mock containers.
   - Sending requests to mock OpenAI and Ollama endpoints returns standard API structures without requiring external API keys.

---

## Test Plan

### Automated Tests
1. **Unit Tests**:
   - Verify environment variable mapping and default value parsing using Viper.
   - Validate circuit breaker state transitions under simulated error rates.
2. **Integration Tests**:
   - Run integration suites against the mock OpenAI and Ollama API endpoints deployed locally, verifying correct parsing of regular and streaming responses.
   - Verify NATS subscriber invokes the application use case service and publishes the response.

### Manual Verification
1. Run local Helm template validation to verify the presence of the mock server pods, services, and associated network configuration:
   ```bash
   helm template my-agent tools/helm/tacito-agent
   ```
2. Inspect values to verify mock servers are enabled and running by default.

---

## Files Affected

- `[NEW] internal/agent/domain/model/brain.go` — stateless LLM request, response, and stream structures.
- `[NEW] internal/agent/application/ports/outbound/brain.go` — driven port abstraction.
- `[NEW] internal/agent/application/ports/inbound/message.go` — driving port for incoming message processing.
- `[NEW] internal/agent/application/service/message_processor.go` — simple orchestrating use case service.
- `[NEW] internal/agent/adapters/outbound/openai/openai_adapter.go` — OpenAI concrete implementation.
- `[NEW] internal/agent/adapters/outbound/ollama/ollama_adapter.go` — Ollama concrete implementation.
- `[MODIFY] internal/agent/adapters/inbound/nats/echo_subscriber.go` — wire subscriber to use the application service pipeline.
- `[MODIFY] tools/helm/tacito-agent/Chart.yaml` — declare mock server sub-components.
- `[MODIFY] tools/helm/tacito-agent/values.yaml` — configure mock API endpoints and selection parameters.
- `[NEW] tools/helm/tacito-agent/templates/mocks.yaml` — deployment and service definitions for local developer mock servers.

