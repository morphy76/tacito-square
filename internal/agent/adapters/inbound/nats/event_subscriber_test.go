package nats_test

import (
	"context"
	"encoding/json"
	"fmt"
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
	mu             sync.Mutex
	RouteEventFunc func(ctx context.Context, event events.DomainEvent) error
	calls          []events.DomainEvent
}

func (m *MockSchemaRouter) RouteEvent(ctx context.Context, event events.DomainEvent) error {
	m.mu.Lock()
	m.calls = append(m.calls, event)
	fn := m.RouteEventFunc
	m.mu.Unlock()

	if fn != nil {
		return fn(ctx, event)
	}
	return nil
}

func (m *MockSchemaRouter) GetCalls() []events.DomainEvent {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]events.DomainEvent, len(m.calls))
	copy(out, m.calls)
	return out
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
		Host:      "127.0.0.1",
		Port:      -1,
		JetStream: true,
		StoreDir:  t.TempDir(),
	}
	ns, err := server.NewServer(opts)
	require.NoError(t, err)

	go ns.Start()
	if !ns.ReadyForConnections(2 * time.Second) {
		t.Fatal("NATS server not ready for connections")
	}

	nc, err := nats.Connect(ns.ClientURL())
	require.NoError(t, err)

	// Pre-create TACITO_EVENTS and TACITO_DLQ streams for tests
	js, err := nc.JetStream()
	require.NoError(t, err)
	_, err = js.AddStream(&nats.StreamConfig{
		Name:     "TACITO_EVENTS",
		Subjects: []string{
			"ts.community.*.agent.*",
			"ts.community.*.agent.*.thread.*.response",
			"ts.community.*.agent.*.thread.*.history",
		},
	})
	require.NoError(t, err)

	_, err = js.AddStream(&nats.StreamConfig{
		Name:     "TACITO_DLQ",
		Subjects: []string{"ts.dlq.community.>"},
	})
	require.NoError(t, err)

	return ns, nc
}

func TestEventSubscriber_Routing(t *testing.T) {
	ns, nc := startTestNatsServer(t)
	defer ns.Shutdown()
	defer nc.Close()

	logger := zerolog.New(nil)
	mockRouter := &MockSchemaRouter{}
	sub := agentnats.NewEventSubscriber(nc, "agent-alpha", "comm-1", "spoke", mockRouter, nil, logger)

	err := sub.Start(context.Background())
	require.NoError(t, err)
	defer func() { _ = sub.Stop() }()

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

	calls := mockRouter.GetCalls()
	require.Len(t, calls, 1)
	assert.Equal(t, events.SchemaConversationalStartThread, calls[0].SchemaRef)
	assert.Equal(t, "evt-123", calls[0].EventID)
	assert.Equal(t, "tenant-1", calls[0].TenantID)
}

func TestEventSubscriber_BroadcastRouting(t *testing.T) {
	ns, nc := startTestNatsServer(t)
	defer ns.Shutdown()
	defer nc.Close()

	logger := zerolog.New(nil)
	mockRouter := &MockSchemaRouter{}
	sub := agentnats.NewEventSubscriber(nc, "agent-alpha", "comm-1", "spoke", mockRouter, nil, logger)

	err := sub.Start(context.Background())
	require.NoError(t, err)
	defer func() { _ = sub.Stop() }()

	payload := events.StartThreadPayload{
		ThreadID:    "thread-abc-broadcast",
		CommunityID: "comm-1",
	}
	payloadBytes, err := json.Marshal(payload)
	require.NoError(t, err)

	evt := events.DomainEvent{
		EventID:    "evt-456",
		SchemaRef:  events.SchemaConversationalStartThread,
		Source:     "keeper",
		TenantID:   "tenant-1",
		OccurredAt: time.Now().UTC().Format(time.RFC3339Nano),
		Payload:    payloadBytes,
	}

	evtBytes, err := json.Marshal(evt)
	require.NoError(t, err)

	// Publish to the broadcast .all topic
	msg := nats.NewMsg("ts.community.comm-1.agent.all")
	msg.Data = evtBytes
	msg.Header.Set("X-Tacito-Schema", events.SchemaConversationalStartThread)
	msg.Header.Set("X-Tacito-Tenant", "tenant-1")

	err = nc.PublishMsg(msg)
	require.NoError(t, err)

	// Wait for processing
	time.Sleep(100 * time.Millisecond)

	calls := mockRouter.GetCalls()
	require.Len(t, calls, 1)
	assert.Equal(t, events.SchemaConversationalStartThread, calls[0].SchemaRef)
	assert.Equal(t, "evt-456", calls[0].EventID)
}

func TestEventSubscriber_InvalidSchema(t *testing.T) {
	ns, nc := startTestNatsServer(t)
	defer ns.Shutdown()
	defer nc.Close()

	logger := zerolog.New(nil)
	mockRouter := &MockSchemaRouter{}
	sub := agentnats.NewEventSubscriber(nc, "agent-alpha", "comm-1", "spoke", mockRouter, nil, logger)

	err := sub.Start(context.Background())
	require.NoError(t, err)
	defer func() { _ = sub.Stop() }()

	msg := nats.NewMsg("ts.community.comm-1.agent.agent-alpha")
	msg.Data = []byte(`{"event_id":"evt-1"}`)

	err = nc.PublishMsg(msg)
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	assert.Empty(t, mockRouter.GetCalls())
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

func TestEventSubscriber_HubQueueGroupLoadBalancing(t *testing.T) {
	ns, nc := startTestNatsServer(t)
	defer ns.Shutdown()
	defer nc.Close()

	logger := zerolog.New(nil)

	// Create 2 mock routers and 2 subscriber instances under the hub role
	mockRouter1 := &MockSchemaRouter{}
	sub1 := agentnats.NewEventSubscriber(nc, "hub-instance-1", "comm-1", "hub", mockRouter1, nil, logger)
	err := sub1.Start(context.Background())
	require.NoError(t, err)
	defer func() { _ = sub1.Stop() }()

	mockRouter2 := &MockSchemaRouter{}
	sub2 := agentnats.NewEventSubscriber(nc, "hub-instance-2", "comm-1", "hub", mockRouter2, nil, logger)
	err = sub2.Start(context.Background())
	require.NoError(t, err)
	defer func() { _ = sub2.Stop() }()

	// Prepare a message payload
	payload := events.StartThreadPayload{
		ThreadID:    "thread-123",
		CommunityID: "comm-1",
	}
	payloadBytes, err := json.Marshal(payload)
	require.NoError(t, err)

	// Publish 10 messages to the hub subject
	for i := 0; i < 10; i++ {
		evt := events.DomainEvent{
			EventID:    fmt.Sprintf("evt-%d", i),
			SchemaRef:  events.SchemaConversationalStartThread,
			Source:     "keeper",
			TenantID:   "tenant-1",
			OccurredAt: time.Now().UTC().Format(time.RFC3339Nano),
			Payload:    payloadBytes,
		}
		evtBytes, err := json.Marshal(evt)
		require.NoError(t, err)

		msg := nats.NewMsg("ts.community.comm-1.agent.hub")
		msg.Data = evtBytes
		msg.Header.Set("X-Tacito-Schema", events.SchemaConversationalStartThread)
		msg.Header.Set("X-Tacito-Tenant", "tenant-1")

		err = nc.PublishMsg(msg)
		require.NoError(t, err)
	}

	// Wait for processing
	time.Sleep(200 * time.Millisecond)

	calls1 := mockRouter1.GetCalls()
	calls2 := mockRouter2.GetCalls()

	totalCalls := len(calls1) + len(calls2)
	assert.Equal(t, 10, totalCalls, "Total processed messages must be exactly 10")

	// Both subscribers should have processed some messages (load-balancing), and neither should have processed all 10 (which would mean broadcast)
	assert.Greater(t, len(calls1), 0, "Subscriber 1 should process at least one message")
	assert.Greater(t, len(calls2), 0, "Subscriber 2 should process at least one message")
}

func TestEventSubscriber_DLQRouting(t *testing.T) {
	ns, nc := startTestNatsServer(t)
	defer ns.Shutdown()
	defer nc.Close()

	logger := zerolog.New(nil)

	// Mock Router that always fails to trigger redeliveries
	mockRouter := &MockSchemaRouter{
		RouteEventFunc: func(ctx context.Context, event events.DomainEvent) error {
			return fmt.Errorf("intentional transient error processing event %s", event.EventID)
		},
	}

	sub := agentnats.NewEventSubscriber(nc, "agent-alpha", "comm-1", "spoke", mockRouter, nil, logger)

	err := sub.Start(context.Background())
	require.NoError(t, err)
	defer func() { _ = sub.Stop() }()

	payload := events.StartThreadPayload{
		ThreadID:    "thread-abc",
		CommunityID: "comm-1",
	}
	payloadBytes, err := json.Marshal(payload)
	require.NoError(t, err)

	evt := events.DomainEvent{
		EventID:    "evt-dlq-123",
		SchemaRef:  events.SchemaConversationalStartThread,
		Source:     "keeper",
		TenantID:   "tenant-1",
		OccurredAt: time.Now().UTC().Format(time.RFC3339Nano),
		Payload:    payloadBytes,
	}

	evtBytes, err := json.Marshal(evt)
	require.NoError(t, err)

	// Publish directly to the subject that is monitored by the stream
	js, err := nc.JetStream()
	require.NoError(t, err)

	msg := nats.NewMsg("ts.community.comm-1.agent.agent-alpha")
	msg.Data = evtBytes
	msg.Header.Set("X-Tacito-Schema", events.SchemaConversationalStartThread)
	msg.Header.Set("X-Tacito-Tenant", "tenant-1")
	msg.Header.Set("Nats-Msg-Id", evt.EventID)

	_, err = js.PublishMsg(msg)
	require.NoError(t, err)

	// Wait for the message to be processed, retried 5 times, and eventually routed to the DLQ stream
	// We check the DLQ stream for the presence of the event
	dlqChan := make(chan *nats.Msg, 1)
	dlqSub, err := nc.Subscribe("ts.dlq.community.>", func(m *nats.Msg) {
		dlqChan <- m
	})
	require.NoError(t, err)
	defer dlqSub.Unsubscribe()

	select {
	case dlqMsg := <-dlqChan:
		var received events.DomainEvent
		err := json.Unmarshal(dlqMsg.Data, &received)
		require.NoError(t, err)
		assert.Equal(t, "evt-dlq-123", received.EventID)
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for message to be routed to DLQ")
	}
}

