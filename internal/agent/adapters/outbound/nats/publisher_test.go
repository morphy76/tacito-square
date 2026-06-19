package nats_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	agentnats "github.com/morphy76/tacito-square/internal/agent/adapters/outbound/nats"
	"github.com/morphy76/tacito-square/pkg/events"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNATSEventPublisher_Publish_JetStream(t *testing.T) {
	// 1. Start NATS Server with JetStream enabled
	opts := &server.Options{
		Host:      "127.0.0.1",
		Port:      -1,
		JetStream: true,
		StoreDir:  t.TempDir(),
	}
	ns, err := server.NewServer(opts)
	require.NoError(t, err)
	go ns.Start()
	defer ns.Shutdown()

	if !ns.ReadyForConnections(2 * time.Second) {
		t.Fatal("NATS server not ready for connections")
	}

	nc, err := nats.Connect(ns.ClientURL())
	require.NoError(t, err)
	defer nc.Close()

	publisher := agentnats.NewNATSEventPublisher(nc)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	payload := map[string]string{"foo": "bar"}
	evt, err := events.NewDomainEvent("urn:tacito:schema:conversational:start-thread:v1", "agent/local", "tenant-123", payload)
	require.NoError(t, err)
	evtBytes, err := json.Marshal(evt)
	require.NoError(t, err)

	// A. Publish to a JetStream subject when the stream does NOT exist
	// Expect this to fail since JetStream publishing checks for stream matching
	jetstreamSubject := "ts.community.test-comm.agent.agent-alpha"
	err = publisher.Publish(ctx, jetstreamSubject, evtBytes)
	assert.Error(t, err, "expected error when publishing to JetStream subject without active stream")

	// B. Create the stream and try again
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

	err = publisher.Publish(ctx, jetstreamSubject, evtBytes)
	require.NoError(t, err, "expected successful publish to JetStream stream")

	// Verify the message was indeed stored in the stream and has the Nats-Msg-Id header
	streamMsg, err := js.GetLastMsg("TACITO_EVENTS", jetstreamSubject)
	require.NoError(t, err)
	assert.Equal(t, evt.EventID, streamMsg.Header.Get("Nats-Msg-Id"), "expected X-Tacito-Event-ID mapped to Nats-Msg-Id")

	// C. Publish a non-stream event (e.g. heartbeat) to verify it passes via standard pub/sub without stream checks
	heartbeatSubject := "ts.agent.heartbeat"
	err = publisher.Publish(ctx, heartbeatSubject, evtBytes)
	require.NoError(t, err, "expected successful publish to non-stream standard subject")
}
