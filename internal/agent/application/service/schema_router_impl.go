package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/morphy76/tacito-square/internal/agent/application/ports/inbound"
	"github.com/morphy76/tacito-square/internal/agent/application/ports/outbound"
	"github.com/morphy76/tacito-square/internal/agent/domain/model"
	"github.com/morphy76/tacito-square/pkg/events"
	"github.com/rs/zerolog"
	"github.com/spf13/viper"
)

type SchemaRouterImpl struct {
	agentID   string
	agentName string
	processor inbound.MessageProcessor
	memory    outbound.ShortTermMemory
	ltm       outbound.LongTermMemory
	embed     outbound.Embedder
	brain     outbound.Brain
	publisher outbound.EventPublisher
	cfg       *viper.Viper
}

func NewSchemaRouterImpl(
	agentID string,
	agentName string,
	processor inbound.MessageProcessor,
	memory outbound.ShortTermMemory,
	ltm outbound.LongTermMemory,
	embed outbound.Embedder,
	brain outbound.Brain,
	publisher outbound.EventPublisher,
	cfg *viper.Viper,
) *SchemaRouterImpl {
	return &SchemaRouterImpl{
		agentID:   agentID,
		agentName: agentName,
		processor: processor,
		memory:    memory,
		ltm:       ltm,
		embed:     embed,
		brain:     brain,
		publisher: publisher,
		cfg:       cfg,
	}
}

func (r *SchemaRouterImpl) RouteEvent(ctx context.Context, event events.DomainEvent) error {
	logger := zerolog.Ctx(ctx).With().
		Str("event_id", event.EventID).
		Str("schema_ref", event.SchemaRef).
		Str("tenant_id", event.TenantID).
		Logger()
	ctx = logger.WithContext(ctx)

	switch event.SchemaRef {
	case events.SchemaConversationalStartThread:
		return r.handleStartThread(ctx, event)
	case events.SchemaConversationalAddUserMessage:
		return r.handleAddUserMessage(ctx, event)
	case events.SchemaConversationalEndThread:
		return r.handleEndThread(ctx, event)
	default:
		logger.Warn().Msg("unsupported event schema, skipping silently")
		return nil
	}
}

func (r *SchemaRouterImpl) handleStartThread(ctx context.Context, event events.DomainEvent) error {
	logger := *zerolog.Ctx(ctx)
	var payload events.StartThreadPayload
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		logger.Error().Err(err).Msg("failed to unmarshal StartThreadPayload")
		return fmt.Errorf("failed to unmarshal start-thread payload: %w", err)
	}

	logger = logger.With().Str("thread_id", payload.ThreadID).Logger()
	ctx = logger.WithContext(ctx)

	logger.Info().Msg("initializing short-term memory for thread")
	if err := r.memory.Clear(ctx, event.TenantID, r.agentID, payload.ThreadID); err != nil {
		logger.Error().Err(err).Msg("failed to clear short-term memory for thread")
		return fmt.Errorf("failed to clear short-term memory: %w", err)
	}

	return nil
}

func (r *SchemaRouterImpl) handleAddUserMessage(ctx context.Context, event events.DomainEvent) error {
	logger := *zerolog.Ctx(ctx)
	var payload events.AddUserMessagePayload
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		logger.Error().Err(err).Msg("failed to unmarshal AddUserMessagePayload")
		return fmt.Errorf("failed to unmarshal add-user-message payload: %w", err)
	}

	logger = logger.With().Str("thread_id", payload.ThreadID).Logger()
	ctx = logger.WithContext(ctx)

	logger.Info().Msg("processing incoming user message event")

	resp, err := r.processor.ProcessIncomingMessage(ctx, event.TenantID, r.agentID, payload.ThreadID, payload.Message)
	if err != nil {
		logger.Warn().Err(err).Msg("failed to process incoming user message, triggering STM rollback")
		// Rollback the user turn from short-term memory
		if rollbackErr := r.memory.RollbackLast(ctx, event.TenantID, r.agentID, payload.ThreadID); rollbackErr != nil {
			logger.Error().Err(rollbackErr).Msg("failed to rollback last memory entry after LLM failure")
		}
		return fmt.Errorf("failed to process message: %w", err)
	}

	// Construct agent response event
	respPayload := events.AgentResponsePayload{
		ThreadID:           payload.ThreadID,
		CommunityID:        payload.CommunityID,
		AgentName:          r.agentName,
		CorrelationEventID: event.EventID,
		Response:           resp,
		Finished:           true,
	}

	sourceIdentity := fmt.Sprintf("agent/%s", r.agentID)
	responseEvent, err := events.NewDomainEvent(
		events.SchemaConversationalAgentResponse,
		sourceIdentity,
		event.TenantID,
		respPayload,
	)
	if err != nil {
		logger.Error().Err(err).Msg("failed to construct agent-response domain event")
		return fmt.Errorf("failed to construct response event: %w", err)
	}

	eventData, err := json.Marshal(responseEvent)
	if err != nil {
		logger.Error().Err(err).Msg("failed to marshal agent-response event to JSON")
		return fmt.Errorf("failed to marshal response event: %w", err)
	}

	subject := fmt.Sprintf("ts.community.%s.agent.%s.response", payload.CommunityID, r.agentName)
	logger.Info().Str("subject", subject).Msg("publishing agent-response event")
	if err := r.publisher.Publish(ctx, subject, eventData); err != nil {
		logger.Error().Err(err).Msg("failed to publish agent-response event to NATS")
		return fmt.Errorf("failed to publish response event: %w", err)
	}

	return nil
}

func (r *SchemaRouterImpl) handleEndThread(ctx context.Context, event events.DomainEvent) error {
	logger := *zerolog.Ctx(ctx)
	var payload events.EndThreadPayload
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		logger.Error().Err(err).Msg("failed to unmarshal EndThreadPayload")
		return fmt.Errorf("failed to unmarshal end-thread payload: %w", err)
	}

	logger = logger.With().Str("thread_id", payload.ThreadID).Logger()
	ctx = logger.WithContext(ctx)

	logger.Info().Msg("ending conversation thread and consolidating memory")

	// 1. Fetch thread history from STM
	history, err := r.memory.Get(ctx, event.TenantID, r.agentID, payload.ThreadID, 0)
	if err != nil {
		logger.Warn().Err(err).Msg("failed to retrieve short-term memory history for end-thread")
	}

	// 2. Emit the thread history event if history exists and publisher is configured
	if len(history) > 0 && r.publisher != nil {
		var turns []events.ThreadTurn
		for _, turn := range history {
			turns = append(turns, events.ThreadTurn{
				Role:      turn.Role,
				Content:   turn.Content,
				Timestamp: turn.Timestamp.Format(time.RFC3339),
				Metadata:  turn.Metadata,
			})
		}
		histPayload := events.ThreadHistoryPayload{
			ThreadID:    payload.ThreadID,
			CommunityID: payload.CommunityID,
			History:     turns,
		}

		sourceIdentity := fmt.Sprintf("agent/%s", r.agentID)
		historyEvent, err := events.NewDomainEvent(
			events.SchemaConversationalThreadHistory,
			sourceIdentity,
			event.TenantID,
			histPayload,
		)
		if err != nil {
			logger.Error().Err(err).Msg("failed to construct thread-history domain event")
		} else {
			eventData, err := json.Marshal(historyEvent)
			if err != nil {
				logger.Error().Err(err).Msg("failed to marshal thread-history event to JSON")
			} else {
				subject := fmt.Sprintf("ts.community.%s.agent.%s.history", payload.CommunityID, r.agentName)
				logger.Info().Str("subject", subject).Msg("publishing thread-history event")
				if err := r.publisher.Publish(ctx, subject, eventData); err != nil {
					logger.Error().Err(err).Msg("failed to publish thread-history event to NATS")
				}
			}
		}
	}

	// 3. Consolidate to LTM if enabled
	bypassLTM := false
	if r.cfg != nil {
		bypassLTM = r.cfg.GetBool("bypass.ltm")
	}
	if !bypassLTM && r.ltm != nil && r.embed != nil && r.brain != nil && len(history) > 0 {
		// Compile turns into text block
		var sb strings.Builder
		sb.WriteString("Summarize and compress the core facts, declarations, and conversational details from these evicted turns:\n")
		for _, turn := range history {
			sb.WriteString(fmt.Sprintf("%s: %s\n", turn.Role, turn.Content))
		}

		// Generate summary via reasoning engine
		summaryResp, err := r.brain.Generate(ctx, model.BrainRequest{
			Prompt: sb.String(),
		})
		if err != nil {
			logger.Warn().Err(err).Msg("failed to summarize thread history for memory consolidation")
		} else {
			// Generate embedding vector
			summaryVector, err := r.embed.CreateEmbedding(ctx, summaryResp.Content)
			if err != nil {
				logger.Warn().Err(err).Msg("failed to generate embedding vector for memory consolidation")
			} else {
				// Save to Qdrant LTM
				ltmEntry := model.LTMEntry{
					ID:        uuid.New().String(),
					Content:   summaryResp.Content,
					Embedding: summaryVector,
					Type:      model.EntryTypeConversation,
					Source:    "thread_consolidator",
					Timestamp: time.Now().UTC(),
					Metadata: map[string]string{
						"visibility": "private",
						"thread_id":  payload.ThreadID,
					},
				}

				err = r.ltm.Save(ctx, event.TenantID, r.agentID, []model.LTMEntry{ltmEntry})
				if err != nil {
					logger.Warn().Err(err).Msg("failed to save consolidated memory to Qdrant LTM")
				} else {
					logger.Debug().Str("memory_id", ltmEntry.ID).Msg("thread memory consolidated and saved to Qdrant LTM")
				}
			}
		}
	}

	// Always clear STM for that thread
	if err := r.memory.Clear(ctx, event.TenantID, r.agentID, payload.ThreadID); err != nil {
		logger.Error().Err(err).Msg("failed to clear short-term memory for thread")
		return fmt.Errorf("failed to clear short-term memory: %w", err)
	}

	return nil
}
