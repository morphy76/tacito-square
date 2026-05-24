package outbound

import (
	"context"

	"github.com/morphy76/tacito-square/internal/keeper/domain/model"
)

// CRDCoordinator defines the outbound port interface for submitting and deleting K8s CRDs.
type CRDCoordinator interface {
	SubmitAgentCRD(ctx context.Context, agent *model.Agent) error
	TeardownAgentCRD(ctx context.Context, agent *model.Agent) error
}
