//go:build integration

package outbound_test

import (
	"context"
	"log"
	"os"
	"testing"
	"time"

	bffoutbound "github.com/morphy76/tacito-square/internal/bff/adapters/outbound"
	"github.com/morphy76/tacito-square/internal/bff/domain/model"
	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	tcredis "github.com/testcontainers/testcontainers-go/modules/redis"
)

var redisAddr string

func TestMain(m *testing.M) {
	ctx := context.Background()

	log.Println("Starting Redis test container...")
	container, err := tcredis.Run(ctx, "redis:7-alpine")
	if err != nil {
		log.Fatalf("Failed to start Redis container: %v", err)
	}
	defer func() {
		if err := container.Terminate(ctx); err != nil {
			log.Printf("Failed to terminate Redis container: %v", err)
		}
	}()

	addr, err := container.Endpoint(ctx, "")
	if err != nil {
		log.Fatalf("Failed to get Redis endpoint: %v", err)
	}

	redisAddr = addr
	log.Printf("Redis container started at %s", redisAddr)

	os.Exit(m.Run())
}

func newTestStore(t *testing.T) *bffoutbound.RedisSessionStore {
	t.Helper()
	client := goredis.NewClient(&goredis.Options{Addr: redisAddr})
	t.Cleanup(func() { client.FlushAll(context.Background()) })
	return bffoutbound.NewRedisSessionStore(client, "test")
}

func newTestSession(t *testing.T) *model.Session {
	t.Helper()
	userInfo := model.UserInfoPayload{
		Sub:      "user-sub-123",
		Email:    "test@tacito.local",
		TenantID: "tenant-a",
	}
	sess, err := model.NewSession(
		"user-sub-123", "tenant-a", "https://issuer.tacito.local",
		"oidc-sid-abc", "access-tok", "refresh-tok",
		userInfo, 1*time.Hour,
	)
	require.NoError(t, err)
	return sess
}

func TestRedisSessionStore_SaveAndGet(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)
	sess := newTestSession(t)

	require.NoError(t, store.Save(ctx, sess, 5*time.Minute))

	got, err := store.Get(ctx, sess.ID)
	require.NoError(t, err)

	assert.Equal(t, sess.ID, got.ID)
	assert.Equal(t, sess.UserID, got.UserID)
	assert.Equal(t, sess.TenantID, got.TenantID)
	assert.Equal(t, sess.Issuer, got.Issuer)
	assert.Equal(t, sess.OIDCSessionID, got.OIDCSessionID)
	assert.Equal(t, sess.UserInfo, got.UserInfo)
}

func TestRedisSessionStore_Expiry(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)
	sess := newTestSession(t)

	require.NoError(t, store.Save(ctx, sess, 1*time.Second))
	time.Sleep(2 * time.Second)

	_, err := store.Get(ctx, sess.ID)
	assert.ErrorIs(t, err, model.ErrSessionNotFound)
}

func TestRedisSessionStore_Delete(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)
	sess := newTestSession(t)

	require.NoError(t, store.Save(ctx, sess, 5*time.Minute))
	require.NoError(t, store.Delete(ctx, sess.ID))

	_, err := store.Get(ctx, sess.ID)
	assert.ErrorIs(t, err, model.ErrSessionNotFound)
}

func TestRedisSessionStore_DeleteByUserID(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)

	userInfo := model.UserInfoPayload{Sub: "shared-user", TenantID: "tenant-a"}
	sess1, err := model.NewSession("shared-user", "tenant-a", "issuer", "sid-1", "tok1", "ref1", userInfo, 1*time.Hour)
	require.NoError(t, err)
	sess2, err := model.NewSession("shared-user", "tenant-a", "issuer", "sid-2", "tok2", "ref2", userInfo, 1*time.Hour)
	require.NoError(t, err)

	require.NoError(t, store.Save(ctx, sess1, 5*time.Minute))
	require.NoError(t, store.Save(ctx, sess2, 5*time.Minute))

	require.NoError(t, store.DeleteByUserID(ctx, "shared-user"))

	_, err = store.Get(ctx, sess1.ID)
	assert.ErrorIs(t, err, model.ErrSessionNotFound)
	_, err = store.Get(ctx, sess2.ID)
	assert.ErrorIs(t, err, model.ErrSessionNotFound)
}

func TestRedisSessionStore_DeleteByOIDCSessionID(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)
	sess := newTestSession(t)

	require.NoError(t, store.Save(ctx, sess, 5*time.Minute))
	require.NoError(t, store.DeleteByOIDCSessionID(ctx, "https://issuer.tacito.local", "oidc-sid-abc"))

	_, err := store.Get(ctx, sess.ID)
	assert.ErrorIs(t, err, model.ErrSessionNotFound)
}
