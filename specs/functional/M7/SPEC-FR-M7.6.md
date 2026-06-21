# SPEC-FR-M7.6: BFF primary API surface to use GraphQL

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | SPEC-FR-M7.6                                |
| Status        | DRAFT                                       |
| Milestone     | M7                                          |
| Component     | bff, ui                                     |
| Depends On    | none                                        |
| Supersedes    | none                                        |

## Context

Migrate the BFF's primary API surface to GraphQL to support flexible UI frontend client queries, while keeping REST APIs to support CLI clients.

## Specification

1. Introduce a GraphQL query/mutation scheme for BFF orchestration.
2. Maintain compatibility with the REST APIs used by CLI/direct consumers.
3. Optimize resolvers to delegate to backing Keeper REST endpoints efficiently.

## Acceptance Criteria

To be defined during spec review.

## Test Plan

To be defined during spec review.

## Files Affected

To be defined during spec review.
