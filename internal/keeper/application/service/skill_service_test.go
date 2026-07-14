package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/morphy76/tacito-square/internal/keeper/domain/model"
	"github.com/morphy76/tacito-square/pkg/agentcard"
	"github.com/stretchr/testify/assert"
)

type mockSkillRepository struct {
	skills                  map[uuid.UUID]*model.Skill
	collections             map[uuid.UUID]*model.SkillCollection
	attachedCollections     map[uuid.UUID][]uuid.UUID // agent -> collections
	collectionSkills        map[uuid.UUID][]uuid.UUID // collection -> skills
	updateCalled            bool
	attachCollectionCalled  bool
	detachCollectionCalled  bool
	addSkillCalled          bool
	removeSkillCalled       bool
}

func newMockSkillRepository() *mockSkillRepository {
	return &mockSkillRepository{
		skills:              make(map[uuid.UUID]*model.Skill),
		collections:         make(map[uuid.UUID]*model.SkillCollection),
		attachedCollections: make(map[uuid.UUID][]uuid.UUID),
		collectionSkills:    make(map[uuid.UUID][]uuid.UUID),
	}
}

func (m *mockSkillRepository) Create(ctx context.Context, skill *model.Skill) error {
	m.skills[skill.ID] = skill
	return nil
}

func (m *mockSkillRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Skill, error) {
	s, ok := m.skills[id]
	if !ok {
		return nil, errors.New("skill not found")
	}
	return s, nil
}

func (m *mockSkillRepository) GetByName(ctx context.Context, name string) (*model.Skill, error) {
	return nil, nil
}

func (m *mockSkillRepository) List(ctx context.Context) ([]*model.Skill, error) {
	return nil, nil
}

func (m *mockSkillRepository) Update(ctx context.Context, skill *model.Skill) error {
	m.updateCalled = true
	m.skills[skill.ID] = skill
	return nil
}

func (m *mockSkillRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (m *mockSkillRepository) AttachSkillToAgent(ctx context.Context, agentID uuid.UUID, skillID uuid.UUID) error {
	return nil
}

func (m *mockSkillRepository) DetachSkillFromAgent(ctx context.Context, agentID uuid.UUID, skillID uuid.UUID) error {
	return nil
}

func (m *mockSkillRepository) ListSkillsByAgent(ctx context.Context, agentID uuid.UUID) ([]*model.Skill, error) {
	return nil, nil
}

func (m *mockSkillRepository) CreateCollection(ctx context.Context, collection *model.SkillCollection) error {
	m.collections[collection.ID] = collection
	return nil
}

func (m *mockSkillRepository) GetCollectionByID(ctx context.Context, id uuid.UUID) (*model.SkillCollection, error) {
	c, ok := m.collections[id]
	if !ok {
		return nil, errors.New("collection not found")
	}
	return c, nil
}

func (m *mockSkillRepository) ListCollections(ctx context.Context) ([]*model.SkillCollection, error) {
	return nil, nil
}

func (m *mockSkillRepository) UpdateCollection(ctx context.Context, collection *model.SkillCollection) error {
	return nil
}

func (m *mockSkillRepository) DeleteCollection(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (m *mockSkillRepository) ResolveCollectionSkills(ctx context.Context, collectionID uuid.UUID) ([]*model.Skill, error) {
	sids := m.collectionSkills[collectionID]
	var res []*model.Skill
	for _, sid := range sids {
		if s, ok := m.skills[sid]; ok {
			res = append(res, s)
		}
	}
	return res, nil
}

func (m *mockSkillRepository) AttachCollectionToAgent(ctx context.Context, agentID uuid.UUID, collectionID uuid.UUID) error {
	m.attachCollectionCalled = true
	m.attachedCollections[agentID] = append(m.attachedCollections[agentID], collectionID)
	return nil
}

func (m *mockSkillRepository) DetachCollectionFromAgent(ctx context.Context, agentID uuid.UUID, collectionID uuid.UUID) error {
	m.detachCollectionCalled = true
	list := m.attachedCollections[agentID]
	for i, cid := range list {
		if cid == collectionID {
			m.attachedCollections[agentID] = append(list[:i], list[i+1:]...)
			break
		}
	}
	return nil
}

func (m *mockSkillRepository) AddSkillToCollection(ctx context.Context, collectionID uuid.UUID, skillID uuid.UUID) error {
	m.addSkillCalled = true
	m.collectionSkills[collectionID] = append(m.collectionSkills[collectionID], skillID)
	return nil
}

func (m *mockSkillRepository) RemoveSkillFromCollection(ctx context.Context, collectionID uuid.UUID, skillID uuid.UUID) error {
	m.removeSkillCalled = true
	list := m.collectionSkills[collectionID]
	for i, sid := range list {
		if sid == skillID {
			m.collectionSkills[collectionID] = append(list[:i], list[i+1:]...)
			break
		}
	}
	return nil
}

type mockAgentRepositoryForSkillService struct {
	agents map[uuid.UUID]*model.Agent
}

func newMockAgentRepositoryForSkillService() *mockAgentRepositoryForSkillService {
	return &mockAgentRepositoryForSkillService{
		agents: make(map[uuid.UUID]*model.Agent),
	}
}

func (m *mockAgentRepositoryForSkillService) Create(ctx context.Context, agent *model.Agent) error {
	m.agents[agent.ID] = agent
	return nil
}

func (m *mockAgentRepositoryForSkillService) GetByID(ctx context.Context, id uuid.UUID) (*model.Agent, error) {
	a, ok := m.agents[id]
	if !ok {
		return nil, errors.New("agent not found")
	}
	return a, nil
}

func (m *mockAgentRepositoryForSkillService) List(ctx context.Context) ([]*model.Agent, error) {
	return nil, nil
}

func (m *mockAgentRepositoryForSkillService) Update(ctx context.Context, agent *model.Agent) error {
	return nil
}

func (m *mockAgentRepositoryForSkillService) Delete(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (m *mockAgentRepositoryForSkillService) GetByName(ctx context.Context, name string) (*model.Agent, error) {
	return nil, nil
}

func (m *mockAgentRepositoryForSkillService) AssignToCommunity(ctx context.Context, agentID uuid.UUID, communityID uuid.UUID) error {
	return nil
}

func (m *mockAgentRepositoryForSkillService) UnassignFromCommunity(ctx context.Context, agentID uuid.UUID, communityID uuid.UUID) error {
	return nil
}

func (m *mockAgentRepositoryForSkillService) UpdateStatus(ctx context.Context, agentID uuid.UUID, status model.AgentStatus) (bool, error) {
	return false, nil
}

func (m *mockAgentRepositoryForSkillService) UpsertRegistration(ctx context.Context, agentID uuid.UUID, communityID uuid.UUID, card *agentcard.AgentCard) error {
	return nil
}

func (m *mockAgentRepositoryForSkillService) GetRegistration(ctx context.Context, agentID uuid.UUID, communityID uuid.UUID) (*agentcard.AgentCard, time.Time, error) {
	return nil, time.Time{}, nil
}

func (m *mockAgentRepositoryForSkillService) GetActiveRegistrationsByCommunity(ctx context.Context, communityID uuid.UUID) ([]*agentcard.AgentCard, time.Time, error) {
	return nil, time.Time{}, nil
}

func (m *mockAgentRepositoryForSkillService) PruneStaleRegistrations(ctx context.Context, threshold time.Duration) ([]agentcard.AgentCommunityRef, error) {
	return nil, nil
}

func (m *mockAgentRepositoryForSkillService) DeleteRegistration(ctx context.Context, agentID uuid.UUID, communityID uuid.UUID) error {
	return nil
}

func TestSkillService_PatchStatus(t *testing.T) {
	skillRepo := newMockSkillRepository()
	agentRepo := newMockAgentRepositoryForSkillService()
	svc := NewSkillService(skillRepo, agentRepo)
	ctx := context.Background()

	sID := uuid.New()
	skill := &model.Skill{
		ID:     sID,
		Name:   "test-skill",
		Status: model.SkillStatusActive,
	}
	_ = skillRepo.Create(ctx, skill)

	// Valid Patch
	updated, err := svc.PatchStatus(ctx, sID, model.SkillStatusSuspended)
	assert.NoError(t, err)
	assert.Equal(t, model.SkillStatusSuspended, updated.Status)
	assert.True(t, skillRepo.updateCalled)

	// Invalid Status
	_, err = svc.PatchStatus(ctx, sID, model.SkillStatus("invalid-status"))
	assert.Error(t, err)
}

func TestSkillService_CollectionMembership(t *testing.T) {
	skillRepo := newMockSkillRepository()
	agentRepo := newMockAgentRepositoryForSkillService()
	svc := NewSkillService(skillRepo, agentRepo)
	ctx := context.Background()

	colID := uuid.New()
	skillID := uuid.New()

	// Add Skill to Collection
	err := svc.AddSkillToCollection(ctx, colID, skillID)
	assert.NoError(t, err)
	assert.True(t, skillRepo.addSkillCalled)
	assert.Contains(t, skillRepo.collectionSkills[colID], skillID)

	// Remove Skill from Collection
	err = svc.RemoveSkillFromCollection(ctx, colID, skillID)
	assert.NoError(t, err)
	assert.True(t, skillRepo.removeSkillCalled)
	assert.NotContains(t, skillRepo.collectionSkills[colID], skillID)
}

func TestSkillService_ResolveAgentSkills(t *testing.T) {
	skillRepo := newMockSkillRepository()
	agentRepo := newMockAgentRepositoryForSkillService()
	svc := NewSkillService(skillRepo, agentRepo)
	ctx := context.Background()

	agentID := uuid.New()
	colID := uuid.New()
	skillID1 := uuid.New()
	skillID2 := uuid.New()

	agent := &model.Agent{
		ID:               agentID,
		Name:             "test-agent",
		SkillCollections: []uuid.UUID{colID},
		Skills:           []uuid.UUID{skillID2},
	}
	_ = agentRepo.Create(ctx, agent)

	s1 := &model.Skill{ID: skillID1, Name: "skill-1", Status: model.SkillStatusActive}
	s2 := &model.Skill{ID: skillID2, Name: "skill-2", Status: model.SkillStatusActive}
	_ = skillRepo.Create(ctx, s1)
	_ = skillRepo.Create(ctx, s2)

	_ = skillRepo.AddSkillToCollection(ctx, colID, skillID1)

	// Resolve Skills
	resolved, err := svc.ResolveAgentSkills(ctx, agentID)
	assert.NoError(t, err)
	assert.Len(t, resolved, 2)
	assert.Equal(t, skillID1, resolved[0].ID)
	assert.Equal(t, "collection", resolved[0].Source)
	assert.Equal(t, skillID2, resolved[1].ID)
	assert.Equal(t, "individual", resolved[1].Source)
}
