package cache_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	agentcache "github.com/morphy76/tacito-square/internal/agent/adapters/outbound/cache"
	"github.com/morphy76/tacito-square/pkg/agentcard"
	"github.com/morphy76/tacito-square/pkg/events"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func startTestNatsServer(t *testing.T) (*server.Server, *nats.Conn) {
	opts := &server.Options{
		Host: "127.0.0.1",
		Port: -1,
	}
	ns, err := server.NewServer(opts)
	require.NoError(t, err)

	go ns.Start()
	if !ns.ReadyForConnections(2 * time.Second) {
		t.Fatal("NATS server not ready for connections")
	}

	nc, err := nats.Connect(ns.ClientURL())
	require.NoError(t, err)

	return ns, nc
}

func TestClientCache_ReactiveUpdates(t *testing.T) {
	ns, nc := startTestNatsServer(t)
	defer ns.Shutdown()
	defer nc.Close()

	logger := zerolog.New(nil)
	communityID := "comm-123"
	tenantID := "tenant-456"

	cache := agentcache.NewClientCache(nc, communityID, tenantID, logger)
	err := cache.Start(context.Background())
	require.NoError(t, err)
	defer cache.Stop()

	// 1. Simulate a heartbeat for agent-1
	card := &agentcard.AgentCard{
		Name:        "agent-alpha",
		Description: "Alpha agent",
		URL:         "http://agent-alpha",
		Version:     "1.0.0",
	}
	evt, err := events.NewDomainEvent(
		events.SchemaInfrastructureAgentHeartbeat,
		"agent/agent-1",
		tenantID,
		card,
	)
	require.NoError(t, err)
	evtBytes, err := json.Marshal(evt)
	require.NoError(t, err)

	hbSubject := fmt.Sprintf("ts.community.%s.agent.agent-1.heartbeat", communityID)
	msg := nats.NewMsg(hbSubject)
	msg.Data = evtBytes
	msg.Header.Set("X-Tacito-Schema", evt.SchemaRef)
	msg.Header.Set("X-Tacito-Source", evt.Source)
	msg.Header.Set("X-Tacito-Tenant", evt.TenantID)
	msg.Header.Set("X-Tacito-Event-ID", evt.EventID)
	msg.Header.Set("X-Tacito-Occurred", evt.OccurredAt)

	err = nc.PublishMsg(msg)
	require.NoError(t, err)

	// Wait for subscription processing
	time.Sleep(100 * time.Millisecond)

	// Card should be resolved from cache without a request-reply call
	fetched, err := cache.GetCardByName(context.Background(), "agent-alpha")
	require.NoError(t, err)
	assert.Equal(t, card.Name, fetched.Name)
	assert.Equal(t, card.Description, fetched.Description)

	// 2. Simulate offline status change
	statusSubject := fmt.Sprintf("ts.community.%s.agent.agent-1.status", communityID)
	statusPayload := []byte(`{"status":"offline"}`)
	err = nc.Publish(statusSubject, statusPayload)
	require.NoError(t, err)

	// Wait for subscription processing
	time.Sleep(100 * time.Millisecond)

	// Set up NATS responder to intercept request-reply call on cache miss now that it's evicted
	requestSubject := fmt.Sprintf("ts.community.%s.registry.request", communityID)
	reqSub, err := nc.Subscribe(requestSubject, func(msg *nats.Msg) {
		// Respond with empty registry
		resp, _ := json.Marshal([]*agentcard.AgentCard{})
		msg.Respond(resp)
	})
	require.NoError(t, err)
	defer reqSub.Unsubscribe()

	// Try fetching again; should result in a cache miss, query NATS request, and return error since we responded empty
	_, err = cache.GetCardByName(context.Background(), "agent-alpha")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found in registry")
}

func TestClientCache_RegistryRequestReply(t *testing.T) {
	ns, nc := startTestNatsServer(t)
	defer ns.Shutdown()
	defer nc.Close()

	logger := zerolog.New(nil)
	communityID := "comm-123"
	tenantID := "tenant-456"

	cache := agentcache.NewClientCache(nc, communityID, tenantID, logger)
	err := cache.Start(context.Background())
	require.NoError(t, err)
	defer cache.Stop()

	// Set up NATS request-reply responder
	requestSubject := fmt.Sprintf("ts.community.%s.registry.request", communityID)
	card := &agentcard.AgentCard{
		Name:        "agent-beta",
		Description: "Beta agent",
		URL:         "http://agent-beta",
		Version:     "2.0.0",
	}
	reqSub, err := nc.Subscribe(requestSubject, func(msg *nats.Msg) {
		assert.Equal(t, tenantID, msg.Header.Get("X-Tacito-Tenant"))
		resp, _ := json.Marshal([]*agentcard.AgentCard{card})
		msg.Respond(resp)
	})
	require.NoError(t, err)
	defer reqSub.Unsubscribe()

	// Retrieve card; should hit cache miss, request from responder, populate cache, and return
	fetched, err := cache.GetCardByName(context.Background(), "agent-beta")
	require.NoError(t, err)
	assert.Equal(t, card.Name, fetched.Name)
	assert.Equal(t, card.Version, fetched.Version)

	// Get all cards; should be returned from cache directly without NATS request (due to short-term cache lock of 30s)
	list, err := cache.GetCards(context.Background())
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, "agent-beta", list[0].Name)
}

func TestClientCache_ResolveAgentID(t *testing.T) {
	ns, nc := startTestNatsServer(t)
	defer ns.Shutdown()
	defer nc.Close()

	logger := zerolog.New(nil)
	communityID := "comm-123"
	tenantID := "tenant-456"

	cache := agentcache.NewClientCache(nc, communityID, tenantID, logger)
	err := cache.Start(context.Background())
	require.NoError(t, err)
	defer cache.Stop()

	// Simulate a heartbeat for agent-uuid-123
	card := &agentcard.AgentCard{
		Name:        "agent-gamma",
		Description: "Gamma agent",
		URL:         "http://agent-gamma",
		Version:     "1.0.0",
	}
	evt, err := events.NewDomainEvent(
		events.SchemaInfrastructureAgentHeartbeat,
		"agent/agent-uuid-123",
		tenantID,
		card,
	)
	require.NoError(t, err)
	evtBytes, err := json.Marshal(evt)
	require.NoError(t, err)

	hbSubject := fmt.Sprintf("ts.community.%s.agent.agent-uuid-123.heartbeat", communityID)
	msg := nats.NewMsg(hbSubject)
	msg.Data = evtBytes
	msg.Header.Set("X-Tacito-Schema", evt.SchemaRef)
	msg.Header.Set("X-Tacito-Source", evt.Source)
	msg.Header.Set("X-Tacito-Tenant", evt.TenantID)
	msg.Header.Set("X-Tacito-Event-ID", evt.EventID)
	msg.Header.Set("X-Tacito-Occurred", evt.OccurredAt)

	err = nc.PublishMsg(msg)
	require.NoError(t, err)

	// Wait for subscription processing
	time.Sleep(100 * time.Millisecond)

	// Resolve agent-gamma -> should return agent-uuid-123
	resolvedID, err := cache.ResolveAgentID(context.Background(), "agent-gamma")
	require.NoError(t, err)
	assert.Equal(t, "agent-uuid-123", resolvedID)

	// Resolve non-existent agent -> should fail
	_, err = cache.ResolveAgentID(context.Background(), "agent-non-existent")
	assert.Error(t, err)
}

func TestClientCache_ResolveAgentID_RefreshCacheMiss(t *testing.T) {
	ns, nc := startTestNatsServer(t)
	defer ns.Shutdown()
	defer nc.Close()

	logger := zerolog.New(nil)
	communityID := "comm-123"
	tenantID := "tenant-456"

	cache := agentcache.NewClientCache(nc, communityID, tenantID, logger)
	err := cache.Start(context.Background())
	require.NoError(t, err)
	defer cache.Stop()

	// Set up NATS responder to simulate registry request-reply
	requestSubject := fmt.Sprintf("ts.community.%s.registry.request", communityID)
	card := &agentcard.AgentCard{
		AgentID:     "agent-uuid-999",
		Name:        "agent-delta",
		Description: "Delta agent",
		URL:         "http://agent-delta",
		Version:     "1.0.0",
	}
	reqSub, err := nc.Subscribe(requestSubject, func(msg *nats.Msg) {
		resp, _ := json.Marshal([]*agentcard.AgentCard{card})
		msg.Respond(resp)
	})
	require.NoError(t, err)
	defer reqSub.Unsubscribe()

	// Resolve agent-delta -> should result in a cache miss, trigger Refresh, and successfully resolve to agent-uuid-999
	resolvedID, err := cache.ResolveAgentID(context.Background(), "agent-delta")
	require.NoError(t, err)
	assert.Equal(t, "agent-uuid-999", resolvedID)
}
