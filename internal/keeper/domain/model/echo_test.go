package model

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSanitizeMessage_StripNonPrintable(t *testing.T) {
	got := SanitizeMessage("hello\x00world\x01")
	assert.Equal(t, "helloworld", got)
}

func TestSanitizeMessage_Truncates(t *testing.T) {
	input := strings.Repeat("a", 1100)
	got := SanitizeMessage(input)
	assert.Len(t, got, 1000)
}

func TestSanitizeMessage_EmptyAfterStrip(t *testing.T) {
	got := SanitizeMessage("\x00\x01\x02\x03")
	assert.Equal(t, "", got)
}

func TestSanitizeMessage_CleanInput(t *testing.T) {
	got := SanitizeMessage("hello world")
	assert.Equal(t, "hello world", got)
}

func TestDecorateMessage(t *testing.T) {
	got := DecorateMessage("agent-alpha", "hello")

	assert.True(t, strings.HasPrefix(got, "[agent:agent-alpha at "), "expected prefix [agent:agent-alpha at ..., got: %s", got)
	assert.True(t, strings.HasSuffix(got, "] hello"), "expected suffix ] hello, got: %s", got)

	// Extract the timestamp portion and verify it's a valid RFC3339 value.
	start := len("[agent:agent-alpha at ")
	end := strings.Index(got, "]")
	assert.Greater(t, end, start, "closing bracket not found")

	ts := got[start:end]
	assert.NotEmpty(t, ts)
	// RFC3339 timestamps contain 'T' and end with 'Z' or offset.
	assert.Contains(t, ts, "T", "timestamp should contain T separator")
}

func TestCommunityEchoResponse_Fields(t *testing.T) {
	t.Run("EchoRequest", func(t *testing.T) {
		original := EchoRequest{
			Message:     "hello",
			CommunityID: "comm-1",
			TenantID:    "tenant-1",
		}
		data, err := json.Marshal(original)
		assert.NoError(t, err)

		var m map[string]interface{}
		err = json.Unmarshal(data, &m)
		assert.NoError(t, err)
		assert.Equal(t, "hello", m["message"])
		assert.Equal(t, "comm-1", m["community_id"])
		assert.Equal(t, "tenant-1", m["tenant_id"])

		var decoded EchoRequest
		err = json.Unmarshal(data, &decoded)
		assert.NoError(t, err)
		assert.Equal(t, original, decoded)
	})

	t.Run("EchoReply", func(t *testing.T) {
		original := EchoReply{
			AgentName: "agent-alpha",
			Decorated: "[agent:agent-alpha at 2026-05-29T18:00:00Z] hello",
			Timestamp: "2026-05-29T18:00:00Z",
		}
		data, err := json.Marshal(original)
		assert.NoError(t, err)

		var m map[string]interface{}
		err = json.Unmarshal(data, &m)
		assert.NoError(t, err)
		assert.Equal(t, "agent-alpha", m["agent_name"])
		assert.Equal(t, "[agent:agent-alpha at 2026-05-29T18:00:00Z] hello", m["decorated"])
		assert.Equal(t, "2026-05-29T18:00:00Z", m["timestamp"])

		var decoded EchoReply
		err = json.Unmarshal(data, &decoded)
		assert.NoError(t, err)
		assert.Equal(t, original, decoded)
	})

	t.Run("AgentEchoResult", func(t *testing.T) {
		original := AgentEchoResult{
			AgentName: "agent-beta",
			Decorated: "",
			Error:     "timeout",
		}
		data, err := json.Marshal(original)
		assert.NoError(t, err)

		var m map[string]interface{}
		err = json.Unmarshal(data, &m)
		assert.NoError(t, err)
		assert.Equal(t, "agent-beta", m["agent_name"])
		assert.Equal(t, "", m["decorated"])
		assert.Equal(t, "timeout", m["error"])

		var decoded AgentEchoResult
		err = json.Unmarshal(data, &decoded)
		assert.NoError(t, err)
		assert.Equal(t, original, decoded)
	})

	t.Run("CommunityEchoResponse", func(t *testing.T) {
		original := CommunityEchoResponse{
			CommunityID:   "comm-1",
			WokeCommunity: true,
			Results: []AgentEchoResult{
				{AgentName: "agent-alpha", Decorated: "decorated-msg", Error: ""},
			},
		}
		data, err := json.Marshal(original)
		assert.NoError(t, err)

		var m map[string]interface{}
		err = json.Unmarshal(data, &m)
		assert.NoError(t, err)
		assert.Equal(t, "comm-1", m["community_id"])
		assert.Equal(t, true, m["woke_community"])
		assert.NotNil(t, m["results"])

		var decoded CommunityEchoResponse
		err = json.Unmarshal(data, &decoded)
		assert.NoError(t, err)
		assert.Equal(t, original, decoded)
	})
}
