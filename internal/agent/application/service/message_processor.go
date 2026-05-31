package service

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/morphy76/tacito-square/internal/agent/application/ports/outbound"
	"github.com/morphy76/tacito-square/internal/agent/domain/model"
	"github.com/rs/zerolog"
)

type MessageProcessorService struct {
	brain     outbound.Brain
	memory    outbound.ShortTermMemory
	ltm       outbound.LongTermMemory
	embed     outbound.Embedder
	cogEngine *CognitiveEngine
	limit     int
}

func NewMessageProcessorService(
	brain outbound.Brain,
	memory outbound.ShortTermMemory,
	ltm outbound.LongTermMemory,
	embed outbound.Embedder,
	cogEngine *CognitiveEngine,
) *MessageProcessorService {
	limit := 10
	if envLimitVal := os.Getenv("TS_AGENT_STM_LIMIT"); envLimitVal != "" {
		if val, err := strconv.Atoi(envLimitVal); err == nil && val > 0 {
			limit = val
		}
	}
	return &MessageProcessorService{
		brain:     brain,
		memory:    memory,
		ltm:       ltm,
		embed:     embed,
		cogEngine: cogEngine,
		limit:     limit,
	}
}

func (s *MessageProcessorService) ProcessIncomingMessage(ctx context.Context, tenantID, agentID, threadID string, payload string) (string, error) {
	logger := zerolog.Ctx(ctx).With().
		Str("tenant_id", tenantID).
		Str("agent_id", agentID).
		Str("thread_id", threadID).
		Logger()
	ctx = logger.WithContext(ctx)

	logger.Info().Str("payload", payload).Msg("processing incoming message via brain reasoning engine with STM and LTM")

	// Step 1: Append User Turn to STM
	userEntry := model.MemoryEntry{
		Role:      "user",
		Content:   payload,
		Timestamp: time.Now().UTC(),
	}
	if err := s.memory.Append(ctx, tenantID, agentID, threadID, userEntry); err != nil {
		logger.Warn().Err(err).Msg("failed to append user message to short-term memory (graceful degradation)")
	}

	// Step 2: Fetch active sliding window history from STM
	history, err := s.memory.Get(ctx, tenantID, agentID, threadID, s.limit)
	if err != nil {
		logger.Warn().Err(err).Msg("failed to retrieve short-term memory history (graceful degradation, falling back to current turn only)")
		history = []model.MemoryEntry{userEntry}
	}

	// Step 3: Trigger active reasoning loop engine
	respContent, err := s.cogEngine.ExecuteReasoningLoop(ctx, tenantID, agentID, threadID, payload, history, "")
	if err != nil {
		logger.Error().Err(err).Msg("message processing failed in cognitive loop engine")
		return "", err
	}

	// Step 4: Append Assistant Turn to STM
	assistantEntry := model.MemoryEntry{
		Role:      "assistant",
		Content:   respContent,
		Timestamp: time.Now().UTC(),
	}
	if err := s.memory.Append(ctx, tenantID, agentID, threadID, assistantEntry); err != nil {
		logger.Warn().Err(err).Msg("failed to append assistant response to short-term memory (graceful degradation)")
	}

	// Step 5: Passive Memory Consolidation on Eviction
	if s.embed != nil && s.ltm != nil {
		s.triggerMemoryConsolidation(ctx, tenantID, agentID, threadID)
	}

	logger.Info().Msg("message processing completed successfully")

	return respContent, nil
}

func (s *MessageProcessorService) triggerMemoryConsolidation(ctx context.Context, tenantID, agentID, threadID string) {
	logger := zerolog.Ctx(ctx)

	// Fetch ALL history turns to check if limit is exceeded
	history, err := s.memory.Get(ctx, tenantID, agentID, threadID, 0)
	if err != nil {
		logger.Warn().Err(err).Msg("failed to fetch STM history for consolidation check")
		return
	}

	if len(history) <= s.limit {
		return
	}

	// Calculate counts to pop
	count := len(history) - s.limit
	evictedTurns := history[:count]

	logger.Info().Int("evicted_turns_count", count).Msg("STM limit exceeded, initiating passive memory consolidation")

	// Trigger consolidation asynchronously in a separate context to avoid blocking the reasoning reply loop
	bgCtx := context.WithoutCancel(ctx)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger.Error().Interface("recover", r).Msg("panic recovered in async memory consolidator")
			}
		}()

		// 1. Compile turns into text block
		var sb strings.Builder
		sb.WriteString("Summarize and compress the core facts, declarations, and conversational details from these evicted turns:\n")
		for _, turn := range evictedTurns {
			sb.WriteString(fmt.Sprintf("%s: %s\n", turn.Role, turn.Content))
		}

		// 2. Summarize via reasoning engine
		summaryResp, err := s.brain.Generate(bgCtx, model.BrainRequest{
			Prompt: sb.String(),
		})
		if err != nil {
			logger.Warn().Err(err).Msg("failed to summarize evicted turns for memory consolidation")
			return
		}

		// 3. Generate embedding vector
		summaryVector, err := s.embed.CreateEmbedding(bgCtx, summaryResp.Content)
		if err != nil {
			logger.Warn().Err(err).Msg("failed to generate embedding vector for memory consolidation")
			return
		}

		// 4. Save to Qdrant LTM
		ltmEntry := model.LTMEntry{
			ID:        uuid.New().String(),
			Content:   summaryResp.Content,
			Embedding: summaryVector,
			Type:      model.EntryTypeConversation,
			Source:    "eviction_consolidator",
			Timestamp: time.Now().UTC(),
			Metadata: map[string]string{
				"visibility": "private",
				"thread_id":  threadID,
			},
		}

		err = s.ltm.Save(bgCtx, tenantID, agentID, []model.LTMEntry{ltmEntry})
		if err != nil {
			logger.Warn().Err(err).Msg("failed to save consolidated memory to Qdrant LTM")
			return
		}

		logger.Info().Str("memory_id", ltmEntry.ID).Msg("passive memory consolidation successfully executed and saved to Qdrant LTM")
	}()
}
