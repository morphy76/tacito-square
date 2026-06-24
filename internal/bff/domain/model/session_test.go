package model_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/morphy76/tacito-square/internal/bff/domain/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testIssuer       = "https://keycloak.example.com/realms/tacito"
	testOIDCsid      = "oidc-sid-abc123"
	testUserID       = "user-123"
	testTenantID     = "tenant-abc"
	testAccessToken  = "access-tok"
	testRefreshToken = "refresh-tok"
	testIDToken      = "id-tok"
)

func newTestSession(t *testing.T, ttl time.Duration) *model.Session {
	t.Helper()
	userInfo := model.UserInfoPayload{
		Sub:            testUserID,
		Email:          "user@example.com",
		TenantID:       testTenantID,
		SubscriptionID: "sub-xyz",
	}
	s, err := model.NewSession(testUserID, testTenantID, testIssuer, testOIDCsid, testAccessToken, testRefreshToken, testIDToken, userInfo, ttl)
	require.NoError(t, err)
	return s
}

func TestSession_NewSession_Success(t *testing.T) {
	before := time.Now().UTC()
	session := newTestSession(t, time.Hour)
	after := time.Now().UTC()

	// ID must be a non-empty UUID v4 format (36 chars with hyphens)
	assert.Len(t, session.ID, 36, "Session ID must be UUID v4 format")
	assert.Contains(t, session.ID, "-", "Session ID must contain hyphens")

	// Core identity fields
	assert.Equal(t, testUserID, session.UserID)
	assert.Equal(t, testTenantID, session.TenantID)

	// OIDC backchannel logout fields
	assert.Equal(t, testIssuer, session.Issuer, "Issuer must be stored for logout token validation")
	assert.Equal(t, testOIDCsid, session.OIDCSessionID, "OIDCSessionID must be stored for sid-based backchannel logout")

	// CreatedAt must be UTC and within the test window
	assert.True(t, session.CreatedAt.Equal(before) || session.CreatedAt.After(before), "CreatedAt must be >= before")
	assert.True(t, session.CreatedAt.Equal(after) || session.CreatedAt.Before(after), "CreatedAt must be <= after")
	assert.Equal(t, time.UTC, session.CreatedAt.Location(), "CreatedAt must be in UTC")

	// A freshly created session must not be expired or invalidated
	assert.False(t, session.IsExpired(), "A newly created session must not be expired")
	assert.False(t, session.Invalidated, "A newly created session must not be invalidated")
}

func TestSession_NewSession_EmptyOIDCSessionID_Allowed(t *testing.T) {
	// The OP may not issue a sid claim — this must be supported gracefully.
	userInfo := model.UserInfoPayload{Sub: testUserID}
	session, err := model.NewSession(testUserID, testTenantID, testIssuer, "", testAccessToken, testRefreshToken, testIDToken, userInfo, time.Hour)
	require.NoError(t, err)
	assert.Empty(t, session.OIDCSessionID, "Empty OIDCSessionID must be accepted when OP does not issue sid")
}

func TestSession_NewSession_UniqueIDs(t *testing.T) {
	userInfo := model.UserInfoPayload{Sub: "u1"}
	s1, err1 := model.NewSession("u1", "t1", testIssuer, "sid-1", "tok", "rtok", "id-tok", userInfo, time.Hour)
	s2, err2 := model.NewSession("u1", "t1", testIssuer, "sid-2", "tok", "rtok", "id-tok", userInfo, time.Hour)

	require.NoError(t, err1)
	require.NoError(t, err2)
	assert.NotEqual(t, s1.ID, s2.ID, "Each session must have a unique BFF session ID")
}

func TestSession_IsExpired_FreshSession(t *testing.T) {
	session := newTestSession(t, time.Hour)
	assert.False(t, session.IsExpired(), "Session with future expiry must not be expired")
}

func TestSession_IsExpired_ExpiredToken(t *testing.T) {
	// TTL of -1 second forces immediate expiry
	session := newTestSession(t, -time.Second)
	assert.True(t, session.IsExpired(), "Session with past expiry must be expired")
}

func TestSession_Invalidate(t *testing.T) {
	session := newTestSession(t, time.Hour)

	session.Invalidate()

	assert.True(t, session.Invalidated, "Session must be marked as invalidated")
	assert.Empty(t, session.AccessToken, "AccessToken must be cleared on Invalidate")
	assert.Empty(t, session.RefreshToken, "RefreshToken must be cleared on Invalidate")

	// Issuer and OIDCSessionID must be preserved after invalidation
	// so that the Redis store can still perform cleanup lookups.
	assert.Equal(t, testIssuer, session.Issuer, "Issuer must be preserved after Invalidate for audit")
	assert.Equal(t, testOIDCsid, session.OIDCSessionID, "OIDCSessionID must be preserved after Invalidate for index cleanup")
}

func TestSession_TokensNotSerializedToJSON(t *testing.T) {
	session := newTestSession(t, time.Hour)

	data, err := json.Marshal(session)
	require.NoError(t, err)
	jsonStr := string(data)

	// Tokens must be absent from the JSON representation.
	assert.NotContains(t, jsonStr, testAccessToken, "AccessToken must not appear in JSON output")
	assert.NotContains(t, jsonStr, testRefreshToken, "RefreshToken must not appear in JSON output")
	assert.NotContains(t, jsonStr, testIDToken, "IDToken must not appear in JSON output")

	// Non-sensitive fields must still be present.
	assert.Contains(t, jsonStr, testIssuer, "Issuer must appear in JSON output")
	assert.Contains(t, jsonStr, testOIDCsid, "OIDCSessionID must appear in JSON output")

	// Direct Go field access must still work.
	assert.Equal(t, testAccessToken, session.AccessToken)
	assert.Equal(t, testRefreshToken, session.RefreshToken)
	assert.Equal(t, testIDToken, session.IDToken)
}
