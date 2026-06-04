// Package s3 implements the BlobStore port using S3-compatible storage (AWS S3 / MinIO).
package s3

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
)

// ErrNotFound is returned when a requested object does not exist.
var ErrNotFound = errors.New("blob not found")

// S3Client abstracts S3 operations for testability.
type S3Client interface {
	PutObject(ctx context.Context, bucket, key string, data io.Reader, contentType string) error
	GetObject(ctx context.Context, bucket, key string) (io.ReadCloser, error)
	DeleteObject(ctx context.Context, bucket, key string) error
	HeadObject(ctx context.Context, bucket, key string) (bool, error)
}

// BlobStoreAdapter implements the BlobStore port using S3.
type BlobStoreAdapter struct {
	client   S3Client
	bucket   string
	endpoint string
}

// NewBlobStoreAdapter creates a new S3 blob store adapter.
func NewBlobStoreAdapter(client S3Client, bucket, endpoint string) *BlobStoreAdapter {
	return &BlobStoreAdapter{
		client:   client,
		bucket:   bucket,
		endpoint: endpoint,
	}
}

// Put stores data in S3 and returns the object URL.
func (a *BlobStoreAdapter) Put(ctx context.Context, key string, data io.Reader, contentType string) (string, error) {
	if err := a.client.PutObject(ctx, a.bucket, key, data, contentType); err != nil {
		return "", fmt.Errorf("s3 put %s: %w", key, err)
	}
	url := fmt.Sprintf("%s/%s/%s", a.endpoint, a.bucket, key)
	return url, nil
}

// Get retrieves data from S3.
func (a *BlobStoreAdapter) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	reader, err := a.client.GetObject(ctx, a.bucket, key)
	if err != nil {
		return nil, fmt.Errorf("s3 get %s: %w", key, err)
	}
	return reader, nil
}

// Delete removes an object from S3.
func (a *BlobStoreAdapter) Delete(ctx context.Context, key string) error {
	if err := a.client.DeleteObject(ctx, a.bucket, key); err != nil {
		return fmt.Errorf("s3 delete %s: %w", key, err)
	}
	return nil
}

// Exists checks if an object exists in S3.
func (a *BlobStoreAdapter) Exists(ctx context.Context, key string) (bool, error) {
	exists, err := a.client.HeadObject(ctx, a.bucket, key)
	if err != nil {
		return false, fmt.Errorf("s3 head %s: %w", key, err)
	}
	return exists, nil
}

// Ping verifies the connectivity to the S3 bucket.
func (a *BlobStoreAdapter) Ping(ctx context.Context) error {
	_, err := a.client.HeadObject(ctx, a.bucket, "health-check-dummy-ping-key")
	if err != nil && !errors.Is(err, ErrNotFound) && !strings.Contains(err.Error(), "404") {
		return err
	}
	return nil
}
