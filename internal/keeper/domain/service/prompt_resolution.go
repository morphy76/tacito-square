package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/morphy76/tacito-square/internal/keeper/domain/model"
	"github.com/rs/zerolog/log"
)

// PromptRepository defines the outbound port interface required by the prompt resolution domain service.
// This local definition adheres to hexagonal architecture guidelines by preventing the domain layer
// from importing application-layer packages.
type PromptRepository interface {
	GetTemplateByID(ctx context.Context, id uuid.UUID) (*model.PromptTemplate, error)
	ResolveCollectionPrompts(ctx context.Context, collectionID uuid.UUID) ([]*model.PromptTemplate, error)
}

// ResolveEffectivePrompts resolves all active prompts associated with an agent, preserving order and removing duplicates.
func ResolveEffectivePrompts(ctx context.Context, agent *model.Agent, promptRepo PromptRepository) ([]*model.ResolvedAgentPrompt, error) {
	var resolved []*model.ResolvedAgentPrompt
	seen := make(map[uuid.UUID]bool)

	// 1. Process prompt collections in agent attachment order
	for _, colID := range agent.PromptCollections {
		colPrompts, err := promptRepo.ResolveCollectionPrompts(ctx, colID)
		if err != nil {
			return nil, err
		}

		for _, p := range colPrompts {
			if seen[p.ID] {
				continue
			}

			if p.Status != model.PromptStatusActive {
				log.Warn().Msgf("skipping non-active prompt %s in agent resolution", p.ID)
				continue
			}

			// Capture colID for closure/loop safety
			cID := colID

			seen[p.ID] = true
			resolved = append(resolved, &model.ResolvedAgentPrompt{
				ID:           p.ID,
				Name:         p.Name,
				Content:      p.Content,
				Status:       p.Status,
				Source:       "collection",
				CollectionID: &cID,
			})
		}
	}

	// 2. Process directly attached prompts in agent attachment order
	for _, pID := range agent.Prompts {
		if seen[pID] {
			continue
		}

		p, err := promptRepo.GetTemplateByID(ctx, pID)
		if err != nil {
			return nil, err
		}

		if p.Status != model.PromptStatusActive {
			log.Warn().Msgf("skipping non-active prompt %s in agent resolution", p.ID)
			continue
		}

		seen[pID] = true
		resolved = append(resolved, &model.ResolvedAgentPrompt{
			ID:           p.ID,
			Name:         p.Name,
			Content:      p.Content,
			Status:       p.Status,
			Source:       "individual",
			CollectionID: nil,
		})
	}

	return resolved, nil
}
