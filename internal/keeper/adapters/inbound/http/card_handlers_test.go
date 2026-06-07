package http_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	keeperhttp "github.com/morphy76/tacito-square/internal/keeper/adapters/inbound/http"
	"github.com/morphy76/tacito-square/internal/keeper/application/ports/outbound"
	"github.com/morphy76/tacito-square/internal/keeper/domain/model"
	"github.com/morphy76/tacito-square/internal/shared/tenant"
	"github.com/morphy76/tacito-square/pkg/agentcard"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockAgentRepo struct {
	outbound.AgentRepository
	mock.Mock
}

func (m *mockAgentRepo) GetRegistration(ctx context.Context, agentID uuid.UUID, communityID uuid.UUID) (*agentcard.AgentCard, time.Time, error) {
	args := m.Called(ctx, agentID, communityID)
	if args.Get(0) == nil {
		return nil, time.Time{}, args.Error(2)
	}
	return args.Get(0).(*agentcard.AgentCard), args.Get(1).(time.Time), args.Error(2)
}

func (m *mockAgentRepo) GetActiveRegistrationsByCommunity(ctx context.Context, communityID uuid.UUID) ([]*agentcard.AgentCard, time.Time, error) {
	args := m.Called(ctx, communityID)
	if args.Get(0) == nil {
		return nil, time.Time{}, args.Error(2)
	}
	return args.Get(0).([]*agentcard.AgentCard), args.Get(1).(time.Time), args.Error(2)
}

type mockCommunityRepo struct {
	outbound.CommunityRepository
	mock.Mock
}

func (m *mockCommunityRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.Community, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Community), args.Error(1)
}

func testTenantMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ten, _ := tenant.New("test-tenant.com", "")
		c.Request = c.Request.WithContext(tenant.ContextWithTenant(c.Request.Context(), ten))
		c.Next()
	}
}

func TestCardHandler_GetAgentCard(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Get Agent Card Successfully with Caching Headers", func(t *testing.T) {
		agentRepo := new(mockAgentRepo)
		commRepo := new(mockCommunityRepo)
		handler := keeperhttp.NewCardHandler(agentRepo, commRepo)

		r := gin.New()
		r.Use(testTenantMiddleware())
		r.GET("/api/v1/communities/:community_id/agents/:agent_id/.well-known/agent-card.json", handler.GetAgentCard)

		commID := uuid.New()
		agentID := uuid.New()

		card := &agentcard.AgentCard{
			Name:        "agent-alpha",
			Description: "Alpha agent",
			URL:         "http://agent-alpha",
			Version:     "1.0.0",
		}

		lastSeen := time.Now().Add(-10 * time.Minute).UTC()

		agentRepo.On("GetRegistration", mock.Anything, agentID, commID).Return(card, lastSeen, nil)

		req, _ := http.NewRequest(http.MethodGet, "/api/v1/communities/"+commID.String()+"/agents/"+agentID.String()+"/.well-known/agent-card.json", nil)
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		assert.Equal(t, "public, max-age=30", resp.Header().Get("Cache-Control"))

		// Verify Expires is set
		assert.NotEmpty(t, resp.Header().Get("Expires"))

		// Verify ETag format
		etag := resp.Header().Get("ETag")
		assert.True(t, strings.HasPrefix(etag, `W/"`))

		// Verify Last-Modified
		lastModified := resp.Header().Get("Last-Modified")
		assert.Equal(t, lastSeen.Format(http.TimeFormat), lastModified)

		var result agentcard.AgentCard
		err := json.Unmarshal(resp.Body.Bytes(), &result)
		require.NoError(t, err)
		assert.Equal(t, card.Name, result.Name)

		// Sub-Test: If-None-Match Content Negotiation
		reqNoneMatch, _ := http.NewRequest(http.MethodGet, "/api/v1/communities/"+commID.String()+"/agents/"+agentID.String()+"/.well-known/agent-card.json", nil)
		reqNoneMatch.Header.Set("If-None-Match", etag)
		respNoneMatch := httptest.NewRecorder()
		r.ServeHTTP(respNoneMatch, reqNoneMatch)

		assert.Equal(t, http.StatusNotModified, respNoneMatch.Code)
		assert.Empty(t, respNoneMatch.Body.String())

		// Sub-Test: If-Modified-Since Content Negotiation (Not Modified)
		reqModSince, _ := http.NewRequest(http.MethodGet, "/api/v1/communities/"+commID.String()+"/agents/"+agentID.String()+"/.well-known/agent-card.json", nil)
		reqModSince.Header.Set("If-Modified-Since", lastSeen.Format(http.TimeFormat))
		respModSince := httptest.NewRecorder()
		r.ServeHTTP(respModSince, reqModSince)

		assert.Equal(t, http.StatusNotModified, respModSince.Code)
		assert.Empty(t, respModSince.Body.String())

		// Sub-Test: If-Modified-Since Content Negotiation (Modified)
		reqModSinceOld, _ := http.NewRequest(http.MethodGet, "/api/v1/communities/"+commID.String()+"/agents/"+agentID.String()+"/.well-known/agent-card.json", nil)
		reqModSinceOld.Header.Set("If-Modified-Since", lastSeen.Add(-1*time.Hour).Format(http.TimeFormat))
		respModSinceOld := httptest.NewRecorder()
		r.ServeHTTP(respModSinceOld, reqModSinceOld)

		assert.Equal(t, http.StatusOK, respModSinceOld.Code)
		assert.NotEmpty(t, respModSinceOld.Body.String())
	})

	t.Run("Get Agent Card Not Found", func(t *testing.T) {
		agentRepo := new(mockAgentRepo)
		commRepo := new(mockCommunityRepo)
		handler := keeperhttp.NewCardHandler(agentRepo, commRepo)

		r := gin.New()
		r.Use(testTenantMiddleware())
		r.GET("/api/v1/communities/:community_id/agents/:agent_id/.well-known/agent-card.json", handler.GetAgentCard)

		commID := uuid.New()
		agentID := uuid.New()

		agentRepo.On("GetRegistration", mock.Anything, agentID, commID).Return(nil, time.Time{}, errors.New("registration not found"))

		req, _ := http.NewRequest(http.MethodGet, "/api/v1/communities/"+commID.String()+"/agents/"+agentID.String()+"/.well-known/agent-card.json", nil)
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusNotFound, resp.Code)
	})
}

func TestCardHandler_GetCommunityCard(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Get Community Card Successfully with Caching Headers", func(t *testing.T) {
		agentRepo := new(mockAgentRepo)
		commRepo := new(mockCommunityRepo)
		handler := keeperhttp.NewCardHandler(agentRepo, commRepo)

		r := gin.New()
		r.Use(testTenantMiddleware())
		r.GET("/api/v1/communities/:community_id/.well-known/community-card.json", handler.GetCommunityCard)

		commID := uuid.New()
		comm := &model.Community{
			ID:          commID,
			Name:        "developer-community",
			Description: "A test community",
			Topology:    "hub-spoke",
			Status:      "active",
			UpdatedAt:   time.Now().Add(-20 * time.Minute).UTC(),
		}

		card := &agentcard.AgentCard{
			Name:        "agent-alpha",
			Description: "Alpha agent",
			Version:     "1.0.0",
			Skills: []agentcard.AgentCardSkill{
				{Name: "code-analysis"},
			},
		}

		latestSeen := time.Now().Add(-5 * time.Minute).UTC()

		commRepo.On("GetByID", mock.Anything, commID).Return(comm, nil)
		agentRepo.On("GetActiveRegistrationsByCommunity", mock.Anything, commID).Return([]*agentcard.AgentCard{card}, latestSeen, nil)

		req, _ := http.NewRequest(http.MethodGet, "/api/v1/communities/"+commID.String()+"/.well-known/community-card.json", nil)
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		assert.Equal(t, "public, max-age=30", resp.Header().Get("Cache-Control"))
		assert.NotEmpty(t, resp.Header().Get("Expires"))
		assert.NotEmpty(t, resp.Header().Get("ETag"))

		// Last-Modified should be the latestSeen timestamp because it's newer than comm.UpdatedAt
		assert.Equal(t, latestSeen.Format(http.TimeFormat), resp.Header().Get("Last-Modified"))

		var result model.CommunityCard
		err := json.Unmarshal(resp.Body.Bytes(), &result)
		require.NoError(t, err)
		assert.Equal(t, comm.Name, result.Name)

		// Sub-Test: If-None-Match
		etag := resp.Header().Get("ETag")
		reqNoneMatch, _ := http.NewRequest(http.MethodGet, "/api/v1/communities/"+commID.String()+"/.well-known/community-card.json", nil)
		reqNoneMatch.Header.Set("If-None-Match", etag)
		respNoneMatch := httptest.NewRecorder()
		r.ServeHTTP(respNoneMatch, reqNoneMatch)

		assert.Equal(t, http.StatusNotModified, respNoneMatch.Code)
		assert.Empty(t, respNoneMatch.Body.String())
	})
}

func TestCardHandler_GetAgentCards(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Get Agent Cards List Successfully with Caching Headers", func(t *testing.T) {
		agentRepo := new(mockAgentRepo)
		commRepo := new(mockCommunityRepo)
		handler := keeperhttp.NewCardHandler(agentRepo, commRepo)

		r := gin.New()
		r.Use(testTenantMiddleware())
		r.GET("/api/v1/communities/:community_id/.well-known/agent-cards.json", handler.GetAgentCards)

		commID := uuid.New()
		comm := &model.Community{
			ID:        commID,
			UpdatedAt: time.Now().Add(-10 * time.Minute).UTC(),
		}

		card := &agentcard.AgentCard{
			Name:    "agent-alpha",
			Version: "1.0.0",
		}

		latestSeen := time.Now().Add(-2 * time.Minute).UTC()

		commRepo.On("GetByID", mock.Anything, commID).Return(comm, nil)
		agentRepo.On("GetActiveRegistrationsByCommunity", mock.Anything, commID).Return([]*agentcard.AgentCard{card}, latestSeen, nil)

		req, _ := http.NewRequest(http.MethodGet, "/api/v1/communities/"+commID.String()+"/.well-known/agent-cards.json", nil)
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		assert.Equal(t, "public, max-age=30", resp.Header().Get("Cache-Control"))
		assert.NotEmpty(t, resp.Header().Get("Expires"))
		assert.NotEmpty(t, resp.Header().Get("ETag"))
		assert.Equal(t, latestSeen.Format(http.TimeFormat), resp.Header().Get("Last-Modified"))

		var result []*agentcard.AgentCard
		err := json.Unmarshal(resp.Body.Bytes(), &result)
		require.NoError(t, err)
		require.Len(t, result, 1)
		assert.Equal(t, "agent-alpha", result[0].Name)

		// Sub-Test: If-None-Match
		etag := resp.Header().Get("ETag")
		reqNoneMatch, _ := http.NewRequest(http.MethodGet, "/api/v1/communities/"+commID.String()+"/.well-known/agent-cards.json", nil)
		reqNoneMatch.Header.Set("If-None-Match", etag)
		respNoneMatch := httptest.NewRecorder()
		r.ServeHTTP(respNoneMatch, reqNoneMatch)

		assert.Equal(t, http.StatusNotModified, respNoneMatch.Code)
	})
}
