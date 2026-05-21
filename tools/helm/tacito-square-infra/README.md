# Tacito Square — Infrastructure Helm Chart

Deploys all external dependencies required by Tacito Square as a self-contained Helm release, separate from the application chart.

## Services Included

| Service | Purpose | Default Port |
|---------|---------|-------------|
| NATS | Messaging backbone (agent-to-agent, keeper-to-agent) | 4222 |
| Redis | Short-term memory (STM) + application cache | 6379 |
| PostgreSQL | Keeper persistence (agents, communities, audit) | 5432 |
| Qdrant | Vector storage for long-term agent memory | 6334 (gRPC) / 6333 (HTTP) |
| OTel Collector | OpenTelemetry trace collection | 4317 (gRPC) / 4318 (HTTP) |
| Keycloak | OIDC identity provider | 8080 |
| MinIO | S3-compatible object storage (opt-in) | 9000 |

## Quick Start

```bash
# Download sub-chart dependencies
make helm-infra-deps

# Lint the chart
make helm-infra-lint

# Install on the current cluster
make helm-infra-install

# Uninstall
make helm-infra-uninstall
```

Or using Helm directly:

```bash
helm dependency update tools/helm/tacito-square-infra/
helm install tacito-infra tools/helm/tacito-square-infra/ --wait
```

## Configuration

Each sub-chart can be independently enabled/disabled:

```bash
# Disable MinIO (opt-in by default)
helm install tacito-infra tools/helm/tacito-square-infra/ --set minio.enabled=true

# Disable Keycloak (e.g., using external IdP)
helm install tacito-infra tools/helm/tacito-square-infra/ --set keycloak.enabled=false
```

### Key Values

| Key | Default | Description |
|-----|---------|-------------|
| `nats.enabled` | `true` | Deploy NATS |
| `redis.enabled` | `true` | Deploy Redis (standalone, no auth) |
| `postgresql.enabled` | `true` | Deploy PostgreSQL |
| `postgresql.auth.username` | `keeper` | PostgreSQL username |
| `postgresql.auth.database` | `tacito_keeper` | PostgreSQL database |
| `qdrant.enabled` | `true` | Deploy Qdrant |
| `otel-collector.enabled` | `true` | Deploy OTel Collector |
| `keycloak.enabled` | `true` | Deploy Keycloak with `tacito` realm |
| `minio.enabled` | `false` | Deploy MinIO (opt-in) |

## Keycloak Pre-configured Realm

The `tacito` realm is automatically provisioned with:

- **Clients**: `tacito-keeper` (confidential), `tacito-ui` (public/PKCE)
- **Roles**: `keeper-admin`, `keeper-viewer`, `user`, `agent-spawner`
- **Dev Users**: `admin` (keeper-admin + agent-spawner), `user` (user)

## Binding to the Application Chart

After deploying the infrastructure chart, configure the application chart to bind:

```bash
helm install tacito tools/helm/tacito-square/ \
  --set keeper.env.TS_KEEPER_DB_HOST=tacito-infra-postgresql \
  --set keeper.env.TS_KEEPER_NATS_URL=nats://tacito-infra-nats:4222
```

Default values in the application chart assume infrastructure release name `tacito-infra`.
