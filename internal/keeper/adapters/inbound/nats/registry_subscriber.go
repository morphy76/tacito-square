package nats

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/morphy76/tacito-square/internal/keeper/application/ports/outbound"
	"github.com/morphy76/tacito-square/internal/keeper/domain/model"
	"github.com/morphy76/tacito-square/internal/shared/observability"
	sharedports "github.com/morphy76/tacito-square/internal/shared/ports/outbound"
	"github.com/morphy76/tacito-square/internal/shared/tenant"
	"github.com/morphy76/tacito-square/pkg/agentcard"
	"github.com/morphy76/tacito-square/pkg/events"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// RegistrySubscriber handles subscribing to agent heartbeats and updating Postgres/Redis.
type RegistrySubscriber struct {
	nc        *nats.Conn
	agentRepo outbound.AgentRepository
	cache     sharedports.Cache
	publisher outbound.EventPublisher
	logger    zerolog.Logger
	sub       *nats.Subscription
	tracer    trace.Tracer
}

// NewRegistrySubscriber creates a new RegistrySubscriber instance.
func NewRegistrySubscriber(
	nc *nats.Conn,
	agentRepo outbound.AgentRepository,
	cache sharedports.Cache,
	publisher outbound.EventPublisher,
	logger zerolog.Logger,
) *RegistrySubscriber {
	return &RegistrySubscriber{
		nc:        nc,
		agentRepo: agentRepo,
		cache:     cache,
		publisher: publisher,
		logger:    logger.With().Str("component", "registry_subscriber").Logger(),
		tracer:    otel.Tracer("registry_subscriber"),
	}
}

// Start registers the wildcard subscription for heartbeats.
func (s *RegistrySubscriber) Start(ctx context.Context) error {
	subject := "ts.community.*.agent.*.heartbeat"
	h := observability.WrapNATSHandler("registry.heartbeat", s.logger, s.handleHeartbeat)
	sub, err := s.nc.Subscribe(subject, h)
	if err != nil {
		return fmt.Errorf("subscribe to heartbeats: %w", err)
	}
	s.sub = sub
	s.logger.Info().Str("subject", subject).Msg("registry heartbeat subscriber started")
	return nil
}

// Stop drains and closes the subscription.
func (s *RegistrySubscriber) Stop() error {
	if s.sub != nil {
		if err := s.sub.Drain(); err != nil {
			return fmt.Errorf("drain heartbeat subscription: %w", err)
		}
		s.sub = nil
	}
	s.logger.Info().Msg("registry heartbeat subscriber stopped")
	return nil
}

func (s *RegistrySubscriber) handleHeartbeat(ctx context.Context, logger zerolog.Logger, msg *nats.Msg) error {
	logger.Trace().Msg("heartbeat message received")
	parts := strings.Split(msg.Subject, ".")
	if len(parts) != 6 {
		logger.Warn().Str("subject", msg.Subject).Msg("malformed heartbeat subject, skipping")
		return nil
	}

	commIDStr := parts[2]
	agentIDStr := parts[4]

	commUUID, err := uuid.Parse(commIDStr)
	if err != nil {
		logger.Warn().Str("community_id", commIDStr).Msg("invalid community uuid in subject, skipping")
		return nil
	}
	agentUUID, err := uuid.Parse(agentIDStr)
	if err != nil {
		logger.Warn().Str("agent_id", agentIDStr).Msg("invalid agent uuid in subject, skipping")
		return nil
	}

	var evt events.DomainEvent
	if err := json.Unmarshal(msg.Data, &evt); err != nil {
		logger.Warn().Err(err).Msg("failed to unmarshal domain event envelope, skipping")
		return nil
	}

	if evt.SchemaRef != events.SchemaInfrastructureAgentHeartbeat {
		logger.Warn().Str("schema_ref", evt.SchemaRef).Msg("unexpected schema in heartbeat NATS subject, skipping")
		return nil
	}

	var card agentcard.AgentCard
	if err := json.Unmarshal(evt.Payload, &card); err != nil {
		logger.Warn().Err(err).Msg("failed to unmarshal agent card from event payload, skipping")
		return nil
	}

	// Resolve tenant from NATS header
	tenantID := msg.Header.Get("X-Tacito-Tenant")
	if tenantID == "" {
		logger.Warn().Msg("heartbeat missing tenant header, ignoring")
		return nil
	}

	ten, err := tenant.New(tenantID, "")
	if err != nil {
		logger.Warn().Err(err).Str("tenant_id", tenantID).Msg("invalid tenant ID format, ignoring")
		return nil
	}
	ctx = tenant.ContextWithTenant(ctx, ten)

	// Update DB registration and agent status
	ctx, span := s.tracer.Start(ctx, "registry.ingest", trace.WithSpanKind(trace.SpanKindConsumer))
	defer span.End()

	err = s.agentRepo.UpsertRegistration(ctx, agentUUID, commUUID, &card)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("upsert agent registration: %w", err)
	}

	changed, err := s.agentRepo.UpdateStatus(ctx, agentUUID, model.AgentStatusRunning)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("update agent status: %w", err)
	}

	if changed && s.publisher != nil {
		// Broadcast status change event to NATS
		subject := fmt.Sprintf("ts.community.%s.agent.%s.status", commIDStr, agentIDStr)
		evt := events.DomainEvent{
			EventID:    uuid.New().String(),
			SchemaRef:  "urn:tacito:schema:conversational:agent-status:v1",
			Source:     "keeper",
			TenantID:   tenantID,
			OccurredAt: time.Now().UTC().Format(time.RFC3339Nano),
			Payload:    []byte(`{"status":"online"}`),
		}
		err = s.publisher.Publish(ctx, subject, evt)
		if err != nil {
			logger.Warn().Err(err).Str("subject", subject).Msg("failed to publish online status event")
		} else {
			logger.Trace().Str("subject", subject).Msg("published online status event successfully")
		}
	}

	// Update cache
	cacheKey := fmt.Sprintf("communities:%s:agents:%s", commIDStr, agentIDStr)
	err = s.cache.Set(ctx, cacheKey, &card, 45*time.Second)
	if err != nil {
		logger.Warn().Err(err).Str("key", cacheKey).Msg("failed to cache agent card in Redis")
	} else {
		logger.Trace().Str("key", cacheKey).Msg("agent card cached successfully")
	}

	return nil
}
