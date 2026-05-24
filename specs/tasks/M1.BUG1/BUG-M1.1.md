# BUG-M1.1: PostgreSQL Server Does Not Support or Accept SSL Connections

| Field         | Value                                                              |
|---------------|--------------------------------------------------------------------|
| ID            | BUG-M1.1                                                           |
| Status        | OPEN                                                               |
| Severity      | HIGH                                                               |
| Milestone     | M1 — Infrastructure Helm Chart                                      |
| Affects       | tools/helm/tacito-square-infra/values.yaml                        |
| Violates      | SPEC-FR-M2.11, SPEC-NFR-STACK                                     |
| Discovered    | Keeper health-check and migration failures in Helm deployments      |

## Problem Statement

The infrastructure's PostgreSQL service (`ts-infra-postgresql`) is running with SSL/TLS disabled (`ssl = off`).
Although the `tacito-square-infra` chart mounts the TLS certificate secret `ts-infra-tacito-square-infra-pg-tls` at `/tls` and provides a custom configuration block containing `ssl = on` inside `/etc/postgresql/postgresql.conf`, the PostgreSQL container ignores this custom config.

The root cause is that the PostgreSQL StatefulSet does not pass the `-c config_file=/etc/postgresql/postgresql.conf` parameter as an entrypoint argument. By default, the official PostgreSQL Docker image ignores the `/etc/postgresql` directory and loads `postgresql.conf` from the data directory (`/var/lib/postgresql/data/pg/postgresql.conf`), resulting in SSL/TLS remaining disabled.

## Affected Components and Files

| Component / File | Location | Issue |
|------------------|----------|-------|
| `values.yaml` (infra) | `tools/helm/tacito-square-infra/values.yaml` | Lacks start `args` instructing PostgreSQL to load `/etc/postgresql/postgresql.conf`, and `customConfig` lacks default configuration inclusions. |

## Impact

1. **Security Violation**: Database connections cannot be encrypted via SSL/TLS.
2. **Broken Deployments**: Applications attempting to connect with `sslmode=require` are rejected immediately with `tls error: server refused TLS connection` errors.
3. **Readiness Probe Failures**: Keeper pods fail readiness probes `/readyz` due to unhealthy database dependencies, causing helm installations to fail.

## Expected Behaviour

1. The PostgreSQL server MUST start with SSL/TLS activated (`ssl = on`).
2. The PostgreSQL StatefulSet container MUST be supplied with arguments (`-c config_file=/etc/postgresql/postgresql.conf`) to load the custom configuration map.
3. The custom configuration block (`customConfig`) MUST use the `include` directive to load the default data-directory configuration (`/var/lib/postgresql/data/pg/postgresql.conf`) before setting custom SSL overrides to ensure standard database parameters are preserved.

## Acceptance Criteria

1. Running `psql -U postgres -c "SHOW ssl;"` inside the running PostgreSQL container returns `on`.
2. Database connections with `sslmode=require` connect successfully.
3. Migration jobs and Keeper components succeed on database connection.
