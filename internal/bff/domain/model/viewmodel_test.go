package model_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/morphy76/tacito-square/internal/bff/domain/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserInfoPayload_JSON_Marshal(t *testing.T) {
	payload := model.UserInfoPayload{
		Sub:            "user-123",
		Email:          "user@test.com",
		TenantID:       "tenant-abc",
		SubscriptionID: "sub-xyz",
	}

	data, err := json.Marshal(payload)
	require.NoError(t, err)

	jsonStr := string(data)
	assert.Contains(t, jsonStr, `"sub":"user-123"`)
	assert.Contains(t, jsonStr, `"email":"user@test.com"`)
	assert.Contains(t, jsonStr, `"tenantid":"tenant-abc"`)
	assert.Contains(t, jsonStr, `"subscriptionid":"sub-xyz"`)
}

func TestUserInfoPayload_JSON_Unmarshal(t *testing.T) {
	raw := `{"sub":"u-456","email":"foo@bar.com","tenantid":"t-999","subscriptionid":"s-000"}`

	var payload model.UserInfoPayload
	err := json.Unmarshal([]byte(raw), &payload)
	require.NoError(t, err)

	assert.Equal(t, "u-456", payload.Sub)
	assert.Equal(t, "foo@bar.com", payload.Email)
	assert.Equal(t, "t-999", payload.TenantID)
	assert.Equal(t, "s-000", payload.SubscriptionID)
}

func TestUserInfoPayload_JSON_PartialFields(t *testing.T) {
	// Fields not present in JSON should marshal to zero values, not cause errors.
	raw := `{"sub":"u-partial"}`

	var payload model.UserInfoPayload
	err := json.Unmarshal([]byte(raw), &payload)
	require.NoError(t, err)

	assert.Equal(t, "u-partial", payload.Sub)
	assert.Empty(t, payload.Email)
	assert.Empty(t, payload.TenantID)
	assert.Empty(t, payload.SubscriptionID)
}

func TestDomainErrors_AreDistinct(t *testing.T) {
	errs := []error{
		model.ErrSessionNotFound,
		model.ErrSessionExpired,
		model.ErrSessionInvalidated,
	}

	for i, e := range errs {
		// Each must implement the error interface
		assert.NotEmpty(t, e.Error(), "Error %d must have a non-empty message", i)

		// Each must be distinct from the others
		for j, other := range errs {
			if i != j {
				assert.False(t, errors.Is(e, other), "Error %d and %d must be distinct sentinel values", i, j)
			}
		}
	}
}

func TestDomainErrors_SentinelIdentity(t *testing.T) {
	// Wrapping and unwrapping must preserve sentinel identity for errors.Is
	wrapped := fmt.Errorf("operation failed: %w", model.ErrSessionNotFound)
	assert.True(t, errors.Is(wrapped, model.ErrSessionNotFound))
}
