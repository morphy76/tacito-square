package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/morphy76/tacito-square/internal/agent/application/ports/outbound"
	"github.com/morphy76/tacito-square/pkg/agentcard"
	"github.com/morphy76/tacito-square/pkg/events"
	natsclient "github.com/nats-io/nats.go"
	"github.com/rs/zerolog"
)

var _ outbound.AgentDiscovery = (*ClientCache)(nil)

// ClientCache implements in-memory caching of community agent cards on the Agent client-side.
type ClientCache struct {
	nc          *natsclient.Conn
	communityID string
	tenantID    string
	logger      zerolog.Logger
	mu          sync.RWMutex
	cards       map[string]*agentcard.AgentCard // key: Name
	idToName    map[string]string               // key: agent_id
	lastFetched time.Time
	subStatus   *natsclient.Subscription
	subHeart    *natsclient.Subscription
}

// NewClientCache creates a new ClientCache.
func NewClientCache(
	nc *natsclient.Conn,
	communityID string,
	tenantID string,
	logger zerolog.Logger,
) *ClientCache {
	return &ClientCache{
		nc:          nc,
		communityID: communityID,
		tenantID:    tenantID,
		logger:      logger.With().Str("component", "client_cache").Logger(),
		cards:       make(map[string]*agentcard.AgentCard),
		idToName:    make(map[string]string),
	}
}

// Start subscribes to community wildcard heartbeats and status events.
func (c *ClientCache) Start(ctx context.Context) error {
	statusSubject := fmt.Sprintf("ts.community.%s.agent.*.status", c.communityID)
	subStatus, err := c.nc.Subscribe(statusSubject, func(msg *natsclient.Msg) {
		c.handleStatusMsg(msg)
	})
	if err != nil {
		return fmt.Errorf("subscribe to status updates: %w", err)
	}
	c.subStatus = subStatus

	heartbeatSubject := fmt.Sprintf("ts.community.%s.agent.*.heartbeat", c.communityID)
	subHeart, err := c.nc.Subscribe(heartbeatSubject, func(msg *natsclient.Msg) {
		c.handleHeartbeatMsg(msg)
	})
	if err != nil {
		_ = c.subStatus.Unsubscribe()
		return fmt.Errorf("subscribe to heartbeats: %w", err)
	}
	c.subHeart = subHeart

	c.logger.Info().
		Str("community_id", c.communityID).
		Msg("client-side registry cache subscriber started")
	return nil
}

// Stop unsubscribes from NATS events.
func (c *ClientCache) Stop() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.subStatus != nil {
		_ = c.subStatus.Unsubscribe()
		c.subStatus = nil
	}
	if c.subHeart != nil {
		_ = c.subHeart.Unsubscribe()
		c.subHeart = nil
	}
	c.logger.Info().Msg("client-side registry cache subscriber stopped")
	return nil
}

func (c *ClientCache) handleStatusMsg(msg *natsclient.Msg) {
	parts := strings.Split(msg.Subject, ".")
	if len(parts) != 6 {
		return
	}
	agentID := parts[4]

	type statusPayload struct {
		Status string `json:"status"`
	}
	var payload statusPayload
	if err := json.Unmarshal(msg.Data, &payload); err != nil {
		c.logger.Warn().Err(err).Msg("failed to unmarshal agent status payload")
		return
	}

	if payload.Status == "offline" {
		c.mu.Lock()
		defer c.mu.Unlock()
		if name, ok := c.idToName[agentID]; ok {
			c.logger.Debug().Str("agent_id", agentID).Str("name", name).Msg("evicting offline agent from local cache")
			delete(c.cards, name)
			delete(c.idToName, agentID)
		}
	}
}

func (c *ClientCache) handleHeartbeatMsg(msg *natsclient.Msg) {
	parts := strings.Split(msg.Subject, ".")
	if len(parts) != 6 {
		return
	}
	agentID := parts[4]

	var evt events.DomainEvent
	if err := json.Unmarshal(msg.Data, &evt); err != nil {
		c.logger.Warn().Err(err).Msg("failed to unmarshal heartbeat domain event envelope")
		return
	}

	if c.tenantID != "" && evt.TenantID != c.tenantID {
		c.logger.Warn().
			Str("event_tenant_id", evt.TenantID).
			Str("configured_tenant_id", c.tenantID).
			Msg("tenant mismatch: ignoring heartbeat")
		return
	}

	if evt.SchemaRef != events.SchemaInfrastructureAgentHeartbeat {
		c.logger.Warn().Str("schema_ref", evt.SchemaRef).Msg("unexpected schema in heartbeat message")
		return
	}

	var card agentcard.AgentCard
	if err := json.Unmarshal(evt.Payload, &card); err != nil {
		c.logger.Warn().Err(err).Msg("failed to unmarshal agent card from heartbeat payload")
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	c.cards[card.Name] = &card
	c.idToName[agentID] = card.Name
	c.logger.Trace().Str("agent_id", agentID).Str("name", card.Name).Msg("cached agent card from heartbeat")
}

// GetCardByName retrieves a card by agent name. It fetches from registry on cache miss.
func (c *ClientCache) GetCardByName(ctx context.Context, name string) (*agentcard.AgentCard, error) {
	c.mu.RLock()
	card, ok := c.cards[name]
	c.mu.RUnlock()
	if ok {
		return card, nil
	}

	if err := c.Refresh(ctx); err != nil {
		return nil, err
	}

	c.mu.RLock()
	defer c.mu.RUnlock()
	card, ok = c.cards[name]
	if !ok {
		return nil, fmt.Errorf("agent card for %s not found in registry", name)
	}
	return card, nil
}

// GetCards retrieves all active agent cards in the community registry.
func (c *ClientCache) GetCards(ctx context.Context) ([]*agentcard.AgentCard, error) {
	c.mu.RLock()
	if time.Since(c.lastFetched) < 30*time.Second && len(c.cards) > 0 {
		list := make([]*agentcard.AgentCard, 0, len(c.cards))
		for _, card := range c.cards {
			list = append(list, card)
		}
		c.mu.RUnlock()
		return list, nil
	}
	c.mu.RUnlock()

	if err := c.Refresh(ctx); err != nil {
		return nil, err
	}

	c.mu.RLock()
	defer c.mu.RUnlock()
	list := make([]*agentcard.AgentCard, 0, len(c.cards))
	for _, card := range c.cards {
		list = append(list, card)
	}
	return list, nil
}

// Refresh issues a NATS request to sync the local cache with the remote registry.
func (c *ClientCache) Refresh(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if time.Since(c.lastFetched) < 1*time.Second {
		return nil
	}

	subject := fmt.Sprintf("ts.community.%s.registry.request", c.communityID)
	msg := &natsclient.Msg{
		Subject: subject,
		Header:  natsclient.Header{},
	}
	msg.Header.Set("X-Tacito-Tenant", c.tenantID)

	reply, err := c.nc.RequestMsgWithContext(ctx, msg)
	if err != nil {
		return fmt.Errorf("registry nats request: %w", err)
	}

	var cards []*agentcard.AgentCard
	if err := json.Unmarshal(reply.Data, &cards); err != nil {
		return fmt.Errorf("unmarshal registry response: %w", err)
	}

	newNames := make(map[string]bool)
	for _, card := range cards {
		c.cards[card.Name] = card
		if card.AgentID != "" {
			c.idToName[card.AgentID] = card.Name
		}
		newNames[card.Name] = true
	}

	for name := range c.cards {
		if !newNames[name] {
			delete(c.cards, name)
			for id, mappedName := range c.idToName {
				if mappedName == name {
					delete(c.idToName, id)
				}
			}
		}
	}

	c.lastFetched = time.Now()
	c.logger.Debug().Int("count", len(cards)).Msg("client cache refreshed from keeper registry")
	return nil
}

// ResolveAgentID looks up the unique agent_id UUID mapped to a human-readable agent name in the cache.
func (c *ClientCache) ResolveAgentID(ctx context.Context, name string) (string, error) {
	c.mu.RLock()
	for id, mappedName := range c.idToName {
		if mappedName == name {
			c.mu.RUnlock()
			return id, nil
		}
	}
	c.mu.RUnlock()

	// Cache miss: refresh from registry
	if err := c.Refresh(ctx); err != nil {
		return "", err
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	for id, mappedName := range c.idToName {
		if mappedName == name {
			return id, nil
		}
	}
	return "", fmt.Errorf("agent ID for name %s not found in cache", name)
}

