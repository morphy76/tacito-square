# TASK-M7.2-T6: UI Configurator Helm Deployment Charts

| Field       | Value                                                              |
|-------------|--------------------------------------------------------------------|
| Task ID     | TASK-M7.2-T6                                                      |
| Spec        | SPEC-FR-M7.2                                                       |
| Boundary    | Deploy — `tools/helm/tacito-square/templates/ui-configurator/`     |
| Status      | DRAFT                                                              |
| Depends On  | SPEC-FR-M7.2, TASK-M7.2-T1                                         |

## Objective

Create the Kubernetes Helm templates for the Configurator UI as a separate deployment (serving static files via a lightweight web server like Nginx or a distroless static web server). Define templates for `Deployment`, `Service`, and Horizontal Pod Autoscaler (`HPA`) configurations under `tools/helm/tacito-square/templates/ui-configurator/`. Configure the parent Helm chart's `values.yaml` and modify the root `Makefile` to include UI container build rules. Create a dedicated Dockerfile for the standalone UI container.

## Files

| File | Action |
|------|--------|
| `tools/docker/Dockerfile.ui-configurator` | NEW |
| `tools/helm/tacito-square/templates/ui-configurator/deployment.yaml` | NEW |
| `tools/helm/tacito-square/templates/ui-configurator/service.yaml` | NEW |
| `tools/helm/tacito-square/templates/ui-configurator/hpa.yaml` | NEW |
| `tools/helm/tacito-square/values.yaml` | MODIFY |
| `Makefile` | MODIFY |

## RED Phase

1. **Helm Template Rendering Assertions**:
   - Run `make helm-template` (which runs `helm template`) and assert that the output contains:
     - A `Deployment` named `{{ .Release.Name }}-ui-configurator` referencing the repository/tag specified in `values.yaml`.
     - A `Service` of type `ClusterIP` named `{{ .Release.Name }}-ui-configurator` mapping port `80` (or `8080`) to the container's static server port.
     - An `HPA` with min/max replica limits configured specifically for the `ui-configurator` deployment.
   - Run `make helm-infra-lint` and check for errors.
   - Confirm that compiling or linting fails before these files exist.

## GREEN Phase

1. **Create UI Dockerfile**:
   - Create `tools/docker/Dockerfile.ui-configurator` using a multi-stage structure:
     - Stage 1: Build the React 19 application using a Node.js builder image.
     - Stage 2: Copy the static built SPA files (`dist/` folder) to a lightweight web server runtime (e.g. `nginx:alpine` or `gcr.io/distroless/static-debian12`).
     - Configure the server stage to serve all non-existent static paths by falling back to `index.html` (SPA routing redirection).

2. **Create Helm templates**:
   - Create `tools/helm/tacito-square/templates/ui-configurator/deployment.yaml` using the configured image. Setup liveness/readiness probes targeting `/index.html` or the base path.
   - Create `tools/helm/tacito-square/templates/ui-configurator/service.yaml` exposing the static server port.
   - Create `tools/helm/tacito-square/templates/ui-configurator/hpa.yaml` configuring autoscaling parameters:
     - `minReplicas` and `maxReplicas` configured via `values.yaml`.
     - Primary metric: CPU utilization at 80% target.

3. **Update `values.yaml`**:
   - Add a `uiConfigurator` configuration section with default parameters:
     ```yaml
     uiConfigurator:
       enabled: true
       replicaCount: 2
       image:
         registry: ""
         name: tacito-square/ui-configurator
         tag: latest
         pullPolicy: IfNotPresent
       service:
         port: 80
       hpa:
         minReplicas: 2
         maxReplicas: 5
       resources:
         requests:
           cpu: "100m"
           memory: "64Mi"
         limits:
           cpu: "300m"
           memory: "128Mi"
     ```

4. **Update root Makefile**:
   - Add target `docker-build-ui-configurator` to build the UI Docker image.
   - Add target `docker-push-ui-configurator` to push the image.
   - Run `make helm-template` and verify that all templates render correctly without errors (GREEN).

## REFACTOR Phase

- Confirm the Nginx/static server config uses non-root permissions and standard distroless or lightweight base image restrictions.
- Ensure the SPA router redirection configuration handles trailing slashes correctly and doesn't conflict with BFF API paths.
