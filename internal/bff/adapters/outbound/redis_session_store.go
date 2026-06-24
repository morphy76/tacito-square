package outbound

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/morphy76/tacito-square/internal/bff/application/ports/outbound"
	"github.com/morphy76/tacito-square/internal/bff/domain/model"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

// Compile-time interface satisfaction assertion.
var _ outbound.SessionStore = (*RedisSessionStore)(nil)

// RedisSessionStore implements outbound.SessionStore using Redis.
// Key layout:
//
//	bff:<prefix>:session:<sessionID>        → JSON-serialized Session
//	bff:<prefix>:user-sessions:<userID>     → Redis Set of sessionIDs
//	bff:<prefix>:oidc-sessions:<issuer>:<sid> → sessionID string
//	bff:<prefix>:pending-state:<state>      → redirect-to URL string (short-lived)
type RedisSessionStore struct {
	client *redis.Client
	prefix string
}

// NewRedisSessionStore creates a new RedisSessionStore.
// prefix is the Viper-configured key namespace (e.g., "bff" or "bff-dev").
func NewRedisSessionStore(client *redis.Client, prefix string) *RedisSessionStore {
	return &RedisSessionStore{client: client, prefix: prefix}
}

func (s *RedisSessionStore) sessionKey(sessionID string) string {
	return fmt.Sprintf("bff:%s:session:%s", s.prefix, sessionID)
}

func (s *RedisSessionStore) userSessionsKey(userID string) string {
	return fmt.Sprintf("bff:%s:user-sessions:%s", s.prefix, userID)
}

func (s *RedisSessionStore) oidcSessionKey(issuer, oidcSessionID string) string {
	return fmt.Sprintf("bff:%s:oidc-session:%s:%s", s.prefix, issuer, oidcSessionID)
}

func (s *RedisSessionStore) pendingStateKey(state string) string {
	return fmt.Sprintf("bff:%s:pending-state:%s", s.prefix, state)
}

// Save persists the session. Note: AccessToken and RefreshToken are serialized via
// a dedicated sessionDTO that preserves the json:"-" tags in Session by using explicit fields.
func (s *RedisSessionStore) Save(ctx context.Context, sess *model.Session, ttl time.Duration) error {
	dto := sessionDTO{
		ID:                   sess.ID,
		UserID:               sess.UserID,
		TenantID:             sess.TenantID,
		Issuer:               sess.Issuer,
		OIDCSessionID:        sess.OIDCSessionID,
		AccessToken:          sess.AccessToken,
		RefreshToken:         sess.RefreshToken,
		IDToken:              sess.IDToken,
		UserInfo:             sess.UserInfo,
		AccessTokenExpiresAt: sess.AccessTokenExpiresAt,
		CreatedAt:            sess.CreatedAt,
		Invalidated:          sess.Invalidated,
	}

	data, err := json.Marshal(dto)
	if err != nil {
		return fmt.Errorf("redis session store: marshal: %w", err)
	}

	pipe := s.client.Pipeline()
	pipe.Set(ctx, s.sessionKey(sess.ID), data, ttl)
	pipe.SAdd(ctx, s.userSessionsKey(sess.UserID), sess.ID)
	if sess.OIDCSessionID != "" {
		pipe.Set(ctx, s.oidcSessionKey(sess.Issuer, sess.OIDCSessionID), sess.ID, ttl)
	}
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("redis session store: pipeline exec: %w", err)
	}

	log.Debug().
		Str("session_id", sess.ID).
		Str("user_id", sess.UserID).
		Msg("Session saved to Redis")

	return nil
}

// Get retrieves a session by its ID.
func (s *RedisSessionStore) Get(ctx context.Context, sessionID string) (*model.Session, error) {
	data, err := s.client.Get(ctx, s.sessionKey(sessionID)).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, model.ErrSessionNotFound
		}
		return nil, fmt.Errorf("redis session store: get: %w", err)
	}

	var dto sessionDTO
	if err := json.Unmarshal(data, &dto); err != nil {
		return nil, fmt.Errorf("redis session store: unmarshal: %w", err)
	}

	return dtoToSession(dto), nil
}

// Delete removes a specific session by its ID.
func (s *RedisSessionStore) Delete(ctx context.Context, sessionID string) error {
	// Retrieve the session first to clean up secondary indexes.
	sess, err := s.Get(ctx, sessionID)
	if err != nil {
		if errors.Is(err, model.ErrSessionNotFound) {
			return nil // idempotent
		}
		return err
	}

	pipe := s.client.Pipeline()
	pipe.Del(ctx, s.sessionKey(sessionID))
	pipe.SRem(ctx, s.userSessionsKey(sess.UserID), sessionID)
	if sess.OIDCSessionID != "" {
		pipe.Del(ctx, s.oidcSessionKey(sess.Issuer, sess.OIDCSessionID))
	}
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("redis session store: delete pipeline: %w", err)
	}

	log.Debug().Str("session_id", sessionID).Msg("Session deleted from Redis")
	return nil
}

// DeleteByUserID removes all sessions associated with the given user ID (sub-based backchannel logout).
func (s *RedisSessionStore) DeleteByUserID(ctx context.Context, userID string) error {
	key := s.userSessionsKey(userID)
	sessionIDs, err := s.client.SMembers(ctx, key).Result()
	if err != nil {
		return fmt.Errorf("redis session store: smembers: %w", err)
	}

	for _, id := range sessionIDs {
		if err := s.Delete(ctx, id); err != nil {
			log.Warn().Str("session_id", id).Err(err).Msg("Failed to delete session during DeleteByUserID")
		}
	}

	s.client.Del(ctx, key)

	log.Info().Str("user_id", userID).Int("count", len(sessionIDs)).Msg("All sessions deleted for user")
	return nil
}

// DeleteByOIDCSessionID removes the session associated with the given OIDC issuer + sid (sid-based backchannel logout).
func (s *RedisSessionStore) DeleteByOIDCSessionID(ctx context.Context, issuer, oidcSessionID string) error {
	oidcKey := s.oidcSessionKey(issuer, oidcSessionID)
	sessionID, err := s.client.Get(ctx, oidcKey).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil // idempotent
		}
		return fmt.Errorf("redis session store: get oidc session key: %w", err)
	}

	if err := s.Delete(ctx, sessionID); err != nil {
		return err
	}

	log.Info().Str("oidc_sid", oidcSessionID).Str("session_id", sessionID).Msg("Session deleted by OIDC session ID")
	return nil
}

// Ping checks the connectivity to the Redis database.
func (s *RedisSessionStore) Ping(ctx context.Context) error {
	return s.client.Ping(ctx).Err()
}

// SavePendingState stores the redirect-to URL keyed by the OIDC state nonce.
func (s *RedisSessionStore) SavePendingState(ctx context.Context, state, redirectTo string, ttl time.Duration) error {
	if err := s.client.Set(ctx, s.pendingStateKey(state), redirectTo, ttl).Err(); err != nil {
		return fmt.Errorf("redis session store: save pending state: %w", err)
	}
	log.Debug().Str("state", state).Msg("Pending OIDC state redirect_to saved to Redis")
	return nil
}

// GetAndDeletePendingState atomically retrieves and removes the redirect-to URL for the given state nonce.
// Returns an empty string (no error) when the key is not found.
func (s *RedisSessionStore) GetAndDeletePendingState(ctx context.Context, state string) (string, error) {
	val, err := s.client.GetDel(ctx, s.pendingStateKey(state)).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return "", nil // no stored redirect — caller falls back to default
		}
		return "", fmt.Errorf("redis session store: get pending state: %w", err)
	}
	log.Debug().Str("state", state).Str("redirect_to", val).Msg("Pending OIDC state redirect_to retrieved from Redis")
	return val, nil
}

// sessionDTO is a serialization-safe representation of a Session that preserves token fields.
type sessionDTO struct {
	ID                   string               `json:"id"`
	UserID               string               `json:"user_id"`
	TenantID             string               `json:"tenant_id"`
	Issuer               string               `json:"issuer"`
	OIDCSessionID        string               `json:"oidc_session_id"`
	AccessToken          string               `json:"access_token"`
	RefreshToken         string               `json:"refresh_token"`
	IDToken              string               `json:"id_token"`
	UserInfo             model.UserInfoPayload `json:"user_info"`
	AccessTokenExpiresAt time.Time            `json:"access_token_expires_at"`
	CreatedAt            time.Time            `json:"created_at"`
	Invalidated          bool                 `json:"invalidated"`
}

func dtoToSession(dto sessionDTO) *model.Session {
	return &model.Session{
		ID:                   dto.ID,
		UserID:               dto.UserID,
		TenantID:             dto.TenantID,
		Issuer:               dto.Issuer,
		OIDCSessionID:        dto.OIDCSessionID,
		AccessToken:          dto.AccessToken,
		RefreshToken:         dto.RefreshToken,
		IDToken:              dto.IDToken,
		UserInfo:             dto.UserInfo,
		AccessTokenExpiresAt: dto.AccessTokenExpiresAt,
		CreatedAt:            dto.CreatedAt,
		Invalidated:          dto.Invalidated,
	}
}
