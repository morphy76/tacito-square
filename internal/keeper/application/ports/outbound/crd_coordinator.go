package outbound

import (
	"context"

	"github.com/google/uuid"
	"github.com/morphy76/tacito-square/internal/keeper/domain/model"
	"github.com/morphy76/tacito-square/pkg/kubernetes/apis/tacito/v1alpha1"
)

// CRDCoordinator defines the outbound port interface for submitting and deleting K8s CRDs.
type CRDCoordinator interface {
	SubmitAgentCRD(ctx context.Context, agent *model.Agent) error
	TeardownAgentCRD(ctx context.Context, agent *model.Agent) error
	GetAgentCRDStatus(ctx context.Context, agentID uuid.UUID) (*v1alpha1.TacitoAgentStatus, error)
}
