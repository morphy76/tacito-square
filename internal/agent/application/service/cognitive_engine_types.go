package service

import (
	"context"
	"strings"
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

// CleanAndExtractJSON attempts to locate the first '{' and the last '}' in the content
// to extract a clean JSON object string, ignoring any leading/trailing conversational text.
// It also strips markdown code block wrappers if they exist.
func CleanAndExtractJSON(content string) string {
	cleaned := content
	if strings.Contains(cleaned, "```json") {
		parts := strings.Split(cleaned, "```json")
		if len(parts) > 1 {
			cleaned = strings.Split(parts[1], "```")[0]
		}
	} else if strings.Contains(cleaned, "```") {
		parts := strings.Split(cleaned, "```")
		if len(parts) > 1 {
			cleaned = strings.Split(parts[1], "```")[0]
		}
	}
	cleaned = strings.TrimSpace(cleaned)

	start := strings.Index(cleaned, "{")
	end := strings.LastIndex(cleaned, "}")
	if start != -1 && end != -1 && end > start {
		return cleaned[start : end+1]
	}
	return cleaned
}
