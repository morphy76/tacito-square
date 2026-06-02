package nats_test

import (
	"bytes"
	"context"
	"encoding/json"
	"regexp"
	"sync"
	"testing"
	"time"

	"io"
	"strings"

	agentnats "github.com/morphy76/tacito-square/internal/agent/adapters/inbound/nats"
	"github.com/morphy76/tacito-square/internal/keeper/domain/model"
	"github.com/morphy76/tacito-square/internal/shared/ports/outbound"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type MockMessageProcessor struct {
	ProcessFunc func(ctx context.Context, tenantID, agentID, threadID string, payload string) (string, error)
}

func (m *MockMessageProcessor) ProcessIncomingMessage(ctx context.Context, tenantID, agentID, threadID string, payload string) (string, error) {
	if m.ProcessFunc != nil {
		return m.ProcessFunc(ctx, tenantID, agentID, threadID, payload)
	}
	return "mocked answer", nil
}

type mockBlobStore struct {
	mu     sync.Mutex
	puts   map[string][]byte
	putErr error
}

var _ outbound.BlobStore = (*mockBlobStore)(nil)

func newMockBlobStore() *mockBlobStore {
	return &mockBlobStore{
		puts: make(map[string][]byte),
	}
}

func (m *mockBlobStore) Put(ctx context.Context, key string, data io.Reader, contentType string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.putErr != nil {
		return "", m.putErr
	}
	b, err := io.ReadAll(data)
	if err != nil {
		return "", err
	}
	m.puts[key] = b
	return "http://mock-s3/test-bucket/" + key, nil
}

func (m *mockBlobStore) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	return nil, nil
}

func (m *mockBlobStore) Delete(ctx context.Context, key string) error {
	return nil
}

func (m *mockBlobStore) Exists(ctx context.Context, key string) (bool, error) {
	return false, nil
}

func startTestNatsServer(t *testing.T) (*server.Server, *nats.Conn) {
	opts := &server.Options{
		Host: "127.0.0.1",
		Port: -1,
	}
	ns, err := server.NewServer(opts)
	require.NoError(t, err)

	go ns.Start()
	if !ns.ReadyForConnections(2 * time.Second) {
		t.Fatal("NATS server not ready for connections")
	}

	nc, err := nats.Connect(ns.ClientURL())
	require.NoError(t, err)

	return ns, nc
}

func TestEchoSubscriber_Replies(t *testing.T) {
	ns, nc := startTestNatsServer(t)
	defer ns.Shutdown()
	defer nc.Close()

	logger := zerolog.New(nil)
	mp := &MockMessageProcessor{
		ProcessFunc: func(ctx context.Context, tenantID, agentID, threadID, payload string) (string, error) {
			return "reasoned hello", nil
		},
	}
	sub := agentnats.NewEchoSubscriber(nc, "agent-alpha", "comm-1", "tenant-1", mp, nil, logger)

	err := sub.Start(context.Background())
	require.NoError(t, err)
	defer sub.Stop()

	req := model.EchoRequest{
		Message:     "hello",
		CommunityID: "comm-1",
		TenantID:    "tenant-1",
	}
	payload, err := json.Marshal(req)
	require.NoError(t, err)

	subject := "ts.community.comm-1.agent.agent-alpha"
	msg, err := nc.Request(subject, payload, 2*time.Second)
	require.NoError(t, err)

	var reply model.EchoReply
	err = json.Unmarshal(msg.Data, &reply)
	require.NoError(t, err)

	assert.Equal(t, "agent-alpha", reply.AgentName)

	pattern := `^\[agent:agent-alpha at \d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z\] reasoned hello$`
	matched, err := regexp.MatchString(pattern, reply.Decorated)
	require.NoError(t, err)
	assert.True(t, matched, "decorated message must match standard envelope format: %s", reply.Decorated)

	_, err = time.Parse(time.RFC3339, reply.Timestamp)
	assert.NoError(t, err, "timestamp must be valid RFC3339 format")
}

type safeBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (s *safeBuffer) Write(p []byte) (n int, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.Write(p)
}

func (s *safeBuffer) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.String()
}

func TestEchoSubscriber_LogsSanitizedMessage(t *testing.T) {
	ns, nc := startTestNatsServer(t)
	defer ns.Shutdown()
	defer nc.Close()

	var buf safeBuffer
	logger := zerolog.New(&buf)
	mp := &MockMessageProcessor{}
	sub := agentnats.NewEchoSubscriber(nc, "agent-alpha", "comm-1", "tenant-1", mp, nil, logger)

	err := sub.Start(context.Background())
	require.NoError(t, err)
	defer sub.Stop()

	req := model.EchoRequest{
		Message:     "hello\x00world\x01",
		CommunityID: "comm-1",
		TenantID:    "tenant-1",
	}
	payload, err := json.Marshal(req)
	require.NoError(t, err)

	subject := "ts.community.comm-1.agent.agent-alpha"
	_, err = nc.Request(subject, payload, 2*time.Second)
	require.NoError(t, err)

	logOutput := buf.String()
	assert.Contains(t, logOutput, `"message":"helloworld"`)
	assert.Contains(t, logOutput, `"agent_name":"agent-alpha"`)
	assert.Contains(t, logOutput, `"community_id":"comm-1"`)
	assert.Contains(t, logOutput, `"tenant_id":"tenant-1"`)
}

func TestEchoSubscriber_Stop(t *testing.T) {
	ns, nc := startTestNatsServer(t)
	defer ns.Shutdown()
	defer nc.Close()

	logger := zerolog.New(nil)
	mp := &MockMessageProcessor{}
	sub := agentnats.NewEchoSubscriber(nc, "agent-alpha", "comm-1", "tenant-1", mp, nil, logger)

	err := sub.Start(context.Background())
	require.NoError(t, err)

	err = sub.Stop()
	require.NoError(t, err)

	req := model.EchoRequest{
		Message:     "hello",
		CommunityID: "comm-1",
		TenantID:    "tenant-1",
	}
	payload, _ := json.Marshal(req)

	subject := "ts.community.comm-1.agent.agent-alpha"
	_, err = nc.Request(subject, payload, 100*time.Millisecond)
	assert.Error(t, err, "request should time out after subscription stopped")
}

func TestEchoSubscriber_MalformedPayload(t *testing.T) {
	ns, nc := startTestNatsServer(t)
	defer ns.Shutdown()
	defer nc.Close()

	var buf safeBuffer
	logger := zerolog.New(&buf)
	mp := &MockMessageProcessor{}
	sub := agentnats.NewEchoSubscriber(nc, "agent-alpha", "comm-1", "tenant-1", mp, nil, logger)

	err := sub.Start(context.Background())
	require.NoError(t, err)
	defer sub.Stop()

	subject := "ts.community.comm-1.agent.agent-alpha"
	_, err = nc.Request(subject, []byte("{brokenJSON"), 100*time.Millisecond)
	assert.Error(t, err, "should time out as malformed payload is ignored and gets no reply")

	assert.Contains(t, buf.String(), `"level":"warn"`)
	assert.Contains(t, buf.String(), "malformed payload")
}

func TestNormalizeBucketName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Acme_Corp & Co.", "acme-corp-co"},
		{"UPPER_case-123", "upper-case-123"},
		{"--leading-and-trailing--", "leading-and-trailing"},
		{"a--b", "a-b"},
		{strings.Repeat("a", 100), strings.Repeat("a", 63)},
	}
	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			result := agentnats.NormalizeBucketName(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestEchoSubscriber_OffloadsLargePayload(t *testing.T) {
	ns, nc := startTestNatsServer(t)
	defer ns.Shutdown()
	defer nc.Close()

	logger := zerolog.New(nil)
	
	// Create a message that exceeds 256KB
	largeMessage := strings.Repeat("A", 300*1024)
	
	var receivedPayload string
	mp := &MockMessageProcessor{
		ProcessFunc: func(ctx context.Context, tenantID, agentID, threadID, payload string) (string, error) {
			receivedPayload = payload
			return "reasoned hello", nil
		},
	}
	
	bs := newMockBlobStore()
	sub := agentnats.NewEchoSubscriber(nc, "agent-alpha", "comm-1", "tenant-1", mp, bs, logger)

	err := sub.Start(context.Background())
	require.NoError(t, err)
	defer sub.Stop()

	req := model.EchoRequest{
		Message:     largeMessage,
		CommunityID: "comm-1",
		TenantID:    "tenant-1",
	}
	payload, err := json.Marshal(req)
	require.NoError(t, err)

	subject := "ts.community.comm-1.agent.agent-alpha"
	msg, err := nc.Request(subject, payload, 2*time.Second)
	require.NoError(t, err)

	// Verify reply
	var reply model.EchoReply
	err = json.Unmarshal(msg.Data, &reply)
	require.NoError(t, err)
	assert.Equal(t, "agent-alpha", reply.AgentName)

	// Verify mock BlobStore recorded the put call
	bs.mu.Lock()
	defer bs.mu.Unlock()
	
	assert.Len(t, bs.puts, 1)
	
	// Find S3 key and assert it contains community ID, ingress functional root, agent name, and thread ID
	var s3Key string
	for k := range bs.puts {
		s3Key = k
	}
	
	assert.Contains(t, s3Key, "comm-1/ingress/agent-alpha/thread-comm-1/")
	assert.Equal(t, largeMessage, string(bs.puts[s3Key]))

	// Verify receivedPayload is substituted with the s3_reference JSON envelope
	var s3Ref struct {
		Type        string `json:"_type"`
		Bucket      string `json:"bucket"`
		Key         string `json:"key"`
		SizeBytes   int64  `json:"size_bytes"`
		ContentType string `json:"content_type"`
	}
	
	err = json.Unmarshal([]byte(receivedPayload), &s3Ref)
	require.NoError(t, err, "Processor must receive valid JSON s3_reference: %s", receivedPayload)
	assert.Equal(t, "s3_reference", s3Ref.Type)
	assert.Equal(t, "tenant-1", s3Ref.Bucket)
	assert.Equal(t, s3Key, s3Ref.Key)
	assert.Equal(t, int64(len(largeMessage)), s3Ref.SizeBytes)
	assert.Equal(t, "text/plain", s3Ref.ContentType)
}
