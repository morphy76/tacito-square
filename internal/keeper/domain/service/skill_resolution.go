package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/morphy76/tacito-square/internal/keeper/domain/model"
	"github.com/rs/zerolog/log"
)

// SkillRepository defines the outbound port interface required by the skill resolution domain service.
// This local definition adheres to hexagonal architecture guidelines by preventing the domain layer
// from importing application-layer packages.
type SkillRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*model.Skill, error)
	ResolveCollectionSkills(ctx context.Context, collectionID uuid.UUID) ([]*model.Skill, error)
}

// ResolveAgentSkills resolves the effective skill list for an agent, preserving order and deduplicating.
func ResolveAgentSkills(ctx context.Context, agent *model.Agent, skillRepo SkillRepository) ([]*model.ResolvedSkill, error) {
	var resolved []*model.ResolvedSkill
	seen := make(map[uuid.UUID]bool)

	// 1. Iterate agent.SkillCollections in slice order
	for _, colID := range agent.SkillCollections {
		colSkills, err := skillRepo.ResolveCollectionSkills(ctx, colID)
		if err != nil {
			return nil, err
		}

		for _, s := range colSkills {
			if seen[s.ID] {
				continue
			}

			if s.Status != model.SkillStatusActive {
				log.Warn().Msgf("skipping non-active skill %s in agent resolution", s.ID)
				continue
			}

			// Capture colID for closure/loop safety
			cID := colID

			seen[s.ID] = true
			resolved = append(resolved, &model.ResolvedSkill{
				ID:           s.ID,
				Name:         s.Name,
				Description:  s.Description,
				Content:      s.Content,
				Status:       s.Status,
				Source:       "collection",
				CollectionID: &cID,
			})
		}
	}

	// 2. Iterate agent.Skills in slice order
	for _, sID := range agent.Skills {
		if seen[sID] {
			continue
		}

		s, err := skillRepo.GetByID(ctx, sID)
		if err != nil {
			return nil, err
		}

		if s.Status != model.SkillStatusActive {
			log.Warn().Msgf("skipping non-active skill %s in agent resolution", s.ID)
			continue
		}

		seen[sID] = true
		resolved = append(resolved, &model.ResolvedSkill{
			ID:           s.ID,
			Name:         s.Name,
			Description:  s.Description,
			Content:      s.Content,
			Status:       s.Status,
			Source:       "individual",
			CollectionID: nil,
		})
	}

	return resolved, nil
}
