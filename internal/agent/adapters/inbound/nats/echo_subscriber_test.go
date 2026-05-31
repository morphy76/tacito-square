package nats_test

import (
	"bytes"
	"context"
	"encoding/json"
	"regexp"
	"sync"
	"testing"
	"time"

	agentnats "github.com/morphy76/tacito-square/internal/agent/adapters/inbound/nats"
	"github.com/morphy76/tacito-square/internal/keeper/domain/model"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type MockMessageProcessor struct {
	ProcessFunc func(ctx context.Context, payload string) (string, error)
}

func (m *MockMessageProcessor) ProcessIncomingMessage(ctx context.Context, payload string) (string, error) {
	if m.ProcessFunc != nil {
		return m.ProcessFunc(ctx, payload)
	}
	return "mocked answer", nil
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
		ProcessFunc: func(ctx context.Context, payload string) (string, error) {
			return "reasoned hello", nil
		},
	}
	sub := agentnats.NewEchoSubscriber(nc, "agent-alpha", "comm-1", "tenant-1", mp, logger)

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
	sub := agentnats.NewEchoSubscriber(nc, "agent-alpha", "comm-1", "tenant-1", mp, logger)

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
	sub := agentnats.NewEchoSubscriber(nc, "agent-alpha", "comm-1", "tenant-1", mp, logger)

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
	sub := agentnats.NewEchoSubscriber(nc, "agent-alpha", "comm-1", "tenant-1", mp, logger)

	err := sub.Start(context.Background())
	require.NoError(t, err)
	defer sub.Stop()

	subject := "ts.community.comm-1.agent.agent-alpha"
	_, err = nc.Request(subject, []byte("{brokenJSON"), 100*time.Millisecond)
	assert.Error(t, err, "should time out as malformed payload is ignored and gets no reply")

	assert.Contains(t, buf.String(), `"level":"warn"`)
	assert.Contains(t, buf.String(), "malformed payload")
}
