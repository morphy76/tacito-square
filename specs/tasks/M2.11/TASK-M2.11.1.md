# TASK-M2.11.1: Secure PostgreSQL & DB Provisioning

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M2.11.1                                |
| Status        | COMPLETE                                    |
| Spec          | SPEC-FR-M2.11                               |
| Depends On    | none                                        |

## Description

Configure PostgreSQL within the `tacito-square-infra` Helm chart to automatically provision the `tacito` database, schema, and user role with restricted permissions, and enforce TLS/SSL-only connections.

## Work Items

1. Update PostgreSQL helm values in `tools/helm/tacito-square-infra/values.yaml` to specify:
   - Database name: `tacito`
   - Master/Superuser configuration with secure credentials.
   - User database user: `tacito` with random/secure password values loaded via K8s Secret bindings.
2. Enable SSL/TLS listener configurations in PostgreSQL sub-chart or custom values, requiring client connections to specify `sslmode=require` or `ssl=true`.
3. Provide a test case in Gin or application code where connection validation fails if SSL/TLS is disabled, and passes when SSL/TLS is configured.

## Acceptance Criteria

1. PostgreSQL boots and only accepts secure SSL/TLS client connections.
2. Database `tacito`, schema `tacito`, and restricted user `tacito` are automatically provisioned at install/upgrade time.
3. Plaintext connections without TLS are strictly rejected by the database.
