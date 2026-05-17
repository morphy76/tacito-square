// Package outbound defines shared outbound port interfaces used across components.
package outbound

import (
	"context"
	"io"
)

// BlobStore is the outbound port for S3-compatible object storage.
type BlobStore interface {
	// Put stores data and returns the object URL.
	Put(ctx context.Context, key string, data io.Reader, contentType string) (string, error)
	// Get retrieves data by key.
	Get(ctx context.Context, key string) (io.ReadCloser, error)
	// Delete removes an object by key.
	Delete(ctx context.Context, key string) error
	// Exists checks if an object exists.
	Exists(ctx context.Context, key string) (bool, error)
}
