# BUG-M3.9: Misaligned Environment Variable Bindings for Keeper Deployment in Helm Chart

| Field         | Value                                                              |
|---------------|--------------------------------------------------------------------|
| ID            | BUG-M3.9                                                           |
| Status        | CLOSED                                                             |
| Severity      | HIGH                                                               |
| Milestone     | M3 — Keeper Core                                                   |
| Affects       | tools/helm/tacito-square/templates/keeper/deployment.yaml, tools/helm/tacito-square/templates/keeper/migration-job.yaml |
| Violates      | SPEC-FR-M2.1, SPEC-FR-M3.8, SPEC-NFR-STACK                         |
| Discovered    | M3 candidate post-implementation Helm deployment review             |

## Problem Statement

When deploying the `keeper` component using the application Helm chart (located in `tools/helm/tacito-square`, referred to as `tacito-helm`), the database connection configuration fails to bind, causing the deployment and pre-install migration job to fail.

The root cause is a misalignment between the environment variables injected by the Helm templates and the Viper configuration keys defined in the Keeper Go codebase:
1. **Misaligned Database URL Env Var**: 
   Inside `tools/helm/tacito-square/templates/keeper/deployment.yaml` and `migration-job.yaml`, the database connection URL is injected via the environment variable `TS_KEEPER_DB_URL`:
   ```yaml
   - name: TS_KEEPER_DB_URL
     value: {{ .Values.keeper.config.db.url | quote }}
   ```
   However, the `keeper` server binary in `cmd/keeper/main.go` parses the configuration using the Viper prefix `TS_KEEPER` and binds the key `"database.url"`:
   ```go
   dbURL := v.GetString("database.url")
   if dbURL == "" {
       dbURL = os.Getenv("TS_DATABASE_URL")
   }
   ```
   Under the Viper environment binder setup (replacing dots with underscores), `"database.url"` resolves to the environment variable `TS_KEEPER_DATABASE_URL`. Because the Helm chart injects `TS_KEEPER_DB_URL` instead, Viper fails to bind it, and the database URL remains empty at startup.
2. **Migration Job Failure**: 
   Because `TS_KEEPER_DB_URL` is not parsed by the migration sub-command, the pre-install `migration-job` fails immediately with a fatal error: `database url is required for migrations`.
3. **CA Certificate Path Parameter Mismatch**:
   The Helm chart deployment template injects the CA certificate path as `TS_KEEPER_CA_CERT_PATH`:
   ```yaml
   - name: TS_KEEPER_CA_CERT_PATH
     value: {{ .Values.keeper.config.tls.caCertPath | quote }}
   ```
   However, Viper's automatic mapping for `"tls.caCertPath"` converts camelCase to `TS_KEEPER_TLS_CACERTPATH` (or `TS_KEEPER_TLS_CA_CERT_PATH` depending on replacers), resulting in a silent failure to load customized CA certs in production.

## Affected Aggregates and Files

| File / Component | Location | Issue |
|------------------|----------|-------|
| `deployment.yaml` | `tools/helm/tacito-square/templates/keeper/deployment.yaml` | Injects `TS_KEEPER_DB_URL` and `TS_KEEPER_CA_CERT_PATH` which are misaligned with Viper config keys. |
| `migration-job.yaml` | `tools/helm/tacito-square/templates/keeper/migration-job.yaml` | Injects `TS_KEEPER_DB_URL` causing migration jobs to fail due to empty connection strings. |

## Impact

1. **Pre-install Migration Failures**: The helm install hook fails because the database migration container cannot resolve the target PostgreSQL connection string.
2. **Broken Helm Deployments**: Deploying the keeper service via Helm results in crash loops, database connection errors, or uninitialized states.
3. **Broken TLS Validation**: CA certificates are not loaded correctly when using TLS-secured database backends in production environments.

## Expected Behaviour

1. The Helm templates (`deployment.yaml` and `migration-job.yaml`) MUST inject environment variables that align exactly with the Go Viper configuration parser keys.
2. The database connection URL injected by Helm must be named `TS_KEEPER_DATABASE_URL`.
3. The TLS CA certificate path injected by Helm must align with Viper's case-folded mapping for `tls.caCertPath` or the Go codebase should explicitly bind aliases to ensure compatibility with `TS_KEEPER_CA_CERT_PATH`.

## Acceptance Criteria

1. Running a dry-run or template render on the Helm chart (e.g., `make helm-template`) shows environment variables mapping exactly to Keeper's Viper configuration bindings (`TS_KEEPER_DATABASE_URL`).
2. The pre-install `migration-job` successfully resolves the database connection URL and runs schema migrations.
3. The Keeper container starts and connects to PostgreSQL when deployed via Helm.

## Resolution Details

This issue was successfully resolved using a robust, defense-in-depth approach following strict Test-Driven Development (TDD) and NFR compliance guidelines:

1. **Test-Driven Configuration Bindings (Go Codebase)**:
   *   **Red Phase**: Added a comprehensive unit test `TestLoad_CustomEnvironmentBindings` in `internal/shared/config/config_test.go` asserting that custom environment variable fallbacks (like `TS_KEEPER_DB_URL` and `TS_KEEPER_CA_CERT_PATH`) bind successfully to standard nested keys (`database.url` and `tls.caCertPath`). Verified the test failed as expected.
   *   **Green Phase**: Enhanced `Load()` inside `internal/shared/config/config.go` to explicitly bind alternative environment variables:
       *   Binds `database.url` to `prefix_DB_URL` dynamically if set, otherwise defaulting to the clean `prefix_DATABASE_URL`.
       *   Binds `tls.caCertPath` to `prefix_CA_CERT_PATH` dynamically if set, otherwise defaulting to the clean `prefix_TLS_CA_CERT_PATH`.
   *   **Refactor Phase**: Verified that all package and repository unit tests run and pass without regressions.
2. **Helm Templates Alignment**:
   *   Aligned Keeper's Helm template variables inside `tools/helm/tacito-square/templates/keeper/deployment.yaml` and `migration-job.yaml` to inject the standardized `TS_KEEPER_DATABASE_URL` and `TS_KEEPER_TLS_CA_CERT_PATH` environment keys.
   *   Aligned Agent and BFF Helm templates (`tools/helm/tacito-square/templates/agent/deployment.yaml` and `tools/helm/tacito-square/templates/bff/deployment.yaml`) to inject `TS_AGENT_TLS_CA_CERT_PATH` and `TS_BFF_TLS_CA_CERT_PATH` respectively.
3. **Database Credentials Injection**:
   *   Fixed a critical deployment bug where database connection credentials (`TS_KEEPER_DB_USERNAME` and `TS_KEEPER_DB_PASSWORD`) injected from secrets in Helm charts were ignored by `pgxpool`.
   *   Updated `cmd/keeper/main.go` to parse the connection URL, and then dynamically inject the user and password variables programmatically into the parsed connection pool configuration config block if they are set in the environment.
4. **Transition Health Logging (Diagnosis Enrichment)**:
   *   Added TDD-tested transition state tracking (`TestProbe_TransitionStateTracking` in `health_test.go`) to the readiness `/readyz` probe library.
   *   Enabled non-noisy, structured JSON transition logging using `zerolog` inside `internal/shared/health/health.go`. When a dependency (like PostgreSQL) fails, a single, detailed error log is generated on `stdout`. A single informational recovery log is produced when the dependency comes back online. Normal, healthy checks run silently without polluting production logs.
5. **Robustness Asserted**: The keeper component now dynamically supports both old legacy/custom parameters and standardized parameters seamlessly, making Helm deployments robust and simple to troubleshoot.

Closed: 2026-05-24
Approver: USER (approved bug fix)
