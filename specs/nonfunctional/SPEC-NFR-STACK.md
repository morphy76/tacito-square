# SPEC-NFR-STACK: Technology Stack

| Field         | Value                              |
|---------------|------------------------------------|
| ID            | SPEC-NFR-STACK                     |
| Status        | DRAFT                              |
| Component     | all                                |

## Specification

These technology choices are LOCKED. Changes require a constitution amendment.

| Layer | Technology | Package/Chart |
|-------|-----------|---------------|
| Language | Go 1.26 | — |
| HTTP Framework | Gin | `github.com/gin-gonic/gin` |
| Logging | zerolog | `github.com/rs/zerolog` |
| Tracing | OpenTelemetry OTLP gRPC | `go.opentelemetry.io/otel` |
| Testing | testify + testcontainers-go | `github.com/stretchr/testify` |
| Config | Viper | `github.com/spf13/viper` |
| LLM Client | OpenAI Go API Library | `github.com/openai/openai-go` |
| STM | Redis | `github.com/redis/go-redis` |
| LTM | Qdrant gRPC | `github.com/qdrant/go-client` |
| Persistence | PostgreSQL | `github.com/jackc/pgx` |
| Migrations | goose | `github.com/pressly/goose` |
| Messaging | NATS | `github.com/nats-io/nats.go` |
| MCP | MCP (client SDKs) | `github.com/modelcontextprotocol/go-sdk` |
| OIDC | Zitadel (client SDKs) | `github.com/zitadel/oidc/v3` |
| Operator | Kubebuilder | `sigs.k8s.io/controller-runtime` |
| Frontend | React 19 + Compiler | — |
| Object Storage | S3-compatible (MinIO for dev) | `github.com/aws/aws-sdk-go-v2` |
| Infra Cache | Redis (shared instance, namespaced) | `github.com/redis/go-redis` |
| Metrics | Prometheus (exposition format) | `github.com/prometheus/client_golang` |
| Autoscaling | HPA + custom metrics | prometheus-adapter or KEDA |
| Deployment | Helm umbrella chart | `deploy/helm/tacito-square/` |
| Docker base | distroless | `gcr.io/distroless/base-nossl-debian13` |

## Acceptance Criteria

1. `go.mod` lists only approved dependencies
2. Helm `Chart.yaml` references only approved sub-charts
3. Dockerfiles use distroless base
