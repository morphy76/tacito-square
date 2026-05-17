# SPEC-FR-12.3: Independent Component Versioning

| Field         | Value                              |
|---------------|------------------------------------|
| ID            | SPEC-FR-12.3                       |
| Status        | VERIFIED                           |
| Milestone     | M1                                 |
| FR/NFR Ref    | FR-12.3                            |
| Component     | all                                |
| Depends On    | —                                  |

## Context

Components (agent, keeper, operator, bff) must evolve independently with their own version lifecycle, enabling contract-based evolution.

## Specification

1. Each component MUST have its own version file: `VERSION.agent`, `VERSION.keeper`, `VERSION.operator`, `VERSION.bff`.
2. The Makefile MUST read version from the component's VERSION file.
3. Docker image tags MUST use the component's version: `{registry}/{name}:{version}`.
4. Helm values MUST support per-component image tag overrides.
5. Version bumps for one component MUST NOT require version bumps for others.

## Acceptance Criteria

1. `VERSION.agent` exists and is read by Makefile
2. `make docker-build-agent` uses `$(AGENT_VERSION)` from file
3. Helm `agent.image.tag` overrides independently
4. Changing keeper version does not affect agent version

## Files

- `VERSION.agent`, `VERSION.keeper`, `VERSION.operator`, `VERSION.bff` ✅
- `Makefile` ✅ (lines 4–7: version variables)
- `deploy/helm/tacito-square/values.yaml` ✅ (per-component `image.tag`)
