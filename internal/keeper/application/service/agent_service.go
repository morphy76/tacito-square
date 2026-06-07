package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/morphy76/tacito-square/internal/keeper/application/ports/outbound"
	"github.com/morphy76/tacito-square/internal/keeper/domain/model"
	sharedports "github.com/morphy76/tacito-square/internal/shared/ports/outbound"
	"github.com/morphy76/tacito-square/internal/shared/tenant"
	"github.com/morphy76/tacito-square/pkg/events"
	"go.opentelemetry.io/otel/trace"
)

// AgentService orchestrates business logic for Agent templates and community assignments.
type AgentService struct {
	repo           outbound.AgentRepository
	crdCoordinator outbound.CRDCoordinator
	cache          sharedports.Cache
	publisher      outbound.EventPublisher
}

// NewAgentService creates a new instance of AgentService.
func NewAgentService(
	repo outbound.AgentRepository,
	crdCoordinator outbound.CRDCoordinator,
	cache sharedports.Cache,
	publisher outbound.EventPublisher,
) *AgentService {
	return &AgentService{
		repo:           repo,
		crdCoordinator: crdCoordinator,
		cache:          cache,
		publisher:      publisher,
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

	// 2. Delete the registration from PostgreSQL agent_registrations table
	if err := s.repo.DeleteRegistration(ctx, agentID, communityID); err != nil {
		return err
	}

	// 3. Detach parent context to avoid cancellation when HTTP request finishes
	bgCtx := context.Background()

	// 4. Propagate Tenant context
	var tenantID string
	if ten := tenant.FromContext(ctx); ten != nil {
		bgCtx = tenant.ContextWithTenant(bgCtx, ten)
		tenantID = ten.FullName()
	}

	// 5. Propagate OpenTelemetry span context
	if span := trace.SpanFromContext(ctx); span.SpanContext().IsValid() {
		bgCtx = trace.ContextWithSpan(bgCtx, span)
	}

	// 6. Execute expensive K8s CRD coordinator hooks asynchronously out-of-band
	go func() {
		_ = s.crdCoordinator.TeardownAgentCRD(bgCtx, agent)
	}()

	// 7. Clear cache keys
	if s.cache != nil {
		agentKey := fmt.Sprintf("communities:%s:agents:%s", communityID.String(), agentID.String())
		_ = s.cache.Invalidate(bgCtx, agentKey)
		registryKey := fmt.Sprintf("communities:%s:registry", communityID.String())
		_ = s.cache.Invalidate(bgCtx, registryKey)
	}

	// 8. Publish NATS offline status change event
	if s.publisher != nil {
		subject := fmt.Sprintf("ts.community.%s.agent.%s.status", communityID.String(), agentID.String())
		evt := events.DomainEvent{
			EventID:    uuid.New().String(),
			SchemaRef:  "urn:tacito:schema:conversational:agent-status:v1",
			Source:     "keeper",
			TenantID:   tenantID,
			OccurredAt: time.Now().UTC().Format(time.RFC3339Nano),
			Payload:    []byte(`{"status":"offline"}`),
		}
		_ = s.publisher.Publish(bgCtx, subject, evt)
	}

	return nil
}
