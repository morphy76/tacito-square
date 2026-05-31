package qdrant

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/morphy76/tacito-square/internal/agent/domain/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestQdrantLTMAdapter(t *testing.T) {
	ctx := context.Background()

	// Spin up Qdrant container
	req := testcontainers.ContainerRequest{
		Image:        "qdrant/qdrant:latest",
		ExposedPorts: []string{"6333/tcp", "6334/tcp"},
		WaitingFor:   wait.ForListeningPort("6334/tcp"),
	}
	qdrantContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Skip("Docker is not available, skipping Qdrant integration test:", err)
	}
	defer func() {
		_ = qdrantContainer.Terminate(ctx)
	}()

	host, err := qdrantContainer.Host(ctx)
	require.NoError(t, err)
	grpcPort, err := qdrantContainer.MappedPort(ctx, "6334")
	require.NoError(t, err)

	qdrantURL := fmt.Sprintf("%s:%s", host, grpcPort.Port())
	collectionName := "ts_agent_memories"
	vectorDim := uint64(3)

	// Instantiate the adapter (should fail to compile right now)
	adapter, err := NewQdrantLTMAdapter(qdrantURL, collectionName, vectorDim)
	require.NoError(t, err)
	defer adapter.Close()

	t.Run("Save, Search and Delete - Multi-tenancy and Permission Isolation", func(t *testing.T) {
		tenantA := "tenant-1"
		tenantB := "tenant-2"
		agent1 := "agent-1"
		agent2 := "agent-2"
		community1 := "community-1"
		community2 := "community-2"

		// 1. Save memories for Tenant A, Agent 1, Community 1
		memories := []model.LTMEntry{
			{
				ID:        "00000000-0000-0000-0000-000000000001",
				Content:   "Private fact for Agent 1",
				Embedding: []float32{0.9, 0.1, 0.0},
				Type:      model.EntryTypeFact,
				Source:    "test",
				Timestamp: time.Now().UTC(),
				Metadata:  map[string]string{"visibility": "private"},
			},
			{
				ID:        "00000000-0000-0000-0000-000000000002",
				Content:   "Shared fact for Community 1",
				Embedding: []float32{0.8, 0.2, 0.0},
				Type:      model.EntryTypeConversation,
				Source:    "test",
				Timestamp: time.Now().UTC(),
				Metadata:  map[string]string{"visibility": "community", "community_id": community1},
			},
			{
				ID:        "00000000-0000-0000-0000-000000000003",
				Content:   "Tenant shared fact for Tenant A",
				Embedding: []float32{0.7, 0.3, 0.0},
				Type:      model.EntryTypeDocument,
				Source:    "test",
				Timestamp: time.Now().UTC(),
				Metadata:  map[string]string{"visibility": "tenant"},
			},
		}

		err = adapter.Save(ctx, tenantA, agent1, memories)
		assert.NoError(t, err)

		// 2. Save memory for Tenant A, Agent 2, Community 2 (Private)
		agent2Mem := []model.LTMEntry{
			{
				ID:        "00000000-0000-0000-0000-000000000004",
				Content:   "Private fact for Agent 2",
				Embedding: []float32{0.85, 0.15, 0.0},
				Type:      model.EntryTypeFact,
				Source:    "test",
				Timestamp: time.Now().UTC(),
				Metadata:  map[string]string{"visibility": "private"},
			},
		}
		err = adapter.Save(ctx, tenantA, agent2, agent2Mem)
		assert.NoError(t, err)

		// 3. Search as Tenant B - should return nothing (Strict Multi-Tenancy Boundary)
		searchVec := []float32{1.0, 0.0, 0.0}
		resB, err := adapter.Search(ctx, tenantB, agent1, searchVec, model.LTMFilter{
			CommunityID: community1,
		}, 10, 0.5)
		assert.NoError(t, err)
		assert.Empty(t, resB)

		// 4. Search as Tenant A, Agent 1, Community 1
		// Should retrieve: Agent 1 Private, Community 1 Shared, and Tenant Shared.
		// Should NOT retrieve: Agent 2 Private.
		resA1, err := adapter.Search(ctx, tenantA, agent1, searchVec, model.LTMFilter{
			CommunityID: community1,
		}, 10, 0.5)
		assert.NoError(t, err)
		require.Len(t, resA1, 3)

		contents := make([]string, len(resA1))
		for i, r := range resA1 {
			contents[i] = r.Content
			assert.NotEmpty(t, r.Score)
		}
		assert.Contains(t, contents, "Private fact for Agent 1")
		assert.Contains(t, contents, "Shared fact for Community 1")
		assert.Contains(t, contents, "Tenant shared fact for Tenant A")
		assert.NotContains(t, contents, "Private fact for Agent 2")

		// 5. Search as Tenant A, Agent 2, Community 2
		// Should retrieve: Agent 2 Private, and Tenant Shared (no community 1 or Agent 1 private)
		resA2, err := adapter.Search(ctx, tenantA, agent2, searchVec, model.LTMFilter{
			CommunityID: community2,
		}, 10, 0.5)
		assert.NoError(t, err)
		require.Len(t, resA2, 2)

		contentsA2 := make([]string, len(resA2))
		for i, r := range resA2 {
			contentsA2[i] = r.Content
		}
		assert.Contains(t, contentsA2, "Private fact for Agent 2")
		assert.Contains(t, contentsA2, "Tenant shared fact for Tenant A")

		// 6. Test Score Threshold pruning (e.g. threshold = 0.95 should drop some)
		resThreshold, err := adapter.Search(ctx, tenantA, agent1, searchVec, model.LTMFilter{
			CommunityID: community1,
		}, 10, 0.995)
		assert.NoError(t, err)
		assert.Len(t, resThreshold, 0) // Should drop everything as similarity score is slightly lower than 0.95

		// 7. Test Delete
		err = adapter.Delete(ctx, tenantA, agent1, model.LTMFilter{
			Types: []model.LTMEntryType{model.EntryTypeFact},
		})
		assert.NoError(t, err)

		// After deleting Fact entries, the Private fact for Agent 1 (type: fact) should be gone,
		// but the other two should remain.
		resAfterDelete, err := adapter.Search(ctx, tenantA, agent1, searchVec, model.LTMFilter{
			CommunityID: community1,
		}, 10, 0.5)
		assert.NoError(t, err)
		require.Len(t, resAfterDelete, 2)
		for _, r := range resAfterDelete {
			assert.NotEqual(t, "Private fact for Agent 1", r.Content)
		}
	})

	t.Run("Ping health check", func(t *testing.T) {
		err := adapter.Ping(ctx)
		assert.NoError(t, err)
	})
}
