package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"unicode"

	"github.com/google/uuid"
	"github.com/morphy76/tacito-square/internal/keeper/application/ports/outbound"
	"github.com/morphy76/tacito-square/internal/keeper/domain/model"
	"github.com/morphy76/tacito-square/internal/shared/tenant"
	"github.com/morphy76/tacito-square/pkg/events"
	"github.com/rs/zerolog/log"
)

// EventServiceImpl implements the EventUseCase and EventStreamUseCase ports.
type EventServiceImpl struct {
	publisher  outbound.EventPublisher
	subscriber outbound.EventSubscriber
	commRepo   outbound.CommunityRepository
}

// NewEventService constructs a new EventServiceImpl.
func NewEventService(publisher outbound.EventPublisher, subscriber outbound.EventSubscriber, commRepo outbound.CommunityRepository) *EventServiceImpl {
	return &EventServiceImpl{
		publisher:  publisher,
		subscriber: subscriber,
		commRepo:   commRepo,
	}
}

// PublishEvent validates, sanitizes, wraps, and dispatches a domain event.
func (s *EventServiceImpl) PublishEvent(ctx context.Context, schemaRef string, payload json.RawMessage) (events.DomainEvent, error) {
	if schemaRef == "" || !strings.HasPrefix(schemaRef, "urn:tacito:schema:") {
		return events.DomainEvent{}, errors.New("invalid schema_ref: must match urn:tacito:schema:*")
	}

	if !json.Valid(payload) {
		return events.DomainEvent{}, errors.New("payload must be a valid JSON object or array")
	}

	// Resolve tenant ID
	ten := tenant.FromContext(ctx)
	var tenantID string
	if ten != nil {
		tenantID = ten.FullName()
	} else if val, ok := ctx.Value("tenant_id").(string); ok {
		tenantID = val
		if parsedTen, err := tenant.New(tenantID, ""); err == nil {
			ctx = tenant.ContextWithTenant(ctx, parsedTen)
		}
	}
	if tenantID == "" {
		return events.DomainEvent{}, errors.New("unauthorized: missing tenant context")
	}

	// Resolve hostname for source
	hostname, err := os.Hostname()
	if err != nil || hostname == "" {
		hostname = "local"
	}
	source := fmt.Sprintf("keeper/%s", hostname)

	// Determine routing and perform schema-specific actions
	var subject string
	if strings.HasPrefix(schemaRef, "urn:tacito:schema:conversational:") {
		var routeInfo struct {
			CommunityID string `json:"community_id"`
			AgentName   string `json:"agent_name"`
		}
		if err := json.Unmarshal(payload, &routeInfo); err != nil {
			return events.DomainEvent{}, fmt.Errorf("failed to parse routing info from payload: %w", err)
		}
		if routeInfo.CommunityID == "" {
			return events.DomainEvent{}, errors.New("community_id is required in conversational event payload")
		}
		// agent_name is optional

		// Handle threadID generation if missing in start-thread
		if schemaRef == events.SchemaConversationalStartThread {
			var stPayload events.StartThreadPayload
			if err := json.Unmarshal(payload, &stPayload); err != nil {
				return events.DomainEvent{}, fmt.Errorf("failed to unmarshal start-thread payload: %w", err)
			}
			if stPayload.ThreadID == "" {
				stPayload.ThreadID = uuid.New().String()
				updatedPayload, err := json.Marshal(stPayload)
				if err != nil {
					return events.DomainEvent{}, fmt.Errorf("failed to marshal updated start-thread payload: %w", err)
				}
				payload = json.RawMessage(updatedPayload)
			}
		}

		if routeInfo.AgentName == "" {
			subject = s.resolveTopologySubject(ctx, routeInfo.CommunityID)
		} else {
			subject = fmt.Sprintf("ts.community.%s.agent.%s", routeInfo.CommunityID, routeInfo.AgentName)
		}

		// Sanitize if add-user-message
		if schemaRef == events.SchemaConversationalAddUserMessage {
			var payloadMap map[string]any
			if err := json.Unmarshal(payload, &payloadMap); err != nil {
				return events.DomainEvent{}, err
			}

			msgVal, ok := payloadMap["message"]
			if !ok {
				return events.DomainEvent{}, errors.New("message field is required")
			}
			msgStr, ok := msgVal.(string)
			if !ok {
				return events.DomainEvent{}, errors.New("message must be a string")
			}

			sanitized := sanitizeMessage(msgStr)
			if len(sanitized) == 0 {
				return events.DomainEvent{}, errors.New("message must not be empty after sanitization")
			}

			payloadMap["message"] = sanitized
			updatedPayload, err := json.Marshal(payloadMap)
			if err != nil {
				return events.DomainEvent{}, fmt.Errorf("failed to marshal sanitized payload: %w", err)
			}
			payload = json.RawMessage(updatedPayload)
		}
	} else {
		subject = fmt.Sprintf("ts.events.%s", tenantID)
	}

	// Construct DomainEvent
	evt, err := events.NewDomainEvent(schemaRef, source, tenantID, json.RawMessage(payload))
	if err != nil {
		return events.DomainEvent{}, fmt.Errorf("failed to construct domain event: %w", err)
	}

	// Publish
	if err := s.publisher.Publish(ctx, subject, evt); err != nil {
		return events.DomainEvent{}, fmt.Errorf("failed to publish event: %w", err)
	}

	return evt, nil
}

// SubscribeEvents registers a handler for real-time community events streaming.
func (s *EventServiceImpl) SubscribeEvents(ctx context.Context, tenantID string, handler func(*events.DomainEvent)) (outbound.EventSubscription, error) {
	return s.subscriber.Subscribe(ctx, "ts.community.>", tenantID, handler)
}

// resolveTopologySubject determines the NATS subject based on community topology.
// For hub-spoke communities, routes to the hub agent. For single-agent or unknown
// communities, falls back to the broadcast subject.
func (s *EventServiceImpl) resolveTopologySubject(ctx context.Context, communityID string) string {
	fallback := fmt.Sprintf("ts.community.%s.agent.all", communityID)
	logger := log.Ctx(ctx)

	if s.commRepo == nil {
		logger.Warn().
			Str("community_id", communityID).
			Msg("community repository not available: routing to broadcast subject")
		return fallback
	}

	commUUID, err := uuid.Parse(communityID)
	if err != nil {
		logger.Warn().
			Str("community_id", communityID).
			Err(err).
			Msg("invalid community UUID: routing to broadcast subject")
		return fallback
	}

	community, err := s.commRepo.GetByID(ctx, commUUID)
	if err != nil {
		logger.Warn().
			Str("community_id", communityID).
			Err(err).
			Msg("failed to resolve community topology: routing to broadcast subject")
		return fallback
	}

	if community.Topology == model.CommunityTopologyHubSpoke {
		subject := fmt.Sprintf("ts.community.%s.agent.hub", communityID)
		logger.Debug().
			Str("community_id", communityID).
			Str("topology", string(community.Topology)).
			Str("subject", subject).
			Msg("resolved hub-spoke routing subject")
		return subject
	}

	logger.Debug().
		Str("community_id", communityID).
		Str("topology", string(community.Topology)).
		Msg("single-agent topology: routing to broadcast subject")
	return fallback
}

func sanitizeMessage(s string) string {
	runes := make([]rune, 0, len(s))
	for _, r := range s {
		if unicode.IsPrint(r) {
			runes = append(runes, r)
		}
	}
	if len(runes) > 4096 {
		runes = runes[:4096]
	}
	return string(runes)
}
