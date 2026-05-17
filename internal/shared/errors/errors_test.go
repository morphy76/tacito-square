package errors

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewNotFound_UnwrapsToSentinel(t *testing.T) {
	err := NewNotFound("Agent", "abc-123")

	assert.True(t, errors.Is(err, ErrNotFound))
	assert.Contains(t, err.Error(), "abc-123")
	assert.Contains(t, err.Error(), "NOT_FOUND")
}

func TestNewAlreadyExists_UnwrapsToSentinel(t *testing.T) {
	err := NewAlreadyExists("Prompt", "default-prompt")

	assert.True(t, errors.Is(err, ErrAlreadyExists))
	assert.Contains(t, err.Error(), "default-prompt")
	assert.Contains(t, err.Error(), "ALREADY_EXISTS")
}

func TestNewInvalidInput_UnwrapsToSentinel(t *testing.T) {
	err := NewInvalidInput("name cannot be empty")

	assert.True(t, errors.Is(err, ErrInvalidInput))
	assert.Contains(t, err.Error(), "name cannot be empty")
}

func TestDomainError_ErrorFormat(t *testing.T) {
	err := &DomainError{
		Err:  ErrInternal,
		Code: "INTERNAL",
	}

	assert.Contains(t, err.Error(), "INTERNAL")
	assert.Contains(t, err.Error(), "internal error")
}
