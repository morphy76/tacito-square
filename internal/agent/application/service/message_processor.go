package service

import (
	"context"

	"github.com/morphy76/tacito-square/internal/agent/application/ports/outbound"
	"github.com/morphy76/tacito-square/internal/agent/domain/model"
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
	req := model.BrainRequest{
		Prompt: payload,
	}
	resp, err := s.brain.Generate(ctx, req)
	if err != nil {
		return "", err
	}
	return resp.Content, nil
}
