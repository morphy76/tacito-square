package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/morphy76/tacito-square/internal/keeper/domain/model"
	"github.com/morphy76/tacito-square/internal/shared/tenant"
	"github.com/morphy76/tacito-square/pkg/agentcard"
	"github.com/morphy76/tacito-square/pkg/events"
	"github.com/morphy76/tacito-square/pkg/kubernetes/apis/tacito/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockAgentRepository struct {
	mock.Mock
}

func (m *mockAgentRepository) Create(ctx context.Context, agent *model.Agent) error {
	return m.Called(ctx, agent).Error(0)
}
func (m *mockAgentRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Agent, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Agent), args.Error(1)
}
func (m *mockAgentRepository) GetByName(ctx context.Context, name string) (*model.Agent, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Agent), args.Error(1)
}
func (m *mockAgentRepository) List(ctx context.Context) ([]*model.Agent, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*model.Agent), args.Error(1)
}
func (m *mockAgentRepository) Update(ctx context.Context, agent *model.Agent) error {
	return m.Called(ctx, agent).Error(0)
}
func (m *mockAgentRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}
func (m *mockAgentRepository) AssignToCommunity(ctx context.Context, agentID uuid.UUID, communityID uuid.UUID) error {
	return m.Called(ctx, agentID, communityID).Error(0)
}

type mockLLMBindingRepository struct {
	mock.Mock
}

func (m *mockLLMBindingRepository) Create(ctx context.Context, binding *model.LLMBinding) error {
	return m.Called(ctx, binding).Error(0)
}
func (m *mockLLMBindingRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.LLMBinding, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.LLMBinding), args.Error(1)
}
func (m *mockLLMBindingRepository) GetByName(ctx context.Context, name string) (*model.LLMBinding, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.LLMBinding), args.Error(1)
}
func (m *mockLLMBindingRepository) List(ctx context.Context) ([]*model.LLMBinding, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.LLMBinding), args.Error(1)
}
func (m *mockLLMBindingRepository) Update(ctx context.Context, binding *model.LLMBinding) error {
	return m.Called(ctx, binding).Error(0)
}
func (m *mockLLMBindingRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}

type mockPromptRepository struct {
	mock.Mock
}

func (m *mockPromptRepository) CreateTemplate(ctx context.Context, template *model.PromptTemplate) error {
	return m.Called(ctx, template).Error(0)
}
func (m *mockPromptRepository) GetTemplateByID(ctx context.Context, id uuid.UUID) (*model.PromptTemplate, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.PromptTemplate), args.Error(1)
}
func (m *mockPromptRepository) ListTemplates(ctx context.Context) ([]*model.PromptTemplate, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.PromptTemplate), args.Error(1)
}
func (m *mockPromptRepository) UpdateTemplate(ctx context.Context, template *model.PromptTemplate) error {
	return m.Called(ctx, template).Error(0)
}
func (m *mockPromptRepository) DeleteTemplate(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}
func (m *mockPromptRepository) CreateCollection(ctx context.Context, collection *model.PromptCollection) error {
	return m.Called(ctx, collection).Error(0)
}
func (m *mockPromptRepository) GetCollectionByID(ctx context.Context, id uuid.UUID) (*model.PromptCollection, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.PromptCollection), args.Error(1)
}
func (m *mockPromptRepository) ListCollections(ctx context.Context) ([]*model.PromptCollection, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.PromptCollection), args.Error(1)
}
func (m *mockPromptRepository) UpdateCollection(ctx context.Context, collection *model.PromptCollection) error {
	return m.Called(ctx, collection).Error(0)
}
func (m *mockPromptRepository) DeleteCollection(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}
func (m *mockPromptRepository) ResolveCollectionPrompts(ctx context.Context, collectionID uuid.UUID) ([]*model.PromptTemplate, error) {
	args := m.Called(ctx, collectionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.PromptTemplate), args.Error(1)
}
func (m *mockPromptRepository) CreateVersion(ctx context.Context, version *model.PromptVersion) error {
	return m.Called(ctx, version).Error(0)
}
func (m *mockPromptRepository) GetLatestVersion(ctx context.Context, promptID uuid.UUID) (*model.PromptVersion, error) {
	args := m.Called(ctx, promptID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.PromptVersion), args.Error(1)
}
func (m *mockPromptRepository) AddPromptToCollection(ctx context.Context, collectionID uuid.UUID, promptID uuid.UUID) error {
	return m.Called(ctx, collectionID, promptID).Error(0)
}
func (m *mockPromptRepository) RemovePromptFromCollection(ctx context.Context, collectionID uuid.UUID, promptID uuid.UUID) error {
	return m.Called(ctx, collectionID, promptID).Error(0)
}
func (m *mockAgentRepository) UnassignFromCommunity(ctx context.Context, agentID uuid.UUID, communityID uuid.UUID) error {
	return m.Called(ctx, agentID, communityID).Error(0)
}
func (m *mockAgentRepository) UpdateStatus(ctx context.Context, agentID uuid.UUID, status model.AgentStatus) (bool, error) {
	args := m.Called(ctx, agentID, status)
	return args.Bool(0), args.Error(1)
}
func (m *mockAgentRepository) UpsertRegistration(ctx context.Context, agentID uuid.UUID, communityID uuid.UUID, card *agentcard.AgentCard) error {
	return m.Called(ctx, agentID, communityID, card).Error(0)
}
func (m *mockAgentRepository) GetRegistration(ctx context.Context, agentID uuid.UUID, communityID uuid.UUID) (*agentcard.AgentCard, time.Time, error) {
	args := m.Called(ctx, agentID, communityID)
	if args.Get(0) == nil {
		return nil, time.Time{}, args.Error(2)
	}
	return args.Get(0).(*agentcard.AgentCard), args.Get(1).(time.Time), args.Error(2)
}
func (m *mockAgentRepository) GetActiveRegistrationsByCommunity(ctx context.Context, communityID uuid.UUID) ([]*agentcard.AgentCard, time.Time, error) {
	args := m.Called(ctx, communityID)
	if args.Get(0) == nil {
		return nil, time.Time{}, args.Error(2)
	}
	return args.Get(0).([]*agentcard.AgentCard), args.Get(1).(time.Time), args.Error(2)
}
func (m *mockAgentRepository) PruneStaleRegistrations(ctx context.Context, threshold time.Duration) ([]agentcard.AgentCommunityRef, error) {
	args := m.Called(ctx, threshold)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]agentcard.AgentCommunityRef), args.Error(1)
}
func (m *mockAgentRepository) DeleteRegistration(ctx context.Context, agentID uuid.UUID, communityID uuid.UUID) error {
	return m.Called(ctx, agentID, communityID).Error(0)
}

type mockCRDCoordinator struct {
	mock.Mock
	submitChan   chan struct{}
	teardownChan chan struct{}
}

func (m *mockCRDCoordinator) SubmitAgentCRD(ctx context.Context, agent *model.Agent) error {
	// Assert tenant context propagated safely
	ten := tenant.FromContext(ctx)
	if ten == nil {
		m.Called(ctx, agent)
		return errors.New("missing tenant context in background call")
	}

	// Simulate slow operation to test non-blocking concurrency
	time.Sleep(50 * time.Millisecond)

	m.Called(ctx, agent)
	close(m.submitChan) // Signal completion AFTER registering the mock call
	return nil
}

func (m *mockCRDCoordinator) TeardownAgentCRD(ctx context.Context, agent *model.Agent) error {
	ten := tenant.FromContext(ctx)
	if ten == nil {
		m.Called(ctx, agent)
		return errors.New("missing tenant context in background call")
	}

	time.Sleep(50 * time.Millisecond)

	m.Called(ctx, agent)
	close(m.teardownChan) // Signal completion AFTER registering the mock call
	return nil
}

func (m *mockCRDCoordinator) GetAgentCRDStatus(ctx context.Context, agentID uuid.UUID) (*v1alpha1.TacitoAgentStatus, error) {
	args := m.Called(ctx, agentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*v1alpha1.TacitoAgentStatus), args.Error(1)
}

type mockCache struct {
	mock.Mock
}

func (m *mockCache) Get(ctx context.Context, key string, dest interface{}) error {
	args := m.Called(ctx, key, dest)
	return args.Error(0)
}

func (m *mockCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	args := m.Called(ctx, key, value, ttl)
	return args.Error(0)
}

func (m *mockCache) Invalidate(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

type mockPublisher struct {
	mock.Mock
}

func (m *mockPublisher) Publish(ctx context.Context, subject string, event events.DomainEvent) error {
	args := m.Called(ctx, subject, event)
	return args.Error(0)
}

type mockCommunityRepository struct {
	mock.Mock
}

func (m *mockCommunityRepository) Create(ctx context.Context, community *model.Community) error {
	return m.Called(ctx, community).Error(0)
}
func (m *mockCommunityRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Community, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Community), args.Error(1)
}
func (m *mockCommunityRepository) GetByName(ctx context.Context, name string) (*model.Community, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Community), args.Error(1)
}
func (m *mockCommunityRepository) List(ctx context.Context) ([]*model.Community, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.Community), args.Error(1)
}
func (m *mockCommunityRepository) Update(ctx context.Context, community *model.Community) error {
	return m.Called(ctx, community).Error(0)
}
func (m *mockCommunityRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}

func TestAgentService_Assign_AsynchronousNonBlocking(t *testing.T) {
	repo := new(mockAgentRepository)
	commRepo := new(mockCommunityRepository)
	submitChan := make(chan struct{})
	crd := &mockCRDCoordinator{submitChan: submitChan}

	svc := NewAgentService(repo, commRepo, crd, nil, nil, new(mockLLMBindingRepository), new(mockPromptRepository))

	ten, _ := tenant.New("acme.com", "")
	ctx := tenant.ContextWithTenant(context.Background(), ten)

	agentID := uuid.New()
	commID := uuid.New()
	agent := &model.Agent{ID: agentID, Name: "reactive-agent"}
	comm := &model.Community{
		ID:       commID,
		Status:   model.CommunityStatusActive,
		Topology: model.CommunityTopologySingleAgent,
	}

	commRepo.On("GetByID", mock.Anything, commID).Return(comm, nil)
	repo.On("GetByID", mock.Anything, agentID).Return(agent, nil)
	repo.On("List", mock.Anything).Return([]*model.Agent{}, nil)
	repo.On("AssignToCommunity", mock.Anything, agentID, commID).Return(nil)
	crd.On("SubmitAgentCRD", mock.Anything, mock.MatchedBy(func(a *model.Agent) bool {
		return a.CommunityID != nil && *a.CommunityID == commID && a.Status == model.AgentStatusPending
	})).Return(nil)

	start := time.Now()
	err := svc.Assign(ctx, commID, agentID)
	duration := time.Since(start)

	assert.NoError(t, err)
	// Assert Assign method returns immediately (well below the 50ms sleep of CRD coordinator)
	assert.Less(t, duration, 20*time.Millisecond, "Assign should return immediately and not block")

	// Wait for background execution to complete
	select {
	case <-submitChan:
		// Background execution finished successfully
	case <-time.After(150 * time.Millisecond):
		t.Fatal("timeout waiting for background SubmitAgentCRD execution")
	}

	repo.AssertExpectations(t)
	crd.AssertExpectations(t)
}

func TestAgentService_Unassign_AsynchronousNonBlocking(t *testing.T) {
	repo := new(mockAgentRepository)
	teardownChan := make(chan struct{})
	crd := &mockCRDCoordinator{teardownChan: teardownChan}

	svc := NewAgentService(repo, nil, crd, nil, nil, new(mockLLMBindingRepository), new(mockPromptRepository))

	ten, _ := tenant.New("acme.com", "")
	ctx := tenant.ContextWithTenant(context.Background(), ten)

	agentID := uuid.New()
	commID := uuid.New()
	agent := &model.Agent{ID: agentID, Name: "reactive-agent"}

	repo.On("GetByID", mock.Anything, agentID).Return(agent, nil)
	repo.On("UnassignFromCommunity", mock.Anything, agentID, commID).Return(nil)
	repo.On("DeleteRegistration", mock.Anything, agentID, commID).Return(nil)
	crd.On("TeardownAgentCRD", mock.Anything, mock.Anything).Return(nil)

	start := time.Now()
	err := svc.Unassign(ctx, commID, agentID)
	duration := time.Since(start)

	assert.NoError(t, err)
	// Assert Unassign method returns immediately (well below the 50ms sleep of CRD coordinator)
	assert.Less(t, duration, 20*time.Millisecond, "Unassign should return immediately and not block")

	// Wait for background execution to complete
	select {
	case <-teardownChan:
		// Background execution finished successfully
	case <-time.After(150 * time.Millisecond):
		t.Fatal("timeout waiting for background TeardownAgentCRD execution")
	}

	repo.AssertExpectations(t)
	crd.AssertExpectations(t)
}

func TestAgentService_Unassign_EvictsAndPublishes(t *testing.T) {
	repo := new(mockAgentRepository)
	teardownChan := make(chan struct{})
	crd := &mockCRDCoordinator{teardownChan: teardownChan}
	cache := new(mockCache)
	publisher := new(mockPublisher)

	svc := NewAgentService(repo, nil, crd, cache, publisher, new(mockLLMBindingRepository), new(mockPromptRepository))

	ten, _ := tenant.New("acme.com", "")
	ctx := tenant.ContextWithTenant(context.Background(), ten)

	agentID := uuid.New()
	commID := uuid.New()
	agent := &model.Agent{ID: agentID, Name: "reactive-agent"}

	repo.On("GetByID", mock.Anything, agentID).Return(agent, nil)
	repo.On("UnassignFromCommunity", mock.Anything, agentID, commID).Return(nil)
	repo.On("DeleteRegistration", mock.Anything, agentID, commID).Return(nil)
	crd.On("TeardownAgentCRD", mock.Anything, mock.Anything).Return(nil)

	// Assertions for cache invalidation
	agentKey := "communities:" + commID.String() + ":agents:" + agentID.String()
	registryKey := "communities:" + commID.String() + ":registry"
	cache.On("Invalidate", mock.Anything, agentKey).Return(nil)
	cache.On("Invalidate", mock.Anything, registryKey).Return(nil)

	// Assertions for NATS event publication
	subject := "ts.community." + commID.String() + ".agent." + agentID.String() + ".status"
	publisher.On("Publish", mock.Anything, subject, mock.Anything).Return(nil)

	err := svc.Unassign(ctx, commID, agentID)
	assert.NoError(t, err)

	// Wait for background execution to complete
	select {
	case <-teardownChan:
		// Background execution finished successfully
	case <-time.After(150 * time.Millisecond):
		t.Fatal("timeout waiting for background TeardownAgentCRD execution")
	}

	repo.AssertExpectations(t)
	crd.AssertExpectations(t)
	cache.AssertExpectations(t)
	publisher.AssertExpectations(t)
}

func TestAgentService_Assign_AlreadyAssigned_Running(t *testing.T) {
	repo := new(mockAgentRepository)
	commRepo := new(mockCommunityRepository)
	crd := new(mockCRDCoordinator)

	svc := NewAgentService(repo, commRepo, crd, nil, nil, new(mockLLMBindingRepository), new(mockPromptRepository))

	ten, _ := tenant.New("acme.com", "")
	ctx := tenant.ContextWithTenant(context.Background(), ten)

	agentID := uuid.New()
	commID := uuid.New()
	agent := &model.Agent{
		ID:          agentID,
		Name:        "reactive-agent",
		CommunityID: &commID,
		Status:      model.AgentStatusRunning,
	}
	comm := &model.Community{
		ID:       commID,
		Status:   model.CommunityStatusActive,
		Topology: model.CommunityTopologySingleAgent,
	}

	commRepo.On("GetByID", mock.Anything, commID).Return(comm, nil)
	repo.On("GetByID", mock.Anything, agentID).Return(agent, nil)
	crd.On("GetAgentCRDStatus", mock.Anything, agentID).Return(&v1alpha1.TacitoAgentStatus{Phase: v1alpha1.PhaseRunning}, nil)

	err := svc.Assign(ctx, commID, agentID)
	assert.NoError(t, err)

	repo.AssertExpectations(t)
	crd.AssertExpectations(t)
}

func TestAgentService_Assign_AlreadyAssigned_NotRunning(t *testing.T) {
	repo := new(mockAgentRepository)
	commRepo := new(mockCommunityRepository)
	submitChan := make(chan struct{})
	crd := &mockCRDCoordinator{submitChan: submitChan}

	svc := NewAgentService(repo, commRepo, crd, nil, nil, new(mockLLMBindingRepository), new(mockPromptRepository))

	ten, _ := tenant.New("acme.com", "")
	ctx := tenant.ContextWithTenant(context.Background(), ten)

	agentID := uuid.New()
	commID := uuid.New()
	agent := &model.Agent{
		ID:          agentID,
		Name:        "reactive-agent",
		CommunityID: &commID,
		Status:      model.AgentStatusStopped,
	}
	comm := &model.Community{
		ID:       commID,
		Status:   model.CommunityStatusActive,
		Topology: model.CommunityTopologySingleAgent,
	}

	commRepo.On("GetByID", mock.Anything, commID).Return(comm, nil)
	repo.On("GetByID", mock.Anything, agentID).Return(agent, nil)
	crd.On("GetAgentCRDStatus", mock.Anything, agentID).Return((*v1alpha1.TacitoAgentStatus)(nil), nil)
	repo.On("Update", mock.Anything, mock.MatchedBy(func(a *model.Agent) bool {
		return a.ID == agentID && a.Status == model.AgentStatusPending
	})).Return(nil)
	crd.On("SubmitAgentCRD", mock.Anything, mock.MatchedBy(func(a *model.Agent) bool {
		return a.CommunityID != nil && *a.CommunityID == commID && a.Status == model.AgentStatusPending
	})).Return(nil)

	err := svc.Assign(ctx, commID, agentID)
	assert.NoError(t, err)

	// Wait for background execution to complete
	select {
	case <-submitChan:
		// Background execution finished successfully
	case <-time.After(150 * time.Millisecond):
		t.Fatal("timeout waiting for background SubmitAgentCRD execution")
	}

	repo.AssertExpectations(t)
	crd.AssertExpectations(t)
}

func TestAgentService_Unassign_AlreadyUnassigned_NotRunning(t *testing.T) {
	repo := new(mockAgentRepository)
	crd := new(mockCRDCoordinator)

	svc := NewAgentService(repo, nil, crd, nil, nil, new(mockLLMBindingRepository), new(mockPromptRepository))

	ten, _ := tenant.New("acme.com", "")
	ctx := tenant.ContextWithTenant(context.Background(), ten)

	agentID := uuid.New()
	commID := uuid.New()
	agent := &model.Agent{
		ID:          agentID,
		Name:        "reactive-agent",
		CommunityID: nil,
		Status:      model.AgentStatusStopped,
	}

	repo.On("GetByID", mock.Anything, agentID).Return(agent, nil)
	crd.On("GetAgentCRDStatus", mock.Anything, agentID).Return((*v1alpha1.TacitoAgentStatus)(nil), nil)

	err := svc.Unassign(ctx, commID, agentID)
	assert.NoError(t, err)

	repo.AssertExpectations(t)
	crd.AssertExpectations(t)
}

func TestAgentService_Unassign_AlreadyUnassigned_Running(t *testing.T) {
	repo := new(mockAgentRepository)
	teardownChan := make(chan struct{})
	crd := &mockCRDCoordinator{teardownChan: teardownChan}
	cache := new(mockCache)
	publisher := new(mockPublisher)

	svc := NewAgentService(repo, nil, crd, cache, publisher, new(mockLLMBindingRepository), new(mockPromptRepository))

	ten, _ := tenant.New("acme.com", "")
	ctx := tenant.ContextWithTenant(context.Background(), ten)

	agentID := uuid.New()
	commID := uuid.New()
	agent := &model.Agent{
		ID:          agentID,
		Name:        "reactive-agent",
		CommunityID: nil,
		Status:      model.AgentStatusStopped,
	}

	repo.On("GetByID", mock.Anything, agentID).Return(agent, nil)
	crd.On("GetAgentCRDStatus", mock.Anything, agentID).Return(&v1alpha1.TacitoAgentStatus{Phase: v1alpha1.PhaseRunning}, nil)
	crd.On("TeardownAgentCRD", mock.Anything, mock.Anything).Return(nil)
	repo.On("DeleteRegistration", mock.Anything, agentID, commID).Return(nil)

	// Assertions for cache invalidation
	agentKey := "communities:" + commID.String() + ":agents:" + agentID.String()
	registryKey := "communities:" + commID.String() + ":registry"
	cache.On("Invalidate", mock.Anything, agentKey).Return(nil)
	cache.On("Invalidate", mock.Anything, registryKey).Return(nil)

	// Assertions for NATS event publication
	subject := "ts.community." + commID.String() + ".agent." + agentID.String() + ".status"
	publisher.On("Publish", mock.Anything, subject, mock.Anything).Return(nil)

	err := svc.Unassign(ctx, commID, agentID)
	assert.NoError(t, err)

	// Wait for background execution to complete
	select {
	case <-teardownChan:
		// Background execution finished successfully
	case <-time.After(150 * time.Millisecond):
		t.Fatal("timeout waiting for background TeardownAgentCRD execution")
	}

	repo.AssertExpectations(t)
	crd.AssertExpectations(t)
	cache.AssertExpectations(t)
	publisher.AssertExpectations(t)
}

func TestAgentService_PromptAttachmentAndResolution(t *testing.T) {
	repo := new(mockAgentRepository)
	promptRepo := new(mockPromptRepository)
	cache := new(mockCache)
	svc := NewAgentService(repo, nil, nil, cache, nil, new(mockLLMBindingRepository), promptRepo)

	ten, _ := tenant.New("acme.com", "")
	ctx := tenant.ContextWithTenant(context.Background(), ten)

	agentID := uuid.New()
	ptID := uuid.New()
	colID := uuid.New()

	agent := &model.Agent{
		ID:                agentID,
		TenantID:          ten.FullName(),
		Name:              "test-agent",
		Prompts:           []uuid.UUID{},
		PromptCollections: []uuid.UUID{},
	}

	repo.On("GetByID", mock.Anything, agentID).Return(agent, nil).Times(5)
	repo.On("Update", mock.Anything, agent).Return(nil).Times(4)

	cacheKey := "agent-prompts:" + ten.FullName() + ":" + agentID.String()
	cache.On("Invalidate", mock.Anything, cacheKey).Return(nil).Times(4)

	// 1. Attach prompt
	err := svc.AttachPromptToAgent(ctx, agentID, ptID)
	assert.NoError(t, err)
	assert.Contains(t, agent.Prompts, ptID)

	// 2. Detach prompt
	err = svc.DetachPromptFromAgent(ctx, agentID, ptID)
	assert.NoError(t, err)
	assert.NotContains(t, agent.Prompts, ptID)

	// 3. Attach collection
	err = svc.AttachCollectionToAgent(ctx, agentID, colID)
	assert.NoError(t, err)
	assert.Contains(t, agent.PromptCollections, colID)

	// 4. Detach collection
	err = svc.DetachCollectionFromAgent(ctx, agentID, colID)
	assert.NoError(t, err)
	assert.NotContains(t, agent.PromptCollections, colID)

	// 5. Resolve prompts
	pt := &model.PromptTemplate{
		ID:      ptID,
		Content: "Hello",
		Status:  model.PromptStatusActive,
	}
	agent.Prompts = []uuid.UUID{ptID}
	promptRepo.On("GetTemplateByID", mock.Anything, ptID).Return(pt, nil)

	cache.On("Get", mock.Anything, cacheKey, mock.Anything).Return(errors.New("cache miss"))
	cache.On("Set", mock.Anything, cacheKey, mock.Anything, mock.Anything).Return(nil)

	resolved, err := svc.ResolveEffectivePrompts(ctx, agentID)
	assert.NoError(t, err)
	assert.Len(t, resolved, 1)
	assert.Equal(t, ptID, resolved[0].ID)

	repo.AssertExpectations(t)
	promptRepo.AssertExpectations(t)
	cache.AssertExpectations(t)
}

