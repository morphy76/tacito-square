package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/morphy76/tacito-square/internal/keeper/domain/model"
	"github.com/morphy76/tacito-square/internal/shared/tenant"
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
func (m *mockAgentRepository) UnassignFromCommunity(ctx context.Context, agentID uuid.UUID, communityID uuid.UUID) error {
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


func TestAgentService_Assign_AsynchronousNonBlocking(t *testing.T) {
	repo := new(mockAgentRepository)
	submitChan := make(chan struct{})
	crd := &mockCRDCoordinator{submitChan: submitChan}

	svc := NewAgentService(repo, crd)

	ten, _ := tenant.New("acme.com", "")
	ctx := tenant.ContextWithTenant(context.Background(), ten)

	agentID := uuid.New()
	commID := uuid.New()
	agent := &model.Agent{ID: agentID, Name: "reactive-agent"}

	repo.On("AssignToCommunity", mock.Anything, agentID, commID).Return(nil)
	repo.On("GetByID", mock.Anything, agentID).Return(agent, nil)
	crd.On("SubmitAgentCRD", mock.Anything, mock.Anything).Return(nil)

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

	svc := NewAgentService(repo, crd)

	ten, _ := tenant.New("acme.com", "")
	ctx := tenant.ContextWithTenant(context.Background(), ten)

	agentID := uuid.New()
	commID := uuid.New()
	agent := &model.Agent{ID: agentID, Name: "reactive-agent"}

	repo.On("GetByID", mock.Anything, agentID).Return(agent, nil)
	repo.On("UnassignFromCommunity", mock.Anything, agentID, commID).Return(nil)
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
