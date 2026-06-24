package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/morphy76/tacito-square/internal/bff/application/ports/outbound"
	"github.com/morphy76/tacito-square/internal/bff/application/service"
	"github.com/morphy76/tacito-square/internal/bff/domain/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockSessionStore struct {
	mock.Mock
}

func (m *mockSessionStore) Save(ctx context.Context, session *model.Session, ttl time.Duration) error {
	return m.Called(ctx, session, ttl).Error(0)
}

func (m *mockSessionStore) Get(ctx context.Context, sessionID string) (*model.Session, error) {
	args := m.Called(ctx, sessionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Session), args.Error(1)
}

func (m *mockSessionStore) Delete(ctx context.Context, sessionID string) error {
	return m.Called(ctx, sessionID).Error(0)
}

func (m *mockSessionStore) DeleteByUserID(ctx context.Context, userID string) error {
	return m.Called(ctx, userID).Error(0)
}

func (m *mockSessionStore) DeleteByOIDCSessionID(ctx context.Context, issuer string, oidcSessionID string) error {
	return m.Called(ctx, issuer, oidcSessionID).Error(0)
}

func (m *mockSessionStore) SavePendingState(ctx context.Context, state, redirectTo string, ttl time.Duration) error {
	return m.Called(ctx, state, redirectTo, ttl).Error(0)
}

func (m *mockSessionStore) GetAndDeletePendingState(ctx context.Context, state string) (string, error) {
	args := m.Called(ctx, state)
	return args.String(0), args.Error(1)
}

type mockOIDCProvider struct {
	mock.Mock
}

func (m *mockOIDCProvider) ExchangeCode(ctx context.Context, code, redirectURI string) (*outbound.TokenSet, error) {
	args := m.Called(ctx, code, redirectURI)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*outbound.TokenSet), args.Error(1)
}

func (m *mockOIDCProvider) RefreshToken(ctx context.Context, refreshToken string) (*outbound.TokenSet, error) {
	args := m.Called(ctx, refreshToken)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*outbound.TokenSet), args.Error(1)
}

func (m *mockOIDCProvider) FetchUserInfo(ctx context.Context, accessToken string) (*model.UserInfoPayload, error) {
	args := m.Called(ctx, accessToken)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.UserInfoPayload), args.Error(1)
}

func (m *mockOIDCProvider) ValidateLogoutToken(ctx context.Context, rawToken string) (sub string, sessionID string, err error) {
	args := m.Called(ctx, rawToken)
	return args.String(0), args.String(1), args.Error(2)
}

func TestSessionService_HandleCallback_Success(t *testing.T) {
	store := &mockSessionStore{}
	oidc := &mockOIDCProvider{}
	cfg := service.SessionConfig{
		ClientID:     "client-id",
		ClientSecret: "client-secret",
		RedirectURI:  "http://localhost:8080/callback",
		SessionTTL:   24 * time.Hour,
		Issuer:       "https://issuer.tacito.local",
	}
	svc := service.NewSessionService(store, oidc, cfg)

	code := "auth-code"
	state := "auth-state"
	ctx := context.Background()

	tokenSet := &outbound.TokenSet{
		AccessToken:  "access-token",
		RefreshToken: "refresh-token",
		ExpiresIn:    1 * time.Hour,
	}

	userInfo := &model.UserInfoPayload{
		Sub:            "user-sub",
		Email:          "user@tacito.local",
		TenantID:       "tenant-1",
		SubscriptionID: "sub-1",
	}

	oidc.On("ExchangeCode", ctx, code, cfg.RedirectURI).Return(tokenSet, nil)
	oidc.On("FetchUserInfo", ctx, "access-token").Return(userInfo, nil)
	store.On("Save", ctx, mock.AnythingOfType("*model.Session"), cfg.SessionTTL).Return(nil).Run(func(args mock.Arguments) {
		sess := args.Get(1).(*model.Session)
		assert.Equal(t, "user-sub", sess.UserID)
		assert.Equal(t, "tenant-1", sess.TenantID)
		assert.Equal(t, "access-token", sess.AccessToken)
		assert.Equal(t, "refresh-token", sess.RefreshToken)
		assert.Equal(t, cfg.Issuer, sess.Issuer)
	})
	store.On("GetAndDeletePendingState", ctx, state).Return("", nil)

	session, _, err := svc.HandleCallback(ctx, code, state)
	assert.NoError(t, err)
	assert.NotNil(t, session)
	assert.Equal(t, "user-sub", session.UserID)
	assert.Equal(t, "tenant-1", session.TenantID)
	mock.AssertExpectationsForObjects(t, store, oidc)
}

func TestSessionService_InitiateLogin_StoresPendingState(t *testing.T) {
	store := &mockSessionStore{}
	oidc := &mockOIDCProvider{}
	cfg := service.SessionConfig{
		ClientID:    "client-id",
		RedirectURI: "http://localhost:8080/callback",
		Issuer:      "https://issuer.tacito.local",
		SessionTTL:  24 * time.Hour,
	}
	svc := service.NewSessionService(store, oidc, cfg)
	ctx := context.Background()

	// Expect SavePendingState to be called with any state value and the given redirectTo
	store.On("SavePendingState", ctx, mock.AnythingOfType("string"), "/ui/secure/page", 5*time.Minute).Return(nil)

	_, state, err := svc.InitiateLogin(ctx, "/ui/secure/page")
	assert.NoError(t, err)
	assert.NotEmpty(t, state)
	mock.AssertExpectationsForObjects(t, store, oidc)
}

func TestSessionService_HandleCallback_ReturnsRedirectTo(t *testing.T) {
	store := &mockSessionStore{}
	oidc := &mockOIDCProvider{}
	cfg := service.SessionConfig{
		ClientID:     "client-id",
		ClientSecret: "client-secret",
		RedirectURI:  "http://localhost:8080/callback",
		SessionTTL:   24 * time.Hour,
		Issuer:       "https://issuer.tacito.local",
	}
	svc := service.NewSessionService(store, oidc, cfg)

	code := "auth-code"
	state := "some-state-nonce"
	ctx := context.Background()

	tokenSet := &outbound.TokenSet{
		AccessToken:  "access-token",
		RefreshToken: "refresh-token",
		ExpiresIn:    1 * time.Hour,
	}
	userInfo := &model.UserInfoPayload{
		Sub:            "user-sub",
		Email:          "user@tacito.local",
		TenantID:       "tenant-1",
		SubscriptionID: "sub-1",
	}

	oidc.On("ExchangeCode", ctx, code, cfg.RedirectURI).Return(tokenSet, nil)
	oidc.On("FetchUserInfo", ctx, "access-token").Return(userInfo, nil)
	store.On("Save", ctx, mock.AnythingOfType("*model.Session"), cfg.SessionTTL).Return(nil)
	store.On("GetAndDeletePendingState", ctx, state).Return("/ui/secure/page", nil)

	session, redirectTo, err := svc.HandleCallback(ctx, code, state)
	assert.NoError(t, err)
	assert.NotNil(t, session)
	assert.Equal(t, "/ui/secure/page", redirectTo)
	mock.AssertExpectationsForObjects(t, store, oidc)
}

func TestSessionService_HandleCallback_ExchangeError(t *testing.T) {
	store := &mockSessionStore{}
	oidc := &mockOIDCProvider{}
	cfg := service.SessionConfig{
		RedirectURI: "http://localhost:8080/callback",
		SessionTTL:  24 * time.Hour,
	}
	svc := service.NewSessionService(store, oidc, cfg)

	ctx := context.Background()
	oidc.On("ExchangeCode", ctx, "bad-code", cfg.RedirectURI).Return(nil, errors.New("exchange failed"))

	session, _, err := svc.HandleCallback(ctx, "bad-code", "state")
	assert.Error(t, err)
	assert.Nil(t, session)
	assert.Contains(t, err.Error(), "exchange failed")
	mock.AssertExpectationsForObjects(t, store, oidc)
}

func TestSessionService_RefreshSession_ExpiredToken(t *testing.T) {
	store := &mockSessionStore{}
	oidc := &mockOIDCProvider{}
	cfg := service.SessionConfig{
		SessionTTL: 24 * time.Hour,
	}
	svc := service.NewSessionService(store, oidc, cfg)

	ctx := context.Background()
	userInfo := model.UserInfoPayload{Sub: "user-sub", TenantID: "tenant-1"}
	// Expired session
	sess, err := model.NewSession("user-sub", "tenant-1", "issuer", "oidc-sid", "old-access", "old-refresh", userInfo, -1*time.Minute)
	assert.NoError(t, err)

	tokenSet := &outbound.TokenSet{
		AccessToken:  "new-access-token",
		RefreshToken: "new-refresh-token",
		ExpiresIn:    1 * time.Hour,
	}

	store.On("Get", ctx, sess.ID).Return(sess, nil)
	oidc.On("RefreshToken", ctx, "old-refresh").Return(tokenSet, nil)
	store.On("Save", ctx, mock.AnythingOfType("*model.Session"), cfg.SessionTTL).Return(nil).Run(func(args mock.Arguments) {
		updated := args.Get(1).(*model.Session)
		assert.Equal(t, "new-access-token", updated.AccessToken)
		assert.Equal(t, "new-refresh-token", updated.RefreshToken)
		assert.True(t, updated.AccessTokenExpiresAt.After(time.Now()))
	})

	refreshed, err := svc.RefreshSession(ctx, sess.ID)
	assert.NoError(t, err)
	assert.NotNil(t, refreshed)
	assert.Equal(t, "new-access-token", refreshed.AccessToken)
	mock.AssertExpectationsForObjects(t, store, oidc)
}

func TestSessionService_RefreshSession_ValidToken(t *testing.T) {
	store := &mockSessionStore{}
	oidc := &mockOIDCProvider{}
	cfg := service.SessionConfig{}
	svc := service.NewSessionService(store, oidc, cfg)

	ctx := context.Background()
	userInfo := model.UserInfoPayload{Sub: "user-sub", TenantID: "tenant-1"}
	// Valid session (1 hour TTL remaining)
	sess, err := model.NewSession("user-sub", "tenant-1", "issuer", "oidc-sid", "valid-access", "valid-refresh", userInfo, 1*time.Hour)
	assert.NoError(t, err)

	store.On("Get", ctx, sess.ID).Return(sess, nil)
	// We assert that RefreshToken and Save are NOT called because the token is still valid.

	refreshed, err := svc.RefreshSession(ctx, sess.ID)
	assert.NoError(t, err)
	assert.NotNil(t, refreshed)
	assert.Equal(t, "valid-access", refreshed.AccessToken)
	mock.AssertExpectationsForObjects(t, store, oidc)
}

func TestSessionService_BackchannelLogout_SubOnly(t *testing.T) {
	store := &mockSessionStore{}
	oidc := &mockOIDCProvider{}
	cfg := service.SessionConfig{}
	svc := service.NewSessionService(store, oidc, cfg)

	ctx := context.Background()
	rawToken := "logout-token-jwt"

	oidc.On("ValidateLogoutToken", ctx, rawToken).Return("user-sub", "", nil)
	store.On("DeleteByUserID", ctx, "user-sub").Return(nil)

	err := svc.BackchannelLogout(ctx, rawToken)
	assert.NoError(t, err)
	mock.AssertExpectationsForObjects(t, store, oidc)
}

func TestSessionService_BackchannelLogout_SessionID(t *testing.T) {
	store := &mockSessionStore{}
	oidc := &mockOIDCProvider{}
	cfg := service.SessionConfig{
		Issuer: "https://issuer.tacito.local",
	}
	svc := service.NewSessionService(store, oidc, cfg)

	ctx := context.Background()
	rawToken := "logout-token-jwt-sid"

	oidc.On("ValidateLogoutToken", ctx, rawToken).Return("", "oidc-sid-val", nil)
	store.On("DeleteByOIDCSessionID", ctx, cfg.Issuer, "oidc-sid-val").Return(nil)

	err := svc.BackchannelLogout(ctx, rawToken)
	assert.NoError(t, err)
	mock.AssertExpectationsForObjects(t, store, oidc)
}

func TestSessionService_Logout_DeletesSession(t *testing.T) {
	store := &mockSessionStore{}
	oidc := &mockOIDCProvider{}
	cfg := service.SessionConfig{}
	svc := service.NewSessionService(store, oidc, cfg)

	ctx := context.Background()
	sessionID := "sess-123"

	store.On("Delete", ctx, sessionID).Return(nil)

	err := svc.Logout(ctx, sessionID)
	assert.NoError(t, err)
	mock.AssertExpectationsForObjects(t, store, oidc)
}
