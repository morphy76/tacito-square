package service

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/morphy76/tacito-square/internal/keeper/application/ports/inbound"
	"github.com/morphy76/tacito-square/internal/keeper/application/ports/outbound"
	"github.com/morphy76/tacito-square/internal/keeper/domain/model"
	"github.com/morphy76/tacito-square/pkg/kubernetes/apis/tacito/v1alpha1"
	"github.com/spf13/viper"
)

var (
	ErrEmptyMessage          = errors.New("message must not be empty after sanitization")
	ErrBroadcasterUnavailable = errors.New("NATS messaging is not available")
	ErrCommunityNotFound     = errors.New("community not found")
	ErrNoRunningAgents       = errors.New("no running agents in community")
)

// EchoServiceImpl implements inbound.EchoUseCase.
type EchoServiceImpl struct {
	commRepo    outbound.CommunityRepository
	agentRepo   outbound.AgentRepository
	crdCoord    outbound.CRDCoordinator
	broadcaster outbound.CommunityBroadcaster
	cfg         *viper.Viper
}

var _ inbound.EchoUseCase = (*EchoServiceImpl)(nil)

// NewEchoService creates a new instance of EchoServiceImpl.
func NewEchoService(
	commRepo outbound.CommunityRepository,
	agentRepo outbound.AgentRepository,
	crdCoord outbound.CRDCoordinator,
	broadcaster outbound.CommunityBroadcaster,
	cfg *viper.Viper,
) *EchoServiceImpl {
	return &EchoServiceImpl{
		commRepo:    commRepo,
		agentRepo:   agentRepo,
		crdCoord:    crdCoord,
		broadcaster: broadcaster,
		cfg:         cfg,
	}
}

// EchoCommunity sanitizes the message, fans it out to all running agents in the community, and returns the aggregated results.
func (s *EchoServiceImpl) EchoCommunity(ctx context.Context, communityID uuid.UUID, message string) (*model.CommunityEchoResponse, error) {
	sanitized := model.SanitizeMessage(message)
	if sanitized == "" {
		return nil, ErrEmptyMessage
	}

	if !s.broadcaster.Available() {
		return nil, ErrBroadcasterUnavailable
	}

	comm, err := s.commRepo.GetByID(ctx, communityID)
	if err != nil || comm == nil {
		return nil, ErrCommunityNotFound
	}

	allAgents, err := s.agentRepo.List(ctx)
	if err != nil {
		return nil, err
	}

	s.syncAgentStatuses(ctx, allAgents, communityID)

	var wokeCommunity bool
	runningAgents := filterRunningAgents(allAgents, communityID)

	if len(runningAgents) == 0 {
		var hasIdleAgent bool
		for _, agent := range allAgents {
			if agent.CommunityID != nil && *agent.CommunityID == communityID {
				status, err := s.crdCoord.GetAgentCRDStatus(ctx, agent.ID)
				if err == nil && status != nil && status.Phase == v1alpha1.PhaseIdle {
					hasIdleAgent = true
					break
				}
			}
		}

		if !hasIdleAgent {
			return nil, ErrNoRunningAgents
		}

		wokeCommunity = true

		wakeupWaitSeconds := 30
		if s.cfg != nil && s.cfg.IsSet("keeper.echo.wakeup_wait_seconds") {
			wakeupWaitSeconds = s.cfg.GetInt("keeper.echo.wakeup_wait_seconds")
		}

		timeout := time.Duration(wakeupWaitSeconds) * time.Second
		timer := time.NewTimer(timeout)
		defer timer.Stop()

		ticker := time.NewTicker(50 * time.Millisecond)
		defer ticker.Stop()

		waitChan := make(chan bool, 1)

		go func() {
			for {
				select {
				case <-timer.C:
					waitChan <- false
					return
				case <-ctx.Done():
					waitChan <- false
					return
				case <-ticker.C:
					agentsList, err := s.agentRepo.List(ctx)
					if err == nil {
						s.syncAgentStatuses(ctx, agentsList, communityID)
						activeRunning := filterRunningAgents(agentsList, communityID)
						if len(activeRunning) > 0 {
							waitChan <- true
							return
						}
					}
				}
			}
		}()

		select {
		case success := <-waitChan:
			if success {
				agentsList, err := s.agentRepo.List(ctx)
				if err == nil {
					s.syncAgentStatuses(ctx, agentsList, communityID)
					runningAgents = filterRunningAgents(agentsList, communityID)
				}
			}
		case <-ctx.Done():
			return nil, ctx.Err()
		}

		if len(runningAgents) == 0 {
			return nil, ErrNoRunningAgents
		}
	}

	agentTimeoutSeconds := 10
	if s.cfg != nil && s.cfg.IsSet("keeper.echo.agent_timeout_seconds") {
		agentTimeoutSeconds = s.cfg.GetInt("keeper.echo.agent_timeout_seconds")
	}

	results := make([]model.AgentEchoResult, len(runningAgents))
	var wg sync.WaitGroup
	var mu sync.Mutex

	for i, agent := range runningAgents {
		wg.Add(1)
		go func(idx int, ag *model.Agent) {
			defer wg.Done()

			agentCtx, agentCancel := context.WithTimeout(ctx, time.Duration(agentTimeoutSeconds)*time.Second)
			defer agentCancel()

			req := model.EchoRequest{
				Message:     sanitized,
				CommunityID: comm.ID.String(),
				TenantID:    ag.TenantID,
			}

			reply, err := s.broadcaster.RequestEcho(agentCtx, comm.ID.String(), ag.Name, req)

			var res model.AgentEchoResult
			res.AgentName = ag.Name

			if err != nil {
				if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
					res.Error = "timeout"
				} else {
					res.Error = err.Error()
				}
			} else if reply != nil {
				res.Decorated = reply.Decorated
			}

			mu.Lock()
			results[idx] = res
			mu.Unlock()
		}(i, agent)
	}

	wg.Wait()

	return &model.CommunityEchoResponse{
		CommunityID:   communityID.String(),
		WokeCommunity: wokeCommunity,
		Results:       results,
	}, nil
}

func filterRunningAgents(agents []*model.Agent, communityID uuid.UUID) []*model.Agent {
	var running []*model.Agent
	for _, a := range agents {
		if a.CommunityID != nil && *a.CommunityID == communityID && a.Status == model.AgentStatusRunning {
			running = append(running, a)
		}
	}
	return running
}

func (s *EchoServiceImpl) syncAgentStatuses(ctx context.Context, agents []*model.Agent, communityID uuid.UUID) {
	for _, a := range agents {
		if a.CommunityID != nil && *a.CommunityID == communityID && a.Status == model.AgentStatusPending {
			crdStatus, err := s.crdCoord.GetAgentCRDStatus(ctx, a.ID)
			if err == nil && crdStatus != nil {
				var mappedStatus model.AgentStatus
				switch crdStatus.Phase {
				case v1alpha1.PhaseRunning:
					mappedStatus = model.AgentStatusRunning
				case v1alpha1.PhasePending:
					mappedStatus = model.AgentStatusPending
				case v1alpha1.PhaseTerminated:
					mappedStatus = model.AgentStatusStopped
				case v1alpha1.PhaseIdle:
					if a.Status == model.AgentStatusRunning {
						mappedStatus = model.AgentStatusPending
					} else {
						mappedStatus = a.Status
					}
				default:
					mappedStatus = model.AgentStatusError
				}

				if mappedStatus != a.Status {
					a.Status = mappedStatus
					a.UpdatedAt = time.Now().UTC()
					_ = s.agentRepo.Update(ctx, a)
				}
			}
		}
	}
}
