# SPEC-FR-M1.2: Makefile Infrastructure Targets

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | SPEC-FR-M1.2                                |
| Status        | DRAFT                                       |
| Milestone     | M1                                          |
| Component     | build                                       |
| Depends On    | SPEC-FR-M1.1                                |
| Supersedes    | none                                        |

## Context

The root Makefile must provide targets to manage the infrastructure Helm chart lifecycle (dependency update, lint, template, install, uninstall). This enables developers to bootstrap a local development environment with a single command.

## Specification

1. The `Makefile` MUST include the following infrastructure targets:
   - `helm-infra-deps`: Run `helm dependency update` on the infra chart.
   - `helm-infra-lint`: Run `helm lint` on the infra chart.
   - `helm-infra-template`: Render infra chart templates locally.
   - `helm-infra-install`: Install/upgrade the infra Helm release on the current cluster.
   - `helm-infra-uninstall`: Uninstall the infra Helm release.
2. The `HELM_INFRA_CHART` variable MUST default to `tools/helm/tacito-square-infra`.
3. The `HELM_INFRA_RELEASE` variable MUST default to `tacito-infra`.
4. All infra targets MUST be declared `.PHONY`.
5. The `help` target MUST include infra targets with documentation.

## Acceptance Criteria

1. `make helm-infra-lint` passes without errors.
2. `make helm-infra-template` renders valid YAML to stdout.
3. `make helm-infra-install` deploys all infrastructure services on minikube.
4. `make helm-infra-uninstall` removes the infrastructure release cleanly.
5. `make help` lists all infra targets with descriptions.

## Test Plan

1. Run `make helm-infra-lint` — exit code 0.
2. Run `make helm-infra-template | kubectl apply --dry-run=client -f -` — validates rendered manifests.
3. Run `make helm-infra-install` followed by `kubectl get pods` — all infra pods Running.
4. Run `make helm-infra-uninstall` followed by `kubectl get pods` — no infra pods remain.

## Files Affected

- `Makefile` (MODIFY — add infra targets)
