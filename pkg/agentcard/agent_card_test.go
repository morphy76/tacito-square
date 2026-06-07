package agentcard_test

import (
	"encoding/json"
	"testing"

	"github.com/morphy76/tacito-square/pkg/agentcard"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAgentCard_JSONRoundTrip(t *testing.T) {
	card := &agentcard.AgentCard{
		Name:        "qa-agent",
		Description: "QA agent template",
		URL:         "http://qa-agent",
		Version:     "1.2.3",
		Provider: &agentcard.AgentCardProvider{
			Organization: "Tacito Organization",
			URL:          "http://tacito.org",
		},
		Capabilities: agentcard.AgentCardCapabilities{
			Streaming:              true,
			PushNotifications:      false,
			StateTransitionHistory: true,
		},
		Authentication: agentcard.AgentCardAuthentication{
			Schemes:     []string{"Bearer"},
			Credentials: "token",
		},
		DefaultInputModes:  []string{"text/plain"},
		DefaultOutputModes: []string{"text/plain"},
		Skills: []agentcard.AgentCardSkill{
			{
				ID:          "skill-1",
				Name:        "code-audit",
				Description: "Audits Python code",
				Tags:        []string{"audit", "python"},
				Examples:    []string{"how is this code?"},
			},
		},
	}

	// Marshal to JSON
	data, err := json.Marshal(card)
	require.NoError(t, err)

	// Unmarshal back
	var roundtrip agentcard.AgentCard
	err = json.Unmarshal(data, &roundtrip)
	require.NoError(t, err)

	// Assert equivalence
	assert.Equal(t, card.Name, roundtrip.Name)
	assert.Equal(t, card.Description, roundtrip.Description)
	assert.Equal(t, card.URL, roundtrip.URL)
	assert.Equal(t, card.Version, roundtrip.Version)
	require.NotNil(t, roundtrip.Provider)
	assert.Equal(t, card.Provider.Organization, roundtrip.Provider.Organization)
	assert.Equal(t, card.Provider.URL, roundtrip.Provider.URL)
	assert.Equal(t, card.Capabilities.Streaming, roundtrip.Capabilities.Streaming)
	assert.Equal(t, card.Capabilities.StateTransitionHistory, roundtrip.Capabilities.StateTransitionHistory)
	assert.Equal(t, card.Authentication.Schemes, roundtrip.Authentication.Schemes)
	assert.Equal(t, card.Authentication.Credentials, roundtrip.Authentication.Credentials)
	assert.Equal(t, card.DefaultInputModes, roundtrip.DefaultInputModes)
	assert.Equal(t, card.DefaultOutputModes, roundtrip.DefaultOutputModes)
	require.Len(t, roundtrip.Skills, 1)
	assert.Equal(t, card.Skills[0].ID, roundtrip.Skills[0].ID)
	assert.Equal(t, card.Skills[0].Name, roundtrip.Skills[0].Name)
	assert.Equal(t, card.Skills[0].Tags, roundtrip.Skills[0].Tags)
}
