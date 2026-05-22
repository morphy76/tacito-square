# Tacito Square — Infrastructure Helm Chart

Deploys all external dependencies required by Tacito Square as a self-contained Helm release, separate from the application chart.

## Services Included

| Service | Purpose | Default Port | Security |
|---------|---------|-------------|----------|
| NATS | Messaging backbone (agent-to-agent, keeper-to-agent) | 4222 | TLS + token auth |
| Redis | Short-term memory (STM) + application cache | 6379 | TLS + password auth |
| PostgreSQL | Keeper persistence (agents, communities, audit) | 5432 | SSL required |
| Qdrant | Vector storage for long-term agent memory | 6334 (gRPC) / 6333 (HTTP) | — |
| OTel Collector | OpenTelemetry trace collection | 4317 (gRPC) / 4318 (HTTP) | — |
| Keycloak | OIDC identity provider | 8443 (HTTPS) / 8080 (HTTP internal) | TLS (production mode) |
| MinIO | S3-compatible object storage | 9000 | TLS |

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
# Disable MinIO
helm install tacito-infra tools/helm/tacito-square-infra/ --set minio.enabled=false

# Disable Keycloak (e.g., using external IdP)
helm install tacito-infra tools/helm/tacito-square-infra/ --set keycloak.enabled=false
```

### Key Values

| Key | Default | Description |
|-----|---------|-------------|
| `nats.enabled` | `true` | Deploy NATS (TLS + token auth) |
| `redis.enabled` | `true` | Deploy Redis (TLS + password auth) |
| `postgresql.enabled` | `true` | Deploy PostgreSQL (SSL required) |
| `postgresql.userDatabase.name.value` | `tacito` | PostgreSQL database |
| `postgresql.userDatabase.user.value` | `tacito` | PostgreSQL username |
| `qdrant.enabled` | `true` | Deploy Qdrant |
| `otel-collector.enabled` | `true` | Deploy OTel Collector |
| `keycloak.enabled` | `true` | Deploy Keycloak with `tacito` realm (HTTPS) |
| `minio.enabled` | `true` | Deploy MinIO (TLS, private `tacito` bucket) |

## Keycloak Pre-configured Realm

The `tacito` realm is automatically provisioned with:

- **Clients**: `tacito-keeper` (confidential), `tacito-ui` (public/PKCE)
- **Roles**: `keeper-admin`, `keeper-viewer`, `user`, `agent-spawner`
- **Dev Users**: `admin` (keeper-admin + agent-spawner), `user` (user)

## TLS Trust

At install time, a Helm hook Job auto-generates a self-signed CA and per-service TLS certificates. The CA public certificate is exported as a **ConfigMap** named `<release>-ca-bundle` (key: `ca.crt`). Mount this ConfigMap in application pods to trust all infrastructure service certificates.

For production, replace the generated secrets with certificates from cert-manager or an external CA. The hook is idempotent — it skips generation if secrets already exist.

## Binding to the Application Chart

After deploying the infrastructure chart, configure the application chart to bind:

```bash
helm install tacito tools/helm/tacito-square/ \
  --set keeper.env.TS_KEEPER_DB_HOST=tacito-infra-postgresql \
  --set keeper.env.TS_KEEPER_DB_SSLMODE=require \
  --set keeper.env.TS_KEEPER_REDIS_URL=rediss://tacito-infra-redis:6379 \
  --set keeper.env.TS_KEEPER_OIDC_ISSUER=https://tacito-infra-keycloak-http:8443/realms/tacito
```

Default values in the application chart assume infrastructure release name `tacito-infra` with TLS-secured endpoints.
