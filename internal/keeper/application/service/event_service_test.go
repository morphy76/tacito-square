package service_test

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/morphy76/tacito-square/internal/keeper/application/ports/outbound"
	"github.com/morphy76/tacito-square/internal/keeper/application/service"
	"github.com/morphy76/tacito-square/internal/keeper/domain/model"
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

// mockCommunityRepo implements outbound.CommunityRepository for unit tests.
type mockCommunityRepo struct {
	communities map[uuid.UUID]*model.Community
}

func newMockCommunityRepo() *mockCommunityRepo {
	return &mockCommunityRepo{
		communities: make(map[uuid.UUID]*model.Community),
	}
}

func (m *mockCommunityRepo) Create(ctx context.Context, community *model.Community) error {
	m.communities[community.ID] = community
	return nil
}

func (m *mockCommunityRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.Community, error) {
	c, ok := m.communities[id]
	if !ok {
		return nil, fmt.Errorf("community not found: %s", id)
	}
	return c, nil
}

func (m *mockCommunityRepo) GetByName(ctx context.Context, name string) (*model.Community, error) {
	for _, c := range m.communities {
		if c.Name == name {
			return c, nil
		}
	}
	return nil, fmt.Errorf("community not found by name: %s", name)
}

func (m *mockCommunityRepo) List(ctx context.Context) ([]*model.Community, error) {
	var result []*model.Community
	for _, c := range m.communities {
		result = append(result, c)
	}
	return result, nil
}

func (m *mockCommunityRepo) Update(ctx context.Context, community *model.Community) error {
	m.communities[community.ID] = community
	return nil
}

func (m *mockCommunityRepo) Delete(ctx context.Context, id uuid.UUID) error {
	delete(m.communities, id)
	return nil
}

func TestPublishEvent_Success_Conversational(t *testing.T) {
	pub := &mockPublisher{}
	sub := &mockSubscriber{}
	commRepo := newMockCommunityRepo()
	svc := service.NewEventService(pub, sub, commRepo)

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

func TestPublishEvent_Success_Conversational_OptionalAgentName(t *testing.T) {
	pub := &mockPublisher{}
	sub := &mockSubscriber{}
	commRepo := newMockCommunityRepo()

	// Register a single-agent community so the topology lookup works
	commID := uuid.New()
	commRepo.communities[commID] = &model.Community{
		ID:       commID,
		TenantID: "tenant-xyz",
		Name:     "test-community",
		Topology: model.CommunityTopologySingleAgent,
		Status:   model.CommunityStatusActive,
	}

	svc := service.NewEventService(pub, sub, commRepo)

	schemaRef := "urn:tacito:schema:conversational:start-thread:v1"
	payload := map[string]any{
		"thread_id":    "thread-123",
		"community_id": commID.String(),
	}
	payloadBytes, _ := json.Marshal(payload)

	ctx := context.Background()
	tenantID := "tenant-xyz"
	ctx = context.WithValue(ctx, "tenant_id", tenantID)

	_, err := svc.PublishEvent(ctx, schemaRef, payloadBytes)
	assert.NoError(t, err)

	// Assert subject routing resolves to agent.all when agent_name is omitted for single-agent topology
	assert.Equal(t, fmt.Sprintf("ts.community.%s.agent.all", commID.String()), pub.publishedSubject)
}

func TestPublishEvent_HubSpoke_RoutesToHub(t *testing.T) {
	pub := &mockPublisher{}
	sub := &mockSubscriber{}
	commRepo := newMockCommunityRepo()

	// Register a hub-spoke community
	commID := uuid.New()
	commRepo.communities[commID] = &model.Community{
		ID:       commID,
		TenantID: "tenant-xyz",
		Name:     "hub-community",
		Topology: model.CommunityTopologyHubSpoke,
		Status:   model.CommunityStatusActive,
	}

	svc := service.NewEventService(pub, sub, commRepo)

	schemaRef := "urn:tacito:schema:conversational:add-user-message:v1"
	payload := map[string]any{
		"thread_id":    "thread-hub-1",
		"community_id": commID.String(),
		"message":      "hello hub",
	}
	payloadBytes, _ := json.Marshal(payload)

	ctx := context.Background()
	ctx = context.WithValue(ctx, "tenant_id", "tenant-xyz")

	_, err := svc.PublishEvent(ctx, schemaRef, payloadBytes)
	assert.NoError(t, err)

	// Assert subject routing resolves to agent.hub for hub-spoke topology
	assert.Equal(t, fmt.Sprintf("ts.community.%s.agent.hub", commID.String()), pub.publishedSubject)
}

func TestPublishEvent_SingleAgent_RoutesToAll(t *testing.T) {
	pub := &mockPublisher{}
	sub := &mockSubscriber{}
	commRepo := newMockCommunityRepo()

	// Register a single-agent community
	commID := uuid.New()
	commRepo.communities[commID] = &model.Community{
		ID:       commID,
		TenantID: "tenant-xyz",
		Name:     "solo-community",
		Topology: model.CommunityTopologySingleAgent,
		Status:   model.CommunityStatusActive,
	}

	svc := service.NewEventService(pub, sub, commRepo)

	schemaRef := "urn:tacito:schema:conversational:add-user-message:v1"
	payload := map[string]any{
		"thread_id":    "thread-solo-1",
		"community_id": commID.String(),
		"message":      "hello solo",
	}
	payloadBytes, _ := json.Marshal(payload)

	ctx := context.Background()
	ctx = context.WithValue(ctx, "tenant_id", "tenant-xyz")

	_, err := svc.PublishEvent(ctx, schemaRef, payloadBytes)
	assert.NoError(t, err)

	// Assert subject routing resolves to agent.all for single-agent topology
	assert.Equal(t, fmt.Sprintf("ts.community.%s.agent.all", commID.String()), pub.publishedSubject)
}

func TestPublishEvent_UnknownCommunity_RoutesToAll(t *testing.T) {
	pub := &mockPublisher{}
	sub := &mockSubscriber{}
	commRepo := newMockCommunityRepo() // empty repo — no communities registered

	svc := service.NewEventService(pub, sub, commRepo)

	unknownID := uuid.New()
	schemaRef := "urn:tacito:schema:conversational:add-user-message:v1"
	payload := map[string]any{
		"thread_id":    "thread-unknown-1",
		"community_id": unknownID.String(),
		"message":      "hello unknown",
	}
	payloadBytes, _ := json.Marshal(payload)

	ctx := context.Background()
	ctx = context.WithValue(ctx, "tenant_id", "tenant-xyz")

	_, err := svc.PublishEvent(ctx, schemaRef, payloadBytes)
	assert.NoError(t, err)

	// Graceful degradation: unknown community falls back to agent.all
	assert.Equal(t, fmt.Sprintf("ts.community.%s.agent.all", unknownID.String()), pub.publishedSubject)
}

func TestPublishEvent_Conversational_StartThread_GenerateThreadID(t *testing.T) {
	pub := &mockPublisher{}
	sub := &mockSubscriber{}
	commRepo := newMockCommunityRepo()
	svc := service.NewEventService(pub, sub, commRepo)

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
	commRepo := newMockCommunityRepo()
	svc := service.NewEventService(pub, sub, commRepo)

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
	commRepo := newMockCommunityRepo()
	svc := service.NewEventService(pub, sub, commRepo)

	_, err := svc.PublishEvent(context.Background(), "invalid-schema-ref", []byte(`{}`))
	assert.Error(t, err)
}

func TestPublishEvent_Sanitization(t *testing.T) {
	pub := &mockPublisher{}
	sub := &mockSubscriber{}
	commRepo := newMockCommunityRepo()
	svc := service.NewEventService(pub, sub, commRepo)

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
	commRepo := newMockCommunityRepo()
	svc := service.NewEventService(pub, sub, commRepo)

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
	commRepo := newMockCommunityRepo()
	svc := service.NewEventService(pub, sub, commRepo)

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
	commRepo := newMockCommunityRepo()
	svc := service.NewEventService(pub, sub, commRepo)

	tenantID := "tenant-xyz"
	handler := func(evt *events.DomainEvent) {}

	_, err := svc.SubscribeEvents(context.Background(), tenantID, handler)
	assert.NoError(t, err)

	assert.Equal(t, "ts.community.>", sub.subscribedSubject)
	assert.Equal(t, tenantID, sub.subscribedTenant)
}
