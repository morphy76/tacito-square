package service

import (
	"context"
)

type tenantCtxKey struct{}
type agentCtxKey struct{}
type threadCtxKey struct{}
type activeToolsKey struct{}
type parsedSkillsKey struct{}

// GetTenantID retrieves the Tenant ID from the given context, if present.
func GetTenantID(ctx context.Context) string {
	if v, ok := ctx.Value(tenantCtxKey{}).(string); ok {
		return v
	}
	return ""
}

// GetAgentID retrieves the Agent ID from the given context, if present.
func GetAgentID(ctx context.Context) string {
	if v, ok := ctx.Value(agentCtxKey{}).(string); ok {
		return v
	}
	return ""
}

// GetThreadID retrieves the Thread ID from the given context, if present.
func GetThreadID(ctx context.Context) string {
	if v, ok := ctx.Value(threadCtxKey{}).(string); ok {
		return v
	}
	return ""
}

// Skill represents a dynamic procedural knowledge source.
type Skill struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Content     string `json:"content"`
}

// PropagatedAgentConfig defines the structured keeper-agent context format.
type PropagatedAgentConfig struct {
	Description string  `json:"description"`
	Directives  string  `json:"directives"`
	Skills      []Skill `json:"skills"`
}

type parsedResponse struct {
	Thought     string          `json:"thought"`
	ToolCall    *toolCallDetail `json:"tool_call,omitempty"`
	FinalAnswer string          `json:"final_answer,omitempty"`
}

type toolCallDetail struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
}
