package service

import (
	"context"

	"github.com/morphy76/tacito-square/internal/operator/application/ports/inbound"
	"github.com/morphy76/tacito-square/pkg/kubernetes/apis/tacito/v1alpha1"
	"github.com/rs/zerolog"
)

// ReconcileAgentServiceImpl implements the ReconcileAgentService use case.
type ReconcileAgentServiceImpl struct {
	logger zerolog.Logger
}

var _ inbound.ReconcileAgentService = (*ReconcileAgentServiceImpl)(nil)

// NewReconcileAgentService constructs a new ReconcileAgentServiceImpl.
func NewReconcileAgentService(logger zerolog.Logger) *ReconcileAgentServiceImpl {
	return &ReconcileAgentServiceImpl{
		logger: logger,
	}
}

// Reconcile coordinates the reconciliation of a TacitoAgent.
func (s *ReconcileAgentServiceImpl) Reconcile(ctx context.Context, agent *v1alpha1.TacitoAgent) error {
	s.logger.Info().
		Str("namespace", agent.Namespace).
		Str("name", agent.Name).
		Str("tenant_id", agent.Spec.TenantID).
		Str("agent_name", agent.Spec.AgentName).
		Msg("reconciling tacito agent resource (stub)")
	return nil
}
