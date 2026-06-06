package service_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/morphy76/tacito-square/internal/keeper/application/ports/outbound"
	"github.com/morphy76/tacito-square/internal/keeper/application/service"
	"github.com/morphy76/tacito-square/pkg/events"
	"github.com/stretchr/testify/assert"
)

type mockPublisher struct {
	publishedSubject string
	publishedEvent   events.DomainEvent
	errToReturn      error
}

func (m *mockPublisher) Publish(ctx context.Context, subject string, event events.DomainEvent) error {
	m.publishedSubject = subject
	m.publishedEvent = event
	return m.errToReturn
}

type mockSubscription struct{}

func (m *mockSubscription) Stop() error { return nil }

type mockSubscriber struct {
	subscribedSubject string
	subscribedTenant  string
	subToReturn       outbound.EventSubscription
	errToReturn       error
}

func (m *mockSubscriber) Subscribe(ctx context.Context, subjectPattern string, tenantID string, handler func(*events.DomainEvent)) (outbound.EventSubscription, error) {
	m.subscribedSubject = subjectPattern
	m.subscribedTenant = tenantID
	return m.subToReturn, m.errToReturn
}

func TestPublishEvent_Success_Conversational(t *testing.T) {
	pub := &mockPublisher{}
	sub := &mockSubscriber{}
	svc := service.NewEventService(pub, sub)

	schemaRef := "urn:tacito:schema:conversational:start-thread:v1"
	payload := map[string]any{
		"thread_id":    "thread-123",
		"community_id": "comm-456",
		"agent_name":   "agent-alpha",
	}
	payloadBytes, _ := json.Marshal(payload)

	ctx := context.Background()
	tenantID := "tenant-xyz"
	ctx = context.WithValue(ctx, "tenant_id", tenantID) // Simulating resolved tenant

	evt, err := svc.PublishEvent(ctx, schemaRef, payloadBytes)
	assert.NoError(t, err)

	// Assert subject routing
	assert.Equal(t, "ts.community.comm-456.agent.agent-alpha", pub.publishedSubject)

	// Assert auto-populated fields
	assert.NotEmpty(t, evt.EventID)
	assert.NotEmpty(t, evt.OccurredAt)
	assert.True(t, strings.HasPrefix(evt.Source, "keeper/")) // resolves dynamically to keeper/{hostname}
	assert.Equal(t, tenantID, evt.TenantID)
	assert.Equal(t, schemaRef, evt.SchemaRef)
}

func TestPublishEvent_Conversational_StartThread_GenerateThreadID(t *testing.T) {
	pub := &mockPublisher{}
	sub := &mockSubscriber{}
	svc := service.NewEventService(pub, sub)

	schemaRef := "urn:tacito:schema:conversational:start-thread:v1"
	payload := map[string]any{
		"community_id": "comm-456",
		"agent_name":   "agent-alpha",
	}
	payloadBytes, _ := json.Marshal(payload)

	ctx := context.Background()
	ctx = context.WithValue(ctx, "tenant_id", "tenant-xyz")

	evt, err := svc.PublishEvent(ctx, schemaRef, payloadBytes)
	assert.NoError(t, err)

	var parsedPayload events.StartThreadPayload
	err = json.Unmarshal(evt.Payload, &parsedPayload)
	assert.NoError(t, err)

	assert.NotEmpty(t, parsedPayload.ThreadID)
	// Must be a valid UUID
	assert.Len(t, parsedPayload.ThreadID, 36)
}

func TestPublishEvent_Conversational_StartThread_PreserveThreadID(t *testing.T) {
	pub := &mockPublisher{}
	sub := &mockSubscriber{}
	svc := service.NewEventService(pub, sub)

	schemaRef := "urn:tacito:schema:conversational:start-thread:v1"
	payload := map[string]any{
		"thread_id":    "thread-existing-123",
		"community_id": "comm-456",
		"agent_name":   "agent-alpha",
	}
	payloadBytes, _ := json.Marshal(payload)

	ctx := context.Background()
	ctx = context.WithValue(ctx, "tenant_id", "tenant-xyz")

	evt, err := svc.PublishEvent(ctx, schemaRef, payloadBytes)
	assert.NoError(t, err)

	var parsedPayload events.StartThreadPayload
	err = json.Unmarshal(evt.Payload, &parsedPayload)
	assert.NoError(t, err)

	assert.Equal(t, "thread-existing-123", parsedPayload.ThreadID)
}

func TestPublishEvent_InvalidSchemaRef(t *testing.T) {
	pub := &mockPublisher{}
	sub := &mockSubscriber{}
	svc := service.NewEventService(pub, sub)

	_, err := svc.PublishEvent(context.Background(), "invalid-schema-ref", []byte(`{}`))
	assert.Error(t, err)
}

func TestPublishEvent_Sanitization(t *testing.T) {
	pub := &mockPublisher{}
	sub := &mockSubscriber{}
	svc := service.NewEventService(pub, sub)

	schemaRef := "urn:tacito:schema:conversational:add-user-message:v1"
	payload := map[string]any{
		"thread_id":    "thread-123",
		"community_id": "comm-456",
		"agent_name":   "agent-alpha",
		"message":      "hello\x00world\x01!",
	}
	payloadBytes, _ := json.Marshal(payload)

	ctx := context.Background()
	ctx = context.WithValue(ctx, "tenant_id", "tenant-xyz")

	evt, err := svc.PublishEvent(ctx, schemaRef, payloadBytes)
	assert.NoError(t, err)

	var parsedPayload events.AddUserMessagePayload
	err = json.Unmarshal(evt.Payload, &parsedPayload)
	assert.NoError(t, err)

	// Verify control characters stripped
	assert.Equal(t, "helloworld!", parsedPayload.Message)
}

func TestPublishEvent_SanitizedEmptyError(t *testing.T) {
	pub := &mockPublisher{}
	sub := &mockSubscriber{}
	svc := service.NewEventService(pub, sub)

	schemaRef := "urn:tacito:schema:conversational:add-user-message:v1"
	payload := map[string]any{
		"thread_id":    "thread-123",
		"community_id": "comm-456",
		"agent_name":   "agent-alpha",
		"message":      "\x00\x01",
	}
	payloadBytes, _ := json.Marshal(payload)

	ctx := context.Background()
	ctx = context.WithValue(ctx, "tenant_id", "tenant-xyz")

	_, err := svc.PublishEvent(ctx, schemaRef, payloadBytes)
	assert.Error(t, err)
}

func TestPublishEvent_UnknownSchema_RoutesToGenericTopic(t *testing.T) {
	pub := &mockPublisher{}
	sub := &mockSubscriber{}
	svc := service.NewEventService(pub, sub)

	schemaRef := "urn:tacito:schema:custom:operation:v1"
	payloadBytes := []byte(`{"some":"data"}`)

	ctx := context.Background()
	tenantID := "tenant-xyz"
	ctx = context.WithValue(ctx, "tenant_id", tenantID)

	_, err := svc.PublishEvent(ctx, schemaRef, payloadBytes)
	assert.NoError(t, err)

	// Routes to generic subject ts.events.{tenantID}
	assert.Equal(t, "ts.events.tenant-xyz", pub.publishedSubject)
}

func TestSubscribeEvents(t *testing.T) {
	pub := &mockPublisher{}
	sub := &mockSubscriber{}
	svc := service.NewEventService(pub, sub)

	tenantID := "tenant-xyz"
	handler := func(evt *events.DomainEvent) {}

	_, err := svc.SubscribeEvents(context.Background(), tenantID, handler)
	assert.NoError(t, err)

	assert.Equal(t, "ts.community.>", sub.subscribedSubject)
	assert.Equal(t, tenantID, sub.subscribedTenant)
}
