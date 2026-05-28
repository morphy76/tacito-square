package inbound

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/morphy76/tacito-square/internal/keeper/domain/model"
)

type AgentStatusDetails struct {
	AgentID   uuid.UUID         `json:"agent_id"`
	Status    model.AgentStatus `json:"status"`
	Message   string            `json:"message"`
	Replicas  int32             `json:"replicas"`
	UpdatedAt time.Time         `json:"updated_at"`
}

type AgentDeploymentResult struct {
	AgentID uuid.UUID `json:"agent_id"`
	Status  string    `json:"status"`
	Error   string    `json:"error,omitempty"`
}

type CommunityDeploymentDetails struct {
	CommunityID uuid.UUID               `json:"community_id"`
	Status      string                  `json:"status"`
	Agents      []AgentDeploymentResult `json:"agents"`
}

type CommunityStatusDetails struct {
	CommunityID uuid.UUID             `json:"community_id"`
	Status      model.CommunityStatus `json:"status"`
	Agents      []AgentStatusDetails  `json:"agents"`
}

type LifecycleUseCase interface {
	DeployAgent(ctx context.Context, agentID uuid.UUID) error
	UndeployAgent(ctx context.Context, agentID uuid.UUID) error
	GetAgentStatus(ctx context.Context, agentID uuid.UUID) (*AgentStatusDetails, error)
	DeployCommunity(ctx context.Context, communityID uuid.UUID) (*CommunityDeploymentDetails, error)
	UndeployCommunity(ctx context.Context, communityID uuid.UUID) (*CommunityDeploymentDetails, error)
	GetCommunityStatus(ctx context.Context, communityID uuid.UUID) (*CommunityStatusDetails, error)
}
