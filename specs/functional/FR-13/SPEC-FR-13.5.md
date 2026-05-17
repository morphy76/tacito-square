# SPEC-FR-13.5: Keycloak Realm Provisioning via Helm

| Field         | Value                              |
|---------------|------------------------------------|
| ID            | SPEC-FR-13.5                       |
| Status        | VERIFIED                           |
| Milestone     | M1                                 |
| FR/NFR Ref    | FR-13.5                            |
| Component     | deploy                             |
| Depends On    | SPEC-FR-12.6                       |

## Context

The dev environment must ship with a fully configured Keycloak realm so that OIDC authentication works out of the box on `helm install`.

## Specification

1. The Helm chart MUST deploy Keycloak in dev mode (`production: false`).
2. Keycloak Config CLI MUST create the `tacito` realm with:
   - 4 RBAC roles: `keeper-admin`, `keeper-viewer`, `user`, `agent-spawner`
   - 2 OIDC clients:
     - `tacito-keeper` (confidential, service account, secret: `keeper-dev-secret`)
     - `tacito-ui` (public, PKCE, SPA-friendly)
   - 2 dev users:
     - `admin` / `admin` (roles: `keeper-admin`, `agent-spawner`)
     - `user` / `user` (roles: `user`)
3. SSL requirement MUST be `none` for dev.
4. Redirect URIs and web origins MUST be wildcard (`*`) for dev.
5. For production, disable Keycloak and point to an external OIDC provider via env vars.

## Acceptance Criteria

1. `helm install` creates Keycloak with `tacito` realm
2. Both OIDC clients are available after install
3. Dev users can authenticate and receive tokens with correct roles
4. Roles are present in JWT `realm_access.roles` claim
5. `keycloak.enabled: false` disables the sub-chart entirely

## Files

- `deploy/helm/tacito-square/values.yaml` ✅ (keycloak section, lines 134–198)
- `deploy/helm/tacito-square/Chart.yaml` ✅ (keycloak dependency)
