package outbound

import (
	"context"
)

// Embedder defines the outbound port interface for generating dense vector embeddings.
type Embedder interface {
	// CreateEmbedding generates a high-dimensional dense vector for the given text.
	CreateEmbedding(ctx context.Context, text string) ([]float32, error)

	// CreateEmbeddingsBatch generates dense vectors for a slice of texts in parallel.
	CreateEmbeddingsBatch(ctx context.Context, texts []string) ([][]float32, error)
}
