# Task SPEC-FR-M10.8: Compact JSON API Responses

## Status
- **Status:** IN_PROGRESS
- **Milestone:** M10

## Tasks

- [ ] Add `omitempty` to JSON tags in domain models
  - [ ] `internal/keeper/domain/model/agent.go` (Skills, MCPClients, CommunityID, optional attributes)
  - [ ] `internal/keeper/domain/model/community.go` (Configuration, Description)
  - [ ] `internal/keeper/domain/model/community_card.go` (Capabilities, Agents, Description)
  - [ ] `internal/keeper/domain/model/mcp_client.go` (Args, Env, Command, URL, AuthSecretRef, Description)
  - [ ] `internal/keeper/domain/model/prompt.go` (Templates, Description)
  - [ ] `internal/keeper/domain/model/skill.go` (Skills, Description, Content)
  - [ ] `internal/keeper/domain/model/echo.go` (Results, Error)
- [ ] Add `omitempty` to JSON tags in package-exposed API contract structs
  - [ ] `pkg/agentcard/agent_card.go` (Provider, URL, Organization, Schemes, Credentials, Tags, Examples, InputModes, OutputModes, Description, etc.)
- [ ] Verify serialization compatibility and run contract tests
  - [ ] Run `make test`
  - [ ] Run `make test-contract`
