package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/morphy76/tacito-square/internal/agent/application/ports/outbound"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type MockToolArgs struct{}

func handleMockTool(ctx context.Context, req *mcp.CallToolRequest, args MockToolArgs) (*mcp.CallToolResult, any, error) {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: "Mock tool execution successful!",
			},
		},
	}, nil, nil
}

func handleFailingTool(ctx context.Context, req *mcp.CallToolRequest, args MockToolArgs) (*mcp.CallToolResult, any, error) {
	return nil, nil, errors.New("simulated tool failure")
}

func TestMCPAdapter_ListAllowedTools_Whitelisting(t *testing.T) {
	t1, t2 := mcp.NewInMemoryTransports()

	server := mcp.NewServer(
		&mcp.Implementation{Name: "mock-server", Version: "1.0.0"},
		nil,
	)

	// Add tools to the mock server using official mcp.AddTool function
	mcp.AddTool(server, &mcp.Tool{
		Name:        "tool-a",
		Description: "Tool A Description",
	}, handleMockTool)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "tool-b",
		Description: "Tool B Description",
	}, handleMockTool)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "tool-c",
		Description: "Tool C Description",
	}, handleMockTool)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		_, _ = server.Connect(ctx, t1, nil)
	}()

	clients := []outbound.MCPClientInfo{
		{
			Name:         "test-client",
			Transport:    "stdio",
			AllowedTools: []string{"tool-a", "tool-c"},
		},
	}

	adapter := NewMCPAdapterWithFactory(clients, 5*time.Second, func(info outbound.MCPClientInfo) (mcp.Transport, error) {
		return t2, nil
	})

	// Run ListAllowedTools
	tools, err := adapter.ListAllowedTools(ctx)
	require.NoError(t, err)
	require.Len(t, tools, 2)
	assert.Equal(t, "tool-a", tools[0].Name)
	assert.Equal(t, "tool-c", tools[1].Name)
}

func TestMCPAdapter_Execute_Timeout(t *testing.T) {
	t1, t2 := mcp.NewInMemoryTransports()
	server := mcp.NewServer(
		&mcp.Implementation{Name: "mock-server", Version: "1.0.0"},
		nil,
	)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "slow-tool",
		Description: "Slow Tool",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args MockToolArgs) (*mcp.CallToolResult, any, error) {
		select {
		case <-time.After(1 * time.Second):
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: "Succeeded after delay"},
				},
			}, nil, nil
		case <-ctx.Done():
			return nil, nil, ctx.Err()
		}
	})

	ctx := context.Background()

	go func() {
		_, _ = server.Connect(ctx, t1, nil)
	}()

	clients := []outbound.MCPClientInfo{
		{
			Name:         "test-client",
			Transport:    "stdio",
			AllowedTools: []string{"slow-tool"},
		},
	}

	// Set a very short timeout of 50ms
	adapter := NewMCPAdapterWithFactory(clients, 50*time.Millisecond, func(info outbound.MCPClientInfo) (mcp.Transport, error) {
		return t2, nil
	})

	// Assert that execution returns a fallback response due to timeout
	resp, err := adapter.Execute(ctx, "slow-tool", map[string]any{})
	require.NoError(t, err)
	assert.Contains(t, resp, "Tool provider temporarily unavailable.")
}

func TestMCPAdapter_Execute_CircuitBreaker(t *testing.T) {
	t1, t2 := mcp.NewInMemoryTransports()
	server := mcp.NewServer(
		&mcp.Implementation{Name: "mock-server", Version: "1.0.0"},
		nil,
	)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "failing-tool",
		Description: "Failing Tool",
	}, handleFailingTool)

	ctx := context.Background()
	go func() {
		_, _ = server.Connect(ctx, t1, nil)
	}()

	clients := []outbound.MCPClientInfo{
		{
			Name:         "test-client",
			Transport:    "stdio",
			AllowedTools: []string{"failing-tool"},
		},
	}

	adapter := NewMCPAdapterWithFactory(clients, 5*time.Second, func(info outbound.MCPClientInfo) (mcp.Transport, error) {
		return t2, nil
	})

	// Assert that after 5 failures, the circuit breaker trips and returns a fallback JSON
	for i := 0; i < 5; i++ {
		resp, err := adapter.Execute(ctx, "failing-tool", map[string]any{})
		require.NoError(t, err)
		assert.Contains(t, resp, "Tool provider temporarily unavailable.")
	}

	// 6th call should immediately return the fallback error block
	resp, err := adapter.Execute(ctx, "failing-tool", map[string]any{})
	require.NoError(t, err)

	var errResp map[string]string
	err = json.Unmarshal([]byte(resp), &errResp)
	require.NoError(t, err)
	assert.Equal(t, "Tool provider temporarily unavailable.", errResp["error"])
}
