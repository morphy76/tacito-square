# SPEC-FR-M2.11: Secured Infrastructure Provisioning (Initial Provisioning & TLS Enforcement)

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | SPEC-FR-M2.11                               |
| Status        | IMPLEMENTED                                 |
| Milestone     | M2                                          |
| Component     | deploy                                      |
| Depends On    | SPEC-FR-M2.10                               |
| Supersedes    | none                                        |

## Context

To transition Tacito Square from a development-centric, unauthenticated environment to a secure-by-default, production-ready system, all infrastructural services deployed by the `tacito-square-infra` Helm chart must enforce secure communication, strong authentication, and confidential data storage. This requires initial provisioning of database schemas, secure object storage buckets, identity provider realms, and the enforcement of encrypted TLS-protected communication channels.

All bootstrapping, tenant isolation, and secure resource provisioning must execute automatically at infrastructure installation/upgrade time (`helm upgrade --install tacito-infra tools/helm/tacito-square-infra`).

## Specification

1. **PostgreSQL Bootstrapping & TLS Security**:
   - **Tenant Provisioning**: The chart must auto-create a dedicated database named `tacito`, a schema named `tacito`, and a custom role/user named `tacito` with restricted, non-superuser privileges.
   - **Confidentiality (TLS)**: PostgreSQL must be configured to require secure, encrypted client connections (`ssl: true` and `sslmode: require`). 
   - **Credentials**: Strong, randomly generated passwords (or pre-configured secure values retrieved from standard K8s secrets) must replace any static default credentials.

2. **Keycloak Realm & HTTPS Configuration**:
   - **Realm Auto-Import**: Auto-provision the `tacito` realm at boot using the Quarkus `--import-realm` feature or custom Helm provisioning jobs.
   - **Confidential Clients**: Pre-configure the realm with OIDC clients:
     - `tacito-keeper`: A confidential client requiring a secure client secret and client credentials grant.
     - `tacito-ui`: A public client configured with secure redirect URIs and strict web origins.
   - **HTTPS/TLS Enforcement**: Keycloak must serve all traffic over TLS/HTTPS (`kc.sh start` with HTTPS bindings enabled, utilizing a generated TLS secret). Wires Gin application endpoints to use secure OIDC issuers (`https://...`).

3. **MinIO Object Storage & HTTPS**:
   - **Bucket Provisioning**: Auto-create a secure, private bucket named `tacito` at installation time (using MinIO Helm's native `buckets` config or custom job hooks).
   - **TLS Encryption**: Enforce TLS on all S3 API interactions (`https://...`) by mounting certificate secrets into the MinIO server pods.

4. **Redis and NATS Security Hardening**:
   - **Redis**: Enable strong password authentication (with credentials managed via Kubernetes Secrets) and enable TLS listener ports (`rediss://`) to prevent plaintext exposure of cached short-term memory data.
   - **NATS**: Enable TLS-protected client connections (`nats://` with TLS enabled) and token/password authentication for internal message routing.

5. **Unified TLS Certificate Generation**:
   - Provision a unified, self-contained mechanism within the chart (e.g., K8s certificate hooks, self-signed issuer configurations, or optional `cert-manager` integration) to automatically generate TLS certificate keys and distribute them to:
     - The servers (Postgres, Keycloak, MinIO, Redis, NATS, Qdrant).
     - The application clients as trusted CA trust-stores.

6. **Install-Time Automation**:
   - All bootstrap operations (e.g. creating databases, importing realms, setting up S3 buckets) must run as part of the `helm upgrade --install` workflow, using chart-native initializers or pre-install/post-install Kubernetes Job hooks.

## Acceptance Criteria

1. Running `helm install` on the infrastructure chart successfully boots PostgreSQL, Keycloak, MinIO, Redis, and NATS with TLS listeners active.
2. All databases, schemas, object buckets, and Keycloak realms are pre-provisioned automatically without needing manual post-install shell configurations.
3. The application services (`tacito-square`) connect to all dependency services exclusively via encrypted protocols (`postgresql://...sslmode=require`, `https://...`, `rediss://...`, `nats://...` with TLS).
4. Statically defined, insecure default database users (like `keeper`) or plaintext credentials are eliminated.
5. Zero warnings or errors are raised when running `make helm-infra-lint` with the new security values enabled.

## Test Plan

1. **Lint & Validation**: Run `make helm-infra-lint` to ensure the security-hardened manifests pass all syntactic validations.
2. **Template Render Check**: Run `make helm-infra-template` and verify that TLS certificates are mounted, environment variables reference TLS paths, and secure endpoints are exposed.
3. **Connectivity Verification**: Run manual helm installs inside a local Kind cluster, and confirm services communicate strictly via SSL/TLS handshakes.

## Files Affected

- `tools/helm/tacito-square-infra/Chart.yaml` (MODIFY)
- `tools/helm/tacito-square-infra/values.yaml` (MODIFY)
- `tools/helm/tacito-square-infra/templates/keycloak-realm-configmap.yaml` (MODIFY)
- `tools/helm/tacito-square/values.yaml` (MODIFY)
