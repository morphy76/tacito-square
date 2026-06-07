package cache_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	agentcache "github.com/morphy76/tacito-square/internal/agent/adapters/outbound/cache"
	"github.com/morphy76/tacito-square/pkg/agentcard"
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
	cardBytes, err := json.Marshal(card)
	require.NoError(t, err)

	hbSubject := fmt.Sprintf("ts.community.%s.agent.agent-1.heartbeat", communityID)
	err = nc.Publish(hbSubject, cardBytes)
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
