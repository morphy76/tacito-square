package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/morphy76/tacito-square/internal/keeper/domain/model"
	"github.com/stretchr/testify/assert"
)

type mockPromptRepository struct {
	updateCalled bool
	deleteCalled bool
}

func (m *mockPromptRepository) CreateTemplate(ctx context.Context, template *model.PromptTemplate) error {
	return nil
}
func (m *mockPromptRepository) GetTemplateByID(ctx context.Context, id uuid.UUID) (*model.PromptTemplate, error) {
	return nil, nil
}
func (m *mockPromptRepository) ListTemplates(ctx context.Context) ([]*model.PromptTemplate, error) {
	return nil, nil
}
func (m *mockPromptRepository) UpdateTemplate(ctx context.Context, template *model.PromptTemplate) error {
	m.updateCalled = true
	return nil
}
func (m *mockPromptRepository) DeleteTemplate(ctx context.Context, id uuid.UUID) error {
	m.deleteCalled = true
	return nil
}
func (m *mockPromptRepository) CreateCollection(ctx context.Context, collection *model.PromptCollection) error {
	return nil
}
func (m *mockPromptRepository) GetCollectionByID(ctx context.Context, id uuid.UUID) (*model.PromptCollection, error) {
	return nil, nil
}
func (m *mockPromptRepository) ListCollections(ctx context.Context) ([]*model.PromptCollection, error) {
	return nil, nil
}
func (m *mockPromptRepository) UpdateCollection(ctx context.Context, collection *model.PromptCollection) error {
	return nil
}
func (m *mockPromptRepository) DeleteCollection(ctx context.Context, id uuid.UUID) error {
	return nil
}
func (m *mockPromptRepository) ResolveCollectionPrompts(ctx context.Context, collectionID uuid.UUID) ([]*model.PromptTemplate, error) {
	return nil, nil
}

func TestPromptSystemLock_IsSystemLocked(t *testing.T) {
	systemID1 := uuid.MustParse("ffffffff-0000-0000-0000-000000000001")
	systemID2 := uuid.MustParse("ffffffff-1234-5678-9abc-def012345678")
	normalID := uuid.MustParse("00000000-1234-5678-9abc-def012345678")

	assert.True(t, model.IsSystemLocked(systemID1))
	assert.True(t, model.IsSystemLocked(systemID2))
	assert.False(t, model.IsSystemLocked(normalID))
}

func TestPromptService_Immutability(t *testing.T) {
	repo := &mockPromptRepository{}
	svc := NewPromptService(repo)
	ctx := context.Background()

	systemID := uuid.MustParse("ffffffff-0000-0000-0000-000000000001")
	normalID := uuid.MustParse("12345678-0000-0000-0000-000000000002")

	// 1. Attempt update on system prompt template -> should fail
	ptSystem := &model.PromptTemplate{
		ID:      systemID,
		Name:    "system-prompt",
		Content: "Locked instructions",
		Status:  model.PromptStatusActive,
	}
	err := svc.UpdateTemplate(ctx, ptSystem)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot update system-locked prompt template")
	assert.False(t, repo.updateCalled)

	// 2. Attempt update on normal template -> should succeed
	ptNormal := &model.PromptTemplate{
		ID:      normalID,
		Name:    "normal-prompt",
		Content: "Editable instructions",
		Status:  model.PromptStatusActive,
	}
	err = svc.UpdateTemplate(ctx, ptNormal)
	assert.NoError(t, err)
	assert.True(t, repo.updateCalled)

	// 3. Attempt delete on system prompt template -> should fail
	err = svc.DeleteTemplate(ctx, systemID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot delete system-locked prompt template")
	assert.False(t, repo.deleteCalled)

	// 4. Attempt delete on normal template -> should succeed
	err = svc.DeleteTemplate(ctx, normalID)
	assert.NoError(t, err)
	assert.True(t, repo.deleteCalled)
}
