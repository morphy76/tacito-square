package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/morphy76/tacito-square/internal/keeper/application/ports/inbound"
	"github.com/morphy76/tacito-square/internal/keeper/application/service"
	"github.com/morphy76/tacito-square/internal/keeper/domain/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockEchoUseCase struct {
	mock.Mock
}

var _ inbound.EchoUseCase = (*mockEchoUseCase)(nil)

func (m *mockEchoUseCase) EchoCommunity(ctx context.Context, communityID uuid.UUID, message string) (*model.CommunityEchoResponse, error) {
	args := m.Called(ctx, communityID, message)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.CommunityEchoResponse), args.Error(1)
}

func TestEchoCommunity_OK(t *testing.T) {
	gin.SetMode(gin.TestMode)
	uc := new(mockEchoUseCase)
	handler := NewEchoHandler(uc)

	r := gin.New()
	r.Use(testTenantMiddleware())
	r.POST("/api/v1/communities/:community_id/echo", handler.EchoCommunity)

	commID := uuid.New()
	reqBody := `{"message": "hello"}`

	expectedResponse := &model.CommunityEchoResponse{
		CommunityID:   commID.String(),
		WokeCommunity: false,
		Results: []model.AgentEchoResult{
			{AgentName: "agent-1", Decorated: "echo-1", Error: ""},
			{AgentName: "agent-2", Decorated: "echo-2", Error: ""},
		},
	}

	uc.On("EchoCommunity", mock.Anything, commID, "hello").Return(expectedResponse, nil)

	req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/communities/%s/echo", commID), bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	r.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)

	var out model.CommunityEchoResponse
	err := json.Unmarshal(resp.Body.Bytes(), &out)
	require.NoError(t, err)
	assert.Equal(t, commID.String(), out.CommunityID)
	assert.False(t, out.WokeCommunity)
	assert.Len(t, out.Results, 2)
	uc.AssertExpectations(t)
}

func TestEchoCommunity_EmptyMessage_BadRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	uc := new(mockEchoUseCase)
	handler := NewEchoHandler(uc)

	r := gin.New()
	r.Use(testTenantMiddleware())
	r.POST("/api/v1/communities/:community_id/echo", handler.EchoCommunity)

	commID := uuid.New()
	reqBody := `{"message": ""}`

	// Binding will fail before use case call since "binding:required" is defined in our request struct,
	// or the use case returns ErrEmptyMessage. Let's verify both handle 400.
	uc.On("EchoCommunity", mock.Anything, commID, "").Return(nil, service.ErrEmptyMessage).Maybe()

	req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/communities/%s/echo", commID), bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	r.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusBadRequest, resp.Code)
	assert.Contains(t, resp.Body.String(), "error")
}

func TestEchoCommunity_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	uc := new(mockEchoUseCase)
	handler := NewEchoHandler(uc)

	r := gin.New()
	r.Use(testTenantMiddleware())
	r.POST("/api/v1/communities/:community_id/echo", handler.EchoCommunity)

	commID := uuid.New()
	reqBody := `{not valid}`

	req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/communities/%s/echo", commID), bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	r.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusBadRequest, resp.Code)
	assert.Contains(t, resp.Body.String(), "error")
}

func TestEchoCommunity_MissingMessageField(t *testing.T) {
	gin.SetMode(gin.TestMode)
	uc := new(mockEchoUseCase)
	handler := NewEchoHandler(uc)

	r := gin.New()
	r.Use(testTenantMiddleware())
	r.POST("/api/v1/communities/:community_id/echo", handler.EchoCommunity)

	commID := uuid.New()
	reqBody := `{}`

	req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/communities/%s/echo", commID), bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	r.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusBadRequest, resp.Code)
	assert.Contains(t, resp.Body.String(), "error")
}

func TestEchoCommunity_CommunityNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	uc := new(mockEchoUseCase)
	handler := NewEchoHandler(uc)

	r := gin.New()
	r.Use(testTenantMiddleware())
	r.POST("/api/v1/communities/:community_id/echo", handler.EchoCommunity)

	commID := uuid.New()
	reqBody := `{"message": "hello"}`

	uc.On("EchoCommunity", mock.Anything, commID, "hello").Return(nil, service.ErrCommunityNotFound)

	req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/communities/%s/echo", commID), bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	r.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusNotFound, resp.Code)
	assert.Contains(t, resp.Body.String(), "error")
	uc.AssertExpectations(t)
}

func TestEchoCommunity_NoRunningAgents(t *testing.T) {
	gin.SetMode(gin.TestMode)
	uc := new(mockEchoUseCase)
	handler := NewEchoHandler(uc)

	r := gin.New()
	r.Use(testTenantMiddleware())
	r.POST("/api/v1/communities/:community_id/echo", handler.EchoCommunity)

	commID := uuid.New()
	reqBody := `{"message": "hello"}`

	uc.On("EchoCommunity", mock.Anything, commID, "hello").Return(nil, service.ErrNoRunningAgents)

	req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/communities/%s/echo", commID), bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	r.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusServiceUnavailable, resp.Code)
	assert.Contains(t, resp.Body.String(), "error")
	uc.AssertExpectations(t)
}

func TestEchoCommunity_NATSUnavailable(t *testing.T) {
	gin.SetMode(gin.TestMode)
	uc := new(mockEchoUseCase)
	handler := NewEchoHandler(uc)

	r := gin.New()
	r.Use(testTenantMiddleware())
	r.POST("/api/v1/communities/:community_id/echo", handler.EchoCommunity)

	commID := uuid.New()
	reqBody := `{"message": "hello"}`

	uc.On("EchoCommunity", mock.Anything, commID, "hello").Return(nil, service.ErrBroadcasterUnavailable)

	req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/communities/%s/echo", commID), bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	r.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusServiceUnavailable, resp.Code)
	assert.Contains(t, resp.Body.String(), "NATS messaging is not available")
	uc.AssertExpectations(t)
}

func TestEchoCommunity_InvalidCommunityID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	uc := new(mockEchoUseCase)
	handler := NewEchoHandler(uc)

	r := gin.New()
	r.Use(testTenantMiddleware())
	r.POST("/api/v1/communities/:community_id/echo", handler.EchoCommunity)

	reqBody := `{"message": "hello"}`

	req, _ := http.NewRequest(http.MethodPost, "/api/v1/communities/invalid-uuid/echo", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	r.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusBadRequest, resp.Code)
	assert.Contains(t, resp.Body.String(), "error")
}

func TestEchoCommunity_MissingTenant(t *testing.T) {
	gin.SetMode(gin.TestMode)
	uc := new(mockEchoUseCase)
	handler := NewEchoHandler(uc)

	r := gin.New()
	r.POST("/api/v1/communities/:community_id/echo", handler.EchoCommunity)

	commID := uuid.New()
	reqBody := `{"message": "hello"}`

	req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/communities/%s/echo", commID), bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	r.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusUnauthorized, resp.Code)
	assert.Contains(t, resp.Body.String(), "error")
}

func TestEchoCommunity_InternalError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	uc := new(mockEchoUseCase)
	handler := NewEchoHandler(uc)

	r := gin.New()
	r.Use(testTenantMiddleware())
	r.POST("/api/v1/communities/:community_id/echo", handler.EchoCommunity)

	commID := uuid.New()
	reqBody := `{"message": "hello"}`

	uc.On("EchoCommunity", mock.Anything, commID, "hello").Return(nil, errors.New("something went wrong"))

	req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/communities/%s/echo", commID), bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	r.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusInternalServerError, resp.Code)
	assert.Contains(t, resp.Body.String(), "error")
	uc.AssertExpectations(t)
}
