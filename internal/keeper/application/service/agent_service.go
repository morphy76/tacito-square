package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/morphy76/tacito-square/internal/keeper/application/ports/outbound"
	"github.com/morphy76/tacito-square/internal/keeper/domain/model"
	"github.com/morphy76/tacito-square/internal/shared/tenant"
	"go.opentelemetry.io/otel/trace"
)

// AgentService orchestrates business logic for Agent templates and community assignments.
type AgentService struct {
	repo           outbound.AgentRepository
	crdCoordinator outbound.CRDCoordinator
}

// NewAgentService creates a new instance of AgentService.
func NewAgentService(repo outbound.AgentRepository, crdCoordinator outbound.CRDCoordinator) *AgentService {
	return &AgentService{
		repo:           repo,
		crdCoordinator: crdCoordinator,
	}
}

func (s *AgentService) Create(ctx context.Context, agent *model.Agent) error {
	return s.repo.Create(ctx, agent)
}

func (s *AgentService) GetByID(ctx context.Context, id uuid.UUID) (*model.Agent, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *AgentService) List(ctx context.Context) ([]*model.Agent, error) {
	return s.repo.List(ctx)
}

func (s *AgentService) Update(ctx context.Context, agent *model.Agent) error {
	return s.repo.Update(ctx, agent)
}

func (s *AgentService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}

// Assign binds an Agent template to a Community asynchronously out-of-band.
func (s *AgentService) Assign(ctx context.Context, communityID uuid.UUID, agentID uuid.UUID) error {
	// 1. Persist the assignment changes synchronously in the repository within current transaction context
	if err := s.repo.AssignToCommunity(ctx, agentID, communityID); err != nil {
		return err
	}

	agent, err := s.repo.GetByID(ctx, agentID)
	if err != nil {
		return err
	}

	// 2. Detach parent context to avoid cancellation when HTTP request finishes
	bgCtx := context.Background()

	// 3. Propagate Tenant context to ensure proper multitenancy scoping in background port calls
	if ten := tenant.FromContext(ctx); ten != nil {
		bgCtx = tenant.ContextWithTenant(bgCtx, ten)
	}

	// 4. Propagate OpenTelemetry span context to correlate async traces
	if span := trace.SpanFromContext(ctx); span.SpanContext().IsValid() {
		bgCtx = trace.ContextWithSpan(bgCtx, span)
	}

	// 5. Execute expensive K8s CRD coordinator hooks asynchronously out-of-band
	go func() {
		_ = s.crdCoordinator.SubmitAgentCRD(bgCtx, agent)
	}()

	return nil
}

// Unassign removes an Agent template assignment from a Community asynchronously out-of-band.
func (s *AgentService) Unassign(ctx context.Context, communityID uuid.UUID, agentID uuid.UUID) error {
	agent, err := s.repo.GetByID(ctx, agentID)
	if err != nil {
		return err
	}

	// 1. Update persistence synchronously
	if err := s.repo.UnassignFromCommunity(ctx, agentID, communityID); err != nil {
		return err
	}

	// 2. Detach parent context to avoid cancellation when HTTP request finishes
	bgCtx := context.Background()

	// 3. Propagate Tenant context
	if ten := tenant.FromContext(ctx); ten != nil {
		bgCtx = tenant.ContextWithTenant(bgCtx, ten)
	}

	// 4. Propagate OpenTelemetry span context
	if span := trace.SpanFromContext(ctx); span.SpanContext().IsValid() {
		bgCtx = trace.ContextWithSpan(bgCtx, span)
	}

	// 5. Execute expensive K8s CRD coordinator hooks asynchronously out-of-band
	go func() {
		_ = s.crdCoordinator.TeardownAgentCRD(bgCtx, agent)
	}()

	return nil
}
