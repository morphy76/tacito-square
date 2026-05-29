# TASK-M4.8-T4: Echo Application Service

| Field       | Value                                                     |
|-------------|-----------------------------------------------------------|
| Task ID     | TASK-M4.8-T4                                             |
| Spec        | SPEC-FR-M4.8                                             |
| Boundary    | Application Service — `internal/keeper/application/service` |
| Status      | TODO                                                     |
| Depends On  | TASK-M4.8-T2, TASK-M4.8-T3                              |

## Objective

Implement `EchoServiceImpl`, the use-case orchestrator for the community echo feature. It depends **only** on outbound port interfaces — no concrete NATS or Kubernetes types.

## Files

| File | Action |
|------|--------|
| `internal/keeper/application/service/echo_service.go` | NEW |
| `internal/keeper/application/service/echo_service_test.go` | NEW |

## RED Phase

Create `internal/keeper/application/service/echo_service_test.go` with hand-rolled mocks for `CommunityBroadcaster`, `CommunityRepository`, and `AgentRepository`:

- `TestEchoCommunity_HappyPath`: Mock broadcaster returns a valid `EchoReply` for each running agent. Assert `CommunityEchoResponse.Results` has one entry per running agent with `Error == ""`.
- `TestEchoCommunity_ExcludesNonRunningAgents`: Agent list contains agents with `status = defined`, `assigned`, `pending`, `stopped`. Assert fanout is only invoked for `running` agents.
- `TestEchoCommunity_ConcurrentFanout`: Use a mock broadcaster with a `sync.WaitGroup` counter. Assert `RequestEcho` is called once per running agent concurrently (not sequentially — verify all calls complete before the first result is returned by checking WaitGroup completes within the service timeout, not agent-count × timeout).
- `TestEchoCommunity_AgentTimeout`: Mock broadcaster returns a context deadline exceeded error. Assert the result entry has `Decorated == ""` and `Error == "timeout"`. Assert HTTP status is still 200 (no domain error returned from service).
- `TestEchoCommunity_EmptyMessageAfterSanitization`: Input `"\x00\x01"` (only control chars). Assert service returns a sentinel domain error (e.g., `ErrEmptyMessage`) before any repository or broadcaster call.
- `TestEchoCommunity_CommunityNotFound`: `CommunityRepository.GetByID` returns not-found error. Assert service returns `ErrCommunityNotFound`.
- `TestEchoCommunity_NATSUnavailable`: `broadcaster.Available()` returns `false`. Assert service returns `ErrBroadcasterUnavailable` before any agent lookup.
- `TestEchoCommunity_NoRunningAgents_NoneIdle`: All agents in `stopped` state; no CRD `Idle` phase detected. Assert service returns `ErrNoRunningAgents`.
- `TestEchoCommunity_WokeCommunity`: At least one agent has CRD phase `Idle`. Assert `CommunityEchoResponse.WokeCommunity == true`.

Run `make test` — tests must fail (RED).

## GREEN Phase

Create `internal/keeper/application/service/echo_service.go`:

**Key design points**:

1. Constructor: `NewEchoService(communityRepo outbound.CommunityRepository, agentRepo outbound.AgentRepository, crdCoord outbound.CRDCoordinator, broadcaster outbound.CommunityBroadcaster, cfg *viper.Viper) *EchoServiceImpl`.

2. Implement `EchoUseCase.EchoCommunity(ctx, communityID, message)`:
   - Sanitize via `model.SanitizeMessage(message)`. If result is `""`, return `ErrEmptyMessage`.
   - Check `broadcaster.Available()`; if false, return `ErrBroadcasterUnavailable`.
   - Load community; if not found, return `ErrCommunityNotFound`.
   - List agents by community; filter those with `Status == AgentStatusRunning`.
   - If no running agents: check each agent's CRD status via `crdCoord.GetAgentCRDStatus`. If any have phase `Idle`, set `wokeCommunity = true` and wait `wakeUpWaitSeconds` (Viper key `keeper.echo.wakeup_wait_seconds`, default 30) using a `time.NewTimer` bounded by `ctx`.
   - If still no running agents after wait (or no idle agents), return `ErrNoRunningAgents`.
   - Fan out concurrently: for each running agent, launch a goroutine that calls `broadcaster.RequestEcho` with a per-agent context timeout (`keeper.echo.agent_timeout_seconds`, default 10s). Collect results into a buffered channel of size `len(agents)`.
   - Build `CommunityEchoResponse` from collected results. A `context.DeadlineExceeded` or `context.Canceled` error from `RequestEcho` maps to `AgentEchoResult.Error = "timeout"`. Any other error maps to `AgentEchoResult.Error = err.Error()`.

3. Define sentinel errors as package-level `var`:
   ```go
   var (
       ErrEmptyMessage          = errors.New("message must not be empty after sanitization")
       ErrBroadcasterUnavailable = errors.New("NATS messaging is not available")
       ErrCommunityNotFound     = errors.New("community not found")
       ErrNoRunningAgents       = errors.New("no running agents in community")
   )
   ```

4. Compile-time guard: `var _ inbound.EchoUseCase = (*EchoServiceImpl)(nil)`.

Run `make test` — tests must pass (GREEN).

## REFACTOR Phase

- Extract `fanOutEcho(ctx, agents, req, broadcaster, timeoutSeconds) []model.AgentEchoResult` as a private function so it can be tested independently.
- Extract `isIdlePhase(status *v1alpha1.TacitoAgentStatus) bool` helper.
- Ensure the wait for `wakeUpWaitSeconds` respects the parent `ctx` cancellation (use `select { case <-timer.C: case <-ctx.Done(): }`).
- Ensure all goroutines in `fanOutEcho` always send to the result channel (no goroutine leak on early context cancellation).
