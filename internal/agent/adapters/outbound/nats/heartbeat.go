package nats

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/morphy76/tacito-square/internal/agent/application/ports/outbound"
	"github.com/morphy76/tacito-square/internal/shared/observability"
	"github.com/morphy76/tacito-square/pkg/agentcard"
	natsclient "github.com/nats-io/nats.go"
	"github.com/rs/zerolog"
	"github.com/spf13/viper"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// HeartbeatPublisher handles the background loop that publishes the agent's card payload to NATS.
type HeartbeatPublisher struct {
	nc          *natsclient.Conn
	cfg         *viper.Viper
	agentID     string
	communityID string
	version     string
	mcpExecutor outbound.ToolExecutor
	interval    time.Duration
	logger      zerolog.Logger
	tracer      trace.Tracer
	running     bool
	cancelCtx   context.Context
	cancelFunc  context.CancelFunc
	mu          sync.Mutex
}

// NewHeartbeatPublisher creates a new HeartbeatPublisher instance.
func NewHeartbeatPublisher(
	nc *natsclient.Conn,
	cfg *viper.Viper,
	version string,
	mcpExecutor outbound.ToolExecutor,
	logger zerolog.Logger,
) *HeartbeatPublisher {
	agentID := cfg.GetString("id")
	communityID := cfg.GetString("community.ref")

	return &HeartbeatPublisher{
		nc:          nc,
		cfg:         cfg,
		agentID:     agentID,
		communityID: communityID,
		version:     version,
		mcpExecutor: mcpExecutor,
		interval:    10 * time.Second, // default interval
		logger:      logger.With().Str("component", "heartbeat_publisher").Logger(),
		tracer:      otel.Tracer("heartbeat_publisher"),
	}
}

// SetInterval configures the heartbeat tick interval.
func (p *HeartbeatPublisher) SetInterval(d time.Duration) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.interval = d
}

// Start kicks off the background heartbeat publishing loop.
func (p *HeartbeatPublisher) Start(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.running {
		return nil
	}

	p.cancelCtx, p.cancelFunc = context.WithCancel(ctx)
	p.running = true

	go p.loop()

	p.logger.Info().
		Str("agent_id", p.agentID).
		Str("community_id", p.communityID).
		Dur("interval", p.interval).
		Msg("heartbeat publisher started")

	return nil
}

// Stop halts the background heartbeat loop.
func (p *HeartbeatPublisher) Stop() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.running {
		return nil
	}

	p.cancelFunc()
	p.running = false
	p.logger.Info().Msg("heartbeat publisher stopped")
	return nil
}

func (p *HeartbeatPublisher) loop() {
	p.mu.Lock()
	interval := p.interval
	p.mu.Unlock()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Publish first heartbeat immediately
	p.publishHeartbeat()

	for {
		select {
		case <-p.cancelCtx.Done():
			return
		case <-ticker.C:
			p.publishHeartbeat()
		}
	}
}

func (p *HeartbeatPublisher) publishHeartbeat() {
	p.logger.Trace().Msg("publishHeartbeat called")
	ctx, span := p.tracer.Start(p.cancelCtx, "agent.heartbeat.publish", trace.WithSpanKind(trace.SpanKindProducer))
	defer span.End()

	card, err := p.compileCard(ctx)
	if err != nil {
		p.logger.Error().Err(err).Msg("failed to compile agent card for heartbeat")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return
	}

	cardBytes, err := json.Marshal(card)
	if err != nil {
		p.logger.Error().Err(err).Msg("failed to marshal agent card for heartbeat")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return
	}

	p.logger.Trace().Int("payload_size_bytes", len(cardBytes)).Msg("marshaled agent card successfully")

	subject := fmt.Sprintf("ts.community.%s.agent.%s.heartbeat", p.communityID, p.agentID)
	msg := natsclient.NewMsg(subject)
	msg.Data = cardBytes

	// Tenant resolution (Rule compliant - single tenant variable search)
	tenantID := p.cfg.GetString("tenant.id")
	if tenantID == "" {
		tenantID = p.cfg.GetString("tenant_id")
	}
	if tenantID == "" {
		tenantID = os.Getenv("TENANT_ID")
	}
	if tenantID != "" {
		msg.Header.Set("X-Tacito-Tenant", tenantID)
	}

	observability.InjectNATSContext(ctx, msg)

	p.logger.Trace().Str("subject", subject).Str("tenant_id", tenantID).Msg("publishing heartbeat NATS message")

	if err := p.nc.PublishMsg(msg); err != nil {
		p.logger.Error().Err(err).Str("subject", subject).Msg("failed to publish heartbeat")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	} else {
		p.logger.Trace().Str("subject", subject).Msg("published heartbeat successfully")
	}
}

func (p *HeartbeatPublisher) compileCard(ctx context.Context) (*agentcard.AgentCard, error) {
	p.logger.Trace().Msg("compiling agent card details")
	card := &agentcard.AgentCard{
		Name:        p.cfg.GetString("name"),
		Description: p.cfg.GetString("description"),
		URL:         p.cfg.GetString("url"),
		Version:     p.version,
		DocumentationURL: p.cfg.GetString("documentation.url"),
		Capabilities: agentcard.AgentCardCapabilities{
			Streaming: p.cfg.GetBool("capabilities.streaming"),
		},
		Authentication: agentcard.AgentCardAuthentication{
			Schemes:     p.cfg.GetStringSlice("capabilities.auth.schemes"),
			Credentials: p.cfg.GetString("capabilities.auth.credentials"),
		},
		DefaultInputModes:  p.cfg.GetStringSlice("capabilities.input.modes"),
		DefaultOutputModes: p.cfg.GetStringSlice("capabilities.output.modes"),
		Skills:             []agentcard.AgentCardSkill{},
	}

	// Apply default capabilities if not explicitly configured
	if len(card.Authentication.Schemes) == 0 {
		card.Authentication.Schemes = []string{"Bearer"}
	}
	if len(card.DefaultInputModes) == 0 {
		card.DefaultInputModes = []string{"text/plain"}
	}
	if len(card.DefaultOutputModes) == 0 {
		card.DefaultOutputModes = []string{"text/plain"}
	}

	// Optional Provider metadata
	org := p.cfg.GetString("provider.organization")
	provURL := p.cfg.GetString("provider.url")
	if org != "" || provURL != "" {
		card.Provider = &agentcard.AgentCardProvider{
			Organization: org,
			URL:          provURL,
		}
	}

	// 1. Static skills defined in configurations (if any)
	var staticSkills []agentcard.AgentCardSkill
	if p.cfg.IsSet("skills") {
		p.logger.Trace().Msg("parsing static skills from configuration")
		if err := p.cfg.UnmarshalKey("skills", &staticSkills); err != nil {
			p.logger.Warn().Err(err).Msg("failed to unmarshal skills list from config")
		}
	}
	card.Skills = append(card.Skills, staticSkills...)

	// 2. Active tools exposed as skills (e.g. MCP tools)
	if p.mcpExecutor != nil {
		p.logger.Trace().Msg("querying active MCP tools list")
		tools, err := p.mcpExecutor.ListAllowedTools(ctx)
		if err == nil {
			p.logger.Trace().Int("tools_count", len(tools)).Msg("retrieved allowed MCP tools list")
			for _, t := range tools {
				card.Skills = append(card.Skills, agentcard.AgentCardSkill{
					ID:          "tool-" + t.Name,
					Name:        t.Name,
					Description: t.Description,
					Tags:        []string{"mcp-tool"},
				})
			}
		} else {
			p.logger.Warn().Err(err).Msg("failed to retrieve MCP tools list for agent card")
		}
	}

	p.logger.Trace().Int("total_skills_and_tools", len(card.Skills)).Msg("agent card compiled successfully")
	return card, nil
}
