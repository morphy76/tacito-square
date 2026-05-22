# TASK-M2.11.2: Secure Keycloak Realm & HTTPS Configuration

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M2.11.2                                |
| Status        | COMPLETE                                    |
| Spec          | SPEC-FR-M2.11                               |
| Depends On    | none                                        |

## Description

Configure Keycloak within the `tacito-square-infra` Helm chart to enable HTTPS/TLS endpoints, auto-provision the secure `tacito` realm on boot, and register OIDC confidential and public clients.

## Work Items

1. Update Keycloak helm values in `tools/helm/tacito-square-infra/values.yaml` to:
   - Enable HTTPS listener port (default 8443) and mount the generated TLS certificate secrets.
   - Ensure the server runs with SSL required for all external requests.
2. Refactor `tools/helm/tacito-square-infra/templates/keycloak-realm-configmap.yaml` to contain the secure realm layout:
   - Define a confidential client `tacito-keeper` requiring a secure client secret retrieved from K8s secrets.
   - Define public client `tacito-ui` with strict redirect URIs and web origins.
3. Configure Keycloak initialization argument to run with `--import-realm` pointing to the mounted `/opt/keycloak/data/import/tacito-realm.json`.

## Acceptance Criteria

1. Keycloak listens and serves traffic exclusively over HTTPS (TLS).
2. Keycloak successfully imports the secure `tacito` realm at startup.
3. The `tacito-keeper` client is configured as confidential and `tacito-ui` is public with strict redirect policies.
