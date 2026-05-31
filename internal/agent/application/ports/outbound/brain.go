package outbound

import (
	"context"

	"github.com/morphy76/tacito-square/internal/agent/domain/model"
)

// Brain defines the interface for outbound LLM adapters.
type Brain interface {
	Generate(ctx context.Context, request model.BrainRequest) (*model.BrainResponse, error)
	GenerateStream(ctx context.Context, request model.BrainRequest) (<-chan model.BrainStreamChunk, <-chan error, error)
}
