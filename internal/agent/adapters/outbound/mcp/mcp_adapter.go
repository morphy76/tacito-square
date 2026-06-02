package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/morphy76/tacito-square/internal/agent/adapters/outbound/resiliency"
	"github.com/morphy76/tacito-square/internal/agent/application/ports/outbound"
)

type TransportFactory func(info outbound.MCPClientInfo) (mcp.Transport, error)

type clientSession struct {
	info         outbound.MCPClientInfo
	mcpClient    *mcp.Client
	session      *mcp.ClientSession
	cb           *resiliency.CircuitBreaker
	allowedTools map[string]bool
	cancel       context.CancelFunc
}

type MCPAdapter struct {
	mu             sync.Mutex
	clients        []outbound.MCPClientInfo
	timeout        time.Duration
	cbThreshold    int
	cbRecoveryTime time.Duration
	sessions       []*clientSession
	factory        TransportFactory
}

var _ outbound.ToolExecutor = (*MCPAdapter)(nil)

func NewMCPAdapter(clients []outbound.MCPClientInfo, timeout time.Duration) *MCPAdapter {
	return NewMCPAdapterWithFactory(clients, timeout, nil)
}

func NewMCPAdapterWithFactory(
	clients []outbound.MCPClientInfo,
	timeout time.Duration,
	factory TransportFactory,
) *MCPAdapter {
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	return &MCPAdapter{
		clients:        clients,
		timeout:        timeout,
		cbThreshold:    5,
		cbRecoveryTime: 15 * time.Second,
		factory:        factory,
	}
}

// WithCircuitBreakerParams configures custom circuit breaker parameters.
func (a *MCPAdapter) WithCircuitBreakerParams(threshold int, recoveryTime time.Duration) *MCPAdapter {
	if threshold > 0 {
		a.cbThreshold = threshold
	}
	if recoveryTime > 0 {
		a.cbRecoveryTime = recoveryTime
	}
	return a
}

func (a *MCPAdapter) init(ctx context.Context) error {
	if len(a.sessions) > 0 {
		return nil
	}

	for _, info := range a.clients {
		allowed := make(map[string]bool)
		for _, t := range info.AllowedTools {
			allowed[t] = true
		}

		cb := resiliency.NewCircuitBreaker(a.cbThreshold, a.cbRecoveryTime)

		var transport mcp.Transport
		var err error
		if a.factory != nil {
			transport, err = a.factory(info)
		} else {
			transport, err = defaultTransportFactory(info)
		}
		if err != nil {
			return fmt.Errorf("failed to create transport for %s: %w", info.Name, err)
		}

		clientCtx, clientCancel := context.WithCancel(context.Background())

		mcpClient := mcp.NewClient(
			&mcp.Implementation{Name: "tacito-agent-mcp", Version: "0.1.0"},
			nil,
		)

		session, err := mcpClient.Connect(clientCtx, transport, nil)
		if err != nil {
			clientCancel()
			return fmt.Errorf("failed to connect to mcp client %s: %w", info.Name, err)
		}

		a.sessions = append(a.sessions, &clientSession{
			info:         info,
			mcpClient:    mcpClient,
			session:      session,
			cb:           cb,
			allowedTools: allowed,
			cancel:       clientCancel,
		})
	}
	return nil
}

func defaultTransportFactory(info outbound.MCPClientInfo) (mcp.Transport, error) {
	if info.Transport == "stdio" {
		cmd := exec.Command(info.Command, info.Args...)
		cmd.Env = os.Environ()
		for k, v := range info.Env {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
		}
		return &mcp.CommandTransport{Command: cmd}, nil
	} else if info.Transport == "sse" {
		return &mcp.StreamableClientTransport{Endpoint: info.URL}, nil
	}
	return nil, fmt.Errorf("unknown transport: %s", info.Transport)
}

func (a *MCPAdapter) ListAllowedTools(ctx context.Context) ([]outbound.ToolDefinition, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if err := a.init(ctx); err != nil {
		return nil, err
	}

	var results []outbound.ToolDefinition

	for _, s := range a.sessions {
		// If allowedTools is empty, it's a deny-all default (as specified in SPEC-FR-M5.5)
		if len(s.info.AllowedTools) == 0 {
			continue
		}

		runCtx, cancel := context.WithTimeout(ctx, a.timeout)
		var mcpToolsResult *mcp.ListToolsResult
		op := func() error {
			var err error
			mcpToolsResult, err = s.session.ListTools(runCtx, nil)
			return err
		}

		err := s.cb.Execute(runCtx, op, nil)
		cancel()
		if err != nil {
			continue // Graceful degradation: continue to next client
		}

		if mcpToolsResult == nil {
			continue
		}

		for _, tool := range mcpToolsResult.Tools {
			// Whitelist check
			if !s.allowedTools[tool.Name] {
				continue
			}

			var inputSchema map[string]any
			if tool.InputSchema != nil {
				if bytes, ok := tool.InputSchema.([]byte); ok {
					_ = json.Unmarshal(bytes, &inputSchema)
				} else if raw, ok := tool.InputSchema.(json.RawMessage); ok {
					_ = json.Unmarshal(raw, &inputSchema)
				} else if m, ok := tool.InputSchema.(map[string]any); ok {
					inputSchema = m
				} else {
					bytes, _ := json.Marshal(tool.InputSchema)
					_ = json.Unmarshal(bytes, &inputSchema)
				}
			}

			results = append(results, outbound.ToolDefinition{
				Name:        tool.Name,
				Description: tool.Description,
				InputSchema: inputSchema,
			})
		}
	}

	return results, nil
}

func (a *MCPAdapter) Execute(ctx context.Context, toolName string, arguments map[string]any) (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if err := a.init(ctx); err != nil {
		return "", err
	}

	var targetSession *clientSession
	for _, s := range a.sessions {
		if s.allowedTools[toolName] {
			targetSession = s
			break
		}
	}

	if targetSession == nil {
		return fmt.Sprintf(`{"error": "tool %s is not registered or allowed"}`, toolName), nil
	}

	var observation string

	op := func() error {
		runCtx, cancel := context.WithTimeout(ctx, a.timeout)
		defer cancel()

		params := &mcp.CallToolParams{
			Name:      toolName,
			Arguments: arguments,
		}

		res, err := targetSession.session.CallTool(runCtx, params)
		if err != nil {
			return err
		}

		if res.IsError {
			return fmt.Errorf("tool execution failed on server")
		}

		var sb []string
		for _, c := range res.Content {
			if tc, ok := c.(*mcp.TextContent); ok {
				sb = append(sb, tc.Text)
			}
		}
		observation = strings.Join(sb, "\n")
		return nil
	}

	fb := func(err error) error {
		return nil // Return nil error so the execution itself completes, returning the fallback JSON
	}

	// Default fallback JSON string
	observation = `{"error": "Tool provider temporarily unavailable."}`

	err := targetSession.cb.Execute(ctx, op, fb)
	if err != nil {
		return "", err
	}

	return observation, nil
}

func (a *MCPAdapter) Close(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	for _, s := range a.sessions {
		s.cancel()
		s.session.Close()
	}
	a.sessions = nil
	return nil
}
