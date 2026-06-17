package events

const (
	// SchemaConversationalStartThread is the URN for starting a thread.
	SchemaConversationalStartThread = "urn:tacito:schema:conversational:start-thread:v1"

	// SchemaConversationalAddUserMessage is the URN for adding a user message.
	SchemaConversationalAddUserMessage = "urn:tacito:schema:conversational:add-user-message:v1"

	// SchemaConversationalEndThread is the URN for ending a thread.
	SchemaConversationalEndThread = "urn:tacito:schema:conversational:end-thread:v1"

	// SchemaConversationalAgentResponse is the URN for agent final responses.
	SchemaConversationalAgentResponse = "urn:tacito:schema:conversational:agent-response:v1"

	// SchemaConversationalThreadHistory is the URN for emitting the full thread history.
	SchemaConversationalThreadHistory = "urn:tacito:schema:conversational:thread-history:v1"

	// SchemaConversationalAgentDelegation is the URN for inter-agent task delegation.
	SchemaConversationalAgentDelegation = "urn:tacito:schema:conversational:agent-delegation:v1"

	// SchemaConversationalAgentReasoning is the URN for agent coordination reasoning updates.
	SchemaConversationalAgentReasoning = "urn:tacito:schema:conversational:agent-reasoning:v1"

	// SchemaConversationalAgentSpokeResponse is the URN for spoke agent responses back to the hub.
	SchemaConversationalAgentSpokeResponse = "urn:tacito:schema:conversational:agent-spoke-response:v1"

	// SchemaConversationalAgentToolEvaluation is the URN for agent tool execution/evaluation updates.
	SchemaConversationalAgentToolEvaluation = "urn:tacito:schema:conversational:agent-tool-evaluation:v1"
)

// ThreadTurn represents a single conversational or execution step in a thread.
type ThreadTurn struct {
	Role      string            `json:"role"`
	Content   string            `json:"content"`
	Timestamp string            `json:"timestamp"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// ThreadHistoryPayload is the payload for urn:tacito:schema:conversational:thread-history:v1.
type ThreadHistoryPayload struct {
	ThreadID    string       `json:"thread_id"`
	CommunityID string       `json:"community_id"`
	History     []ThreadTurn `json:"history"`
}

// StartThreadPayload is the payload for urn:tacito:schema:conversational:start-thread:v1.
type StartThreadPayload struct {
	// ThreadID is a client-supplied or keeper-generated UUID for this conversation thread.
	ThreadID string `json:"thread_id"`

	// CommunityID is the UUID of the target community.
	CommunityID string `json:"community_id"`

	// Metadata is an optional arbitrary key-value map for caller-defined context.
	Metadata map[string]string `json:"metadata,omitempty"`
}

// AddUserMessagePayload is the payload for urn:tacito:schema:conversational:add-user-message:v1.
type AddUserMessagePayload struct {
	// ThreadID identifies the active conversation thread.
	ThreadID string `json:"thread_id"`

	// CommunityID is the UUID of the target community.
	CommunityID string `json:"community_id"`

	// Message is the user's input text.
	Message string `json:"message"`
}

// EndThreadPayload is the payload for urn:tacito:schema:conversational:end-thread:v1.
type EndThreadPayload struct {
	// ThreadID identifies the conversation thread to close.
	ThreadID string `json:"thread_id"`

	// CommunityID is the UUID of the target community.
	CommunityID string `json:"community_id"`

	// Reason is an optional human-readable string describing why the thread ended.
	Reason string `json:"reason,omitempty"`
}

// AgentResponsePayload is the payload for agent-response, agent-reasoning, and agent-spoke-response schemas.
type AgentResponsePayload struct {
	// ThreadID correlates this response to its originating thread.
	ThreadID string `json:"thread_id"`

	// CommunityID is the UUID of the agent's community.
	CommunityID string `json:"community_id"`

	// AgentName is the name of the responding agent.
	AgentName string `json:"agent_name"`

	// CorrelationEventID is the EventID of the add-user-message event that triggered this response.
	CorrelationEventID string `json:"correlation_event_id"`

	// Response is the agent's LLM-generated reply text.
	Response string `json:"response"`

	// Finished indicates whether this is the final (complete) response chunk.
	Finished bool `json:"finished"`
}

// AgentDelegationPayload is the payload for urn:tacito:schema:conversational:agent-delegation:v1.
type AgentDelegationPayload struct {
	// ThreadID identifies the active conversation thread.
	ThreadID string `json:"thread_id"`

	// CommunityID is the UUID of the target community.
	CommunityID string `json:"community_id"`

	// DelegatingAgent is the name of the Hub agent delegating the task.
	DelegatingAgent string `json:"delegating_agent"`

	// TargetAgent is the name of the Spoke agent receiving the task.
	TargetAgent string `json:"target_agent"`

	// Message is the task description for the Spoke agent.
	Message string `json:"message"`
}
