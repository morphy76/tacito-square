package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/morphy76/tacito-square/internal/keeper/domain/model"
	"github.com/stretchr/testify/assert"
)

type mockSkillRepository struct {
	skills      map[uuid.UUID]*model.Skill
	collections map[uuid.UUID]*model.SkillCollection
}

func (m *mockSkillRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Skill, error) {
	s, ok := m.skills[id]
	if !ok {
		return nil, errors.New("not found")
	}
	return s, nil
}

func (m *mockSkillRepository) ResolveCollectionSkills(ctx context.Context, collectionID uuid.UUID) ([]*model.Skill, error) {
	c, ok := m.collections[collectionID]
	if !ok {
		return nil, errors.New("not found")
	}
	var res []*model.Skill
	for _, sid := range c.Skills {
		s, err := m.GetByID(ctx, sid)
		if err == nil {
			res = append(res, s)
		}
	}
	return res, nil
}

func TestResolveSkills_UnionWithoutDuplication(t *testing.T) {
	ctx := context.Background()

	s1 := &model.Skill{ID: uuid.New(), TenantID: "t1", Name: "S1", Status: model.SkillStatusActive}
	s2 := &model.Skill{ID: uuid.New(), TenantID: "t1", Name: "S2", Status: model.SkillStatusActive}

	col1 := &model.SkillCollection{
		ID:       uuid.New(),
		TenantID: "t1",
		Name:     "Col1",
		Skills:   []uuid.UUID{s1.ID, s2.ID},
	}

	repo := &mockSkillRepository{
		skills: map[uuid.UUID]*model.Skill{
			s1.ID: s1,
			s2.ID: s2,
		},
		collections: map[uuid.UUID]*model.SkillCollection{
			col1.ID: col1,
		},
	}

	agent := &model.Agent{
		ID:               uuid.New(),
		TenantID:         "t1",
		Name:             "Test Agent",
		Skills:           []uuid.UUID{s1.ID}, // s1 attached directly
		SkillCollections: []uuid.UUID{col1.ID}, // col1 attached, containing s1 and s2
		Status:           model.AgentStatusDefined,
		ShortTermMemory:  model.ShortTermMemoryConfig{TTLSeconds: 3600},
	}

	resolved, err := ResolveAgentSkills(ctx, agent, repo)
	assert.NoError(t, err)

	// Result should have S1 and S2, without duplicating S1.
	// Since col1 is processed first, S1's source should be "collection".
	assert.Len(t, resolved, 2)
	assert.Equal(t, s1.ID, resolved[0].ID)
	assert.Equal(t, "collection", resolved[0].Source)
	assert.Equal(t, &col1.ID, resolved[0].CollectionID)
	assert.Equal(t, s2.ID, resolved[1].ID)
	assert.Equal(t, "collection", resolved[1].Source)
	assert.Equal(t, &col1.ID, resolved[1].CollectionID)
}

func TestResolveSkills_CollectionOrder(t *testing.T) {
	ctx := context.Background()

	s1 := &model.Skill{ID: uuid.New(), TenantID: "t1", Name: "S1", Status: model.SkillStatusActive}
	s2 := &model.Skill{ID: uuid.New(), TenantID: "t1", Name: "S2", Status: model.SkillStatusActive}
	s3 := &model.Skill{ID: uuid.New(), TenantID: "t1", Name: "S3", Status: model.SkillStatusActive}

	col1 := &model.SkillCollection{
		ID:       uuid.New(),
		TenantID: "t1",
		Name:     "Col1",
		Skills:   []uuid.UUID{s2.ID, s1.ID}, // definition order inside collection: S2 first, then S1
	}

	repo := &mockSkillRepository{
		skills: map[uuid.UUID]*model.Skill{
			s1.ID: s1,
			s2.ID: s2,
			s3.ID: s3,
		},
		collections: map[uuid.UUID]*model.SkillCollection{
			col1.ID: col1,
		},
	}

	agent := &model.Agent{
		ID:               uuid.New(),
		TenantID:         "t1",
		Name:             "Test Agent",
		Skills:           []uuid.UUID{s3.ID},
		SkillCollections: []uuid.UUID{col1.ID},
		Status:           model.AgentStatusDefined,
		ShortTermMemory:  model.ShortTermMemoryConfig{TTLSeconds: 3600},
	}

	resolved, err := ResolveAgentSkills(ctx, agent, repo)
	assert.NoError(t, err)

	// Order: col1 skills in definition order (S2, S1), then directly attached skills (S3)
	assert.Len(t, resolved, 3)
	assert.Equal(t, s2.ID, resolved[0].ID)
	assert.Equal(t, s1.ID, resolved[1].ID)
	assert.Equal(t, s3.ID, resolved[2].ID)
}

func TestResolveSkills_SkipsSuspended(t *testing.T) {
	ctx := context.Background()

	activeS := &model.Skill{ID: uuid.New(), TenantID: "t1", Name: "Active", Status: model.SkillStatusActive}
	suspendedS := &model.Skill{ID: uuid.New(), TenantID: "t1", Name: "Suspended", Status: model.SkillStatusSuspended}

	repo := &mockSkillRepository{
		skills: map[uuid.UUID]*model.Skill{
			activeS.ID:    activeS,
			suspendedS.ID: suspendedS,
		},
	}

	agent := &model.Agent{
		ID:              uuid.New(),
		TenantID:        "t1",
		Name:            "Test Agent",
		Skills:          []uuid.UUID{activeS.ID, suspendedS.ID},
		Status:          model.AgentStatusDefined,
		ShortTermMemory: model.ShortTermMemoryConfig{TTLSeconds: 3600},
	}

	resolved, err := ResolveAgentSkills(ctx, agent, repo)
	assert.NoError(t, err)

	// Should only resolve the active skill
	assert.Len(t, resolved, 1)
	assert.Equal(t, activeS.ID, resolved[0].ID)
}

func TestResolveSkills_SkipsInactive(t *testing.T) {
	ctx := context.Background()

	activeS := &model.Skill{ID: uuid.New(), TenantID: "t1", Name: "Active", Status: model.SkillStatusActive}
	inactiveS := &model.Skill{ID: uuid.New(), TenantID: "t1", Name: "Inactive", Status: model.SkillStatusInactive}

	repo := &mockSkillRepository{
		skills: map[uuid.UUID]*model.Skill{
			activeS.ID:   activeS,
			inactiveS.ID: inactiveS,
		},
	}

	agent := &model.Agent{
		ID:              uuid.New(),
		TenantID:        "t1",
		Name:            "Test Agent",
		Skills:          []uuid.UUID{activeS.ID, inactiveS.ID},
		Status:          model.AgentStatusDefined,
		ShortTermMemory: model.ShortTermMemoryConfig{TTLSeconds: 3600},
	}

	resolved, err := ResolveAgentSkills(ctx, agent, repo)
	assert.NoError(t, err)

	// Should only resolve the active skill
	assert.Len(t, resolved, 1)
	assert.Equal(t, activeS.ID, resolved[0].ID)
}

func TestResolveSkills_MultipleCollections(t *testing.T) {
	ctx := context.Background()

	s1 := &model.Skill{ID: uuid.New(), TenantID: "t1", Name: "S1", Status: model.SkillStatusActive}

	col1 := &model.SkillCollection{
		ID:       uuid.New(),
		TenantID: "t1",
		Name:     "Col1",
		Skills:   []uuid.UUID{s1.ID},
	}
	col2 := &model.SkillCollection{
		ID:       uuid.New(),
		TenantID: "t1",
		Name:     "Col2",
		Skills:   []uuid.UUID{s1.ID},
	}

	repo := &mockSkillRepository{
		skills: map[uuid.UUID]*model.Skill{
			s1.ID: s1,
		},
		collections: map[uuid.UUID]*model.SkillCollection{
			col1.ID: col1,
			col2.ID: col2,
		},
	}

	agent := &model.Agent{
		ID:               uuid.New(),
		TenantID:         "t1",
		Name:             "Test Agent",
		SkillCollections: []uuid.UUID{col1.ID, col2.ID},
		Status:           model.AgentStatusDefined,
		ShortTermMemory:  model.ShortTermMemoryConfig{TTLSeconds: 3600},
	}

	resolved, err := ResolveAgentSkills(ctx, agent, repo)
	assert.NoError(t, err)

	// Should only appear once, associated with col1 (first processed)
	assert.Len(t, resolved, 1)
	assert.Equal(t, s1.ID, resolved[0].ID)
	assert.Equal(t, "collection", resolved[0].Source)
	assert.Equal(t, &col1.ID, resolved[0].CollectionID)
}

func TestResolveSkills_EmptyAgent(t *testing.T) {
	ctx := context.Background()
	repo := &mockSkillRepository{}

	agent := &model.Agent{
		ID:              uuid.New(),
		TenantID:        "t1",
		Name:            "Test Agent",
		Status:          model.AgentStatusDefined,
		ShortTermMemory: model.ShortTermMemoryConfig{TTLSeconds: 3600},
	}

	resolved, err := ResolveAgentSkills(ctx, agent, repo)
	assert.NoError(t, err)
	assert.Empty(t, resolved)
}
