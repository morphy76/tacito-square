package nats

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/morphy76/tacito-square/internal/keeper/application/ports/outbound"
	"github.com/morphy76/tacito-square/internal/shared/observability"
	sharedports "github.com/morphy76/tacito-square/internal/shared/ports/outbound"
	"github.com/morphy76/tacito-square/internal/shared/tenant"
	"github.com/morphy76/tacito-square/pkg/agentcard"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// RegistryHandler handles NATS request-reply queries for community registries.
type RegistryHandler struct {
	nc        *nats.Conn
	agentRepo outbound.AgentRepository
	cache     sharedports.Cache
	logger    zerolog.Logger
	sub       *nats.Subscription
	tracer    trace.Tracer
}

// NewRegistryHandler creates a new RegistryHandler.
func NewRegistryHandler(
	nc *nats.Conn,
	agentRepo outbound.AgentRepository,
	cache sharedports.Cache,
	logger zerolog.Logger,
) *RegistryHandler {
	return &RegistryHandler{
		nc:        nc,
		agentRepo: agentRepo,
		cache:     cache,
		logger:    logger.With().Str("component", "registry_handler").Logger(),
		tracer:    otel.Tracer("registry_handler"),
	}
}

// Start subscribes to the registry request subject.
func (h *RegistryHandler) Start(ctx context.Context) error {
	subject := "ts.community.*.registry.request"
	handler := observability.WrapNATSHandler("registry.request", h.logger, h.handleRequest)
	sub, err := h.nc.Subscribe(subject, handler)
	if err != nil {
		return fmt.Errorf("subscribe to registry requests: %w", err)
	}
	h.sub = sub
	h.logger.Info().Str("subject", subject).Msg("registry request handler started")
	return nil
}

// Stop unsubscribes from the subject.
func (h *RegistryHandler) Stop() error {
	if h.sub != nil {
		if err := h.sub.Drain(); err != nil {
			return fmt.Errorf("drain registry subscription: %w", err)
		}
		h.sub = nil
	}
	h.logger.Info().Msg("registry request handler stopped")
	return nil
}

func (h *RegistryHandler) handleRequest(ctx context.Context, logger zerolog.Logger, msg *nats.Msg) error {
	logger.Trace().Msg("registry request received")
	parts := strings.Split(msg.Subject, ".")
	if len(parts) != 5 {
		logger.Warn().Str("subject", msg.Subject).Msg("malformed registry request subject, skipping")
		return nil
	}

	commIDStr := parts[2]
	commUUID, err := uuid.Parse(commIDStr)
	if err != nil {
		logger.Warn().Str("community_id", commIDStr).Msg("invalid community uuid in subject, skipping")
		return nil
	}

	// Resolve tenant from NATS headers or default
	tenantID := msg.Header.Get("X-Tacito-Tenant")
	if tenantID == "" {
		tenantID = "default"
	}
	ten, err := tenant.New(tenantID, "")
	if err != nil {
		logger.Warn().Err(err).Str("tenant_id", tenantID).Msg("invalid tenant ID format, ignoring")
		return nil
	}
	ctx = tenant.ContextWithTenant(ctx, ten)

	ctx, span := h.tracer.Start(ctx, "registry.resolve", trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	// Try fetching from cache
	cacheKey := fmt.Sprintf("communities:%s:registry", commIDStr)
	var cards []*agentcard.AgentCard
	err = h.cache.Get(ctx, cacheKey, &cards)
	if err == nil {
		logger.Trace().Str("community_id", commIDStr).Msg("registry cache hit")
	} else {
		logger.Trace().Str("community_id", commIDStr).Err(err).Msg("registry cache miss, fetching from database")
		cards, _, err = h.agentRepo.GetActiveRegistrationsByCommunity(ctx, commUUID)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return fmt.Errorf("get active registrations: %w", err)
		}

		// Cache for a short duration to prevent DB load under spike conditions
		if errCache := h.cache.Set(ctx, cacheKey, cards, 10*time.Second); errCache != nil {
			logger.Warn().Err(errCache).Str("key", cacheKey).Msg("failed to cache registry cards list")
		}
	}

	respBytes, err := json.Marshal(cards)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("marshal registry response: %w", err)
	}

	replyMsg := &nats.Msg{
		Subject: msg.Reply,
		Data:    respBytes,
		Header:  nats.Header{},
	}
	replyMsg.Header.Set("X-Tacito-Tenant", tenantID)

	if err := h.nc.PublishMsg(replyMsg); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("publish registry response: %w", err)
	}

	logger.Trace().Str("community_id", commIDStr).Msg("registry response published")
	return nil
}
