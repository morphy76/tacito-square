package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/morphy76/tacito-square/internal/keeper/application/ports/outbound"
	sharedports "github.com/morphy76/tacito-square/internal/shared/ports/outbound"
	"github.com/morphy76/tacito-square/pkg/events"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// RegistryPruner runs a background loop checking for stale registrations and setting them offline.
type RegistryPruner struct {
	agentRepo  outbound.AgentRepository
	cache      sharedports.Cache
	publisher  outbound.EventPublisher
	interval   time.Duration
	logger     zerolog.Logger
	tracer     trace.Tracer
	running    bool
	cancelCtx  context.Context
	cancelFunc context.CancelFunc
	mu         sync.Mutex
}

// NewRegistryPruner creates a new RegistryPruner instance.
func NewRegistryPruner(
	agentRepo outbound.AgentRepository,
	cache sharedports.Cache,
	publisher outbound.EventPublisher,
	logger zerolog.Logger,
) *RegistryPruner {
	return &RegistryPruner{
		agentRepo: agentRepo,
		cache:     cache,
		publisher: publisher,
		interval:  10 * time.Second, // default interval
		logger:    logger.With().Str("component", "registry_pruner").Logger(),
		tracer:    otel.Tracer("registry_pruner"),
	}
}

// SetInterval configures the pruning scan interval.
func (p *RegistryPruner) SetInterval(d time.Duration) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.interval = d
}

// Start launches the background pruner ticker loop.
func (p *RegistryPruner) Start(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.running {
		return nil
	}

	p.cancelCtx, p.cancelFunc = context.WithCancel(ctx)
	p.running = true

	go p.loop()

	p.logger.Info().Dur("interval", p.interval).Msg("registry pruner started")
	return nil
}

// Stop stops the background loop.
func (p *RegistryPruner) Stop() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.running {
		return nil
	}

	p.cancelFunc()
	p.running = false
	p.logger.Info().Msg("registry pruner stopped")
	return nil
}

func (p *RegistryPruner) loop() {
	p.mu.Lock()
	interval := p.interval
	p.mu.Unlock()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-p.cancelCtx.Done():
			return
		case <-ticker.C:
			p.prune()
		}
	}
}

func (p *RegistryPruner) prune() {
	p.logger.Trace().Msg("starting registry pruning cycle")
	ctx, span := p.tracer.Start(p.cancelCtx, "registry.pruner.tick", trace.WithSpanKind(trace.SpanKindInternal))
	defer span.End()

	// 1. Prune DB (sets status offline and deletes registrations older than 30s)
	prunedRefs, err := p.agentRepo.PruneStaleRegistrations(ctx, 30*time.Second)
	if err != nil {
		p.logger.Error().Err(err).Msg("failed to prune stale registrations in database")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return
	}

	if len(prunedRefs) == 0 {
		p.logger.Trace().Msg("no stale registrations to prune")
		return
	}

	p.logger.Info().Int("count", len(prunedRefs)).Msg("pruned stale registrations successfully")

	for _, ref := range prunedRefs {
		p.logger.Trace().
			Str("agent_id", ref.AgentID).
			Str("community_id", ref.CommunityID).
			Msg("handling stale agent eviction and status broadcast")

		// 2. Invalidate card from cache
		cacheKey := fmt.Sprintf("communities:%s:agents:%s", ref.CommunityID, ref.AgentID)
		err = p.cache.Invalidate(ctx, cacheKey)
		if err != nil {
			p.logger.Warn().Err(err).Str("key", cacheKey).Msg("failed to invalidate stale agent card from cache")
		} else {
			p.logger.Trace().Str("key", cacheKey).Msg("evicted stale agent card from cache")
		}

		// 3. Broadcast status change event to NATS
		subject := fmt.Sprintf("ts.community.%s.agent.%s.status", ref.CommunityID, ref.AgentID)
		evt := events.DomainEvent{
			EventID:    uuid.New().String(),
			SchemaRef:  "urn:tacito:schema:conversational:agent-status:v1",
			Source:     "keeper",
			TenantID:   "default", // tenant is dynamically propagated or default
			OccurredAt: time.Now().UTC().Format(time.RFC3339Nano),
			Payload:    []byte(`{"status":"offline"}`),
		}

		err = p.publisher.Publish(ctx, subject, evt)
		if err != nil {
			p.logger.Warn().Err(err).Str("subject", subject).Msg("failed to publish offline status event")
		} else {
			p.logger.Trace().Str("subject", subject).Msg("published offline status event successfully")
		}
	}
}
