# TASK-M3.8.3: Helm Application Chart DB Secret & Connection Environment Bindings

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M3.8.3                                 |
| Status        | IMPLEMENTED                                 |
| Spec          | SPEC-FR-M3.8                                |
| Depends On    | none                                        |

## Description

Design and implement the Helm templates to bind the database persistent configuration and connection secrets. The charts must expose database URLs and passwords securely using Kubernetes Opaque Secrets, injecting them safely into the Keeper Deployment pods using container environment variables, and enforcing TLS configuration.

## Boundary & Target Templates

- **Chart Path**: `tools/helm/tacito-square/`
- **Target Templates**:
  - `templates/keeper/secret-db.yaml` (Database secret credentials allocation)
  - `templates/keeper/deployment.yaml` (Keeper Pod environment configuration)
  - `values.yaml` (Sensible defaults for connection URL and passwords)

## Work Items

1. **RED Phase**:
   - Create rendering and dry-run validation scripts in the workspace test harness (e.g. validating YAML properties).
   - Verify that running `helm template` lint checks fail when required database secret parameters are completely omitted or structurally broken.

2. **GREEN Phase**:
   - Implement `templates/keeper/secret-db.yaml` to define a Kubernetes `Secret` mapping `.Values.keeper.config.db.password` securely with base64 encoding.
   - Edit `templates/keeper/deployment.yaml` to configure environment variables `TS_KEEPER_DB_URL` mapping from values and `TS_KEEPER_DB_PASSWORD` mapping from the password key in the secret.
   - Update `values.yaml` with standard secure defaults, appending `sslmode=require` parameters to database connection URLs to assert TLS enforcement.

3. **REFACTOR Phase**:
   - Refactor Helm templates to leverage standard helpers (`include "tacito-square.fullname" .`) to ensure clean namespace and resource naming consistency across components.

## Acceptance Criteria

1. Running `helm lint tools/helm/tacito-square/` finishes with zero warnings or errors.
2. Running `helm template` correctly renders the Secret and Deployment manifests.
3. Database passwords are fully obfuscated inside standard secret resource definitions rather than hardcoded in Deployment specs.
