package s3

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockS3Client is a test double for S3 operations.
type mockS3Client struct {
	objects map[string][]byte
}

func newMockS3Client() *mockS3Client {
	return &mockS3Client{
		objects: make(map[string][]byte),
	}
}

func (m *mockS3Client) PutObject(_ context.Context, bucket, key string, data io.Reader, contentType string) error {
	b, _ := io.ReadAll(data)
	m.objects[bucket+"/"+key] = b
	return nil
}

func (m *mockS3Client) GetObject(_ context.Context, bucket, key string) (io.ReadCloser, error) {
	data, ok := m.objects[bucket+"/"+key]
	if !ok {
		return nil, ErrNotFound
	}
	return io.NopCloser(bytes.NewReader(data)), nil
}

func (m *mockS3Client) DeleteObject(_ context.Context, bucket, key string) error {
	delete(m.objects, bucket+"/"+key)
	return nil
}

func (m *mockS3Client) HeadObject(_ context.Context, bucket, key string) (bool, error) {
	_, ok := m.objects[bucket+"/"+key]
	return ok, nil
}

func TestBlobStoreAdapter_Put_Success(t *testing.T) {
	mock := newMockS3Client()
	adapter := NewBlobStoreAdapter(mock, "test-bucket", "http://localhost:9000")

	data := bytes.NewReader([]byte("hello world"))
	url, err := adapter.Put(context.Background(), "community/thread/agent/file.txt", data, "text/plain")

	require.NoError(t, err)
	assert.Contains(t, url, "community/thread/agent/file.txt")
	assert.Contains(t, url, "http://localhost:9000")
}

func TestBlobStoreAdapter_Get_Success(t *testing.T) {
	mock := newMockS3Client()
	adapter := NewBlobStoreAdapter(mock, "test-bucket", "http://localhost:9000")

	// Store first
	data := bytes.NewReader([]byte("stored content"))
	_, err := adapter.Put(context.Background(), "mykey", data, "text/plain")
	require.NoError(t, err)

	// Retrieve
	reader, err := adapter.Get(context.Background(), "mykey")
	require.NoError(t, err)
	defer reader.Close()

	content, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Equal(t, "stored content", string(content))
}

func TestBlobStoreAdapter_Get_NotFound(t *testing.T) {
	mock := newMockS3Client()
	adapter := NewBlobStoreAdapter(mock, "test-bucket", "http://localhost:9000")

	_, err := adapter.Get(context.Background(), "nonexistent")

	require.Error(t, err)
}

func TestBlobStoreAdapter_Delete_Success(t *testing.T) {
	mock := newMockS3Client()
	adapter := NewBlobStoreAdapter(mock, "test-bucket", "http://localhost:9000")

	data := bytes.NewReader([]byte("to delete"))
	_, err := adapter.Put(context.Background(), "delkey", data, "text/plain")
	require.NoError(t, err)

	err = adapter.Delete(context.Background(), "delkey")
	require.NoError(t, err)

	exists, _ := adapter.Exists(context.Background(), "delkey")
	assert.False(t, exists)
}

func TestBlobStoreAdapter_Exists_True(t *testing.T) {
	mock := newMockS3Client()
	adapter := NewBlobStoreAdapter(mock, "test-bucket", "http://localhost:9000")

	data := bytes.NewReader([]byte("exists"))
	_, err := adapter.Put(context.Background(), "existskey", data, "text/plain")
	require.NoError(t, err)

	exists, err := adapter.Exists(context.Background(), "existskey")
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestBlobStoreAdapter_Exists_False(t *testing.T) {
	mock := newMockS3Client()
	adapter := NewBlobStoreAdapter(mock, "test-bucket", "http://localhost:9000")

	exists, err := adapter.Exists(context.Background(), "nope")
	require.NoError(t, err)
	assert.False(t, exists)
}
