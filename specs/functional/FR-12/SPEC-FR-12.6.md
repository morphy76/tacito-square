# SPEC-FR-12.6: Helm Umbrella Chart with Sub-Charts

| Field         | Value                              |
|---------------|------------------------------------|
| ID            | SPEC-FR-12.6                       |
| Status        | VERIFIED                           |
| Milestone     | M1                                 |
| FR/NFR Ref    | FR-12.6                            |
| Component     | deploy                             |
| Depends On    | —                                  |

## Context

The entire Tacito Square platform is installable via a single `helm install` command using an umbrella chart that bundles all infrastructure dependencies.

## Specification

1. `deploy/helm/tacito-square/Chart.yaml` MUST define an umbrella chart.
2. Infrastructure dependencies MUST be declared as sub-chart dependencies with `condition` toggles:
   - NATS (`nats.enabled`)
   - Redis (`redis.enabled`)
   - PostgreSQL (`postgresql.enabled`)
   - Qdrant (`qdrant.enabled`)
   - OpenTelemetry Collector (`otel-collector.enabled`)
   - Keycloak (`keycloak.enabled`)
   - MinIO (`minio.enabled`, opt-in, default: false)
3. Each dependency MUST be individually disable-able for production externalization.
4. `helm install tacito-square deploy/helm/tacito-square` MUST deploy a functional platform.
5. Image resolution: `{component.image.registry || global.imageRegistry}/{component.image.name}:{component.image.tag}`.

## Acceptance Criteria

1. `helm lint` passes (excluding missing deps warning)
2. All sub-charts have `condition` toggles
3. Disabling all infra charts still renders component templates
4. Image resolution falls back to `global.imageRegistry`

## Files

- `deploy/helm/tacito-square/Chart.yaml` ✅ (8 dependencies)
- `deploy/helm/tacito-square/values.yaml` ✅ (per-component + infra config)
- `deploy/helm/tacito-square/templates/_helpers.tpl` ✅ (image resolution)
- `deploy/helm/tacito-square/README.md` ✅ (documentation)
