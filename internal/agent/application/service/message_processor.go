package service

import (
	"context"

	"github.com/morphy76/tacito-square/internal/agent/application/ports/outbound"
	"github.com/morphy76/tacito-square/internal/agent/domain/model"
	"github.com/rs/zerolog"
)

type MessageProcessorService struct {
	brain outbound.Brain
}

func NewMessageProcessorService(brain outbound.Brain) *MessageProcessorService {
	return &MessageProcessorService{
		brain: brain,
	}
}

func (s *MessageProcessorService) ProcessIncomingMessage(ctx context.Context, payload string) (string, error) {
	logger := zerolog.Ctx(ctx)
	logger.Info().Str("payload", payload).Msg("processing incoming message via brain reasoning engine")

	req := model.BrainRequest{
		Prompt: payload,
	}
	resp, err := s.brain.Generate(ctx, req)
	if err != nil {
		logger.Error().Err(err).Msg("message processing failed in brain reasoning engine")
		return "", err
	}

	logger.Info().
		Int("prompt_tokens", resp.Usage.PromptTokens).
		Int("completion_tokens", resp.Usage.CompletionTokens).
		Int("total_tokens", resp.Usage.TotalTokens).
		Str("finish_reason", resp.FinishReason).
		Msg("message processing completed successfully")

	return resp.Content, nil
}
