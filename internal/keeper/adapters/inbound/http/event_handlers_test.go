package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/morphy76/tacito-square/internal/keeper/application/ports/outbound"
	"github.com/morphy76/tacito-square/pkg/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockEventUseCase struct {
	mock.Mock
}

func (m *mockEventUseCase) PublishEvent(ctx context.Context, schemaRef string, payload json.RawMessage) (events.DomainEvent, error) {
	args := m.Called(ctx, schemaRef, payload)
	if args.Get(0) == nil {
		return events.DomainEvent{}, args.Error(1)
	}
	return args.Get(0).(events.DomainEvent), args.Error(1)
}

type mockSubscription struct {
	mock.Mock
}

func (m *mockSubscription) Stop() error {
	args := m.Called()
	return args.Error(0)
}

type mockEventStreamUseCase struct {
	mock.Mock
}

func (m *mockEventStreamUseCase) SubscribeEvents(ctx context.Context, tenantID string, handler func(*events.DomainEvent)) (outbound.EventSubscription, error) {
	args := m.Called(ctx, tenantID, handler)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(outbound.EventSubscription), args.Error(1)
}

func TestPublishEvent_HTTP_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	pubUC := new(mockEventUseCase)
	streamUC := new(mockEventStreamUseCase)
	handler := NewEventHandler(pubUC, streamUC)

	r := gin.New()
	r.Use(testTenantMiddleware())
	r.POST("/api/v1/events", handler.PublishEvent)

	reqBody := `{"schema_ref":"urn:tacito:schema:conversational:start-thread:v1","payload":{"thread_id":"t1","community_id":"c1","agent_name":"a1"}}`
	simulatedEvent := events.DomainEvent{
		EventID:    "evt-123",
		SchemaRef:  "urn:tacito:schema:conversational:start-thread:v1",
		Source:     "keeper/local",
		TenantID:   "tenant-1",
		OccurredAt: "2026-06-06T08:00:00Z",
		Payload:    json.RawMessage(`{"thread_id":"t1","community_id":"c1","agent_name":"a1"}`),
	}

	pubUC.On("PublishEvent", mock.Anything, "urn:tacito:schema:conversational:start-thread:v1", json.RawMessage(`{"thread_id":"t1","community_id":"c1","agent_name":"a1"}`)).
		Return(simulatedEvent, nil)

	req, _ := http.NewRequest(http.MethodPost, "/api/v1/events", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	r.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusAccepted, resp.Code)
	assert.Equal(t, "/api/v1/communities/c1/threads/t1", resp.Header().Get("Location"))

	var out events.DomainEvent
	err := json.Unmarshal(resp.Body.Bytes(), &out)
	require.NoError(t, err)
	assert.Equal(t, "evt-123", out.EventID)
	assert.Equal(t, "tenant-1", out.TenantID)
	pubUC.AssertExpectations(t)
}

func TestPublishEvent_HTTP_SanitizationEmptyError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	pubUC := new(mockEventUseCase)
	streamUC := new(mockEventStreamUseCase)
	handler := NewEventHandler(pubUC, streamUC)

	r := gin.New()
	r.Use(testTenantMiddleware())
	r.POST("/api/v1/events", handler.PublishEvent)

	reqBody := `{"schema_ref":"urn:tacito:schema:conversational:add-user-message:v1","payload":{"message":"\u0000"}}`

	pubUC.On("PublishEvent", mock.Anything, "urn:tacito:schema:conversational:add-user-message:v1", json.RawMessage(`{"message":"\u0000"}`)).
		Return(events.DomainEvent{}, errors.New("message must not be empty after sanitization"))

	req, _ := http.NewRequest(http.MethodPost, "/api/v1/events", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	r.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusBadRequest, resp.Code)
	assert.Contains(t, resp.Body.String(), "message must not be empty after sanitization")
	pubUC.AssertExpectations(t)
}

func TestPublishEvent_HTTP_InvalidSchemaRef(t *testing.T) {
	gin.SetMode(gin.TestMode)
	pubUC := new(mockEventUseCase)
	streamUC := new(mockEventStreamUseCase)
	handler := NewEventHandler(pubUC, streamUC)

	r := gin.New()
	r.Use(testTenantMiddleware())
	r.POST("/api/v1/events", handler.PublishEvent)

	reqBody := `{"schema_ref":"invalid-urn","payload":{}}`

	pubUC.On("PublishEvent", mock.Anything, "invalid-urn", json.RawMessage(`{}`)).
		Return(events.DomainEvent{}, errors.New("invalid schema_ref: must match urn:tacito:schema:*"))

	req, _ := http.NewRequest(http.MethodPost, "/api/v1/events", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	r.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusUnprocessableEntity, resp.Code)
	assert.Contains(t, resp.Body.String(), "invalid schema_ref")
}

type closeNotifyingRecorder struct {
	*httptest.ResponseRecorder
	closed chan bool
}

func (c *closeNotifyingRecorder) CloseNotify() <-chan bool {
	return c.closed
}

func TestStreamEvents_HTTP_SSE(t *testing.T) {
	gin.SetMode(gin.TestMode)
	pubUC := new(mockEventUseCase)
	streamUC := new(mockEventStreamUseCase)
	handler := NewEventHandler(pubUC, streamUC)

	r := gin.New()
	r.Use(testTenantMiddleware())
	r.GET("/api/v1/events/stream", handler.StreamEvents)

	sub := new(mockSubscription)
	sub.On("Stop").Return(nil)

	// Mock SubscribeEvents to invoke handler immediately
	streamUC.On("SubscribeEvents", mock.Anything, "test-tenant.com", mock.Anything).
		Run(func(args mock.Arguments) {
			h := args.Get(2).(func(*events.DomainEvent))
			evt := &events.DomainEvent{
				EventID:    "evt-123",
				SchemaRef:  "urn:tacito:schema:conversational:start-thread:v1",
				Source:     "keeper/local",
				TenantID:   "test-tenant.com",
				OccurredAt: "2026-06-06T08:00:00Z",
				Payload:    json.RawMessage(`{}`),
			}
			h(evt)
		}).
		Return(sub, nil)

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/events/stream", nil)
	resp := httptest.NewRecorder()
	closedChan := make(chan bool, 1)
	customWriter := &closeNotifyingRecorder{
		ResponseRecorder: resp,
		closed:           closedChan,
	}

	// Since SSE stream keeps running, we need to run in a goroutine or set a timeout context
	ctx, cancel := context.WithCancel(context.Background())
	req = req.WithContext(ctx)

	// We close the context after a brief period to stop the stream
	go func() {
		time.Sleep(100 * time.Millisecond)
		closedChan <- true
		cancel()
	}()

	r.ServeHTTP(customWriter, req)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, "text/event-stream", resp.Header().Get("Content-Type"))
	assert.Equal(t, "no-cache", resp.Header().Get("Cache-Control"))
	assert.Equal(t, "keep-alive", resp.Header().Get("Connection"))
	assert.Contains(t, resp.Body.String(), "event: start-thread")
	assert.Contains(t, resp.Body.String(), "id: evt-123")
	assert.Contains(t, resp.Body.String(), `data: {`)
}

func TestStreamEvents_HTTP_SSE_Blacklisted(t *testing.T) {
	gin.SetMode(gin.TestMode)
	pubUC := new(mockEventUseCase)
	streamUC := new(mockEventStreamUseCase)
	handler := NewEventHandler(pubUC, streamUC)

	r := gin.New()
	r.Use(testTenantMiddleware())
	r.GET("/api/v1/events/stream", handler.StreamEvents)

	sub := new(mockSubscription)
	sub.On("Stop").Return(nil)

	// Mock SubscribeEvents to invoke handler immediately with a blacklisted heartbeat event
	streamUC.On("SubscribeEvents", mock.Anything, "test-tenant.com", mock.Anything).
		Run(func(args mock.Arguments) {
			h := args.Get(2).(func(*events.DomainEvent))
			evt := &events.DomainEvent{
				EventID:    "evt-hb",
				SchemaRef:  events.SchemaInfrastructureAgentHeartbeat,
				Source:     "agent/1",
				TenantID:   "test-tenant.com",
				OccurredAt: "2026-06-06T08:00:00Z",
				Payload:    json.RawMessage(`{}`),
			}
			h(evt)
		}).
		Return(sub, nil)

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/events/stream", nil)
	resp := httptest.NewRecorder()
	closedChan := make(chan bool, 1)
	customWriter := &closeNotifyingRecorder{
		ResponseRecorder: resp,
		closed:           closedChan,
	}

	ctx, cancel := context.WithCancel(context.Background())
	req = req.WithContext(ctx)

	go func() {
		time.Sleep(100 * time.Millisecond)
		closedChan <- true
		cancel()
	}()

	r.ServeHTTP(customWriter, req)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.NotContains(t, resp.Body.String(), "evt-hb")
	assert.NotContains(t, resp.Body.String(), "agent-heartbeat")
}

func TestStreamEvents_HTTP_SSE_AgentDelegation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	pubUC := new(mockEventUseCase)
	streamUC := new(mockEventStreamUseCase)
	handler := NewEventHandler(pubUC, streamUC)

	r := gin.New()
	r.Use(testTenantMiddleware())
	r.GET("/api/v1/events/stream", handler.StreamEvents)

	sub := new(mockSubscription)
	sub.On("Stop").Return(nil)

	// Mock SubscribeEvents to invoke handler with an agent-delegation event
	streamUC.On("SubscribeEvents", mock.Anything, "test-tenant.com", mock.Anything).
		Run(func(args mock.Arguments) {
			h := args.Get(2).(func(*events.DomainEvent))
			evt := &events.DomainEvent{
				EventID:    "evt-deleg-1",
				SchemaRef:  events.SchemaConversationalAgentDelegation,
				Source:     "agent/hub-123",
				TenantID:   "test-tenant.com",
				OccurredAt: "2026-06-06T08:00:00Z",
				Payload:    json.RawMessage(`{"thread_id":"t-1","community_id":"c-1","delegating_agent":"hub","target_agent":"writer","message":"write"}`),
			}
			h(evt)
		}).
		Return(sub, nil)

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/events/stream", nil)
	resp := httptest.NewRecorder()
	closedChan := make(chan bool, 1)
	customWriter := &closeNotifyingRecorder{
		ResponseRecorder: resp,
		closed:           closedChan,
	}

	ctx, cancel := context.WithCancel(context.Background())
	req = req.WithContext(ctx)

	go func() {
		time.Sleep(100 * time.Millisecond)
		closedChan <- true
		cancel()
	}()

	r.ServeHTTP(customWriter, req)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Contains(t, resp.Body.String(), "event: agent-delegation")
	assert.Contains(t, resp.Body.String(), "id: evt-deleg-1")
	assert.Contains(t, resp.Body.String(), `data: {`)
}
