package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TacitoAgentPhase represents the lifecycle phase of a TacitoAgent.
type TacitoAgentPhase string

const (
	// PhasePending indicates the agent has been created but is not yet running.
	PhasePending TacitoAgentPhase = "Pending"
	// PhaseRunning indicates the agent is actively processing.
	PhaseRunning TacitoAgentPhase = "Running"
	// PhaseIdle indicates the agent is running but has no active work.
	PhaseIdle TacitoAgentPhase = "Idle"
	// PhaseTerminated indicates the agent has been shut down.
	PhaseTerminated TacitoAgentPhase = "Terminated"
)

// LLMConfig defines the LLM parameters for the agent's brain.
type LLMConfig struct {
	// Model is the LLM model identifier (e.g., "gpt-4o", "gemini-1.5-pro").
	// +kubebuilder:validation:MinLength=1
	Model string `json:"model"`

	// Temperature controls the randomness of the LLM output.
	// +optional
	// +kubebuilder:validation:Pattern="^[0-9]+(\\.[0-9]+)?$"
	// +kubebuilder:default="0.7"
	Temperature *string `json:"temperature,omitempty"`

	// MaxTokens is the maximum number of tokens the LLM can generate.
	// +optional
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=8192
	// +kubebuilder:default=2048
	MaxTokens *int32 `json:"maxTokens,omitempty"`
}

// MCPClientSpec defines the configuration of an attached MCP client gateway.
type MCPClientSpec struct {
	// Name is the unique identifier of the MCP client config.
	Name string `json:"name"`

	// Transport is the protocol transport type ("stdio" or "sse").
	Transport string `json:"transport"`

	// Command is the subprocess execution command (for stdio).
	// +optional
	Command string `json:"command,omitempty"`

	// Args is the command-line arguments (for stdio).
	// +optional
	Args []string `json:"args,omitempty"`

	// Env is the custom environment overrides (for stdio).
	// +optional
	Env map[string]string `json:"env,omitempty"`

	// URL is the SSE event connection endpoint (for sse).
	// +optional
	URL string `json:"url,omitempty"`

	// AllowedTools is the whitelist of tool names permitted for execution.
	AllowedTools []string `json:"allowedTools"`
}

// TacitoAgentSpec defines the desired state of a TacitoAgent.
type TacitoAgentSpec struct {
	// TenantID is the identifier of the tenant owning this agent.
	// +kubebuilder:validation:MinLength=1
	TenantID string `json:"tenantId"`

	// AgentName is the unique name of the agent.
	// +kubebuilder:validation:MinLength=1
	AgentName string `json:"agentName"`

	// CommunityRef is the reference to the parent community.
	// +kubebuilder:validation:MinLength=1
	CommunityRef string `json:"communityRef"`

	// LLMConfig holds the LLM parameters for the agent's brain.
	LLMConfig LLMConfig `json:"llmConfig"`

	// SystemPrompt is the fully synthesized system prompt for the agent.
	// +optional
	SystemPrompt string `json:"systemPrompt,omitempty"`

	// MCPClients holds the list of configured MCP client connections.
	// +optional
	MCPClients []MCPClientSpec `json:"mcpClients,omitempty"`

	// Replicas is the desired number of agent pod replicas.
	// +optional
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=10
	// +kubebuilder:default=1
	Replicas *int32 `json:"replicas,omitempty"`

	// Tier is the logical runtime tier name used by the Operator to select the
	// appropriate container image, resource requests/limits, and probe timings.
	// An empty string signals the Operator to apply the implicit default profile.
	// +optional
	Tier string `json:"tier,omitempty"`
}

// TacitoAgentStatus defines the observed state of a TacitoAgent.
type TacitoAgentStatus struct {
	// Phase is the current lifecycle phase of the agent.
	// +optional
	// +kubebuilder:validation:Enum=Pending;Running;Idle;Terminated
	Phase TacitoAgentPhase `json:"phase,omitempty"`

	// Conditions represent the latest available observations of an agent's state.
	// +optional
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// LastHeartbeat is the timestamp of the last agent heartbeat.
	// +optional
	LastHeartbeat metav1.Time `json:"lastHeartbeat,omitempty"`

	// Selector is the label selector for scaling.
	// +optional
	Selector string `json:"selector,omitempty"`

	// Replicas is the current number of running replicas.
	Replicas int32 `json:"replicas"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:subresource:scale:specpath=.spec.replicas,statuspath=.status.replicas,selectorpath=.status.selector
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Tenant",type=string,JSONPath=`.spec.tenantId`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// TacitoAgent is the Schema for the tacitoagents API.
type TacitoAgent struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TacitoAgentSpec   `json:"spec,omitempty"`
	Status TacitoAgentStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// TacitoAgentList contains a list of TacitoAgent resources.
type TacitoAgentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []TacitoAgent `json:"items"`
}
