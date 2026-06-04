// Package s3 implements the BlobStore port using S3-compatible storage (AWS S3 / MinIO).
package s3

import (
	"context"
	"fmt"
	"io"
	"net/http"
)

// HTTPClient abstracts net/http.Client for instrumentation and mockability.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// S3HTTPClient implements S3Client interface using HTTP REST calls.
type S3HTTPClient struct {
	endpoint string
	client   HTTPClient
}

// NewS3HTTPClient constructs a new S3HTTPClient.
func NewS3HTTPClient(endpoint string, client HTTPClient) *S3HTTPClient {
	if client == nil {
		client = http.DefaultClient
	}
	return &S3HTTPClient{
		endpoint: endpoint,
		client:   client,
	}
}

// PutObject uploads an object using PUT request.
func (c *S3HTTPClient) PutObject(ctx context.Context, bucket, key string, data io.Reader, contentType string) error {
	url := fmt.Sprintf("%s/%s/%s", c.endpoint, bucket, key)
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, data)
	if err != nil {
		return err
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("s3 put failed with status %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

// GetObject downloads an object using GET request.
func (c *S3HTTPClient) GetObject(ctx context.Context, bucket, key string) (io.ReadCloser, error) {
	url := fmt.Sprintf("%s/%s/%s", c.endpoint, bucket, key)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusNotFound {
		resp.Body.Close()
		return nil, ErrNotFound
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("s3 get failed with status %d: %s", resp.StatusCode, string(body))
	}

	return resp.Body, nil
}

// DeleteObject removes an object using DELETE request.
func (c *S3HTTPClient) DeleteObject(ctx context.Context, bucket, key string) error {
	url := fmt.Sprintf("%s/%s/%s", c.endpoint, bucket, key)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 && resp.StatusCode != http.StatusNotFound {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("s3 delete failed with status %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

// HeadObject checks if an object exists using HEAD request.
func (c *S3HTTPClient) HeadObject(ctx context.Context, bucket, key string) (bool, error) {
	url := fmt.Sprintf("%s/%s/%s", c.endpoint, bucket, key)
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, url, nil)
	if err != nil {
		return false, err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return true, nil
	}
	if resp.StatusCode == http.StatusNotFound {
		return false, nil
	}
	return false, fmt.Errorf("s3 head failed with status %d", resp.StatusCode)
}
