# Tacito Square Helm Chart

Umbrella Helm chart for deploying the complete Tacito Square multi-agent platform on Kubernetes.

## Components Deployed

| Component | Default | Description |
|-----------|---------|-------------|
| Keeper | Enabled | Control plane for agent lifecycle management |
| Agent template | ConfigMap | Template config used by Keeper to spawn agents |
| NATS | Enabled | Inter-agent and agent-keeper messaging |
| Redis | Enabled | Agent short-term memory |
| PostgreSQL | Enabled | Keeper persistence |
| Qdrant | Enabled | Agent long-term vector memory |
| OpenTelemetry Collector | Enabled | Span/metrics collection and export |
| Keycloak | Enabled (dev) | OIDC identity provider with realm config |
| MinIO | Disabled | S3-compatible object storage (opt-in) |
| Operator | Disabled | CRD-based agent management (enable in M5) |
| BFF | Disabled | Backend-for-frontend (enable in M6) |

## Quick Start — Minimal Deployment

```bash
# Create local Kind cluster
kind create cluster --config deploy/dev/kind-config.yaml

# Install with defaults (all infrastructure included)
helm dependency update deploy/helm/tacito-square
helm install tacito-square deploy/helm/tacito-square --wait
```

This deploys the Keeper + all infrastructure dependencies in a single command.
Images default to `localhost:5000` registry — suitable for Kind with a local registry.

## Values Reference

### Global

| Key | Default | Description |
|-----|---------|-------------|
| `global.imageRegistry` | `localhost:5000` | Default container registry (local) |
| `global.imagePullSecrets` | `[]` | Pull secrets for private registries |

### Image Resolution

Each component's image is resolved as: `{registry}/{name}:{tag}` where registry falls back: **per-component** → **global.imageRegistry** → **localhost:5000**.

### Keeper

| Key | Default | Description |
|-----|---------|-------------|
| `keeper.replicaCount` | `1` | Number of Keeper replicas |
| `keeper.image.registry` | `""` | Override `global.imageRegistry` |
| `keeper.image.name` | `tacito-square/keeper` | Image name (without registry) |
| `keeper.image.tag` | `0.1.0` | Image version tag |
| `keeper.resources.requests.cpu` | `100m` | CPU request |
| `keeper.resources.limits.memory` | `256Mi` | Memory limit |
| `keeper.env.TS_KEEPER_OIDC_ISSUER` | `...keycloak.../realms/tacito` | OIDC issuer URL |
| `keeper.env.TS_KEEPER_OIDC_CLIENT_ID` | `tacito-keeper` | OIDC client ID |
| `keeper.service.port` | `8080` | Service port |

### Agent (Template)

| Key | Default | Description |
|-----|---------|-------------|
| `agent.image.registry` | `""` | Override `global.imageRegistry` |
| `agent.image.name` | `tacito-square/agent` | Image name (without registry) |
| `agent.image.tag` | `0.1.0` | Agent image version |
| `agent.resources.limits.memory` | `512Mi` | Memory limit per agent |

### Infrastructure

| Key | Default | Description |
|-----|---------|-------------|
| `nats.enabled` | `true` | Deploy NATS |
| `redis.enabled` | `true` | Deploy Redis |
| `postgresql.enabled` | `true` | Deploy PostgreSQL |
| `qdrant.enabled` | `true` | Deploy Qdrant |
| `otel-collector.enabled` | `true` | Deploy OpenTelemetry Collector |
| `keycloak.enabled` | `true` | Deploy Keycloak (dev mode) |
| `minio.enabled` | `false` | Deploy MinIO (S3-compatible, opt-in) |
| `minio.auth.rootUser` | `tacito-dev` | MinIO root user |
| `minio.auth.rootPassword` | `tacito-dev-secret` | MinIO root password |
| `minio.defaultBuckets` | `tacito-artifacts` | Auto-created bucket |
| `minio.persistence.size` | `5Gi` | Storage size |

### IAM (Keycloak)

The chart ships with a Keycloak dev instance pre-configured with:

| Resource | Value | Description |
|----------|-------|-------------|
| Realm | `tacito` | OIDC realm for all Tacito Square components |
| Client: `tacito-keeper` | Confidential | Keeper service account (secret: `keeper-dev-secret`) |
| Client: `tacito-ui` | Public | SPA/UI PKCE client |
| Role: `keeper-admin` | — | Full Keeper administration |
| Role: `keeper-viewer` | — | Read-only Keeper access |
| Role: `user` | — | End user agent interaction |
| Role: `agent-spawner` | — | Can spawn agents |
| User: `admin` / `admin` | `keeper-admin`, `agent-spawner` | Dev admin user |
| User: `user` / `user` | `user` | Dev end user |

## Externalizing Dependencies

For production deployments, disable bundled infrastructure and point to external services:

```yaml
# values-production.yaml

# Use private registry
global:
  imageRegistry: "ghcr.io/morphy76"

# Disable bundled infrastructure
nats:
  enabled: false
redis:
  enabled: false
postgresql:
  enabled: false
qdrant:
  enabled: false
keycloak:
  enabled: false
otel-collector:
  enabled: false
minio:
  enabled: false

# Point to external services
keeper:
  env:
    TS_KEEPER_DB_HOST: "my-rds-instance.region.rds.amazonaws.com"
    TS_KEEPER_NATS_URL: "nats://my-nats-cluster:4222"
    TS_KEEPER_OIDC_ISSUER: "https://my-keycloak.example.com/realms/tacito"
    TS_KEEPER_OIDC_CLIENT_ID: "tacito-keeper-prod"
    TS_KEEPER_OTEL_ENDPOINT: "my-otel-collector:4317"
    TS_KEEPER_S3_ENDPOINT: "https://s3.us-east-1.amazonaws.com"
    TS_KEEPER_S3_BUCKET: "my-tacito-artifacts"
    TS_KEEPER_S3_REGION: "us-east-1"

agent:
  env:
    TS_AGENT_REDIS_URL: "redis://my-elasticache:6379"
    TS_AGENT_QDRANT_URL: "http://my-qdrant:6334"
    TS_AGENT_NATS_URL: "nats://my-nats-cluster:4222"
    TS_AGENT_OTEL_ENDPOINT: "my-otel-collector:4317"
```

Deploy with overrides:

```bash
helm install tacito-square deploy/helm/tacito-square -f values-production.yaml
```

## Health Probes

All components expose health endpoints validated by K8s probes:

| Endpoint | Type | Checks |
|----------|------|--------|
| `/healthz` | Liveness | Process is alive |
| `/readyz` | Readiness | All dependencies reachable (DB, NATS, Redis, Qdrant) |

Readiness checks per component:

- **Keeper**: PostgreSQL ping + NATS connection
- **Agent**: NATS + Redis + Qdrant + LLM reachability
- **BFF**: Keeper API reachability
- **Operator**: K8s API server connectivity

## Upgrading

```bash
# Update a single component version
helm upgrade tacito-square deploy/helm/tacito-square \
  --set keeper.image.tag=0.2.0

# Switch to remote registry
helm upgrade tacito-square deploy/helm/tacito-square \
  --set global.imageRegistry=ghcr.io/morphy76
```
