# TASK-M1.2.2: Add infrastructure Makefile targets (GREEN)

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M1.2.2                                 |
| Status        | TODO                                        |
| Spec          | SPEC-FR-M1.2                                |
| Phase         | GREEN                                       |
| Depends On    | TASK-M1.2.1, TASK-M1.1.2                   |

## Description

Add infrastructure Helm chart targets to the root Makefile. Implement the minimum to make the RED tests pass.

## Work Items

1. Add variables to `Makefile`:
   ```makefile
   HELM_INFRA_CHART   := tools/helm/tacito-square-infra
   HELM_INFRA_RELEASE ?= tacito-infra
   ```
2. Add infrastructure targets:
   - `helm-infra-deps`: `helm dependency update $(HELM_INFRA_CHART)`
   - `helm-infra-lint`: `helm lint $(HELM_INFRA_CHART)`
   - `helm-infra-template`: `helm template $(HELM_INFRA_RELEASE) $(HELM_INFRA_CHART)`
   - `helm-infra-install`: `helm upgrade --install $(HELM_INFRA_RELEASE) $(HELM_INFRA_CHART) --wait`
   - `helm-infra-uninstall`: `helm uninstall $(HELM_INFRA_RELEASE)`
3. Add all targets to `.PHONY` declaration.
4. Add `## ` comments for the `help` target auto-documentation.
5. Run `test/make/test_infra_targets.sh` — all checks MUST pass.

## Acceptance Criteria

1. `make help` lists all 5 infrastructure targets with descriptions.
2. `make -n helm-infra-lint` succeeds (dry run validates target existence).
3. `make helm-infra-lint` passes (requires TASK-M1.1.2 chart to exist).
4. `make helm-infra-template` renders valid YAML to stdout.
