package model

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestAgentReasoningStepPayloadValidation(t *testing.T) {
	t.Run("valid payload with thought and action", func(t *testing.T) {
		payload := AgentReasoningStepPayload{
			StepIndex: 1,
			Thought:   "Thinking about searching memories",
			Action: &ToolCallAction{
				Tool: "recall_memory",
				Input: map[string]any{
					"query": "database pooling",
				},
			},
			Timestamp: time.Now().UTC(),
		}
		err := payload.Validate()
		assert.NoError(t, err)
	})

	t.Run("valid payload with final answer text only", func(t *testing.T) {
		payload := AgentReasoningStepPayload{
			StepIndex: 2,
			Thought:   "Got the final answer",
			Timestamp: time.Now().UTC(),
		}
		err := payload.Validate()
		assert.NoError(t, err)
	})

	t.Run("invalid step index", func(t *testing.T) {
		payload := AgentReasoningStepPayload{
			StepIndex: 0,
			Thought:   "Thinking...",
			Timestamp: time.Now().UTC(),
		}
		err := payload.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "step index must be greater than zero")
	})

	t.Run("missing timestamp", func(t *testing.T) {
		payload := AgentReasoningStepPayload{
			StepIndex: 1,
			Thought:   "Thinking...",
		}
		err := payload.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "timestamp must not be zero")
	})

	t.Run("missing cognitive content", func(t *testing.T) {
		payload := AgentReasoningStepPayload{
			StepIndex: 1,
			Timestamp: time.Now().UTC(),
		}
		err := payload.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "payload must contain at least one of thought, action, or observation")
	})

	t.Run("invalid action tool name", func(t *testing.T) {
		payload := AgentReasoningStepPayload{
			StepIndex: 1,
			Action: &ToolCallAction{
				Tool: "",
			},
			Timestamp: time.Now().UTC(),
		}
		err := payload.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "action tool name must not be empty")
	})
}

func TestAgentReasoningStepPayloadJSON(t *testing.T) {
	t.Run("json marshal and unmarshal", func(t *testing.T) {
		payload := AgentReasoningStepPayload{
			StepIndex: 1,
			Thought:   "Checking database config",
			Action: &ToolCallAction{
				Tool: "recall_memory",
				Input: map[string]any{
					"query": "db",
				},
			},
			Timestamp: time.Date(2026, 6, 1, 1, 15, 0, 0, time.UTC),
		}

		data, err := json.Marshal(payload)
		assert.NoError(t, err)

		var unmarshaled AgentReasoningStepPayload
		err = json.Unmarshal(data, &unmarshaled)
		assert.NoError(t, err)

		assert.Equal(t, payload.StepIndex, unmarshaled.StepIndex)
		assert.Equal(t, payload.Thought, unmarshaled.Thought)
		assert.Equal(t, payload.Action.Tool, unmarshaled.Action.Tool)
		assert.Equal(t, "db", unmarshaled.Action.Input["query"])
		assert.True(t, payload.Timestamp.Equal(unmarshaled.Timestamp))
	})
}
