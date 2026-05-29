# BUG-M1.1: Infrastructure Services Do Not Enforce SSL/TLS or Authenticated Connections

| Field         | Value                                                              |
|---------------|-------------------------------------------------------------------|
| ID            | BUG-M1.1                                                          |
| Status        | OPEN                                                              |
| Severity      | HIGH                                                              |
| Milestone     | M1 — Infrastructure Helm Chart                                    |
| Affects       | `tools/helm/tacito-square-infra/values.yaml`                      |
| Violates      | SPEC-FR-M2.11, SPEC-NFR-STACK, SPEC-NFR-CLOUD                    |
| Discovered    | Keeper health-check and migration failures in Helm deployments    |

## Problem Statement

The `tacito-square-infra` Helm chart deploys seven infrastructure services — PostgreSQL, Redis, NATS, Qdrant, Keycloak, MinIO, and the OpenTelemetry Collector — but **none** of them currently run with SSL/TLS or proper authentication enabled.
The TLS generation job (`tls-generate-job.yaml`) and per-service TLS secret generation are already implemented but **not wired up**: every service has its secure configuration blocks commented out in `values.yaml`, and the `tls.generate` flag defaults to `false`.

This violates `SPEC-FR-M2.11` (Secured Infrastructure Provisioning), which mandates that all services communicate over encrypted channels with authenticated connections.

## Affected Components and Files

| Service | Config Location | Issue |
|---------|-----------------|-------|
| **PostgreSQL** | `postgresql.customConfig` in `values.yaml` | `ssl = off` hardcoded; `args` to load `/etc/postgresql/postgresql.conf` are missing; the StatefulSet never passes `-c config_file=...` so the custom config is ignored entirely; TLS secret mount is commented out. |
| **Redis** | `redis.args` in `values.yaml` | TLS port (`--tls-port 6479`) and TLS cert/key/ca args are commented out; only plaintext `--port 6379` is active; TLS secret mount is commented out. |
| **NATS** | `nats.config.nats.tls` in `values.yaml` | TLS block (`enabled: false`); `secretName`, `cert`, and `key` fields are commented out; token authentication only (no TLS layer). |
| **Qdrant** | `qdrant.config.service` in `values.yaml` | `enable_tls: false`; TLS cert/key/ca_cert entries and TLS secret mount are commented out. |
| **Keycloak** | `keycloak.args` and `keycloak.extraVolumes` in `values.yaml` | HTTPS cert/key args (`--https-certificate-file`, `--https-certificate-key-file`) are commented out; `--http-enabled=true` forces plaintext; Keycloak TLS volume/mount is commented out; Keycloak also connects to PostgreSQL without `sslmode=require`. |
| **MinIO** | `minio.tls` in `values.yaml` | `enabled: false`; `certSecret`, `publicCrt`, and `privateKey` fields are commented out. |
| **`tls.generate` flag** | Top-level `tls.generate` in `values.yaml` | Defaults to `false`, so the TLS certificate-generation pre-install Job never runs, meaning no secrets are ever created for any of the above. |

## Impact

1. **Security Violation**: All inter-service traffic is unencrypted. Data in transit (credentials, tokens, agent memory, vectors) is exposed in plaintext.
2. **Broken Deployments in Secure Environments**: Any client (Keeper, Agent) configured with `sslmode=require`, `rediss://`, or NATS TLS will be immediately rejected.
3. **Readiness Probe Failures**: Keeper pods fail `/readyz` because their dependency checks connect with TLS expectations that the servers do not satisfy, causing `helm install` rollouts to time out.
4. **No Authentication on Redis, Qdrant, NATS**: While Redis has a password arg, Qdrant and NATS (beyond the token) currently operate without TLS-backed identity verification.
5. **Cascading Keycloak Failure**: Keycloak connects to PostgreSQL over plaintext, making the entire OIDC trust chain insecure.

## Expected Behaviour

### PostgreSQL
1. StatefulSet container args must include `-c config_file=/etc/postgresql/postgresql.conf` so the custom config map is actually loaded.
2. `customConfig` must include a `include` directive loading `/var/lib/postgresql/data/pg/postgresql.conf` first, then override with `ssl = on`, `ssl_cert_file`, `ssl_key_file`, and `ssl_ca_file` pointing to `/tls/`.
3. TLS secret (`${RELEASE_NAME}-pg-tls`) must be mounted at `/tls` in the PostgreSQL container.

### Redis
1. Redis args must activate the TLS listener: `--tls-port 6479`, `--port 0` (disable plaintext), and `--tls-cert-file`, `--tls-key-file`, `--tls-ca-cert-file` pointing to `/tls/`.
2. TLS secret (`${RELEASE_NAME}-redis-tls`) must be mounted at `/tls`.

### NATS
1. NATS TLS must be enabled (`tls.enabled: true`) with `secretName`, `cert`, and `key` referencing the generated `${RELEASE_NAME}-nats-tls` secret.

### Qdrant
1. `service.enable_tls` must be set to `true`; `tls.cert`, `tls.key`, and `tls.ca_cert` must point to `/tls/`.
2. TLS secret (`${RELEASE_NAME}-qdrant-tls`) must be mounted at `/tls`.

### Keycloak
1. HTTPS bindings must be enabled: `--https-certificate-file=/tls/tls.crt` and `--https-certificate-key-file=/tls/tls.key` passed as startup args; `--http-enabled=true` removed or replaced with `--http-enabled=false`.
2. Keycloak TLS secret (`${RELEASE_NAME}-keycloak-tls`) must be mounted at `/tls`.
3. Keycloak's database connection must use `sslmode=require` (via appropriate Keycloak JDBC properties or Quarkus config).

### MinIO
1. `minio.tls.enabled` must be set to `true` with `certSecret: ${RELEASE_NAME}-minio-tls`, `publicCrt: tls.crt`, and `privateKey: tls.key`.

### TLS Generation
1. `tls.generate` must be set to `true` (or at minimum, the README/dev-values must document and default to `true`), so the pre-install Job runs and populates all required `*-tls` secrets before service pods start.

## Acceptance Criteria

1. `tls.generate: true` causes the pre-install Job to create all six per-service TLS secrets and the CA bundle ConfigMap without error.
2. Running `psql "sslmode=require host=ts-infra-postgresql ..."` inside the cluster connects successfully; `SHOW ssl;` returns `on`.
3. `redis-cli --tls --cacert /path/to/ca.crt -h ts-infra-redis -p 6479 PING` returns `PONG`.
4. A NATS client connecting with TLS authenticates and publishes a message without error.
5. Qdrant health endpoint is reachable only via HTTPS (`https://ts-infra-qdrant:6333/healthz`).
6. Keycloak's admin console and OIDC discovery document are served exclusively over HTTPS.
7. MinIO S3 API is reachable only via `https://ts-infra-minio:9000`.
8. Migration jobs and Keeper components succeed on all dependency connections.
9. `make helm-infra-lint` passes with the fully secured `values.yaml`.
