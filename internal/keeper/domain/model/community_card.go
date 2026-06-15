package model

import "github.com/google/uuid"

// CommunityAgentSummary represents a lightweight description of a member agent in a CommunityCard.
type CommunityAgentSummary struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Version      string   `json:"version"`
	Description  string   `json:"description,omitempty"`
	Capabilities []string `json:"capabilities,omitempty"`
}

// CommunityCard represents the collective capabilities metadata of a community.
type CommunityCard struct {
	CommunityID uuid.UUID               `json:"community_id"`
	Name        string                  `json:"name"`
	Description string                  `json:"description,omitempty"`
	Topology    string                  `json:"topology"`
	Status      string                  `json:"status"`
	Agents      []CommunityAgentSummary `json:"agents,omitempty"`
}
