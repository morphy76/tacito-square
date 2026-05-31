package model

import (
	"errors"
	"time"
)

// ToolCallAction represents a specific tool invocation requested by the brain.
type ToolCallAction struct {
	Tool  string         `json:"tool"`
	Input map[string]any `json:"input,omitempty"`
}

// AgentReasoningStepPayload represents a single granular step in the active reasoning loop.
type AgentReasoningStepPayload struct {
	StepIndex   int             `json:"step_index"`
	Thought     string          `json:"thought,omitempty"`
	Action      *ToolCallAction `json:"action,omitempty"`
	Observation string          `json:"observation,omitempty"`
	Timestamp   time.Time       `json:"timestamp"`
}

// Validate asserts the structural correctness and constraints of a reasoning step payload.
func (p *AgentReasoningStepPayload) Validate() error {
	if p.StepIndex <= 0 {
		return errors.New("step index must be greater than zero")
	}
	if p.Timestamp.IsZero() {
		return errors.New("timestamp must not be zero")
	}
	if p.Thought == "" && p.Action == nil && p.Observation == "" {
		return errors.New("payload must contain at least one of thought, action, or observation")
	}
	if p.Action != nil && p.Action.Tool == "" {
		return errors.New("action tool name must not be empty")
	}
	return nil
}
