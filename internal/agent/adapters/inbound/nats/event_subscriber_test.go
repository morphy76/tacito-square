package nats_test

import (
	"context"
	"encoding/json"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	agentnats "github.com/morphy76/tacito-square/internal/agent/adapters/inbound/nats"
	"github.com/morphy76/tacito-square/internal/shared/ports/outbound"
	"github.com/morphy76/tacito-square/pkg/events"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type MockSchemaRouter struct {
	RouteEventFunc func(ctx context.Context, event events.DomainEvent) error
	Calls          []events.DomainEvent
}

func (m *MockSchemaRouter) RouteEvent(ctx context.Context, event events.DomainEvent) error {
	m.Calls = append(m.Calls, event)
	if m.RouteEventFunc != nil {
		return m.RouteEventFunc(ctx, event)
	}
	return nil
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

func TestEventSubscriber_Routing(t *testing.T) {
	ns, nc := startTestNatsServer(t)
	defer ns.Shutdown()
	defer nc.Close()

	logger := zerolog.New(nil)
	mockRouter := &MockSchemaRouter{}
	sub := agentnats.NewEventSubscriber(nc, "agent-alpha", "comm-1", mockRouter, nil, logger)

	err := sub.Start(context.Background())
	require.NoError(t, err)
	defer sub.Stop()

	payload := events.StartThreadPayload{
		ThreadID:    "thread-abc",
		CommunityID: "comm-1",
	}
	payloadBytes, err := json.Marshal(payload)
	require.NoError(t, err)

	evt := events.DomainEvent{
		EventID:    "evt-123",
		SchemaRef:  events.SchemaConversationalStartThread,
		Source:     "keeper",
		TenantID:   "tenant-1",
		OccurredAt: time.Now().UTC().Format(time.RFC3339Nano),
		Payload:    payloadBytes,
	}

	evtBytes, err := json.Marshal(evt)
	require.NoError(t, err)

	msg := nats.NewMsg("ts.community.comm-1.agent.agent-alpha")
	msg.Data = evtBytes
	msg.Header.Set("X-Tacito-Schema", events.SchemaConversationalStartThread)
	msg.Header.Set("X-Tacito-Tenant", "tenant-1")

	err = nc.PublishMsg(msg)
	require.NoError(t, err)

	// Wait for processing
	time.Sleep(100 * time.Millisecond)

	require.Len(t, mockRouter.Calls, 1)
	assert.Equal(t, events.SchemaConversationalStartThread, mockRouter.Calls[0].SchemaRef)
	assert.Equal(t, "evt-123", mockRouter.Calls[0].EventID)
	assert.Equal(t, "tenant-1", mockRouter.Calls[0].TenantID)
}

func TestEventSubscriber_InvalidSchema(t *testing.T) {
	ns, nc := startTestNatsServer(t)
	defer ns.Shutdown()
	defer nc.Close()

	logger := zerolog.New(nil)
	mockRouter := &MockSchemaRouter{}
	sub := agentnats.NewEventSubscriber(nc, "agent-alpha", "comm-1", mockRouter, nil, logger)

	err := sub.Start(context.Background())
	require.NoError(t, err)
	defer sub.Stop()

	msg := nats.NewMsg("ts.community.comm-1.agent.agent-alpha")
	msg.Data = []byte(`{"event_id":"evt-1"}`)

	err = nc.PublishMsg(msg)
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	assert.Empty(t, mockRouter.Calls)
}

func TestOffloadUtility(t *testing.T) {
	t.Run("NormalizeBucketName", func(t *testing.T) {
		assert.Equal(t, "acme-corp-co", agentnats.NormalizeBucketName("Acme_Corp & Co."))
		assert.Equal(t, "leading-and-trailing", agentnats.NormalizeBucketName("--leading-and-trailing--"))
	})

	t.Run("OffloadPayload", func(t *testing.T) {
		bs := newMockBlobStore()
		ctx := context.Background()

		largePayload := []byte(strings.Repeat("A", 300*1024))
		refStr, err := agentnats.OffloadPayload(ctx, bs, "comm-1", "agent-alpha", "thread-123", "tenant-1", largePayload)
		require.NoError(t, err)

		var ref agentnats.S3Reference
		err = json.Unmarshal([]byte(refStr), &ref)
		require.NoError(t, err)

		assert.Equal(t, "s3_reference", ref.Type)
		assert.Equal(t, "tenant-1", ref.Bucket)
		assert.Equal(t, int64(len(largePayload)), ref.SizeBytes)

		bs.mu.Lock()
		defer bs.mu.Unlock()
		assert.Len(t, bs.puts, 1)
	})
}
