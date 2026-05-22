# Tacito Square — Application Helm Chart

Deploys the Tacito Square application components (Keeper, Operator, BFF, and the Agent characterization templates) on Kubernetes.

> [!IMPORTANT]
> **Decoupled Architecture**: This chart does NOT bundle database or messaging infrastructure. It depends on external infrastructure services. The [Infrastructure Helm Chart](file:///Users/R.Pasquini/Projects/side/tacito-square/tools/helm/tacito-square-infra/README.md) is a hard prerequisite and must be installed first.

## Components Deployed

| Component          | Default                 | Description                                            |
| ------------------ | ----------------------- | ------------------------------------------------------ |
| **Keeper**         | Enabled                 | Control plane for agent lifecycle management           |
| **Agent template** | ConfigMap               | Base template used to characterize spawned agents      |
| **Operator**       | Disabled (enable in M4) | CRD reconciliation controller watching Agent resources |
| **BFF**            | Disabled (enable in M7) | Backend-for-frontend routing and API translation layer |

## Prerequisites

1. **Kubernetes Cluster** (v1.29+ or local Kind/Minikube)
2. **Infrastructure Release** (`tacito-square-infra` installed and running)
3. **Local Registry** (default `tacito-square` or custom configured registry)

## Quick Start

```bash
# Step 1: Add external infrastructure chart dependencies & install them
make helm-infra-deps
make helm-infra-install

# Step 2: Install application components
make helm-install
```

Using Helm directly (assuming default release name `tacito-infra` for infrastructure):

```bash
helm install tacito tools/helm/tacito-square/ --wait
```

## Binding Interfaces

The application components connect to database, messaging, and storage via binding environment variables in `values.yaml`. Default values in `values.yaml` are auto-configured to bind to an infrastructure release named `tacito-infra`.

If your infrastructure release has a different name or is hosted externally, adjust the binding variables accordingly:

### Keeper Bindings

| Key                                  | Default                                                    | Description                 |
| ------------------------------------ | ---------------------------------------------------------- | --------------------------- |
| `keeper.env.TS_KEEPER_DB_HOST`       | `tacito-infra-postgresql`                                  | PostgreSQL host             |
| `keeper.env.TS_KEEPER_DB_NAME`       | `tacito`                                                   | PostgreSQL database         |
| `keeper.env.TS_KEEPER_DB_USER`       | `tacito`                                                   | PostgreSQL user             |
| `keeper.env.TS_KEEPER_DB_SSLMODE`    | `require`                                                  | PostgreSQL SSL mode         |
| `keeper.env.TS_KEEPER_NATS_URL`      | `nats://tacito-infra-nats:4222`                            | NATS connection URL         |
| `keeper.env.TS_KEEPER_REDIS_URL`     | `rediss://tacito-infra-redis:6379`                         | Redis connection URL (TLS)  |
| `keeper.env.TS_KEEPER_OTEL_ENDPOINT` | `tacito-infra-otel-collector:4317`                         | OpenTelemetry gRPC endpoint |
| `keeper.env.TS_KEEPER_OIDC_ISSUER`   | `https://tacito-infra-keycloak-http:8443/realms/tacito`    | OIDC provider token issuer  |
| `keeper.env.TS_KEEPER_S3_ENDPOINT`   | `https://tacito-infra-minio:9000`                          | S3 endpoint (MinIO, TLS)    |
| `keeper.env.TS_KEEPER_CA_CERT_PATH`  | `/etc/ssl/tacito/ca.crt`                                   | CA trust bundle mount path  |

### Agent Template Bindings

When the Keeper and Operator spawn agents, they receive configurations defined under the `agent.env` block:

| Key                                | Default                                    | Description                  |
| ---------------------------------- | ------------------------------------------ | ---------------------------- |
| `agent.env.TS_AGENT_NATS_URL`      | `nats://tacito-infra-nats:4222`            | NATS connection URL          |
| `agent.env.TS_AGENT_REDIS_URL`     | `rediss://tacito-infra-redis:6379`         | Redis connection URL (TLS)   |
| `agent.env.TS_AGENT_QDRANT_URL`    | `http://tacito-infra-qdrant:6334`          | Qdrant gRPC vector store URL |
| `agent.env.TS_AGENT_OTEL_ENDPOINT` | `tacito-infra-otel-collector:4317`         | OpenTelemetry gRPC endpoint  |
| `agent.env.TS_AGENT_S3_ENDPOINT`   | `https://tacito-infra-minio:9000`          | S3 endpoint (MinIO, TLS)     |
| `agent.env.TS_AGENT_CA_CERT_PATH`  | `/etc/ssl/tacito/ca.crt`                   | CA trust bundle mount path   |

## Values Reference

### Global Resolution

| Key                       | Default          | Description                                |
| ------------------------- | ---------------- | ------------------------------------------ |
| `global.imageRegistry`    | `""`             | Fallback container registry                |
| `global.infraReleaseName` | `tacito-infra`   | Default prefix for infrastructure services |

### Keeper Settings

| Key                     | Default                | Description             |
| ----------------------- | ---------------------- | ----------------------- |
| `keeper.replicaCount`   | `1`                    | Replica count           |
| `keeper.image.registry` | `""`                   | Registry override       |
| `keeper.image.name`     | `tacito-square/keeper` | Container image name    |
| `keeper.image.tag`      | `0.1.0`                | Image version tag       |
| `keeper.service.port`   | `8080`                 | Port exposed by service |

### Operator Settings

| Key                   | Default                  | Description                        |
| --------------------- | ------------------------ | ---------------------------------- |
| `operator.enabled`    | `false`                  | Enable the CRD Operator controller |
| `operator.image.name` | `tacito-square/operator` | Image name                         |
| `operator.image.tag`  | `0.1.0`                  | Version tag                        |

### BFF Settings

| Key                | Default             | Description               |
| ------------------ | ------------------- | ------------------------- |
| `bff.enabled`      | `false`             | Enable the BFF API bridge |
| `bff.image.name`   | `tacito-square/bff` | Image name                |
| `bff.image.tag`    | `0.1.0`             | Version tag               |
| `bff.service.port` | `8083`              | Port exposed by service   |

## Health Probes

All running components expose structured health endpoints validated by Kubernetes probes:
- `/healthz` (liveness): returns `{"status":"alive"}` when the engine is running.
- `/readyz` (readiness): returns `{"status":"ready"}` if external services are reachable.
