# TASK-M4.BUG1.1: Operator Reconciler Implementation

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M4.BUG1.1                              |
| Status        | VERIFIED                                    |
| Spec          | BUG-M4.1                                    |
| Depends On    | none                                        |

## Description

Implement the active `Reconcile` method in `internal/operator/application/service/reconcile_service.go` to construct the backing Deployment and headless Service configurations using `BuildDeployment` and `BuildHeadlessService`, apply them to the Kubernetes cluster using the controller-runtime `client.Client`, set correct `OwnerReferences` for cascade deletion, and update the Custom Resource's `status.phase` and conditions.

## Work Items

1. **RED Phase**:
   - [x] Add unit tests in `internal/operator/application/service/reconcile_service_test.go` (or `reconciler_test.go` if applicable) to assert that a call to `Reconcile` correctly invokes Kubernetes Client `Create` or `Update` / `Get` operations on Deployment and Service.
   - [x] Confirm that the new tests fail (RED) when executing `make test` against the current stub implementation.

2. **GREEN Phase**:
   - [x] Implement the `Reconcile` method in `internal/operator/application/service/reconcile_service.go`:
     - [x] Retrieve/get the parent `TacitoAgent` resource.
     - [x] Call `BuildDeployment` and `BuildHeadlessService` to generate child manifests.
     - [x] Apply/Create the Deployment and headless Service using the `client.Client` API (upsert logic utilizing standard Kubernetes update/create or Server-Side Apply / Get-then-Update patterns).
     - [x] Update the status phase of the parent `TacitoAgent` Custom Resource to `Running` (or `Pending` if not ready, using Deployment replica readiness checks).
   - [x] Ensure the tests compile and go GREEN.

3. **REFACTOR Phase**:
   - [x] Ensure clean Hexagonal architecture decoupling.
   - [x] Optimize client operations and error handling.

## Acceptance Criteria

1. Reconciling a `TacitoAgent` CRD applies the backing Deployment and headless Service resources to Kubernetes.
2. The agent container is scheduled with all required multi-tenancy and connection parameters.
3. Unit and integration tests for the reconciler pass successfully.
