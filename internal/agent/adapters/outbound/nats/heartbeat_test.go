package nats_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	agentnats "github.com/morphy76/tacito-square/internal/agent/adapters/outbound/nats"
	agentmodel "github.com/morphy76/tacito-square/internal/agent/domain/model"
	"github.com/morphy76/tacito-square/pkg/agentcard"
	"github.com/morphy76/tacito-square/pkg/events"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog"
	"github.com/spf13/viper"
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

type mockMCPExecutor struct {
	tools []agentmodel.ToolDefinition
	err   error
}

func (m *mockMCPExecutor) ListAllowedTools(ctx context.Context) ([]agentmodel.ToolDefinition, error) {
	return m.tools, m.err
}

func (m *mockMCPExecutor) Execute(ctx context.Context, toolName string, arguments map[string]any) (string, error) {
	return "", nil
}

func (m *mockMCPExecutor) Close(ctx context.Context) error {
	return nil
}

func TestHeartbeatPublisher_PublishesCard(t *testing.T) {
	ns, nc := startTestNatsServer(t)
	defer ns.Shutdown()
	defer nc.Close()

	logger := zerolog.New(nil)
	cfg := viper.New()
	cfg.Set("name", "test-agent-name")
	cfg.Set("description", "A test agent")
	cfg.Set("url", "http://test-agent-url")
	cfg.Set("tenant.id", "test-tenant")
	cfg.Set("id", "test-agent")
	cfg.Set("community.ref", "test-comm")

	// Set up a subscriber on NATS heartbeat subject
	subject := "ts.community.test-comm.agent.test-agent.heartbeat"
	ch := make(chan []byte, 1)
	sub, err := nc.Subscribe(subject, func(msg *nats.Msg) {
		ch <- msg.Data
	})
	require.NoError(t, err)
	defer func() { _ = sub.Unsubscribe() }()

	// Initialize the heartbeat publisher with a fast tick interval (50ms)
	pub := agentnats.NewHeartbeatPublisher(nc, cfg, "0.1.0", nil, logger)
	pub.SetInterval(50 * time.Millisecond)

	ctx := context.Background()
	err = pub.Start(ctx)
	require.NoError(t, err)
	defer func() { _ = pub.Stop() }()

	// Assert message is received and valid
	select {
	case data := <-ch:
		var evt events.DomainEvent
		err := json.Unmarshal(data, &evt)
		require.NoError(t, err)
		assert.Equal(t, events.SchemaInfrastructureAgentHeartbeat, evt.SchemaRef)

		var card agentcard.AgentCard
		err = json.Unmarshal(evt.Payload, &card)
		require.NoError(t, err)
		assert.Equal(t, "test-agent-name", card.Name)
		assert.Equal(t, "A test agent", card.Description)
		assert.Equal(t, "http://test-agent-url", card.URL)
		assert.Equal(t, "0.1.0", card.Version)
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for heartbeat message")
	}
}

func TestHeartbeatPublisher_IncludesMCPTools(t *testing.T) {
	ns, nc := startTestNatsServer(t)
	defer ns.Shutdown()
	defer nc.Close()

	logger := zerolog.New(nil)
	cfg := viper.New()
	cfg.Set("name", "test-agent-name")
	cfg.Set("description", "A test agent")
	cfg.Set("id", "test-agent")
	cfg.Set("community.ref", "test-comm")

	mcpMock := &mockMCPExecutor{
		tools: []agentmodel.ToolDefinition{
			{
				Name:        "get_weather",
				Description: "Gets current weather conditions",
			},
		},
	}

	subject := "ts.community.test-comm.agent.test-agent.heartbeat"
	ch := make(chan []byte, 1)
	sub, err := nc.Subscribe(subject, func(msg *nats.Msg) {
		ch <- msg.Data
	})
	require.NoError(t, err)
	defer func() { _ = sub.Unsubscribe() }()

	pub := agentnats.NewHeartbeatPublisher(nc, cfg, "0.1.0", mcpMock, logger)
	pub.SetInterval(50 * time.Millisecond)

	ctx := context.Background()
	err = pub.Start(ctx)
	require.NoError(t, err)
	defer func() { _ = pub.Stop() }()

	select {
	case data := <-ch:
		var evt events.DomainEvent
		err := json.Unmarshal(data, &evt)
		require.NoError(t, err)
		assert.Equal(t, events.SchemaInfrastructureAgentHeartbeat, evt.SchemaRef)

		var card agentcard.AgentCard
		err = json.Unmarshal(evt.Payload, &card)
		require.NoError(t, err)
		assert.Len(t, card.Skills, 1)
		assert.Equal(t, "tool-get_weather", card.Skills[0].ID)
		assert.Equal(t, "get_weather", card.Skills[0].Name)
		assert.Equal(t, "Gets current weather conditions", card.Skills[0].Description)
		assert.Contains(t, card.Skills[0].Tags, "mcp-tool")
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for heartbeat message")
	}
}

func TestHeartbeatPublisher_ParsesSystemPrompt(t *testing.T) {
	ns, nc := startTestNatsServer(t)
	defer ns.Shutdown()
	defer nc.Close()

	logger := zerolog.New(nil)
	cfg := viper.New()
	cfg.Set("name", "test-agent-name")
	cfg.Set("id", "test-agent")
	cfg.Set("community.ref", "test-comm")
	cfg.Set("port", "8081")
	cfg.Set("system.prompt", `{"description":"Dynamic description from prompt","directives":"Be helpful","skills":[{"name":"DynamicSkill1","description":"A dynamic skill description","content":"some content"}]}`)
	cfg.Set("capabilities.pushNotifications", true)
	cfg.Set("capabilities.stateTransitionHistory", true)

	subject := "ts.community.test-comm.agent.test-agent.heartbeat"
	ch := make(chan []byte, 1)
	sub, err := nc.Subscribe(subject, func(msg *nats.Msg) {
		ch <- msg.Data
	})
	require.NoError(t, err)
	defer func() { _ = sub.Unsubscribe() }()

	pub := agentnats.NewHeartbeatPublisher(nc, cfg, "0.1.0", nil, logger)
	pub.SetInterval(50 * time.Millisecond)

	ctx := context.Background()
	err = pub.Start(ctx)
	require.NoError(t, err)
	defer func() { _ = pub.Stop() }()

	select {
	case data := <-ch:
		var evt events.DomainEvent
		err := json.Unmarshal(data, &evt)
		require.NoError(t, err)
		assert.Equal(t, events.SchemaInfrastructureAgentHeartbeat, evt.SchemaRef)

		var card agentcard.AgentCard
		err = json.Unmarshal(evt.Payload, &card)
		require.NoError(t, err)
		assert.Equal(t, "test-agent-name", card.Name)
		assert.Equal(t, "Dynamic description from prompt", card.Description)
		assert.Equal(t, "http://localhost:8081", card.URL)
		assert.Equal(t, "0.1.0", card.Version)
		assert.True(t, card.Capabilities.PushNotifications)
		assert.True(t, card.Capabilities.StateTransitionHistory)
		
		// Assert dynamic skill is parsed
		require.Len(t, card.Skills, 1)
		assert.Equal(t, "skill-DynamicSkill1", card.Skills[0].ID)
		assert.Equal(t, "DynamicSkill1", card.Skills[0].Name)
		assert.Equal(t, "A dynamic skill description", card.Skills[0].Description)
		assert.Contains(t, card.Skills[0].Tags, "dynamic-skill")
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for heartbeat message")
	}
}

