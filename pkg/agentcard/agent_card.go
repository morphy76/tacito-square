package agentcard

// AgentCardProvider defines the provider metadata for an AgentCard.
type AgentCardProvider struct {
	Organization string `json:"organization"`
	URL          string `json:"url"`
}

// AgentCardCapabilities defines the capabilities flags for an AgentCard.
type AgentCardCapabilities struct {
	Streaming              bool `json:"streaming,omitempty"`
	PushNotifications      bool `json:"pushNotifications,omitempty"`
	StateTransitionHistory bool `json:"stateTransitionHistory,omitempty"`
}

// AgentCardAuthentication defines the authentication details for an AgentCard.
type AgentCardAuthentication struct {
	Schemes     []string `json:"schemes"`
	Credentials string   `json:"credentials,omitempty"`
}

// AgentCardSkill defines a specific unit of capability an agent possesses.
type AgentCardSkill struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
	Examples    []string `json:"examples,omitempty"`
	InputModes  []string `json:"inputModes,omitempty"`
	OutputModes []string `json:"outputModes,omitempty"`
}

// AgentCard represents the A2A AgentCard discovery structure.
type AgentCard struct {
	Name               string                  `json:"name"`
	Description        string                  `json:"description"`
	URL                string                  `json:"url"`
	Provider           *AgentCardProvider      `json:"provider,omitempty"`
	Version            string                  `json:"version"`
	DocumentationURL   string                  `json:"documentationUrl,omitempty"`
	Capabilities       AgentCardCapabilities   `json:"capabilities"`
	Authentication     AgentCardAuthentication `json:"authentication"`
	DefaultInputModes  []string                `json:"defaultInputModes"`
	DefaultOutputModes []string                `json:"defaultOutputModes"`
	Skills             []AgentCardSkill        `json:"skills"`
}

// AgentCommunityRef represents an agent-community relation for status updates.
type AgentCommunityRef struct {
	AgentID     string `json:"agent_id"`
	CommunityID string `json:"community_id"`
	TenantID    string `json:"tenant_id"`
}
