package nats_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	natsadapter "github.com/morphy76/tacito-square/internal/keeper/adapters/outbound/nats"
	"github.com/morphy76/tacito-square/internal/keeper/domain/model"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog"
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

func TestRequestEcho_Success(t *testing.T) {
	ns, nc := startTestNatsServer(t)
	defer ns.Shutdown()
	defer nc.Close()

	logger := zerolog.New(nil)
	broadcaster := natsadapter.NewNATSCommunityBroadcaster(nc, logger)

	subject := "ts.community.test-comm.agent.agent-alpha"
	sub, err := nc.Subscribe(subject, func(msg *nats.Msg) {
		var req model.EchoRequest
		err := json.Unmarshal(msg.Data, &req)
		if err != nil {
			return
		}

		reply := model.EchoReply{
			AgentName: "agent-alpha",
			Decorated: "[agent:agent-alpha at 2026-05-30T00:00:00Z] hello",
			Timestamp: "2026-05-30T00:00:00Z",
		}
		replyBytes, _ := json.Marshal(reply)
		_ = msg.Respond(replyBytes)
	})
	require.NoError(t, err)
	defer sub.Unsubscribe()

	req := model.EchoRequest{
		Message:     "hello",
		CommunityID: "test-comm",
		TenantID:    "tenant-1",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	res, err := broadcaster.RequestEcho(ctx, "test-comm", "agent-alpha", req)
	require.NoError(t, err)
	assert.Equal(t, "agent-alpha", res.AgentName)
	assert.Equal(t, "[agent:agent-alpha at 2026-05-30T00:00:00Z] hello", res.Decorated)
	assert.Equal(t, "2026-05-30T00:00:00Z", res.Timestamp)
}

func TestRequestEcho_Timeout(t *testing.T) {
	ns, nc := startTestNatsServer(t)
	defer ns.Shutdown()
	defer nc.Close()

	logger := zerolog.New(nil)
	broadcaster := natsadapter.NewNATSCommunityBroadcaster(nc, logger)

	req := model.EchoRequest{
		Message:     "hello",
		CommunityID: "test-comm",
		TenantID:    "tenant-1",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := broadcaster.RequestEcho(ctx, "test-comm", "agent-alpha", req)
	assert.Error(t, err)
}

func TestRequestEcho_MarshalPayload(t *testing.T) {
	ns, nc := startTestNatsServer(t)
	defer ns.Shutdown()
	defer nc.Close()

	logger := zerolog.New(nil)
	broadcaster := natsadapter.NewNATSCommunityBroadcaster(nc, logger)

	subject := "ts.community.test-comm.agent.agent-alpha"
	reqReceivedChan := make(chan model.EchoRequest, 1)

	sub, err := nc.Subscribe(subject, func(msg *nats.Msg) {
		var req model.EchoRequest
		_ = json.Unmarshal(msg.Data, &req)
		reqReceivedChan <- req

		reply := model.EchoReply{AgentName: "agent-alpha"}
		replyBytes, _ := json.Marshal(reply)
		_ = msg.Respond(replyBytes)
	})
	require.NoError(t, err)
	defer sub.Unsubscribe()

	req := model.EchoRequest{
		Message:     "sanitized message",
		CommunityID: "test-comm",
		TenantID:    "tenant-1",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err = broadcaster.RequestEcho(ctx, "test-comm", "agent-alpha", req)
	require.NoError(t, err)

	select {
	case received := <-reqReceivedChan:
		assert.Equal(t, req.Message, received.Message)
		assert.Equal(t, req.CommunityID, received.CommunityID)
		assert.Equal(t, req.TenantID, received.TenantID)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for echo request payload")
	}
}

func TestAvailable_Connected(t *testing.T) {
	ns, nc := startTestNatsServer(t)
	defer ns.Shutdown()
	defer nc.Close()

	logger := zerolog.New(nil)
	broadcaster := natsadapter.NewNATSCommunityBroadcaster(nc, logger)

	assert.True(t, broadcaster.Available())
}

func TestAvailable_NilConn(t *testing.T) {
	logger := zerolog.New(nil)
	broadcaster := natsadapter.NewNATSCommunityBroadcaster(nil, logger)

	assert.False(t, broadcaster.Available())
}
