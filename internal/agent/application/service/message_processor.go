package service

import (
	"context"
	"os"
	"strconv"
	"time"

	"github.com/morphy76/tacito-square/internal/agent/application/ports/outbound"
	"github.com/morphy76/tacito-square/internal/agent/domain/model"
	"github.com/rs/zerolog"
)

type MessageProcessorService struct {
	brain  outbound.Brain
	memory outbound.ShortTermMemory
	limit  int
}

func NewMessageProcessorService(brain outbound.Brain, memory outbound.ShortTermMemory) *MessageProcessorService {
	limit := 10
	if envLimitVal := os.Getenv("TS_AGENT_STM_LIMIT"); envLimitVal != "" {
		if val, err := strconv.Atoi(envLimitVal); err == nil && val > 0 {
			limit = val
		}
	}
	return &MessageProcessorService{
		brain:  brain,
		memory: memory,
		limit:  limit,
	}
}

func (s *MessageProcessorService) ProcessIncomingMessage(ctx context.Context, tenantID, agentID, threadID string, payload string) (string, error) {
	logger := zerolog.Ctx(ctx).With().
		Str("tenant_id", tenantID).
		Str("agent_id", agentID).
		Str("thread_id", threadID).
		Logger()
	ctx = logger.WithContext(ctx)

	logger.Info().Str("payload", payload).Msg("processing incoming message via brain reasoning engine with STM")

	// Step 1: Append User Turn
	userEntry := model.MemoryEntry{
		Role:      "user",
		Content:   payload,
		Timestamp: time.Now().UTC(),
	}
	if err := s.memory.Append(ctx, tenantID, agentID, threadID, userEntry); err != nil {
		logger.Warn().Err(err).Msg("failed to append user message to short-term memory (graceful degradation)")
	}

	// Step 2: Fetch history
	history, err := s.memory.Get(ctx, tenantID, agentID, threadID, s.limit)
	if err != nil {
		logger.Warn().Err(err).Msg("failed to retrieve short-term memory history (graceful degradation, falling back to current turn only)")
		history = []model.MemoryEntry{userEntry}
	}

	// Step 3: Trigger reasoning engine
	req := model.BrainRequest{
		Prompt:  payload,
		History: history,
	}
	resp, err := s.brain.Generate(ctx, req)
	if err != nil {
		logger.Error().Err(err).Msg("message processing failed in brain reasoning engine")
		return "", err
	}

	// Step 4: Append Assistant Turn
	assistantEntry := model.MemoryEntry{
		Role:      "assistant",
		Content:   resp.Content,
		Timestamp: time.Now().UTC(),
	}
	if err := s.memory.Append(ctx, tenantID, agentID, threadID, assistantEntry); err != nil {
		logger.Warn().Err(err).Msg("failed to append assistant response to short-term memory (graceful degradation)")
	}

	logger.Info().
		Int("prompt_tokens", resp.Usage.PromptTokens).
		Int("completion_tokens", resp.Usage.CompletionTokens).
		Int("total_tokens", resp.Usage.TotalTokens).
		Str("finish_reason", resp.FinishReason).
		Msg("message processing completed successfully")

	return resp.Content, nil
}
