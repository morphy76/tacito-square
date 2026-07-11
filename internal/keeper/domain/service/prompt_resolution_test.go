package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/morphy76/tacito-square/internal/keeper/domain/model"
	"github.com/stretchr/testify/assert"
)

type mockPromptRepository struct {
	templates   map[uuid.UUID]*model.PromptTemplate
	collections map[uuid.UUID]*model.PromptCollection
}

func (m *mockPromptRepository) GetTemplateByID(ctx context.Context, id uuid.UUID) (*model.PromptTemplate, error) {
	t, ok := m.templates[id]
	if !ok {
		return nil, errors.New("not found")
	}
	return t, nil
}

func (m *mockPromptRepository) ResolveCollectionPrompts(ctx context.Context, collectionID uuid.UUID) ([]*model.PromptTemplate, error) {
	c, ok := m.collections[collectionID]
	if !ok {
		return nil, errors.New("not found")
	}
	var res []*model.PromptTemplate
	for _, tid := range c.Templates {
		t, err := m.GetTemplateByID(ctx, tid)
		if err == nil {
			res = append(res, t)
		}
	}
	return res, nil
}

func TestResolvePrompts_UnionWithoutDuplication(t *testing.T) {
	ctx := context.Background()

	p1 := &model.PromptTemplate{ID: uuid.New(), TenantID: "t1", Name: "P1", Status: model.PromptStatusActive}
	p2 := &model.PromptTemplate{ID: uuid.New(), TenantID: "t1", Name: "P2", Status: model.PromptStatusActive}

	col1 := &model.PromptCollection{
		ID:        uuid.New(),
		TenantID:  "t1",
		Name:      "Col1",
		Templates: []uuid.UUID{p1.ID, p2.ID},
	}

	repo := &mockPromptRepository{
		templates: map[uuid.UUID]*model.PromptTemplate{
			p1.ID: p1,
			p2.ID: p2,
		},
		collections: map[uuid.UUID]*model.PromptCollection{
			col1.ID: col1,
		},
	}

	agent := &model.Agent{
		ID:                uuid.New(),
		TenantID:          "t1",
		Name:              "Test Agent",
		Prompts:           []uuid.UUID{p1.ID}, // p1 attached directly
		PromptCollections: []uuid.UUID{col1.ID}, // col1 attached, containing p1 and p2
		Status:            model.AgentStatusDefined,
		ShortTermMemory:   model.ShortTermMemoryConfig{TTLSeconds: 3600},
	}

	resolved, err := ResolveEffectivePrompts(ctx, agent, repo)
	assert.NoError(t, err)

	// Result should have P1 and P2, without duplicating P1
	assert.Len(t, resolved, 2)
	assert.Equal(t, p1.ID, resolved[0].ID)
	assert.Equal(t, p2.ID, resolved[1].ID)
}

func TestResolvePrompts_CollectionOrder(t *testing.T) {
	ctx := context.Background()

	p1 := &model.PromptTemplate{ID: uuid.New(), TenantID: "t1", Name: "P1", Status: model.PromptStatusActive}
	p2 := &model.PromptTemplate{ID: uuid.New(), TenantID: "t1", Name: "P2", Status: model.PromptStatusActive}
	p3 := &model.PromptTemplate{ID: uuid.New(), TenantID: "t1", Name: "P3", Status: model.PromptStatusActive}

	col1 := &model.PromptCollection{
		ID:        uuid.New(),
		TenantID:  "t1",
		Name:      "Col1",
		Templates: []uuid.UUID{p2.ID, p1.ID}, // definition order inside collection: P2 first, then P1
	}

	repo := &mockPromptRepository{
		templates: map[uuid.UUID]*model.PromptTemplate{
			p1.ID: p1,
			p2.ID: p2,
			p3.ID: p3,
		},
		collections: map[uuid.UUID]*model.PromptCollection{
			col1.ID: col1,
		},
	}

	agent := &model.Agent{
		ID:                uuid.New(),
		TenantID:          "t1",
		Name:              "Test Agent",
		Prompts:           []uuid.UUID{p3.ID},
		PromptCollections: []uuid.UUID{col1.ID},
		Status:            model.AgentStatusDefined,
		ShortTermMemory:   model.ShortTermMemoryConfig{TTLSeconds: 3600},
	}

	resolved, err := ResolveEffectivePrompts(ctx, agent, repo)
	assert.NoError(t, err)

	// Order: col1 templates in definition order (P2, P1), then directly attached templates (P3)
	assert.Len(t, resolved, 3)
	assert.Equal(t, p2.ID, resolved[0].ID)
	assert.Equal(t, p1.ID, resolved[1].ID)
	assert.Equal(t, p3.ID, resolved[2].ID)
}

func TestResolvePrompts_SkipsDraftAndArchived(t *testing.T) {
	ctx := context.Background()

	activeP := &model.PromptTemplate{ID: uuid.New(), TenantID: "t1", Name: "Active", Status: model.PromptStatusActive}
	draftP := &model.PromptTemplate{ID: uuid.New(), TenantID: "t1", Name: "Draft", Status: model.PromptStatusDraft}
	archivedP := &model.PromptTemplate{ID: uuid.New(), TenantID: "t1", Name: "Archived", Status: model.PromptStatusArchived}

	repo := &mockPromptRepository{
		templates: map[uuid.UUID]*model.PromptTemplate{
			activeP.ID:   activeP,
			draftP.ID:    draftP,
			archivedP.ID: archivedP,
		},
	}

	agent := &model.Agent{
		ID:              uuid.New(),
		TenantID:        "t1",
		Name:            "Test Agent",
		Prompts:         []uuid.UUID{activeP.ID, draftP.ID, archivedP.ID},
		Status:          model.AgentStatusDefined,
		ShortTermMemory: model.ShortTermMemoryConfig{TTLSeconds: 3600},
	}

	resolved, err := ResolveEffectivePrompts(ctx, agent, repo)
	assert.NoError(t, err)

	// Should only resolve the active prompt
	assert.Len(t, resolved, 1)
	assert.Equal(t, activeP.ID, resolved[0].ID)
}
