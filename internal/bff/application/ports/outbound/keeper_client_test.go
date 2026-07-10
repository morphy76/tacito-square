package outbound_test

import (
	"context"
	"testing"

	"github.com/morphy76/tacito-square/internal/bff/application/ports/outbound"
)

type mockKeeperClient struct{}

func (m *mockKeeperClient) Ping(ctx context.Context) error {
	return nil
}
func (m *mockKeeperClient) LLMBindings() outbound.LLMBindingClient { return nil }
func (m *mockKeeperClient) MCPServers() outbound.MCPServerClient   { return nil }
func (m *mockKeeperClient) Skills() outbound.SkillClient           { return nil }
func (m *mockKeeperClient) Prompts() outbound.PromptClient         { return nil }
func (m *mockKeeperClient) Agents() outbound.AgentClient           { return nil }
func (m *mockKeeperClient) Communities() outbound.CommunityClient { return nil }

// Compile-time assertion
var _ outbound.KeeperClient = (*mockKeeperClient)(nil)

func TestKeeperClientInterface(t *testing.T) {
	// Assertions are checked at compile time
}
