package http_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	bffhttp "github.com/morphy76/tacito-square/internal/bff/adapters/inbound/http"
	"github.com/morphy76/tacito-square/internal/bff/application/ports/inbound"
	"github.com/morphy76/tacito-square/internal/bff/domain/model"
)

type mockEventStreamUseCase struct {
	StreamEventsFunc func(ctx context.Context, tenantID string) (<-chan []byte, error)
}

func (m *mockEventStreamUseCase) StreamEvents(ctx context.Context, tenantID string) (<-chan []byte, error) {
	if m.StreamEventsFunc != nil {
		return m.StreamEventsFunc(ctx, tenantID)
	}
	return nil, nil
}

var _ inbound.EventStreamUseCase = (*mockEventStreamUseCase)(nil)

func TestSSEHandler_StreamEvents_ForwardsEvents(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	mockSessionUC := &mockSessionUseCase{
		GetSessionFunc: func(ctx context.Context, sessionID string) (*model.Session, error) {
			return &model.Session{
				ID:       "session-123",
				TenantID: "tenant-789",
			}, nil
		},
	}

	ch := make(chan []byte, 3)
	ch <- []byte("event-1")
	ch <- []byte("event-2")
	ch <- []byte("event-3")
	close(ch)

	mockEventUC := &mockEventStreamUseCase{
		StreamEventsFunc: func(ctx context.Context, tenantID string) (<-chan []byte, error) {
			assert.Equal(t, "tenant-789", tenantID)
			return ch, nil
		},
	}

	bffhttp.RegisterRoutes(r, mockSessionUC, mockEventUC, "/ui")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/ui/api/v1/events/stream", nil)
	req.AddCookie(&http.Cookie{
		Name:  "bff_session_id",
		Value: "session-123",
	})

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "text/event-stream", w.Header().Get("Content-Type"))
	assert.Equal(t, "no-cache", w.Header().Get("Cache-Control"))
	assert.Equal(t, "keep-alive", w.Header().Get("Connection"))

	body := w.Body.String()
	assert.Contains(t, body, "data: event-1\n\n")
	assert.Contains(t, body, "data: event-2\n\n")
	assert.Contains(t, body, "data: event-3\n\n")
}

func TestSSEHandler_StreamEvents_RequiresAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	mockSessionUC := &mockSessionUseCase{
		GetSessionFunc: func(ctx context.Context, sessionID string) (*model.Session, error) {
			return nil, errors.New("unauthorized")
		},
	}
	mockEventUC := &mockEventStreamUseCase{}

	bffhttp.RegisterRoutes(r, mockSessionUC, mockEventUC, "/ui")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/ui/api/v1/events/stream", nil)
	// No session cookie

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
