package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/morphy76/tacito-square/internal/keeper/domain/model"
	cacheport "github.com/morphy76/tacito-square/internal/shared/ports/outbound"
	"github.com/morphy76/tacito-square/internal/shared/tenant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestLLMBindingService_GetByID_CacheHit(t *testing.T) {
	repo := new(mockLLMBindingRepository)
	cache := new(mockCache)
	svc := NewLLMBindingService(repo, cache) // Fails to compile under old constructor

	ten, _ := tenant.New("test-tenant.com", "")
	ctx := tenant.ContextWithTenant(context.Background(), ten)
	id := uuid.New()
	expected := &model.LLMBinding{ID: id, TenantID: "test-tenant.com", Name: "cached-binding"}

	cache.On("Get", mock.Anything, "test-tenant.com:llm-bindings:"+id.String(), mock.Anything).
		Return(nil).
		Run(func(args mock.Arguments) {
			dest := args.Get(2).(*model.LLMBinding)
			*dest = *expected
		})

	result, err := svc.GetByID(ctx, id)
	assert.NoError(t, err)
	assert.Equal(t, expected.ID, result.ID)
	assert.Equal(t, expected.Name, result.Name)

	repo.AssertNotCalled(t, "GetByID", mock.Anything, mock.Anything)
}

func TestLLMBindingService_GetByID_CacheMiss(t *testing.T) {
	repo := new(mockLLMBindingRepository)
	cache := new(mockCache)
	svc := NewLLMBindingService(repo, cache)

	ten, _ := tenant.New("test-tenant.com", "")
	ctx := tenant.ContextWithTenant(context.Background(), ten)
	id := uuid.New()
	expected := &model.LLMBinding{ID: id, TenantID: "test-tenant.com", Name: "db-binding"}

	cache.On("Get", mock.Anything, "test-tenant.com:llm-bindings:"+id.String(), mock.Anything).
		Return(cacheport.ErrCacheMiss)
	repo.On("GetByID", mock.Anything, id).Return(expected, nil)
	cache.On("Set", mock.Anything, "test-tenant.com:llm-bindings:"+id.String(), expected, 300*time.Second).
		Return(nil)

	result, err := svc.GetByID(ctx, id)
	assert.NoError(t, err)
	assert.Equal(t, expected.ID, result.ID)
	assert.Equal(t, expected.Name, result.Name)

	repo.AssertExpectations(t)
	cache.AssertExpectations(t)
}

func TestLLMBindingService_Mutations_Invalidate(t *testing.T) {
	repo := new(mockLLMBindingRepository)
	cache := new(mockCache)
	svc := NewLLMBindingService(repo, cache)

	ten, _ := tenant.New("test-tenant.com", "")
	ctx := tenant.ContextWithTenant(context.Background(), ten)
	id := uuid.New()
	binding := &model.LLMBinding{ID: id, TenantID: "test-tenant.com", Name: "db-binding"}

	t.Run("Update invalidates cache", func(t *testing.T) {
		repo.On("Update", mock.Anything, binding).Return(nil)
		cache.On("Invalidate", mock.Anything, "test-tenant.com:llm-bindings:"+id.String()).Return(nil)

		err := svc.Update(ctx, binding)
		assert.NoError(t, err)
		repo.AssertExpectations(t)
		cache.AssertExpectations(t)
	})

	t.Run("Delete invalidates cache", func(t *testing.T) {
		repo.On("Delete", mock.Anything, id).Return(nil)
		cache.On("Invalidate", mock.Anything, "test-tenant.com:llm-bindings:"+id.String()).Return(nil)

		err := svc.Delete(ctx, id)
		assert.NoError(t, err)
		repo.AssertExpectations(t)
		cache.AssertExpectations(t)
	})
}
