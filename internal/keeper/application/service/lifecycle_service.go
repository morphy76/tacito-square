package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/morphy76/tacito-square/internal/keeper/application/ports/inbound"
	"github.com/morphy76/tacito-square/internal/keeper/application/ports/outbound"
	"github.com/morphy76/tacito-square/internal/keeper/domain/model"
	"github.com/morphy76/tacito-square/internal/shared/tenant"
	"github.com/morphy76/tacito-square/pkg/kubernetes/apis/tacito/v1alpha1"
	"github.com/nats-io/nats.go"
	"golang.org/x/sync/errgroup"
)

type LifecycleService struct {
	agentRepo      outbound.AgentRepository
	commRepo       outbound.CommunityRepository
	crdCoordinator outbound.CRDCoordinator
	natsConn       *nats.Conn
}

var _ inbound.LifecycleUseCase = (*LifecycleService)(nil)

// NewLifecycleService creates a new instance of LifecycleService.
func NewLifecycleService(
	agentRepo outbound.AgentRepository,
	commRepo outbound.CommunityRepository,
	crdCoord outbound.CRDCoordinator,
	nc *nats.Conn,
) *LifecycleService {
	return &LifecycleService{
		agentRepo:      agentRepo,
		commRepo:       commRepo,
		crdCoordinator: crdCoord,
		natsConn:       nc,
	}
}

// DeployAgent triggers agent CRD creation and database state transitions.
func (s *LifecycleService) DeployAgent(ctx context.Context, agentID uuid.UUID) error {
	ten := tenant.FromContext(ctx)
	if ten == nil {
		return errors.New("unauthorized: tenant context is required")
	}

	agent, err := s.agentRepo.GetByID(ctx, agentID)
	if err != nil {
		return err
	}

	if agent.TenantID != ten.FullName() {
		return fmt.Errorf("agent not found: %s", agentID)
	}

	if agent.CommunityID == nil {
		return fmt.Errorf("agent must be assigned to a community before deploying")
	}

	if agent.Status == model.AgentStatusPending || agent.Status == model.AgentStatusRunning {
		return fmt.Errorf("agent is already pending or running")
	}

	// Work on a shallow copy to avoid mutating the repository-returned pointer,
	// which would cause data races when DeployAgent is called concurrently
	// from DeployCommunity's errgroup goroutines.
	agentCopy := *agent
	agentCopy.Status = model.AgentStatusPending
	agentCopy.UpdatedAt = time.Now().UTC()
	if err := s.agentRepo.Update(ctx, &agentCopy); err != nil {
		return fmt.Errorf("updating agent status to pending: %w", err)
	}

	// Submit CRD which asynchronously triggers K8s and publishes started/completed NATS events
	if err := s.crdCoordinator.SubmitAgentCRD(ctx, &agentCopy); err != nil {
		return fmt.Errorf("submitting agent CRD: %w", err)
	}

	return nil
}

// UndeployAgent deletes the agent CRD and gracefully shuts down conversation loops.
func (s *LifecycleService) UndeployAgent(ctx context.Context, agentID uuid.UUID) error {
	ten := tenant.FromContext(ctx)
	if ten == nil {
		return errors.New("unauthorized: tenant context is required")
	}

	agent, err := s.agentRepo.GetByID(ctx, agentID)
	if err != nil {
		return err
	}

	if agent.TenantID != ten.FullName() {
		return fmt.Errorf("agent not found: %s", agentID)
	}

	if agent.Status != model.AgentStatusRunning && agent.Status != model.AgentStatusPending && agent.Status != model.AgentStatusError {
		return fmt.Errorf("agent is already undeployed/stopped")
	}

	// Publish NATS started event
	s.publishNatsEvent(ctx, "agent.provisioning.started", agent, nil)

	if err := s.crdCoordinator.TeardownAgentCRD(ctx, agent); err != nil {
		s.publishNatsEvent(ctx, "agent.provisioning.failed", agent, err)
		return fmt.Errorf("tearing down agent CRD: %w", err)
	}

	agent.Status = model.AgentStatusStopped
	agent.UpdatedAt = time.Now().UTC()
	if err := s.agentRepo.Update(ctx, agent); err != nil {
		s.publishNatsEvent(ctx, "agent.provisioning.failed", agent, err)
		return fmt.Errorf("updating agent status to stopped: %w", err)
	}

	// Publish NATS completed event
	s.publishNatsEvent(ctx, "agent.provisioning.completed", agent, nil)

	return nil
}

// GetAgentStatus queries the real-time observed status of a running TacitoAgent.
func (s *LifecycleService) GetAgentStatus(ctx context.Context, agentID uuid.UUID) (*inbound.AgentStatusDetails, error) {
	ten := tenant.FromContext(ctx)
	if ten == nil {
		return nil, errors.New("unauthorized: tenant context is required")
	}

	agent, err := s.agentRepo.GetByID(ctx, agentID)
	if err != nil {
		return nil, err
	}

	if agent.TenantID != ten.FullName() {
		return nil, fmt.Errorf("agent not found: %s", agentID)
	}

	// Query real-time status from CRD coordinator
	crdStatus, err := s.crdCoordinator.GetAgentCRDStatus(ctx, agentID)
	if err != nil {
		return nil, fmt.Errorf("querying CRD status: %w", err)
	}

	details := &inbound.AgentStatusDetails{
		AgentID:   agentID,
		Status:    agent.Status,
		Message:   "Pod stopped",
		Replicas:  0,
		UpdatedAt: agent.UpdatedAt,
	}

	if crdStatus != nil {
		// Map CRD phase to AgentStatus
		switch crdStatus.Phase {
		case v1alpha1.PhasePending:
			details.Status = model.AgentStatusPending
			details.Message = "Pod scheduling or pulling image"
		case v1alpha1.PhaseRunning, v1alpha1.PhaseIdle:
			details.Status = model.AgentStatusRunning
			details.Message = "Pod healthy and running"
		case v1alpha1.PhaseTerminated:
			details.Status = model.AgentStatusStopped
			details.Message = "Pod terminated"
		default:
			details.Status = model.AgentStatusError
			details.Message = "Pod error or crashed"
		}
		details.Replicas = crdStatus.Replicas
		
		if len(crdStatus.Conditions) > 0 {
			latest := crdStatus.Conditions[len(crdStatus.Conditions)-1]
			details.Message = fmt.Sprintf("%s: %s", latest.Reason, latest.Message)
		}
	} else if agent.Status != model.AgentStatusStopped && agent.Status != model.AgentStatusDefined {
		// If K8s resource is missing but DB status indicates deployment, it is an error state
		details.Status = model.AgentStatusError
		details.Message = "K8s resource missing"
	}

	return details, nil
}

// DeployCommunity deploys logical logical resources of a community and deploys all assigned agents in parallel.
func (s *LifecycleService) DeployCommunity(ctx context.Context, communityID uuid.UUID) (*inbound.CommunityDeploymentDetails, error) {
	ten := tenant.FromContext(ctx)
	if ten == nil {
		return nil, errors.New("unauthorized: tenant context is required")
	}

	comm, err := s.commRepo.GetByID(ctx, communityID)
	if err != nil {
		return nil, err
	}

	if comm.TenantID != ten.FullName() {
		return nil, fmt.Errorf("community not found: %s", communityID)
	}

	// Fetch all agents assigned to this community
	allAgents, err := s.agentRepo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing agents: %w", err)
	}

	var assignedAgents []*model.Agent
	for _, a := range allAgents {
		if a.CommunityID != nil && *a.CommunityID == communityID {
			assignedAgents = append(assignedAgents, a)
		}
	}

	details := &inbound.CommunityDeploymentDetails{
		CommunityID: communityID,
		Status:      "success",
		Agents:      []inbound.AgentDeploymentResult{},
	}

	if len(assignedAgents) == 0 {
		comm.Status = model.CommunityStatusActive
		comm.UpdatedAt = time.Now().UTC()
		if err := s.commRepo.Update(ctx, comm); err != nil {
			return nil, fmt.Errorf("updating community status: %w", err)
		}
		return details, nil
	}

	var eg errgroup.Group
	var mu sync.Mutex
	hasFailures := false

	for _, a := range assignedAgents {
		agentToDeploy := a
		eg.Go(func() error {
			deployErr := s.DeployAgent(ctx, agentToDeploy.ID)
			
			mu.Lock()
			defer mu.Unlock()
			
			res := inbound.AgentDeploymentResult{
				AgentID: agentToDeploy.ID,
				Status:  "deployed",
			}
			if deployErr != nil {
				res.Status = "failed"
				res.Error = deployErr.Error()
				hasFailures = true
			}
			details.Agents = append(details.Agents, res)
			return nil
		})
	}

	_ = eg.Wait()

	comm.Status = model.CommunityStatusActive
	comm.UpdatedAt = time.Now().UTC()
	if err := s.commRepo.Update(ctx, comm); err != nil {
		return nil, fmt.Errorf("updating community status to active: %w", err)
	}

	if hasFailures {
		details.Status = "partial_success"
	}

	return details, nil
}

// UndeployCommunity terminates all assigned agents within the community in parallel.
func (s *LifecycleService) UndeployCommunity(ctx context.Context, communityID uuid.UUID) (*inbound.CommunityDeploymentDetails, error) {
	ten := tenant.FromContext(ctx)
	if ten == nil {
		return nil, errors.New("unauthorized: tenant context is required")
	}

	comm, err := s.commRepo.GetByID(ctx, communityID)
	if err != nil {
		return nil, err
	}

	if comm.TenantID != ten.FullName() {
		return nil, fmt.Errorf("community not found: %s", communityID)
	}

	allAgents, err := s.agentRepo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing agents: %w", err)
	}

	var assignedAgents []*model.Agent
	for _, a := range allAgents {
		if a.CommunityID != nil && *a.CommunityID == communityID {
			if a.Status == model.AgentStatusRunning || a.Status == model.AgentStatusPending || a.Status == model.AgentStatusError {
				assignedAgents = append(assignedAgents, a)
			}
		}
	}

	details := &inbound.CommunityDeploymentDetails{
		CommunityID: communityID,
		Status:      "success",
		Agents:      []inbound.AgentDeploymentResult{},
	}

	if len(assignedAgents) == 0 {
		comm.Status = model.CommunityStatusInactive
		comm.UpdatedAt = time.Now().UTC()
		if err := s.commRepo.Update(ctx, comm); err != nil {
			return nil, fmt.Errorf("updating community status: %w", err)
		}
		return details, nil
	}

	var eg errgroup.Group
	var mu sync.Mutex
	hasFailures := false

	for _, a := range assignedAgents {
		agentToUndeploy := a
		eg.Go(func() error {
			undeployErr := s.UndeployAgent(ctx, agentToUndeploy.ID)
			
			mu.Lock()
			defer mu.Unlock()
			
			res := inbound.AgentDeploymentResult{
				AgentID: agentToUndeploy.ID,
				Status:  "stopped",
			}
			if undeployErr != nil {
				res.Status = "failed"
				res.Error = undeployErr.Error()
				hasFailures = true
			}
			details.Agents = append(details.Agents, res)
			return nil
		})
	}

	_ = eg.Wait()

	comm.Status = model.CommunityStatusInactive
	comm.UpdatedAt = time.Now().UTC()
	if err := s.commRepo.Update(ctx, comm); err != nil {
		return nil, fmt.Errorf("updating community status to inactive: %w", err)
	}

	if hasFailures {
		details.Status = "partial_success"
	}

	return details, nil
}

// GetCommunityStatus aggregates K8s and database statuses of all agents in the community.
func (s *LifecycleService) GetCommunityStatus(ctx context.Context, communityID uuid.UUID) (*inbound.CommunityStatusDetails, error) {
	ten := tenant.FromContext(ctx)
	if ten == nil {
		return nil, errors.New("unauthorized: tenant context is required")
	}

	comm, err := s.commRepo.GetByID(ctx, communityID)
	if err != nil {
		return nil, err
	}

	if comm.TenantID != ten.FullName() {
		return nil, fmt.Errorf("community not found: %s", communityID)
	}

	allAgents, err := s.agentRepo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing agents: %w", err)
	}

	var assignedAgents []*model.Agent
	for _, a := range allAgents {
		if a.CommunityID != nil && *a.CommunityID == communityID {
			assignedAgents = append(assignedAgents, a)
		}
	}

	details := &inbound.CommunityStatusDetails{
		CommunityID: communityID,
		Status:      comm.Status,
		Agents:      []inbound.AgentStatusDetails{},
	}

	var eg errgroup.Group
	var mu sync.Mutex

	for _, a := range assignedAgents {
		agentToQuery := a
		eg.Go(func() error {
			agentStatus, queryErr := s.GetAgentStatus(ctx, agentToQuery.ID)
			
			mu.Lock()
			defer mu.Unlock()
			
			if queryErr == nil && agentStatus != nil {
				details.Agents = append(details.Agents, *agentStatus)
			} else {
				fallbackMsg := "query failed"
				if queryErr != nil {
					fallbackMsg = queryErr.Error()
				}
				details.Agents = append(details.Agents, inbound.AgentStatusDetails{
					AgentID:   agentToQuery.ID,
					Status:    agentToQuery.Status,
					Message:   fallbackMsg,
					UpdatedAt: agentToQuery.UpdatedAt,
				})
			}
			return nil
		})
	}

	_ = eg.Wait()

	return details, nil
}

// publishNatsEvent is a helper to manually stream provisioning status events.
func (s *LifecycleService) publishNatsEvent(ctx context.Context, subject string, agent *model.Agent, errVal error) {
	if s.natsConn == nil {
		return
	}

	var errMsg string
	if errVal != nil {
		errMsg = errVal.Error()
	}

	var communityID string
	if agent.CommunityID != nil {
		communityID = agent.CommunityID.String()
	}

	event := struct {
		TenantID    string `json:"tenant_id"`
		AgentID     string `json:"agent_id"`
		CommunityID string `json:"community_id"`
		Timestamp   string `json:"timestamp"`
		Error       string `json:"error,omitempty"`
	}{
		TenantID:    agent.TenantID,
		AgentID:     agent.ID.String(),
		CommunityID: communityID,
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
		Error:       errMsg,
	}

	data, err := json.Marshal(event)
	if err != nil {
		return
	}

	_ = s.natsConn.Publish(subject, data)
}
