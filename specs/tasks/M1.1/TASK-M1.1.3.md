# TASK-M1.1.3: Configure Keycloak realm (GREEN)

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M1.1.3                                 |
| Status        | TODO                                        |
| Spec          | SPEC-FR-M1.1                                |
| Phase         | GREEN                                       |
| Depends On    | TASK-M1.1.2                                 |

## Description

Move the Keycloak realm configuration from the application chart to the infrastructure chart. The `tacito` realm must include pre-configured clients and roles for dev use.

## Work Items

1. Move the Keycloak realm JSON configuration from `tools/helm/tacito-square/values.yaml` to `tools/helm/tacito-square-infra/values.yaml` under the `keycloak` key.
2. Ensure the realm includes:
   - Realm: `tacito` (enabled, sslRequired: none for dev)
   - Clients: `tacito-keeper` (confidential, service account), `tacito-ui` (public, PKCE)
   - Roles: `keeper-admin`, `keeper-viewer`, `user`, `agent-spawner`
   - Dev users: `admin` (keeper-admin, agent-spawner), `user` (user)
3. Verify `helm template` renders the Keycloak config with the realm JSON.

## Acceptance Criteria

1. Keycloak realm configuration is present in the infra chart values.
2. `helm template` output includes the realm JSON with all clients, roles, and users.
3. Application chart no longer contains Keycloak realm configuration.
