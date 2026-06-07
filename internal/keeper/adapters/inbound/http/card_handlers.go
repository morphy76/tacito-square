package http

import (
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/morphy76/tacito-square/internal/keeper/application/ports/outbound"
	"github.com/morphy76/tacito-square/internal/keeper/domain/model"
	"github.com/morphy76/tacito-square/internal/shared/observability"
	"github.com/morphy76/tacito-square/internal/shared/tenant"
	"github.com/morphy76/tacito-square/pkg/agentcard"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

// CardHandler implements A2A discovery well-known endpoints with standard caching headers.
type CardHandler struct {
	agentRepo outbound.AgentRepository
	commRepo  outbound.CommunityRepository
}

// NewCardHandler creates a new CardHandler.
func NewCardHandler(agentRepo outbound.AgentRepository, commRepo outbound.CommunityRepository) *CardHandler {
	return &CardHandler{
		agentRepo: agentRepo,
		commRepo:  commRepo,
	}
}

// GetAgentCard handles GET /api/v1/communities/:community_id/agents/:agent_id/.well-known/agent-card.json
func (h *CardHandler) GetAgentCard(c *gin.Context) {
	ctx, span := otel.Tracer("keeper").Start(c.Request.Context(), "http.get_agent_card", trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logger := observability.NewLogger("info", os.Stdout)
	reqLogger := observability.WithContext(logger, ctx)

	ten := tenant.FromContext(ctx)
	if ten == nil {
		reqLogger.Warn().Msg("unauthorized: missing tenant context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant is required"})
		return
	}

	commIDStr := c.Param("community_id")
	commID, err := uuid.Parse(commIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid community_id UUID"})
		return
	}

	agentIDStr := c.Param("agent_id")
	agentID, err := uuid.Parse(agentIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid agent_id UUID"})
		return
	}

	card, lastSeen, err := h.agentRepo.GetRegistration(ctx, agentID, commID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "agent card registration not found"})
			return
		}
		reqLogger.Error().Err(err).Msg("failed to get agent card registration")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Serialize card to JSON bytes to compute ETag
	cardBytes, err := json.Marshal(card)
	if err != nil {
		reqLogger.Error().Err(err).Msg("failed to marshal card for ETag")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to serialize card"})
		return
	}

	etag := fmt.Sprintf(`W/"%x"`, sha1.Sum(cardBytes))
	c.Header("ETag", etag)

	lastModifiedStr := lastSeen.UTC().Format(http.TimeFormat)
	c.Header("Last-Modified", lastModifiedStr)

	expiresStr := time.Now().UTC().Add(30 * time.Second).Format(http.TimeFormat)
	c.Header("Expires", expiresStr)
	c.Header("Cache-Control", "public, max-age=30")

	// If-None-Match negotiation
	if c.GetHeader("If-None-Match") == etag {
		c.Status(http.StatusNotModified)
		return
	}

	// If-Modified-Since negotiation
	ifModSince := c.GetHeader("If-Modified-Since")
	if ifModSince != "" {
		parsedTime, err := time.Parse(http.TimeFormat, ifModSince)
		if err == nil {
			if !lastSeen.Truncate(time.Second).After(parsedTime.Truncate(time.Second)) {
				c.Status(http.StatusNotModified)
				return
			}
		}
	}

	c.Data(http.StatusOK, "application/json; charset=utf-8", cardBytes)
}

// GetCommunityCard handles GET /api/v1/communities/:community_id/.well-known/community-card.json
func (h *CardHandler) GetCommunityCard(c *gin.Context) {
	ctx, span := otel.Tracer("keeper").Start(c.Request.Context(), "http.get_community_card", trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logger := observability.NewLogger("info", os.Stdout)
	reqLogger := observability.WithContext(logger, ctx)

	ten := tenant.FromContext(ctx)
	if ten == nil {
		reqLogger.Warn().Msg("unauthorized: missing tenant context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant is required"})
		return
	}

	commIDStr := c.Param("community_id")
	commID, err := uuid.Parse(commIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid community_id UUID"})
		return
	}

	comm, err := h.commRepo.GetByID(ctx, commID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "community not found"})
			return
		}
		reqLogger.Error().Err(err).Msg("failed to get community")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	cards, latestTime, err := h.agentRepo.GetActiveRegistrationsByCommunity(ctx, commID)
	if err != nil {
		reqLogger.Error().Err(err).Msg("failed to get active registrations")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	agentSummaries := make([]model.CommunityAgentSummary, 0, len(cards))
	for _, card := range cards {
		caps := make([]string, 0)
		for _, skill := range card.Skills {
			caps = append(caps, skill.Name)
		}
		agentSummaries = append(agentSummaries, model.CommunityAgentSummary{
			ID:           "urn:agent:" + card.Name,
			Name:         card.Name,
			Version:      card.Version,
			Description:  card.Description,
			Capabilities: caps,
		})
	}

	commCard := &model.CommunityCard{
		CommunityID: comm.ID,
		Name:        comm.Name,
		Description: comm.Description,
		Topology:    string(comm.Topology),
		Status:      string(comm.Status),
		Agents:      agentSummaries,
	}

	commCardBytes, err := json.Marshal(commCard)
	if err != nil {
		reqLogger.Error().Err(err).Msg("failed to marshal community card")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to serialize community card"})
		return
	}

	// Dynamic caching headers
	lastModified := comm.UpdatedAt
	if latestTime.After(lastModified) {
		lastModified = latestTime
	}

	etag := fmt.Sprintf(`W/"%x"`, sha1.Sum(commCardBytes))
	c.Header("ETag", etag)

	lastModifiedStr := lastModified.UTC().Format(http.TimeFormat)
	c.Header("Last-Modified", lastModifiedStr)

	expiresStr := time.Now().UTC().Add(30 * time.Second).Format(http.TimeFormat)
	c.Header("Expires", expiresStr)
	c.Header("Cache-Control", "public, max-age=30")

	// If-None-Match negotiation
	if c.GetHeader("If-None-Match") == etag {
		c.Status(http.StatusNotModified)
		return
	}

	// If-Modified-Since negotiation
	ifModSince := c.GetHeader("If-Modified-Since")
	if ifModSince != "" {
		parsedTime, err := time.Parse(http.TimeFormat, ifModSince)
		if err == nil {
			if !lastModified.Truncate(time.Second).After(parsedTime.Truncate(time.Second)) {
				c.Status(http.StatusNotModified)
				return
			}
		}
	}

	c.Data(http.StatusOK, "application/json; charset=utf-8", commCardBytes)
}

// GetAgentCards handles GET /api/v1/communities/:community_id/.well-known/agent-cards.json
func (h *CardHandler) GetAgentCards(c *gin.Context) {
	ctx, span := otel.Tracer("keeper").Start(c.Request.Context(), "http.get_agent_cards", trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logger := observability.NewLogger("info", os.Stdout)
	reqLogger := observability.WithContext(logger, ctx)

	ten := tenant.FromContext(ctx)
	if ten == nil {
		reqLogger.Warn().Msg("unauthorized: missing tenant context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant is required"})
		return
	}

	commIDStr := c.Param("community_id")
	commID, err := uuid.Parse(commIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid community_id UUID"})
		return
	}

	// Verify community exists
	comm, err := h.commRepo.GetByID(ctx, commID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "community not found"})
			return
		}
		reqLogger.Error().Err(err).Msg("failed to get community")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	cards, latestTime, err := h.agentRepo.GetActiveRegistrationsByCommunity(ctx, commID)
	if err != nil {
		reqLogger.Error().Err(err).Msg("failed to get active registrations")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if cards == nil {
		cards = make([]*agentcard.AgentCard, 0)
	}

	cardsBytes, err := json.Marshal(cards)
	if err != nil {
		reqLogger.Error().Err(err).Msg("failed to marshal cards list")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to serialize cards list"})
		return
	}

	// Dynamic caching headers
	lastModified := latestTime
	if lastModified.IsZero() {
		lastModified = comm.UpdatedAt
	}

	etag := fmt.Sprintf(`W/"%x"`, sha1.Sum(cardsBytes))
	c.Header("ETag", etag)

	lastModifiedStr := lastModified.UTC().Format(http.TimeFormat)
	c.Header("Last-Modified", lastModifiedStr)

	expiresStr := time.Now().UTC().Add(30 * time.Second).Format(http.TimeFormat)
	c.Header("Expires", expiresStr)
	c.Header("Cache-Control", "public, max-age=30")

	// If-None-Match negotiation
	if c.GetHeader("If-None-Match") == etag {
		c.Status(http.StatusNotModified)
		return
	}

	// If-Modified-Since negotiation
	ifModSince := c.GetHeader("If-Modified-Since")
	if ifModSince != "" {
		parsedTime, err := time.Parse(http.TimeFormat, ifModSince)
		if err == nil {
			if !lastModified.Truncate(time.Second).After(parsedTime.Truncate(time.Second)) {
				c.Status(http.StatusNotModified)
				return
			}
		}
	}

	c.Data(http.StatusOK, "application/json; charset=utf-8", cardsBytes)
}
