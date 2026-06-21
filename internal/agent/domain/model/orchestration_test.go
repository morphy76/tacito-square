package model_test

import (
	"testing"

	"github.com/morphy76/tacito-square/internal/agent/domain/model"
	"github.com/stretchr/testify/assert"
)

func TestOrchestrationState_Validate(t *testing.T) {
	t.Run("happy path: valid orchestration state", func(t *testing.T) {
		state := model.OrchestrationState{
			ThreadID:    "thread-123",
			CommunityID: "comm-abc",
			Status:      model.StatusWaitingSpoke,
		}
		err := state.Validate()
		assert.NoError(t, err)
	})

	t.Run("missing thread ID", func(t *testing.T) {
		state := model.OrchestrationState{
			CommunityID: "comm-abc",
			Status:      model.StatusWaitingSpoke,
		}
		err := state.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "thread_id must not be empty")
	})

	t.Run("missing community ID", func(t *testing.T) {
		state := model.OrchestrationState{
			ThreadID: "thread-123",
			Status:   model.StatusWaitingSpoke,
		}
		err := state.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "community_id must not be empty")
	})

	t.Run("missing status", func(t *testing.T) {
		state := model.OrchestrationState{
			ThreadID:    "thread-123",
			CommunityID: "comm-abc",
		}
		err := state.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "status must not be empty")
	})
}
