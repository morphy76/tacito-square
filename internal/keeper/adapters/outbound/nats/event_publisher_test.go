package nats_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	natsadapter "github.com/morphy76/tacito-square/internal/keeper/adapters/outbound/nats"
	"github.com/morphy76/tacito-square/pkg/events"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

func TestNATSEventPublisher_Publish(t *testing.T) {
	ns, nc := startTestNatsServer(t)
	defer ns.Shutdown()
	defer nc.Close()

	publisher := natsadapter.NewNATSEventPublisher(nc)

	subject := "ts.community.test-comm.agent.agent-alpha"
	msgChan := make(chan *nats.Msg, 1)

	sub, err := nc.Subscribe(subject, func(msg *nats.Msg) {
		msgChan <- msg
	})
	require.NoError(t, err)
	defer sub.Unsubscribe()

	payload := map[string]string{"foo": "bar"}
	evt, err := events.NewDomainEvent("urn:tacito:schema:conversational:start-thread:v1", "keeper/local", "tenant-123", payload)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err = publisher.Publish(ctx, subject, evt)
	require.NoError(t, err)

	select {
	case msg := <-msgChan:
		// Verify body
		var received events.DomainEvent
		err := json.Unmarshal(msg.Data, &received)
		require.NoError(t, err)
		assert.Equal(t, evt.EventID, received.EventID)
		assert.Equal(t, evt.SchemaRef, received.SchemaRef)

		// Verify NATS headers
		assert.Equal(t, evt.SchemaRef, msg.Header.Get("X-Tacito-Schema"))
		assert.Equal(t, evt.Source, msg.Header.Get("X-Tacito-Source"))
		assert.Equal(t, evt.TenantID, msg.Header.Get("X-Tacito-Tenant"))
		assert.Equal(t, evt.EventID, msg.Header.Get("X-Tacito-Event-ID"))
		assert.Equal(t, evt.OccurredAt, msg.Header.Get("X-Tacito-Occurred"))
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for NATS message")
	}
}

func TestNATSEventSubscriber_Subscribe(t *testing.T) {
	ns, nc := startTestNatsServer(t)
	defer ns.Shutdown()
	defer nc.Close()

	subscriber := natsadapter.NewNATSEventSubscriber(nc)

	tenantA := "tenant-A"
	tenantB := "tenant-B"

	eventsA := make(chan *events.DomainEvent, 10)
	eventsB := make(chan *events.DomainEvent, 10)

	ctx := context.Background()

	// Subscribe for tenantA
	subA, err := subscriber.Subscribe(ctx, "ts.community.>", tenantA, func(evt *events.DomainEvent) {
		eventsA <- evt
	})
	require.NoError(t, err)
	defer subA.Stop()

	// Subscribe for tenantB
	subB, err := subscriber.Subscribe(ctx, "ts.community.>", tenantB, func(evt *events.DomainEvent) {
		eventsB <- evt
	})
	require.NoError(t, err)
	defer subB.Stop()

	// Helper to publish with headers
	pubHelper := func(subject string, evt events.DomainEvent) {
		data, _ := json.Marshal(evt)
		msg := nats.NewMsg(subject)
		msg.Data = data
		msg.Header.Set("X-Tacito-Tenant", evt.TenantID)
		_ = nc.PublishMsg(msg)
	}

	evtA, _ := events.NewDomainEvent("urn:tacito:schema:conversational:start-thread:v1", "keeper/local", tenantA, map[string]string{})
	evtB, _ := events.NewDomainEvent("urn:tacito:schema:conversational:start-thread:v1", "keeper/local", tenantB, map[string]string{})

	pubHelper("ts.community.comm-1.agent.agent-1", evtA)
	pubHelper("ts.community.comm-1.agent.agent-1", evtB)

	// Verify tenantA subscriber only got evtA
	select {
	case rec := <-eventsA:
		assert.Equal(t, tenantA, rec.TenantID)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for tenantA event")
	}

	// Verify tenantB subscriber only got evtB
	select {
	case rec := <-eventsB:
		assert.Equal(t, tenantB, rec.TenantID)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for tenantB event")
	}
}
