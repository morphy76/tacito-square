# TASK-M3.8.4: Helm Pre-Deployment Database Migration Hook Job

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M3.8.4                                 |
| Status        | IMPLEMENTED                                 |
| Spec          | SPEC-FR-M3.8                                |
| Depends On    | TASK-M3.8.3                                 |

## Description

Design and implement a dedicated database migration Job inside the Keeper Helm chart templates using Helm hooks. The Job must be executed as a pre-install/pre-upgrade phase to run database migrations successfully using the Keeper container image before any application Web Server Deployment pods start up, preventing schema inconsistency failures.

## Boundary & Target Templates

- **Chart Path**: `tools/helm/tacito-square/`
- **Target Templates**:
  - `templates/keeper/migration-job.yaml` (Pre-deployment hook Job specification)

## Work Items

1. **RED Phase**:
   - Write rendering unit test scripts to assert that the rendered Job manifest contains correct Helm hook annotations and executes before the standard keeper deployment.
   - Assert that if the database is down or unconfigured, the job will fail rendering/validation.

2. **GREEN Phase**:
   - Create `templates/keeper/migration-job.yaml` specifying a Kubernetes `Job` resource.
   - Attach necessary Helm Hook annotations:
     - `"helm.sh/hook": pre-install,pre-upgrade`
     - `"helm.sh/hook-weight": "5"`
     - `"helm.sh/hook-delete-policy": before-hook-creation,hook-succeeded`
   - Bind the Job's restart policy to `OnFailure` and map database connection environment variables securely from the password Secret.
   - Direct the Job's container command to execute the migration logic built into the image (e.g. `/app/keeper migrate up` or similar commands running goose).

3. **REFACTOR Phase**:
   - Verify label parity across the migration job and standard keeper templates using Gopher-style clean conventions.
   - Optimize container resource limits for the migration Job to prevent OOM termination issues in restricted clusters.

## Acceptance Criteria

1. Rendered Helm templates successfully include the pre-deployment database migration Job.
2. The Job cleanly carries valid hook and deletion policy annotations.
3. Running `helm lint` returns 0 issues or template warnings.
