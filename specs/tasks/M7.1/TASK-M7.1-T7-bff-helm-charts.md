# TASK-M7.1-T7: BFF Helm Deployment Charts (`deploy/helm/tacito-square/templates/bff/`)

| Field       | Value                                                      |
|-------------|------------------------------------------------------------|
| Task ID     | TASK-M7.1-T7                                               |
| Spec        | SPEC-FR-M7.1                                               |
| Boundary    | Deploy — `deploy/helm/tacito-square/templates/bff/`        |
| Status      | TODO                                                       |
| Depends On  | TASK-M7.1-T6                                               |

## Objective

Create the Kubernetes Helm templates for the BFF component: a `Deployment`, a `Service`, a `ConfigMap` for Viper configuration, a `Secret` template for OIDC credentials and the Redis session key prefix, and an `HPA` configured for HTTP-request-rate-based autoscaling. Update the parent `Chart.yaml` and `values.yaml` accordingly.

## Files

| File | Action |
|------|--------|
| `deploy/helm/tacito-square/templates/bff/deployment.yaml` | NEW |
| `deploy/helm/tacito-square/templates/bff/service.yaml` | NEW |
| `deploy/helm/tacito-square/templates/bff/configmap.yaml` | NEW |
| `deploy/helm/tacito-square/templates/bff/secret.yaml` | NEW |
| `deploy/helm/tacito-square/templates/bff/hpa.yaml` | NEW |
| `deploy/helm/tacito-square/values.yaml` | MODIFY |
| `deploy/helm/tacito-square/Chart.yaml` | MODIFY (no version bump — add `bff` to dependency metadata only) |

## RED Phase

Validate Helm templates render correctly using `helm template` dry-run assertions:

- Run `make helm-template` and assert the output contains:
  - A `Deployment` named `{{ .Release.Name }}-bff` using the distroless base image `gcr.io/distroless/base-nossl-debian13`.
  - An `HPA` with `minReplicas: 2`, `maxReplicas: 5`, targeting the `{{ .Release.Name }}-bff` Deployment.
  - A `Service` of type `ClusterIP` exposing port `8080`.
  - A `ConfigMap` mounting BFF Viper configuration (OIDC issuer, Keeper base URL, Redis address, SSE heartbeat interval).
  - A `Secret` template for OIDC client ID/secret and session encryption key (values sourced from `.Values.bff.oidc.*` and `.Values.bff.session.*`).
- Run `make helm-infra-lint` and assert zero errors.

These are not Go unit tests; verification is done via `make helm-template` and `make helm-infra-lint` as part of the GREEN phase.

## GREEN Phase

1. **`deployment.yaml`**:
   - Multi-container spec: single `bff` container using `gcr.io/distroless/base-nossl-debian13` as runtime base.
   - Image tag sourced from `.Values.bff.image.tag`.
   - Mount `ConfigMap` as environment variables; mount `Secret` as environment variables.
   - Liveness probe: `GET /healthz`, `initialDelaySeconds: 5`, `periodSeconds: 10`.
   - Readiness probe: `GET /readyz`, `initialDelaySeconds: 10`, `periodSeconds: 15`, `failureThreshold: 3`.
   - Resource requests/limits configured via `.Values.bff.resources`.
   - `RollingUpdate` strategy with `maxUnavailable: 0` and `maxSurge: 1`.

2. **`service.yaml`**:
   - `ClusterIP` type.
   - Port `8080` → container port `8080`.
   - Selector: `app: {{ .Release.Name }}-bff`.

3. **`configmap.yaml`**:
   - Keys: `BFF_OIDC_ISSUER_URL`, `BFF_KEEPER_BASE_URL`, `BFF_REDIS_ADDR`, `BFF_REDIS_KEY_PREFIX`, `BFF_SSE_HEARTBEAT_SECONDS`, `BFF_SESSION_TTL_MINUTES`.
   - Values sourced from `.Values.bff.config.*`.

4. **`secret.yaml`**:
   - Keys: `BFF_OIDC_CLIENT_ID`, `BFF_OIDC_CLIENT_SECRET`, `BFF_SESSION_ENCRYPTION_KEY`.
   - Values sourced from `.Values.bff.oidc.clientID`, `.Values.bff.oidc.clientSecret`, `.Values.bff.session.encryptionKey` (base64 encoded).

5. **`hpa.yaml`** (per SPEC-NFR-HPA for BFF):
   - `minReplicas: {{ .Values.bff.hpa.minReplicas }}` (default `2`)
   - `maxReplicas: {{ .Values.bff.hpa.maxReplicas }}` (default `5`)
   - Primary metric: HTTP request rate (Prometheus custom metric `bff_http_requests_per_second`)
   - Secondary metric: CPU utilization at 70% target
   - `scaleDown.stabilizationWindowSeconds: 300` to prevent flapping.

6. **Update `values.yaml`**: Add `bff:` section with defaults for all the above values, including:
   ```yaml
   bff:
     image:
       repository: ghcr.io/morphy76/tacito-square/bff
       tag: latest
     hpa:
       minReplicas: 2
       maxReplicas: 5
     resources:
       requests:
         cpu: "100m"
         memory: "128Mi"
       limits:
         cpu: "500m"
         memory: "256Mi"
     config:
       keeperBaseURL: "http://keeper:8080"
       redisAddr: "redis:6379"
       redisKeyPrefix: "bff"
       sseHeartbeatSeconds: 30
       sessionTTLMinutes: 60
     oidc:
       issuerURL: ""
       clientID: ""
       clientSecret: ""
     session:
       encryptionKey: ""
   ```

Run `make helm-template` and `make helm-infra-lint` — must pass with zero errors (GREEN).

## REFACTOR Phase

- Confirm the `Deployment` does NOT use `hostNetwork` or `privileged` security context.
- Verify `readinessProbe` `failureThreshold` is at least `3` to tolerate brief Redis hiccups without premature pod removal.
- Add `podDisruptionBudget.yaml` with `minAvailable: 1` to prevent complete BFF downtime during node drain events.
- Confirm all Helm templates use `helm.sh/chart` and `app.kubernetes.io/*` standard labels.
- Confirm the Dockerfile for BFF (in `build/` or root `Dockerfile.bff`) uses the multi-stage build pattern: Go SDK builder → distroless runtime (per SPEC-NFR-STACK).
