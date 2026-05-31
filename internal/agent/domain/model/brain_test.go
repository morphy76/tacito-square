package model_test

import (
	"testing"

	"github.com/morphy76/tacito-square/internal/agent/domain/model"
	"github.com/stretchr/testify/assert"
)

func TestBrainRequest_Validate(t *testing.T) {
	t.Run("empty prompt should return error", func(t *testing.T) {
		req := model.BrainRequest{
			Prompt: "",
		}
		err := req.Validate()
		assert.Error(t, err)
		assert.Equal(t, "prompt must not be empty", err.Error())
	})

	t.Run("non-empty prompt should be valid", func(t *testing.T) {
		req := model.BrainRequest{
			Prompt: "Explain quantum physics",
		}
		err := req.Validate()
		assert.NoError(t, err)
	})
}
