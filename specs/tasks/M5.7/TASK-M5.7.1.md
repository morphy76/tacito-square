# TASK-M5.7.1: Scaffolding and Configuration of Standalone Agent Helm Chart

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M5.7.1                                 |
| Status        | VERIFIED                                    |
| Spec          | SPEC-FR-M5.7                                |
| Depends On    | none                                        |

## Description

Scaffold a standalone Helm 3 chart for the Tacito Agent at `tools/helm/tacito-agent`. Configure standard `Chart.yaml`, structured `values.yaml` defaults, and standard helper functions in `templates/_helpers.tpl`. Implement the `templates/configmap.yaml` template to dynamically serialize configuration values into the canonical `TS_AGENT_*` environment variable block.

## Work Items

1. **RED Phase**:
   - Write dry-run template rendering checks in `test/helm/test_agent_standalone_chart.sh` that assert the existence and YAML validity of a generated ConfigMap named `my-agent-tacito-agent-config` containing required properties (e.g., `TS_AGENT_NAME`, `TS_AGENT_COMMUNITY_REF`).
   - Run the script and observe failure (RED) because the `tools/helm/tacito-agent/` chart does not exist.

2. **GREEN Phase**:
   - Scaffold the directory structure under `tools/helm/tacito-agent/`.
   - Create `Chart.yaml` defining `tacito-agent` with version `0.1.0` and application version `0.1.0` (independent standalone release).
   - Define structured values in `values.yaml` matching all agent properties, LLM parameters, memory parameters (Redis/Qdrant), S3 settings, and NATS url.
   - Implement `templates/_helpers.tpl` for standard name/labels helpers.
   - Implement `templates/configmap.yaml` mapping all `agent.*`, `redis.*`, `qdrant.*`, `nats.*`, and `s3.*` values to flat environment variables.
   - Verify dry-run template rendering passes (GREEN).

3. **REFACTOR Phase**:
   - Ensure clean formatting of `values.yaml` and appropriate multi-line serialization of `TS_AGENT_SYSTEM_PROMPT`.
   - Verify zero redundant or hardcoded fields.

## Acceptance Criteria

1. Running `helm template` on `tools/helm/tacito-agent` renders a valid Kubernetes ConfigMap containing all required `TS_AGENT_*` keys.
2. The `TS_AGENT_COMMUNITY_REF` is mapped to `agent.communityRef` values.
3. No external dependency on the umbrella or infrastructure Helm charts exists.
