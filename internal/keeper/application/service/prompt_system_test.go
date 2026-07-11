package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/morphy76/tacito-square/internal/keeper/domain/model"
	"github.com/stretchr/testify/assert"
)

type mockPromptRepositoryForPromptService struct {
	updateCalled               bool
	deleteCalled               bool
	addPromptCalled            bool
	removePromptCalled         bool
	addPromptCollectionID      uuid.UUID
	addPromptTemplateID        uuid.UUID
	removePromptCollectionID   uuid.UUID
	removePromptTemplateID     uuid.UUID
}

func (m *mockPromptRepositoryForPromptService) CreateTemplate(ctx context.Context, template *model.PromptTemplate) error {
	return nil
}
func (m *mockPromptRepositoryForPromptService) GetTemplateByID(ctx context.Context, id uuid.UUID) (*model.PromptTemplate, error) {
	return nil, nil
}
func (m *mockPromptRepositoryForPromptService) ListTemplates(ctx context.Context) ([]*model.PromptTemplate, error) {
	return nil, nil
}
func (m *mockPromptRepositoryForPromptService) UpdateTemplate(ctx context.Context, template *model.PromptTemplate) error {
	m.updateCalled = true
	return nil
}
func (m *mockPromptRepositoryForPromptService) DeleteTemplate(ctx context.Context, id uuid.UUID) error {
	m.deleteCalled = true
	return nil
}
func (m *mockPromptRepositoryForPromptService) CreateCollection(ctx context.Context, collection *model.PromptCollection) error {
	return nil
}
func (m *mockPromptRepositoryForPromptService) GetCollectionByID(ctx context.Context, id uuid.UUID) (*model.PromptCollection, error) {
	return nil, nil
}
func (m *mockPromptRepositoryForPromptService) ListCollections(ctx context.Context) ([]*model.PromptCollection, error) {
	return nil, nil
}
func (m *mockPromptRepositoryForPromptService) UpdateCollection(ctx context.Context, collection *model.PromptCollection) error {
	return nil
}
func (m *mockPromptRepositoryForPromptService) DeleteCollection(ctx context.Context, id uuid.UUID) error {
	return nil
}
func (m *mockPromptRepositoryForPromptService) ResolveCollectionPrompts(ctx context.Context, collectionID uuid.UUID) ([]*model.PromptTemplate, error) {
	return nil, nil
}
func (m *mockPromptRepositoryForPromptService) CreateVersion(ctx context.Context, version *model.PromptVersion) error {
	return nil
}
func (m *mockPromptRepositoryForPromptService) GetLatestVersion(ctx context.Context, promptID uuid.UUID) (*model.PromptVersion, error) {
	return nil, nil
}
func (m *mockPromptRepositoryForPromptService) AddPromptToCollection(ctx context.Context, collectionID uuid.UUID, promptID uuid.UUID) error {
	m.addPromptCalled = true
	m.addPromptCollectionID = collectionID
	m.addPromptTemplateID = promptID
	return nil
}
func (m *mockPromptRepositoryForPromptService) RemovePromptFromCollection(ctx context.Context, collectionID uuid.UUID, promptID uuid.UUID) error {
	m.removePromptCalled = true
	m.removePromptCollectionID = collectionID
	m.removePromptTemplateID = promptID
	return nil
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
	repo := &mockPromptRepositoryForPromptService{}
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

func TestPromptService_CollectionMembership(t *testing.T) {
	repo := &mockPromptRepositoryForPromptService{}
	svc := NewPromptService(repo)
	ctx := context.Background()

	colID := uuid.New()
	ptID := uuid.New()

	// 1. Add prompt to collection
	err := svc.AddPromptToCollection(ctx, colID, ptID)
	assert.NoError(t, err)
	assert.True(t, repo.addPromptCalled)
	assert.Equal(t, colID, repo.addPromptCollectionID)
	assert.Equal(t, ptID, repo.addPromptTemplateID)

	// 2. Remove prompt from collection
	err = svc.RemovePromptFromCollection(ctx, colID, ptID)
	assert.NoError(t, err)
	assert.True(t, repo.removePromptCalled)
	assert.Equal(t, colID, repo.removePromptCollectionID)
	assert.Equal(t, ptID, repo.removePromptTemplateID)
}
