# TASK-M2.11.3: Secure Object Storage, Cache, and Messaging

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M2.11.3                                |
| Status        | COMPLETE                                    |
| Spec          | SPEC-FR-M2.11                               |
| Depends On    | TASK-M2.11.1, TASK-M2.11.2                  |

## Description

Configure MinIO object storage, Redis short-term memory, NATS message broker, and certificates to require strong authentication and secure TLS transport.

## Work Items

1. Update MinIO helm values in `tools/helm/tacito-square-infra/values.yaml` to:
   - Auto-create the `tacito` bucket with secure private permissions.
   - Mount certificate secrets to enable secure HTTPS bindings on the S3 API endpoint.
2. Update Redis helm values in `tools/helm/tacito-square-infra/values.yaml` to:
   - Enable strong password authentication (managed via K8s Secret references).
   - Enable TLS support to encrypt all cached short-term memory transitions.
3. Configure NATS helm values in `tools/helm/tacito-square-infra/values.yaml` to:
   - Enforce client token/credentials authentication.
   - Enforce TLS-protected socket connections for all publisher/subscriber workloads.
4. Implement a certificate helper mechanism (e.g. self-signed generator or cert secrets deployment template) within `tacito-square-infra/templates/` to issue TLS keys for local testing.
5. Re-align application chart binding values in `tools/helm/tacito-square/values.yaml` to connect to:
   - Postgres via `sslmode=require`
   - Keycloak via `https://`
   - MinIO via `https://`
   - Redis via `rediss://`
   - NATS via `nats://` with TLS enabled

## Acceptance Criteria

1. MinIO auto-creates the private `tacito` bucket, exposed only via HTTPS.
2. Redis and NATS reject unauthenticated connections and require TLS.
3. Application templates render secure endpoints (`https`, `rediss`, `sslmode=require`).
4. The entire infrastructure chart passes `make helm-infra-lint` cleanly.
