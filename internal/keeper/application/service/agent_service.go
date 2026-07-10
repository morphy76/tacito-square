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
	"github.com/morphy76/tacito-square/pkg/kubernetes/apis/tacito/v1alpha1"
	"go.opentelemetry.io/otel/trace"
)

// AgentService orchestrates business logic for Agent templates and community assignments.
type AgentService struct {
	repo           outbound.AgentRepository
	communityRepo  outbound.CommunityRepository
	crdCoordinator outbound.CRDCoordinator
	cache          sharedports.Cache
	publisher      outbound.EventPublisher
	llmBindingRepo outbound.LLMBindingRepository
}

// NewAgentService creates a new instance of AgentService.
func NewAgentService(
	repo outbound.AgentRepository,
	communityRepo outbound.CommunityRepository,
	crdCoordinator outbound.CRDCoordinator,
	cache sharedports.Cache,
	publisher outbound.EventPublisher,
	llmBindingRepo outbound.LLMBindingRepository,
) *AgentService {
	return &AgentService{
		repo:           repo,
		communityRepo:  communityRepo,
		crdCoordinator: crdCoordinator,
		cache:          cache,
		publisher:      publisher,
		llmBindingRepo: llmBindingRepo,
	}
}

func (s *AgentService) Create(ctx context.Context, agent *model.Agent) error {
	if _, err := s.llmBindingRepo.GetByID(ctx, agent.Brain.LLMBindingID); err != nil {
		return fmt.Errorf("llm binding does not exist: %w", err)
	}
	return s.repo.Create(ctx, agent)
}

func (s *AgentService) GetByID(ctx context.Context, id uuid.UUID) (*model.Agent, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *AgentService) List(ctx context.Context) ([]*model.Agent, error) {
	return s.repo.List(ctx)
}

func (s *AgentService) Update(ctx context.Context, agent *model.Agent) error {
	if _, err := s.llmBindingRepo.GetByID(ctx, agent.Brain.LLMBindingID); err != nil {
		return fmt.Errorf("llm binding does not exist: %w", err)
	}
	return s.repo.Update(ctx, agent)
}

func (s *AgentService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}

// Assign binds an Agent template to a Community asynchronously out-of-band.
func (s *AgentService) Assign(ctx context.Context, communityID uuid.UUID, agentID uuid.UUID) error {
	ten := tenant.FromContext(ctx)
	if ten == nil {
		return fmt.Errorf("tenant resolution failed")
	}

	// 1. Verify community exists, belongs to tenant, and is active/created
	community, err := s.communityRepo.GetByID(ctx, communityID)
	if err != nil {
		return err
	}
	if community.Status != model.CommunityStatusActive && community.Status != model.CommunityStatusCreated {
		return fmt.Errorf("community status is %s, must be active or created", community.Status)
	}

	// 2. Verify agent exists, belongs to tenant
	agent, err := s.repo.GetByID(ctx, agentID)
	if err != nil {
		return err
	}

	// Reconcile: If already assigned to the same community, verify deployment status
	if agent.CommunityID != nil {
		if *agent.CommunityID != communityID {
			return fmt.Errorf("agent already assigned to community: %s", agent.CommunityID)
		}

		crdStatus, err := s.crdCoordinator.GetAgentCRDStatus(ctx, agentID)
		if err != nil {
			return fmt.Errorf("check crd status: %w", err)
		}

		isCRDActive := crdStatus != nil && (crdStatus.Phase == v1alpha1.PhaseRunning || crdStatus.Phase == v1alpha1.PhaseIdle || crdStatus.Phase == v1alpha1.PhasePending)

		if !isCRDActive {
			// Trigger deployment reconciliation!
			agent.Status = model.AgentStatusPending
			if err := s.repo.Update(ctx, agent); err != nil {
				return err
			}

			bgCtx := context.Background()
			bgCtx = tenant.ContextWithTenant(bgCtx, ten)
			if span := trace.SpanFromContext(ctx); span.SpanContext().IsValid() {
				bgCtx = trace.ContextWithSpan(bgCtx, span)
			}
			go func() {
				_ = s.crdCoordinator.SubmitAgentCRD(bgCtx, agent)
			}()
		}
		return nil
	}

	// 3. Verify topology constraints
	if community.Topology == model.CommunityTopologySingleAgent {
		agents, err := s.repo.List(ctx)
		if err != nil {
			return err
		}
		count := 0
		for _, a := range agents {
			if a.CommunityID != nil && *a.CommunityID == communityID {
				count++
			}
		}
		if count >= 1 {
			return fmt.Errorf("community with single-agent topology cannot have more than one agent assigned")
		}
	}
	// Note: hub-spoke role validation is delegated to the CommunityAssignmentRepository (SPEC-FR-M6.5.1)

	// 4. Persist the assignment changes synchronously in the repository within current transaction context
	if err := s.repo.AssignToCommunity(ctx, agentID, communityID); err != nil {
		return err
	}

	// Update in-memory state to reflect the assignment and pending status before passing to background task
	agent.CommunityID = &communityID
	agent.Status = model.AgentStatusPending

	// 5. Detach parent context to avoid cancellation when HTTP request finishes
	bgCtx := context.Background()

	// 6. Propagate Tenant context to ensure proper multitenancy scoping in background port calls
	bgCtx = tenant.ContextWithTenant(bgCtx, ten)

	// 7. Propagate OpenTelemetry span context to correlate async traces
	if span := trace.SpanFromContext(ctx); span.SpanContext().IsValid() {
		bgCtx = trace.ContextWithSpan(bgCtx, span)
	}

	// 8. Execute expensive K8s CRD coordinator hooks asynchronously out-of-band
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

	// Reconcile: If already unassigned, verify deployment status
	if agent.CommunityID == nil && agent.Status != "" {
		crdStatus, err := s.crdCoordinator.GetAgentCRDStatus(ctx, agentID)
		if err != nil {
			return fmt.Errorf("check crd status: %w", err)
		}

		isCRDActive := crdStatus != nil && crdStatus.Phase != v1alpha1.PhaseTerminated

		if isCRDActive {
			// Trigger undeployment reconciliation!
			if err := s.repo.DeleteRegistration(ctx, agentID, communityID); err != nil {
				return err
			}

			bgCtx := context.Background()
			var tenantID string
			if ten := tenant.FromContext(ctx); ten != nil {
				bgCtx = tenant.ContextWithTenant(bgCtx, ten)
				tenantID = ten.FullName()
			}
			if span := trace.SpanFromContext(ctx); span.SpanContext().IsValid() {
				bgCtx = trace.ContextWithSpan(bgCtx, span)
			}

			go func() {
				_ = s.crdCoordinator.TeardownAgentCRD(bgCtx, agent)
			}()

			if s.cache != nil {
				agentKey := fmt.Sprintf("communities:%s:agents:%s", communityID.String(), agentID.String())
				_ = s.cache.Invalidate(bgCtx, agentKey)
				registryKey := fmt.Sprintf("communities:%s:registry", communityID.String())
				_ = s.cache.Invalidate(bgCtx, registryKey)
			}

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
		}
		return nil
	}

	// 1. Update persistence synchronously
	if err := s.repo.UnassignFromCommunity(ctx, agentID, communityID); err != nil {
		return err
	}

	// 2. Delete the registration from PostgreSQL agent_registrations table
	if err := s.repo.DeleteRegistration(ctx, agentID, communityID); err != nil {
		return err
	}

	// Update in-memory state to reflect the unassignment and defined status
	agent.CommunityID = nil
	agent.Status = model.AgentStatusDefined

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
