package events_test

import (
	"encoding/json"
	"testing"

	"github.com/morphy76/tacito-square/pkg/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConstantsExist(t *testing.T) {
	assert.Equal(t, "urn:tacito:schema:conversational:start-thread:v1", events.SchemaConversationalStartThread)
	assert.Equal(t, "urn:tacito:schema:conversational:add-user-message:v1", events.SchemaConversationalAddUserMessage)
	assert.Equal(t, "urn:tacito:schema:conversational:end-thread:v1", events.SchemaConversationalEndThread)
	assert.Equal(t, "urn:tacito:schema:conversational:agent-response:v1", events.SchemaConversationalAgentResponse)
	assert.Equal(t, "urn:tacito:schema:conversational:agent-delegation:v1", events.SchemaConversationalAgentDelegation)
	assert.Equal(t, "urn:tacito:schema:conversational:agent-reasoning:v1", events.SchemaConversationalAgentReasoning)
	assert.Equal(t, "urn:tacito:schema:conversational:agent-spoke-response:v1", events.SchemaConversationalAgentSpokeResponse)
	assert.Equal(t, "urn:tacito:schema:conversational:agent-tool-evaluation:v1", events.SchemaConversationalAgentToolEvaluation)
}

func TestStartThreadPayload_JSON(t *testing.T) {
	// Full payload
	p1 := events.StartThreadPayload{
		ThreadID:    "t-123",
		CommunityID: "c-456",
		Metadata:    map[string]string{"user": "alice"},
	}
	data, err := json.Marshal(p1)
	assert.NoError(t, err)
	assert.Contains(t, string(data), `"metadata"`)

	var p1Decoded events.StartThreadPayload
	err = json.Unmarshal(data, &p1Decoded)
	assert.NoError(t, err)
	assert.Equal(t, p1.ThreadID, p1Decoded.ThreadID)
	assert.Equal(t, p1.CommunityID, p1Decoded.CommunityID)
	assert.Equal(t, p1.Metadata["user"], p1Decoded.Metadata["user"])

	// Empty metadata omission
	p2 := events.StartThreadPayload{
		ThreadID:    "t-123",
		CommunityID: "c-456",
	}
	data, err = json.Marshal(p2)
	assert.NoError(t, err)
	assert.NotContains(t, string(data), `"metadata"`)
}

func TestAddUserMessagePayload_JSON(t *testing.T) {
	p := events.AddUserMessagePayload{
		ThreadID:    "t-123",
		CommunityID: "c-456",
		Message:     "hello world",
	}
	data, err := json.Marshal(p)
	assert.NoError(t, err)

	var pDecoded events.AddUserMessagePayload
	err = json.Unmarshal(data, &pDecoded)
	assert.NoError(t, err)
	assert.Equal(t, p.ThreadID, pDecoded.ThreadID)
	assert.Equal(t, p.CommunityID, pDecoded.CommunityID)
	assert.Equal(t, p.Message, pDecoded.Message)
}

func TestEndThreadPayload_JSON(t *testing.T) {
	p := events.EndThreadPayload{
		ThreadID:    "t-123",
		CommunityID: "c-456",
		Reason:      "finished conversation",
	}
	data, err := json.Marshal(p)
	assert.NoError(t, err)

	var pDecoded events.EndThreadPayload
	err = json.Unmarshal(data, &pDecoded)
	assert.NoError(t, err)
	assert.Equal(t, p.ThreadID, pDecoded.ThreadID)
	assert.Equal(t, p.CommunityID, pDecoded.CommunityID)
	assert.Equal(t, p.Reason, pDecoded.Reason)
}

func TestAgentResponsePayload_JSON(t *testing.T) {
	p := events.AgentResponsePayload{
		ThreadID:           "t-123",
		CommunityID:        "c-456",
		AgentName:          "agent-alpha",
		CorrelationEventID: "e-789",
		Response:           "this is agent response",
		Finished:           true,
	}
	data, err := json.Marshal(p)
	assert.NoError(t, err)
	assert.NotContains(t, string(data), `"message_type"`)

	var pDecoded events.AgentResponsePayload
	err = json.Unmarshal(data, &pDecoded)
	assert.NoError(t, err)
	assert.Equal(t, p.ThreadID, pDecoded.ThreadID)
	assert.Equal(t, p.CommunityID, pDecoded.CommunityID)
	assert.Equal(t, p.AgentName, pDecoded.AgentName)
	assert.Equal(t, p.CorrelationEventID, pDecoded.CorrelationEventID)
	assert.Equal(t, p.Response, pDecoded.Response)
	assert.Equal(t, p.Finished, pDecoded.Finished)
}

func TestAgentDelegationPayload_JSON(t *testing.T) {
	p := events.AgentDelegationPayload{
		ThreadID:        "t-123",
		CommunityID:     "c-456",
		DelegatingAgent: "hub-agent",
		TargetAgent:     "writer",
		Message:         "write about a dragon",
		ContextHistory: []events.ThreadTurn{
			{Role: "user", Content: "hello", Timestamp: "2026-06-20T09:30:00Z"},
			{Role: "assistant", Content: "hi", Timestamp: "2026-06-20T09:30:05Z"},
		},
	}
	data, err := json.Marshal(p)
	assert.NoError(t, err)
	assert.Contains(t, string(data), `"context_history"`)

	var pDecoded events.AgentDelegationPayload
	err = json.Unmarshal(data, &pDecoded)
	assert.NoError(t, err)
	assert.Equal(t, p.ThreadID, pDecoded.ThreadID)
	assert.Equal(t, p.CommunityID, pDecoded.CommunityID)
	assert.Equal(t, p.DelegatingAgent, pDecoded.DelegatingAgent)
	assert.Equal(t, p.TargetAgent, pDecoded.TargetAgent)
	assert.Equal(t, p.Message, pDecoded.Message)
	require.Len(t, pDecoded.ContextHistory, 2)
	assert.Equal(t, p.ContextHistory[0].Role, pDecoded.ContextHistory[0].Role)
	assert.Equal(t, p.ContextHistory[0].Content, pDecoded.ContextHistory[0].Content)
	assert.Equal(t, p.ContextHistory[0].Timestamp, pDecoded.ContextHistory[0].Timestamp)
}
