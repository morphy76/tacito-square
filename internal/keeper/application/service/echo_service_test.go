package service_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/morphy76/tacito-square/internal/keeper/application/service"
	"github.com/morphy76/tacito-square/internal/keeper/domain/model"
	"github.com/morphy76/tacito-square/pkg/kubernetes/apis/tacito/v1alpha1"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockCommunityBroadcaster struct {
	mock.Mock
}

func (m *mockCommunityBroadcaster) RequestEcho(ctx context.Context, communityID, agentName string, req model.EchoRequest) (*model.EchoReply, error) {
	args := m.Called(ctx, communityID, agentName, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.EchoReply), args.Error(1)
}

func (m *mockCommunityBroadcaster) Available() bool {
	return m.Called().Bool(0)
}

func TestEchoCommunity_HappyPath(t *testing.T) {
	commRepo := new(mockCommunityRepository)
	agentRepo := new(mockAgentRepository)
	broadcaster := new(mockCommunityBroadcaster)
	crdCoord := new(mockCRDCoordinator)
	cfg := viper.New()

	svc := service.NewEchoService(commRepo, agentRepo, crdCoord, broadcaster, cfg)

	communityID := uuid.New()
	comm := &model.Community{ID: communityID, Name: "test-comm", Status: "active"}
	agents := []*model.Agent{
		{ID: uuid.New(), Name: "agent-1", Status: model.AgentStatusRunning, CommunityID: &communityID},
		{ID: uuid.New(), Name: "agent-2", Status: model.AgentStatusRunning, CommunityID: &communityID},
	}

	commRepo.On("GetByID", mock.Anything, communityID).Return(comm, nil)
	agentRepo.On("List", mock.Anything).Return(agents, nil)
	broadcaster.On("Available").Return(true)

	broadcaster.On("RequestEcho", mock.Anything, "test-comm", "agent-1", mock.Anything).Return(&model.EchoReply{
		AgentName: "agent-1",
		Decorated: "[agent:agent-1 at 2026-05-30T00:00:00Z] hello",
		Timestamp: "2026-05-30T00:00:00Z",
	}, nil)

	broadcaster.On("RequestEcho", mock.Anything, "test-comm", "agent-2", mock.Anything).Return(&model.EchoReply{
		AgentName: "agent-2",
		Decorated: "[agent:agent-2 at 2026-05-30T00:00:00Z] hello",
		Timestamp: "2026-05-30T00:00:00Z",
	}, nil)

	res, err := svc.EchoCommunity(context.Background(), communityID, "hello")
	require.NoError(t, err)
	assert.Equal(t, communityID.String(), res.CommunityID)
	assert.False(t, res.WokeCommunity)
	assert.Len(t, res.Results, 2)

	assert.Equal(t, "agent-1", res.Results[0].AgentName)
	assert.Equal(t, "[agent:agent-1 at 2026-05-30T00:00:00Z] hello", res.Results[0].Decorated)
	assert.Empty(t, res.Results[0].Error)

	assert.Equal(t, "agent-2", res.Results[1].AgentName)
	assert.Equal(t, "[agent:agent-2 at 2026-05-30T00:00:00Z] hello", res.Results[1].Decorated)
	assert.Empty(t, res.Results[1].Error)
}

func TestEchoCommunity_ExcludesNonRunningAgents(t *testing.T) {
	commRepo := new(mockCommunityRepository)
	agentRepo := new(mockAgentRepository)
	broadcaster := new(mockCommunityBroadcaster)
	crdCoord := new(mockCRDCoordinator)
	cfg := viper.New()

	svc := service.NewEchoService(commRepo, agentRepo, crdCoord, broadcaster, cfg)

	communityID := uuid.New()
	comm := &model.Community{ID: communityID, Name: "test-comm", Status: "active"}
	agents := []*model.Agent{
		{ID: uuid.New(), Name: "agent-running", Status: model.AgentStatusRunning, CommunityID: &communityID},
		{ID: uuid.New(), Name: "agent-defined", Status: model.AgentStatusDefined, CommunityID: &communityID},
	}

	commRepo.On("GetByID", mock.Anything, communityID).Return(comm, nil)
	agentRepo.On("List", mock.Anything).Return(agents, nil)
	broadcaster.On("Available").Return(true)

	broadcaster.On("RequestEcho", mock.Anything, "test-comm", "agent-running", mock.Anything).Return(&model.EchoReply{
		AgentName: "agent-running",
		Decorated: "[agent:agent-running] hello",
	}, nil)

	res, err := svc.EchoCommunity(context.Background(), communityID, "hello")
	require.NoError(t, err)
	assert.Len(t, res.Results, 1)
	assert.Equal(t, "agent-running", res.Results[0].AgentName)
}

func TestEchoCommunity_ConcurrentFanout(t *testing.T) {
	commRepo := new(mockCommunityRepository)
	agentRepo := new(mockAgentRepository)
	broadcaster := new(mockCommunityBroadcaster)
	crdCoord := new(mockCRDCoordinator)
	cfg := viper.New()

	svc := service.NewEchoService(commRepo, agentRepo, crdCoord, broadcaster, cfg)

	communityID := uuid.New()
	comm := &model.Community{ID: communityID, Name: "test-comm", Status: "active"}
	agents := []*model.Agent{
		{ID: uuid.New(), Name: "agent-1", Status: model.AgentStatusRunning, CommunityID: &communityID},
		{ID: uuid.New(), Name: "agent-2", Status: model.AgentStatusRunning, CommunityID: &communityID},
	}

	commRepo.On("GetByID", mock.Anything, communityID).Return(comm, nil)
	agentRepo.On("List", mock.Anything).Return(agents, nil)
	broadcaster.On("Available").Return(true)

	var wg sync.WaitGroup
	wg.Add(2)

	broadcaster.On("RequestEcho", mock.Anything, "test-comm", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			wg.Done()
			time.Sleep(50 * time.Millisecond)
		}).
		Return(&model.EchoReply{Decorated: "echo"}, nil)

	start := time.Now()
	res, err := svc.EchoCommunity(context.Background(), communityID, "hello")
	duration := time.Since(start)

	require.NoError(t, err)
	assert.Len(t, res.Results, 2)
	assert.Less(t, duration, 90*time.Millisecond)
}

func TestEchoCommunity_AgentTimeout(t *testing.T) {
	commRepo := new(mockCommunityRepository)
	agentRepo := new(mockAgentRepository)
	broadcaster := new(mockCommunityBroadcaster)
	crdCoord := new(mockCRDCoordinator)
	cfg := viper.New()

	svc := service.NewEchoService(commRepo, agentRepo, crdCoord, broadcaster, cfg)

	communityID := uuid.New()
	comm := &model.Community{ID: communityID, Name: "test-comm", Status: "active"}
	agents := []*model.Agent{
		{ID: uuid.New(), Name: "agent-alpha", Status: model.AgentStatusRunning, CommunityID: &communityID},
	}

	commRepo.On("GetByID", mock.Anything, communityID).Return(comm, nil)
	agentRepo.On("List", mock.Anything).Return(agents, nil)
	broadcaster.On("Available").Return(true)

	broadcaster.On("RequestEcho", mock.Anything, "test-comm", "agent-alpha", mock.Anything).Return(nil, context.DeadlineExceeded)

	res, err := svc.EchoCommunity(context.Background(), communityID, "hello")
	require.NoError(t, err)
	assert.Len(t, res.Results, 1)
	assert.Equal(t, "agent-alpha", res.Results[0].AgentName)
	assert.Empty(t, res.Results[0].Decorated)
	assert.Equal(t, "timeout", res.Results[0].Error)
}

func TestEchoCommunity_EmptyMessageAfterSanitization(t *testing.T) {
	commRepo := new(mockCommunityRepository)
	agentRepo := new(mockAgentRepository)
	broadcaster := new(mockCommunityBroadcaster)
	crdCoord := new(mockCRDCoordinator)
	cfg := viper.New()

	svc := service.NewEchoService(commRepo, agentRepo, crdCoord, broadcaster, cfg)

	_, err := svc.EchoCommunity(context.Background(), uuid.New(), "\x00\x01")
	assert.ErrorIs(t, err, service.ErrEmptyMessage)
}

func TestEchoCommunity_CommunityNotFound(t *testing.T) {
	commRepo := new(mockCommunityRepository)
	agentRepo := new(mockAgentRepository)
	broadcaster := new(mockCommunityBroadcaster)
	crdCoord := new(mockCRDCoordinator)
	cfg := viper.New()

	svc := service.NewEchoService(commRepo, agentRepo, crdCoord, broadcaster, cfg)

	communityID := uuid.New()
	broadcaster.On("Available").Return(true)
	commRepo.On("GetByID", mock.Anything, communityID).Return(nil, errors.New("not found"))

	_, err := svc.EchoCommunity(context.Background(), communityID, "hello")
	assert.ErrorIs(t, err, service.ErrCommunityNotFound)
}

func TestEchoCommunity_NATSUnavailable(t *testing.T) {
	commRepo := new(mockCommunityRepository)
	agentRepo := new(mockAgentRepository)
	broadcaster := new(mockCommunityBroadcaster)
	crdCoord := new(mockCRDCoordinator)
	cfg := viper.New()

	svc := service.NewEchoService(commRepo, agentRepo, crdCoord, broadcaster, cfg)

	broadcaster.On("Available").Return(false)

	_, err := svc.EchoCommunity(context.Background(), uuid.New(), "hello")
	assert.ErrorIs(t, err, service.ErrBroadcasterUnavailable)
}

func TestEchoCommunity_NoRunningAgents_NoneIdle(t *testing.T) {
	commRepo := new(mockCommunityRepository)
	agentRepo := new(mockAgentRepository)
	broadcaster := new(mockCommunityBroadcaster)
	crdCoord := new(mockCRDCoordinator)
	cfg := viper.New()

	svc := service.NewEchoService(commRepo, agentRepo, crdCoord, broadcaster, cfg)

	communityID := uuid.New()
	comm := &model.Community{ID: communityID, Name: "test-comm", Status: "active"}
	agents := []*model.Agent{
		{ID: uuid.New(), Name: "agent-1", Status: model.AgentStatusStopped, CommunityID: &communityID},
	}

	commRepo.On("GetByID", mock.Anything, communityID).Return(comm, nil)
	agentRepo.On("List", mock.Anything).Return(agents, nil)
	broadcaster.On("Available").Return(true)

	crdCoord.On("GetAgentCRDStatus", mock.Anything, mock.Anything).Return(nil, errors.New("not found"))

	_, err := svc.EchoCommunity(context.Background(), communityID, "hello")
	assert.ErrorIs(t, err, service.ErrNoRunningAgents)
}

func TestEchoCommunity_WokeCommunity(t *testing.T) {
	commRepo := new(mockCommunityRepository)
	agentRepo := new(mockAgentRepository)
	broadcaster := new(mockCommunityBroadcaster)
	crdCoord := new(mockCRDCoordinator)
	cfg := viper.New()
	cfg.Set("keeper.echo.wakeup_wait_seconds", 1)

	svc := service.NewEchoService(commRepo, agentRepo, crdCoord, broadcaster, cfg)

	communityID := uuid.New()
	comm := &model.Community{ID: communityID, Name: "test-comm", Status: "active"}
	
	agentID1 := uuid.New()
	agentsBefore := []*model.Agent{
		{ID: agentID1, Name: "agent-1", Status: model.AgentStatusStopped, CommunityID: &communityID},
	}
	agentsAfter := []*model.Agent{
		{ID: agentID1, Name: "agent-1", Status: model.AgentStatusRunning, CommunityID: &communityID},
	}

	commRepo.On("GetByID", mock.Anything, communityID).Return(comm, nil)
	
	agentRepo.On("List", mock.Anything).Return(agentsBefore, nil).Once()
	agentRepo.On("List", mock.Anything).Return(agentsAfter, nil)

	broadcaster.On("Available").Return(true)

	crdCoord.On("GetAgentCRDStatus", mock.Anything, agentID1).Return(&v1alpha1.TacitoAgentStatus{
		Phase: v1alpha1.PhaseIdle,
	}, nil)

	broadcaster.On("RequestEcho", mock.Anything, "test-comm", "agent-1", mock.Anything).Return(&model.EchoReply{
		AgentName: "agent-1",
		Decorated: "[agent:agent-1] hello",
	}, nil)

	res, err := svc.EchoCommunity(context.Background(), communityID, "hello")
	require.NoError(t, err)
	assert.True(t, res.WokeCommunity)
	assert.Len(t, res.Results, 1)
	assert.Equal(t, "agent-1", res.Results[0].AgentName)
}

func TestEchoCommunity_PendingDatabaseStatus_RunningCRDStatus(t *testing.T) {
	commRepo := new(mockCommunityRepository)
	agentRepo := new(mockAgentRepository)
	broadcaster := new(mockCommunityBroadcaster)
	crdCoord := new(mockCRDCoordinator)
	cfg := viper.New()

	svc := service.NewEchoService(commRepo, agentRepo, crdCoord, broadcaster, cfg)

	communityID := uuid.New()
	comm := &model.Community{ID: communityID, Name: "test-comm", Status: "active"}
	
	agentID1 := uuid.New()
	agents := []*model.Agent{
		{ID: agentID1, Name: "agent-1", Status: model.AgentStatusPending, CommunityID: &communityID},
	}

	commRepo.On("GetByID", mock.Anything, communityID).Return(comm, nil)
	agentRepo.On("List", mock.Anything).Return(agents, nil)
	broadcaster.On("Available").Return(true)

	// Mock GetAgentCRDStatus to return PhaseRunning
	crdCoord.On("GetAgentCRDStatus", mock.Anything, agentID1).Return(&v1alpha1.TacitoAgentStatus{
		Phase: v1alpha1.PhaseRunning,
	}, nil)

	// Expecting that GetAgentStatus / status sync will call agentRepo.Update to sync the status to running!
	agentRepo.On("Update", mock.Anything, mock.MatchedBy(func(a *model.Agent) bool {
		return a.ID == agentID1 && a.Status == model.AgentStatusRunning
	})).Return(nil)

	broadcaster.On("RequestEcho", mock.Anything, "test-comm", "agent-1", mock.Anything).Return(&model.EchoReply{
		AgentName: "agent-1",
		Decorated: "[agent:agent-1] hello",
	}, nil)

	res, err := svc.EchoCommunity(context.Background(), communityID, "hello")
	require.NoError(t, err)
	assert.Len(t, res.Results, 1)
	assert.Equal(t, "agent-1", res.Results[0].AgentName)
	assert.Empty(t, res.Results[0].Error)
}

