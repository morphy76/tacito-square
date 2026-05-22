# TASK-M2.1.3: Add CRD templates (GREEN)

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M2.1.3                                 |
| Status        | TODO                                        |
| Spec          | SPEC-FR-M2.1                                |
| Phase         | GREEN                                       |
| Depends On    | TASK-M2.1.2                                 |

## Description

Add placeholder CRD definitions for TacitoAgent and TacitoCommunity to the application Helm chart. These are skeleton CRDs that will be fully fleshed out in M4 (Operator Core) but must be present now for chart completeness.

## Work Items

1. Create `tools/helm/tacito-square/crds/tacitoagent-crd.yaml`:
   - apiVersion: `apiextensions.k8s.io/v1`, kind: `CustomResourceDefinition`
   - Group: `tacito.square.io`, version: `v1alpha1`, kind: `TacitoAgent`
   - Minimal spec schema (name, communityRef, replicas as required fields).
2. Create `tools/helm/tacito-square/crds/tacitocommunity-crd.yaml`:
   - Group: `tacito.square.io`, version: `v1alpha1`, kind: `TacitoCommunity`
   - Minimal spec schema (communityName, topology as required fields).
3. Verify CRDs are rendered by `helm template`.

## Acceptance Criteria

1. `helm template` output contains `TacitoAgent` CRD.
2. `helm template` output contains `TacitoCommunity` CRD.
3. CRDs use API group `tacito.square.io/v1alpha1`.
